package redis

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/hghukasyan/flowcore/store"
	goredis "github.com/redis/go-redis/v9"
)

func newTestStore(t *testing.T) (*Store, func()) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	cl := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	s, err := New(cl)
	if err != nil {
		t.Fatal(err)
	}
	cleanup := func() {
		_ = cl.Close()
		mr.Close()
	}
	return s, cleanup
}

func TestRedisPutGetWorkflow(t *testing.T) {
	ctx := context.Background()
	s, cleanup := newTestStore(t)
	defer cleanup()

	if err := s.PutWorkflow(ctx, "wf1", []string{"a", "b"}); err != nil {
		t.Fatal(err)
	}
	w, err := s.GetWorkflow(ctx, "wf1")
	if err != nil {
		t.Fatal(err)
	}
	if w.ID != "wf1" {
		t.Fatalf("ID: got %q", w.ID)
	}
	if w.Steps["a"].Status != store.StatusPending || w.Steps["b"].Status != store.StatusPending {
		t.Fatalf("steps: %+v", w.Steps)
	}
}

func TestRedisGetWorkflowNotFound(t *testing.T) {
	s, cleanup := newTestStore(t)
	defer cleanup()
	_, err := s.GetWorkflow(context.Background(), "missing")
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestRedisSetStep(t *testing.T) {
	ctx := context.Background()
	s, cleanup := newTestStore(t)
	defer cleanup()
	_ = s.PutWorkflow(ctx, "wf1", []string{"x"})
	if err := s.SetStep(ctx, "wf1", "x", store.StatusRunning, 0); err != nil {
		t.Fatal(err)
	}
	w, err := s.GetWorkflow(ctx, "wf1")
	if err != nil {
		t.Fatal(err)
	}
	if w.Steps["x"].Status != store.StatusRunning || w.Steps["x"].RetryCount != 0 {
		t.Fatalf("got %+v", w.Steps["x"])
	}
	if err := s.SetStep(ctx, "wf1", "x", store.StatusDone, 2); err != nil {
		t.Fatal(err)
	}
	w2, _ := s.GetWorkflow(ctx, "wf1")
	if w2.Steps["x"].Status != store.StatusDone || w2.Steps["x"].RetryCount != 2 {
		t.Fatalf("after done: %+v", w2.Steps["x"])
	}
}

func TestRedisSetStepNotFound(t *testing.T) {
	ctx := context.Background()
	s, cleanup := newTestStore(t)
	defer cleanup()
	if err := s.SetStep(ctx, "nope", "x", store.StatusDone, 0); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("unknown workflow: got %v", err)
	}
	_ = s.PutWorkflow(ctx, "wf1", []string{"a"})
	if err := s.SetStep(ctx, "wf1", "missing", store.StatusDone, 0); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("unknown step: got %v", err)
	}
}

func TestRedisGetWorkflowSnapshot(t *testing.T) {
	ctx := context.Background()
	s, cleanup := newTestStore(t)
	defer cleanup()
	_ = s.PutWorkflow(ctx, "wf1", []string{"a"})
	snap, _ := s.GetWorkflow(ctx, "wf1")
	_ = s.SetStep(ctx, "wf1", "a", store.StatusFailed, 3)
	if snap.Steps["a"].Status != store.StatusPending {
		t.Fatal("snapshot should not mutate when store updates")
	}
}

func TestRedisIdempotencySuccessSkipsSecond(t *testing.T) {
	ctx := context.Background()
	s, cleanup := newTestStore(t)
	defer cleanup()
	const key = "order-123"
	ok, err := s.TryIdempotencyStart(ctx, key, "wf-a")
	if err != nil || !ok {
		t.Fatalf("first claim: ok=%v err=%v", ok, err)
	}
	if err := s.FinishIdempotency(ctx, key, true); err != nil {
		t.Fatal(err)
	}
	ok2, err := s.TryIdempotencyStart(ctx, key, "wf-b")
	if err != nil || ok2 {
		t.Fatalf("second after success: shouldRun=%v err=%v (want false, nil)", ok2, err)
	}
}

func TestRedisIdempotencyFailedAllowsRetry(t *testing.T) {
	ctx := context.Background()
	s, cleanup := newTestStore(t)
	defer cleanup()
	const key = "k"
	if ok, err := s.TryIdempotencyStart(ctx, key, "wf1"); !ok || err != nil {
		t.Fatal(err)
	}
	if err := s.FinishIdempotency(ctx, key, false); err != nil {
		t.Fatal(err)
	}
	ok, err := s.TryIdempotencyStart(ctx, key, "wf2")
	if err != nil || !ok {
		t.Fatalf("retry after fail: ok=%v err=%v", ok, err)
	}
}

func TestRedisIdempotencyInProgress(t *testing.T) {
	ctx := context.Background()
	s, cleanup := newTestStore(t)
	defer cleanup()
	const key = "k"
	if ok, err := s.TryIdempotencyStart(ctx, key, "wf1"); !ok || err != nil {
		t.Fatal(err)
	}
	ok2, err := s.TryIdempotencyStart(ctx, key, "wf2")
	if ok2 || !errors.Is(err, store.ErrIdempotencyInProgress) {
		t.Fatalf("want ErrIdempotencyInProgress, ok=%v err=%v", ok2, err)
	}
}

func TestRedisConcurrentIdempotencyStart(t *testing.T) {
	ctx := context.Background()
	s, cleanup := newTestStore(t)
	defer cleanup()
	const key = "same"
	var wg sync.WaitGroup
	var wins int
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := s.TryIdempotencyStart(ctx, key, "wf")
			if ok && err == nil {
				wins++
				_ = s.FinishIdempotency(ctx, key, true)
			}
		}()
	}
	wg.Wait()
	if wins != 1 {
		t.Fatalf("want exactly one winning start, got %d", wins)
	}
}

func TestNewNilClient(t *testing.T) {
	_, err := New(nil)
	if err == nil {
		t.Fatal("want error")
	}
}
