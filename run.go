package flowcore

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/hghukasyan/flowcore/store"
)

// RunConfig controls persistence and logging for a single run.
type RunConfig struct {
	Store  store.Store
	Logger Logger
}

// DefaultRunConfig uses an in-memory store and [PrintLogger].
func DefaultRunConfig() RunConfig {
	return RunConfig{
		Store:  store.NewMemory(),
		Logger: PrintLogger{},
	}
}

// RunWithConfig runs a workflow with explicit store and logger. Nil store defaults to memory; nil logger skips hooks.
func RunWithConfig(ctx context.Context, w *Workflow, cfg RunConfig) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if cfg.Store == nil {
		cfg.Store = store.NewMemory()
	}
	steps := w.steps
	if len(steps) == 0 {
		return nil
	}
	layers, err := computeLayers(steps)
	if err != nil {
		return err
	}
	byName := make(map[string]*Step, len(steps))
	names := make([]string, 0, len(steps))
	for _, s := range steps {
		byName[s.Name] = s
		names = append(names, s.Name)
	}
	wfID, err := newWorkflowID()
	if err != nil {
		return err
	}
	if err := cfg.Store.PutWorkflow(ctx, wfID, names); err != nil {
		return err
	}

	wctx := NewContext(ctx)
	var succeeded []string
	var succMu sync.Mutex

	for _, layer := range layers {
		layerCtx, cancel := context.WithCancel(ctx)
		var wg sync.WaitGroup
		var errOnce sync.Once
		var runErr error
		for _, st := range layer {
			wg.Add(1)
			go func(s *Step) {
				defer wg.Done()
				if layerCtx.Err() != nil {
					return
				}
				e := executeOne(layerCtx, wfID, s, wctx, cfg)
				if e != nil {
					errOnce.Do(func() {
						runErr = e
						cancel()
					})
					return
				}
				succMu.Lock()
				succeeded = append(succeeded, s.Name)
				succMu.Unlock()
			}(st)
		}
		wg.Wait()
		cancel()
		if runErr != nil {
			runCompensations(ctx, wfID, wctx, cfg, succeeded, byName)
			return runErr
		}
	}
	return nil
}

func executeOne(goCtx context.Context, wfID string, s *Step, wctx *Context, cfg RunConfig) error {
	attempts := s.attempts()
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			if err := backoffSleep(goCtx, s, attempt); err != nil {
				return err
			}
		}
		if goCtx.Err() != nil {
			return goCtx.Err()
		}
		if cfg.Store != nil {
			_ = cfg.Store.SetStep(goCtx, wfID, s.Name, store.StatusRunning, attempt)
		}
		if cfg.Logger != nil {
			cfg.Logger.StepStarted(wfID, s.Name)
		}

		runGoCtx := goCtx
		var attemptCancel context.CancelFunc
		if s.Timeout > 0 {
			runGoCtx, attemptCancel = context.WithTimeout(goCtx, s.Timeout)
		}
		stepCtx := wctx.branch(runGoCtx)
		err := s.Run(stepCtx)
		if attemptCancel != nil {
			attemptCancel()
		}

		if err == nil {
			if cfg.Store != nil {
				_ = cfg.Store.SetStep(goCtx, wfID, s.Name, store.StatusDone, attempt)
			}
			if cfg.Logger != nil {
				cfg.Logger.StepSucceeded(wfID, s.Name)
			}
			return nil
		}
		lastErr = err
		if cfg.Logger != nil {
			cfg.Logger.StepFailed(wfID, s.Name, err)
		}
		if attempt+1 >= attempts {
			if cfg.Store != nil {
				_ = cfg.Store.SetStep(goCtx, wfID, s.Name, store.StatusFailed, attempt)
			}
			return err
		}
	}
	return lastErr
}

func backoffSleep(ctx context.Context, s *Step, afterFailureIndex int) error {
	var d time.Duration
	switch s.Backoff.Kind {
	case BackoffFixed:
		d = s.Backoff.BaseDelay
	case BackoffExponential:
		mult := s.backoffMultiplier()
		// afterFailureIndex is the Run attempt index (1 = first retry); first wait is BaseDelay.
		f := 1.0
		for i := 1; i < afterFailureIndex; i++ {
			f *= mult
		}
		d = time.Duration(float64(s.Backoff.BaseDelay) * f)
	default:
		return nil
	}
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func runCompensations(ctx context.Context, wfID string, wctx *Context, cfg RunConfig, succeeded []string, byName map[string]*Step) {
	for i := len(succeeded) - 1; i >= 0; i-- {
		name := succeeded[i]
		st := byName[name]
		if st == nil || st.Compensate == nil {
			continue
		}
		if cfg.Logger != nil {
			cfg.Logger.StepStarted(wfID, name+":compensate")
		}
		err := st.Compensate(wctx.branch(ctx))
		if err != nil {
			if cfg.Logger != nil {
				cfg.Logger.StepFailed(wfID, name+":compensate", err)
			}
			continue
		}
		if cfg.Logger != nil {
			cfg.Logger.StepSucceeded(wfID, name+":compensate")
		}
	}
}

func computeLayers(steps []*Step) ([][]*Step, error) {
	if len(steps) == 0 {
		return nil, nil
	}
	byName := make(map[string]*Step, len(steps))
	for _, s := range steps {
		if _, dup := byName[s.Name]; dup {
			return nil, fmt.Errorf("flowcore: duplicate step name %q", s.Name)
		}
		byName[s.Name] = s
	}
	inDegree := make(map[string]int, len(steps))
	successors := make(map[string][]string)
	for _, s := range steps {
		inDegree[s.Name] = len(s.DependsOn)
		for _, d := range s.DependsOn {
			if _, ok := byName[d]; !ok {
				return nil, fmt.Errorf("flowcore: step %q depends on unknown step %q", s.Name, d)
			}
			successors[d] = append(successors[d], s.Name)
		}
	}
	var layers [][]*Step
	var current []*Step
	for _, s := range steps {
		if inDegree[s.Name] == 0 {
			current = append(current, s)
		}
	}
	sort.Slice(current, func(i, j int) bool { return current[i].Name < current[j].Name })
	seen := 0
	for len(current) > 0 {
		layer := make([]*Step, len(current))
		copy(layer, current)
		layers = append(layers, layer)
		seen += len(current)
		nextSet := make(map[string]*Step)
		for _, s := range current {
			for _, succ := range successors[s.Name] {
				inDegree[succ]--
				if inDegree[succ] == 0 {
					nextSet[succ] = byName[succ]
				}
			}
		}
		current = current[:0]
		for _, s := range nextSet {
			current = append(current, s)
		}
		sort.Slice(current, func(i, j int) bool { return current[i].Name < current[j].Name })
	}
	if seen != len(steps) {
		return nil, errors.New("flowcore: dependency cycle or unreachable steps")
	}
	return layers, nil
}

func newWorkflowID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
