package flowcore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hghukasyan/flowcore/store"
)

func TestRunEmptyWorkflow(t *testing.T) {
	w := New()
	if err := RunWithConfig(context.Background(), w, RunConfig{}); err != nil {
		t.Fatal(err)
	}
}

func TestComputeLayersDuplicateStepName(t *testing.T) {
	w := New()
	w.Step("x", func(*Context) error { return nil })
	w.Step("x", func(*Context) error { return nil })
	_, err := computeLayers(w.steps)
	if err == nil {
		t.Fatal("expected error for duplicate name")
	}
}

func TestComputeLayersUnknownDependency(t *testing.T) {
	w := New()
	w.Step("a", func(*Context) error { return nil }, DependsOn("ghost"))
	_, err := computeLayers(w.steps)
	if err == nil {
		t.Fatal("expected error for unknown dependency")
	}
}

func TestComputeLayersCycle(t *testing.T) {
	w := New()
	w.Step("a", func(*Context) error { return nil }, DependsOn("b"))
	w.Step("b", func(*Context) error { return nil }, DependsOn("a"))
	_, err := computeLayers(w.steps)
	if err == nil {
		t.Fatal("expected error for cycle")
	}
}

func TestWorkflowExecutionLayers(t *testing.T) {
	w := New()
	w.Step("a", func(*Context) error { return nil })
	w.Step("b", func(*Context) error { return nil }, DependsOn("a"))
	ls, err := w.ExecutionLayers()
	if err != nil {
		t.Fatal(err)
	}
	if len(ls) != 2 || len(ls[0]) != 1 || ls[0][0].Name != "a" {
		t.Fatalf("layer0: %v", layerNames(ls))
	}
}

func TestRunRetryThenSuccess(t *testing.T) {
	var calls int
	w := New()
	w.Step("flaky", func(*Context) error {
		calls++
		if calls < 2 {
			return errors.New("temporary")
		}
		return nil
	}, RetryWithBackoff(3, Backoff{Kind: BackoffFixed, BaseDelay: time.Millisecond}))

	err := RunWithConfig(context.Background(), w, RunConfig{Logger: nil, Store: nil})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("want 2 calls, got %d", calls)
	}
}

func TestRunCompensationReverseOrder(t *testing.T) {
	var comp []string
	w := New()
	w.Step("a", func(*Context) error { return nil },
		WithCompensation(func(*Context) error {
			comp = append(comp, "a")
			return nil
		}))
	w.Step("b", func(*Context) error { return nil },
		WithCompensation(func(*Context) error {
			comp = append(comp, "b")
			return nil
		}),
		DependsOn("a"))
	w.Step("c", func(*Context) error { return errors.New("fail") }, DependsOn("b"))

	err := RunWithConfig(context.Background(), w, RunConfig{Logger: nil, Store: nil})
	if err == nil || err.Error() != "fail" {
		t.Fatalf("err = %v", err)
	}
	if len(comp) != 2 || comp[0] != "b" || comp[1] != "a" {
		t.Fatalf("compensation order: %v", comp)
	}
}

func TestRunCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w := New()
	w.Step("x", func(*Context) error { return nil })
	err := RunWithConfig(ctx, w, RunConfig{Logger: nil, Store: nil})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}

type captureWorkflowID struct {
	*store.Memory
	id string
}

func (c *captureWorkflowID) PutWorkflow(ctx context.Context, id string, stepNames []string) error {
	c.id = id
	return c.Memory.PutWorkflow(ctx, id, stepNames)
}

func TestRunUpdatesStoreStepStatus(t *testing.T) {
	mem := store.NewMemory()
	spy := &captureWorkflowID{Memory: mem}
	w := New()
	w.Step("only", func(*Context) error { return nil })

	err := RunWithConfig(context.Background(), w, RunConfig{Store: spy, Logger: nil})
	if err != nil {
		t.Fatal(err)
	}
	snap, err := mem.GetWorkflow(context.Background(), spy.id)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Steps["only"].Status != store.StatusDone {
		t.Fatalf("status = %q", snap.Steps["only"].Status)
	}
}
