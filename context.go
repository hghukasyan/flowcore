package flowcore

import (
	"context"
	"sync"
)

type kvStore struct {
	mu sync.RWMutex
	m  map[string]any
}

// Context carries workflow-scoped key/value data and the underlying [context.Context].
type Context struct {
	ctx context.Context
	kv  *kvStore
}

// NewContext wraps a standard context for step execution.
func NewContext(ctx context.Context) *Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Context{ctx: ctx, kv: &kvStore{m: make(map[string]any)}}
}

func (c *Context) branch(goCtx context.Context) *Context {
	return &Context{ctx: goCtx, kv: c.kv}
}

// GoContext returns the underlying context for deadlines and cancellation.
func (c *Context) GoContext() context.Context {
	return c.ctx
}

// Set stores a value by key. Safe for concurrent use when steps share the same workflow context.
func (c *Context) Set(key string, value any) {
	c.kv.mu.Lock()
	defer c.kv.mu.Unlock()
	c.kv.m[key] = value
}

// Get returns a value or nil if missing.
func (c *Context) Get(key string) any {
	c.kv.mu.RLock()
	defer c.kv.mu.RUnlock()
	return c.kv.m[key]
}

// MustGet returns a value or panics if the key is missing.
func (c *Context) MustGet(key string) any {
	c.kv.mu.RLock()
	v, ok := c.kv.m[key]
	c.kv.mu.RUnlock()
	if !ok {
		panic("flowcore: missing key " + key)
	}
	return v
}
