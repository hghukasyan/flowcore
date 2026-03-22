package flowcore

import (
	"context"
	"testing"
)

func TestContextSetGet(t *testing.T) {
	c := NewContext(context.Background())
	c.Set("k", 42)
	if v := c.Get("k"); v != 42 {
		t.Fatalf("Get: got %v", v)
	}
	if c.Get("missing") != nil {
		t.Fatalf("missing: got %v", c.Get("missing"))
	}
}

func TestContextMustGet(t *testing.T) {
	c := NewContext(context.Background())
	c.Set("ok", "yes")
	if c.MustGet("ok") != "yes" {
		t.Fatal("MustGet ok")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on missing key")
		}
	}()
	_ = c.MustGet("nope")
}

func TestContextBranchSharesKV(t *testing.T) {
	parent := NewContext(context.Background())
	parent.Set("x", 1)
	child := parent.branch(context.Background())
	child.Set("y", 2)
	if parent.Get("y") != 2 {
		t.Fatal("branch write should be visible on parent kv store")
	}
}
