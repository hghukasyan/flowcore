package engine

import (
	"context"

	"github.com/hghukasyan/flowcore"
	"github.com/hghukasyan/flowcore/store"
)

// Engine runs workflows with pluggable storage and logging.
type Engine struct {
	store  store.Store
	logger flowcore.Logger
}

// Option configures an [Engine].
type Option func(*Engine)

// WithStore sets persistence for workflow and step state.
func WithStore(s store.Store) Option {
	return func(e *Engine) {
		e.store = s
	}
}

// WithLogger sets optional lifecycle hooks. Pass nil to disable logging.
func WithLogger(l flowcore.Logger) Option {
	return func(e *Engine) {
		e.logger = l
	}
}

// New builds an engine. Defaults: in-memory store and no logger unless [WithLogger] is used.
func New(opts ...Option) *Engine {
	e := &Engine{store: store.NewMemory()}
	for _, o := range opts {
		o(e)
	}
	return e
}

func (e *Engine) runConfig() flowcore.RunConfig {
	return flowcore.RunConfig{
		Store:  e.store,
		Logger: e.logger,
	}
}

// Run executes the workflow synchronously.
func (e *Engine) Run(ctx context.Context, wf *flowcore.Workflow) error {
	return flowcore.RunWithConfig(ctx, wf, e.runConfig())
}

// RunAsync starts a run in a new goroutine. The returned channel receives exactly one result.
func (e *Engine) RunAsync(ctx context.Context, wf *flowcore.Workflow) <-chan error {
	ch := make(chan error, 1)
	go func() {
		ch <- e.Run(ctx, wf)
	}()
	return ch
}
