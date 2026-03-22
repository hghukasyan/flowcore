package store

import (
	"context"
	"sync"
)

// Memory is an in-memory Store for development and embedded use.
type Memory struct {
	mu    sync.RWMutex
	flows map[string]*WorkflowState
}

// NewMemory returns a new empty memory store.
func NewMemory() *Memory {
	return &Memory{flows: make(map[string]*WorkflowState)}
}

func (m *Memory) PutWorkflow(ctx context.Context, id string, stepNames []string) error {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	steps := make(map[string]*StepState, len(stepNames))
	for _, n := range stepNames {
		steps[n] = &StepState{Name: n, Status: StatusPending, RetryCount: 0}
	}
	m.flows[id] = &WorkflowState{ID: id, Steps: steps}
	return nil
}

func (m *Memory) SetStep(ctx context.Context, workflowID, stepName string, status StepStatus, retryCount int) error {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	w, ok := m.flows[workflowID]
	if !ok {
		return ErrNotFound
	}
	s, ok := w.Steps[stepName]
	if !ok {
		return ErrNotFound
	}
	s.Status = status
	s.RetryCount = retryCount
	return nil
}

func (m *Memory) GetWorkflow(ctx context.Context, id string) (*WorkflowState, error) {
	_ = ctx
	m.mu.RLock()
	defer m.mu.RUnlock()
	w, ok := m.flows[id]
	if !ok {
		return nil, ErrNotFound
	}
	// Shallow copy for a stable snapshot.
	out := &WorkflowState{ID: w.ID, Steps: make(map[string]*StepState, len(w.Steps))}
	for k, v := range w.Steps {
		cp := *v
		out.Steps[k] = &cp
	}
	return out, nil
}
