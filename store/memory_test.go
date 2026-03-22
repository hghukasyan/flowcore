package store

import (
	"context"
	"errors"
	"testing"
)

func TestMemoryPutGetWorkflow(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	if err := m.PutWorkflow(ctx, "wf1", []string{"a", "b"}); err != nil {
		t.Fatal(err)
	}
	w, err := m.GetWorkflow(ctx, "wf1")
	if err != nil {
		t.Fatal(err)
	}
	if w.ID != "wf1" {
		t.Fatalf("ID: got %q", w.ID)
	}
	if w.Steps["a"].Status != StatusPending || w.Steps["b"].Status != StatusPending {
		t.Fatalf("steps: %+v", w.Steps)
	}
}

func TestMemoryGetWorkflowNotFound(t *testing.T) {
	_, err := NewMemory().GetWorkflow(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestMemorySetStep(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	_ = m.PutWorkflow(ctx, "wf1", []string{"x"})
	if err := m.SetStep(ctx, "wf1", "x", StatusRunning, 0); err != nil {
		t.Fatal(err)
	}
	w, err := m.GetWorkflow(ctx, "wf1")
	if err != nil {
		t.Fatal(err)
	}
	if w.Steps["x"].Status != StatusRunning || w.Steps["x"].RetryCount != 0 {
		t.Fatalf("got %+v", w.Steps["x"])
	}
	if err := m.SetStep(ctx, "wf1", "x", StatusDone, 2); err != nil {
		t.Fatal(err)
	}
	w2, _ := m.GetWorkflow(ctx, "wf1")
	if w2.Steps["x"].Status != StatusDone || w2.Steps["x"].RetryCount != 2 {
		t.Fatalf("after done: %+v", w2.Steps["x"])
	}
}

func TestMemorySetStepNotFound(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	if err := m.SetStep(ctx, "nope", "x", StatusDone, 0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown workflow: got %v", err)
	}
	_ = m.PutWorkflow(ctx, "wf1", []string{"a"})
	if err := m.SetStep(ctx, "wf1", "missing", StatusDone, 0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown step: got %v", err)
	}
}

func TestMemoryGetWorkflowSnapshot(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	_ = m.PutWorkflow(ctx, "wf1", []string{"a"})
	snap, _ := m.GetWorkflow(ctx, "wf1")
	_ = m.SetStep(ctx, "wf1", "a", StatusFailed, 3)
	if snap.Steps["a"].Status != StatusPending {
		t.Fatal("snapshot should not mutate when store updates")
	}
}
