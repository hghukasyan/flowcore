// Package redis provides a Redis-backed [store.Store] and [store.IdempotencyStore].
// Pass a [github.com/redis/go-redis/v9] client you already configure (pool, TLS, ACLs).
package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hghukasyan/flowcore/store"
	"github.com/redis/go-redis/v9"
)

// Store persists workflow and idempotency state in Redis.
type Store struct {
	r      redis.UniversalClient
	prefix string
}

// Option configures [New].
type Option func(*Store)

// WithPrefix sets a key namespace (default "flowcore"). Use this when several apps share one Redis.
func WithPrefix(prefix string) Option {
	return func(s *Store) {
		if strings.TrimSpace(prefix) != "" {
			s.prefix = strings.TrimSpace(prefix)
		}
	}
}

// New builds a store. The client must outlive the store; [Store] does not close it.
func New(c redis.UniversalClient, opts ...Option) (*Store, error) {
	if c == nil {
		return nil, errors.New("redis store: nil client")
	}
	s := &Store{r: c, prefix: "flowcore"}
	for _, o := range opts {
		o(s)
	}
	return s, nil
}

func (s *Store) wfMarkKey(workflowID string) string {
	return fmt.Sprintf("%s:w:%s", s.prefix, workflowID)
}

func (s *Store) stepsKey(workflowID string) string {
	return fmt.Sprintf("%s:w:%s:steps", s.prefix, workflowID)
}

func (s *Store) idemRedisKey(idempotencyKey string) string {
	h := sha256.Sum256([]byte(idempotencyKey))
	return fmt.Sprintf("%s:idem:%s", s.prefix, hex.EncodeToString(h[:]))
}

func encodeStep(status store.StepStatus, retry int) string {
	return fmt.Sprintf("%s|%d", status, retry)
}

func decodeStep(field, raw string) (*store.StepState, error) {
	parts := strings.SplitN(raw, "|", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("redis store: bad step value for %q", field)
	}
	rc, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}
	return &store.StepState{
		Name:       field,
		Status:     store.StepStatus(parts[0]),
		RetryCount: rc,
	}, nil
}

const tryIdemLua = `
if redis.call('EXISTS', KEYS[1]) == 0 then
  redis.call('HSET', KEYS[1], 'state', 'running', 'workflow_id', ARGV[1], 'updated_at', ARGV[2])
  return 'RUN'
end
local st = redis.call('HGET', KEYS[1], 'state')
if st == false then
  redis.call('HSET', KEYS[1], 'state', 'running', 'workflow_id', ARGV[1], 'updated_at', ARGV[2])
  return 'RUN'
end
if st == 'completed' then return 'SKIP' end
if st == 'running' then return 'BUSY' end
if st == 'failed' then
  redis.call('HSET', KEYS[1], 'state', 'running', 'workflow_id', ARGV[1], 'updated_at', ARGV[2])
  return 'RUN'
end
return 'BUSY'
`

const finishIdemLua = `
if redis.call('EXISTS', KEYS[1]) == 0 then return 'OK' end
local state = 'failed'
if ARGV[1] == '1' then state = 'completed' end
redis.call('HSET', KEYS[1], 'state', state, 'updated_at', ARGV[2])
return 'OK'
`

// TryIdempotencyStart implements [store.IdempotencyStore].
func (s *Store) TryIdempotencyStart(ctx context.Context, key, workflowID string) (bool, error) {
	if strings.TrimSpace(key) == "" {
		return true, nil
	}
	key = strings.TrimSpace(key)
	rk := s.idemRedisKey(key)
	now := strconv.FormatInt(time.Now().Unix(), 10)

	v, err := s.r.Eval(ctx, tryIdemLua, []string{rk}, workflowID, now).Result()
	if err != nil {
		return false, err
	}
	code, ok := v.(string)
	if !ok {
		return false, fmt.Errorf("redis store: unexpected idempotency script reply %T", v)
	}
	switch code {
	case "RUN":
		return true, nil
	case "SKIP":
		return false, nil
	case "BUSY":
		return false, store.ErrIdempotencyInProgress
	default:
		return false, fmt.Errorf("redis store: unknown idempotency code %q", code)
	}
}

// FinishIdempotency implements [store.IdempotencyStore].
func (s *Store) FinishIdempotency(ctx context.Context, key string, success bool) error {
	if strings.TrimSpace(key) == "" {
		return nil
	}
	rk := s.idemRedisKey(strings.TrimSpace(key))
	now := strconv.FormatInt(time.Now().Unix(), 10)
	flag := "0"
	if success {
		flag = "1"
	}
	_, err := s.r.Eval(ctx, finishIdemLua, []string{rk}, flag, now).Result()
	return err
}

// PutWorkflow implements [store.Store].
func (s *Store) PutWorkflow(ctx context.Context, id string, stepNames []string) error {
	pipe := s.r.Pipeline()
	pipe.Del(ctx, s.stepsKey(id))
	pipe.Set(ctx, s.wfMarkKey(id), "1", 0)
	for _, n := range stepNames {
		pipe.HSet(ctx, s.stepsKey(id), n, encodeStep(store.StatusPending, 0))
	}
	_, err := pipe.Exec(ctx)
	return err
}

// SetStep implements [store.Store].
func (s *Store) SetStep(ctx context.Context, workflowID, stepName string, status store.StepStatus, retryCount int) error {
	n, err := s.r.Exists(ctx, s.wfMarkKey(workflowID)).Result()
	if err != nil {
		return err
	}
	if n == 0 {
		return store.ErrNotFound
	}
	_, err = s.r.HGet(ctx, s.stepsKey(workflowID), stepName).Result()
	if err == redis.Nil {
		return store.ErrNotFound
	}
	if err != nil {
		return err
	}
	return s.r.HSet(ctx, s.stepsKey(workflowID), stepName, encodeStep(status, retryCount)).Err()
}

// GetWorkflow implements [store.Store].
func (s *Store) GetWorkflow(ctx context.Context, id string) (*store.WorkflowState, error) {
	n, err := s.r.Exists(ctx, s.wfMarkKey(id)).Result()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, store.ErrNotFound
	}
	m, err := s.r.HGetAll(ctx, s.stepsKey(id)).Result()
	if err != nil {
		return nil, err
	}
	steps := make(map[string]*store.StepState, len(m))
	for name, raw := range m {
		st, err := decodeStep(name, raw)
		if err != nil {
			return nil, err
		}
		steps[name] = st
	}
	return &store.WorkflowState{ID: id, Steps: steps}, nil
}
