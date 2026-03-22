package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/hghukasyan/flowcore"
	"github.com/hghukasyan/flowcore/store"
)

func TestEngineRunSuccess(t *testing.T) {
	wf := flowcore.New()
	var ran bool
	wf.Step("only", func(*flowcore.Context) error {
		ran = true
		return nil
	})
	e := New()
	if err := e.Run(context.Background(), wf); err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Fatal("step did not run")
	}
}

func TestEngineRunError(t *testing.T) {
	wf := flowcore.New()
	wf.Step("bad", func(*flowcore.Context) error {
		return errors.New("boom")
	})
	err := New().Run(context.Background(), wf)
	if err == nil || err.Error() != "boom" {
		t.Fatalf("got %v", err)
	}
}

func TestEngineRunAsync(t *testing.T) {
	wf := flowcore.New()
	wf.Step("x", func(*flowcore.Context) error { return nil })
	ch := New().RunAsync(context.Background(), wf)
	if err := <-ch; err != nil {
		t.Fatal(err)
	}
}

type countingStore struct {
	*store.Memory
	putWorkflowCalls int
}

func (c *countingStore) PutWorkflow(ctx context.Context, id string, stepNames []string) error {
	c.putWorkflowCalls++
	return c.Memory.PutWorkflow(ctx, id, stepNames)
}

func TestEngineWithStoreUsesInjectedBackend(t *testing.T) {
	base := store.NewMemory()
	spy := &countingStore{Memory: base}
	e := New(WithStore(spy))
	wf := flowcore.New()
	wf.Step("a", func(*flowcore.Context) error { return nil })
	if err := e.Run(context.Background(), wf); err != nil {
		t.Fatal(err)
	}
	if spy.putWorkflowCalls != 1 {
		t.Fatalf("PutWorkflow calls = %d", spy.putWorkflowCalls)
	}
}
