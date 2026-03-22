package flowcore

import (
	"context"
)

// Workflow is an ordered collection of steps with optional dependencies.
type Workflow struct {
	steps          []*Step
	idempotencyKey string
}

// New builds an empty workflow. Pass [IdempotencyKey] and other options as needed.
func New(opts ...WorkflowOption) *Workflow {
	w := &Workflow{}
	for _, o := range opts {
		o(w)
	}
	return w
}

// Step registers a named step. Options configure retries, dependencies, timeouts, and compensation.
func (w *Workflow) Step(name string, fn StepFunc, opts ...StepOption) {
	st := &Step{Name: name, Run: fn, MaxAttempts: 1}
	for _, o := range opts {
		o(st)
	}
	w.steps = append(w.steps, st)
}

// Steps returns a defensive copy of registered steps for inspection or custom scheduling.
func (w *Workflow) Steps() []*Step {
	out := make([]*Step, len(w.steps))
	copy(out, w.steps)
	return out
}

// Run executes the workflow with a default in-memory store and console logger.
func (w *Workflow) Run(ctx context.Context) error {
	cfg := DefaultRunConfig()
	if w.idempotencyKey != "" {
		cfg.IdempotencyKey = w.idempotencyKey
	}
	return RunWithConfig(ctx, w, cfg)
}

// ExecutionLayers returns batches of steps that may run concurrently; order between layers is sequential.
func (w *Workflow) ExecutionLayers() ([][]*Step, error) {
	return computeLayers(w.steps)
}
