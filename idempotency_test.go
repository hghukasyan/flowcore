package flowcore

import (
	"context"
	"errors"
	"testing"

	"github.com/hghukasyan/flowcore/store"
)

func TestIdempotencySecondRunSkipsSteps(t *testing.T) {
	mem := store.NewMemory()
	cfg := RunConfig{Store: mem, Logger: nil}
	w := New(IdempotencyKey("order-42"))
	var calls int
	w.Step("pay", func(*Context) error {
		calls++
		return nil
	})
	if err := RunWithConfig(context.Background(), w, cfg); err != nil {
		t.Fatal(err)
	}
	if err := RunWithConfig(context.Background(), w, cfg); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("want step once, got %d", calls)
	}
}

func TestIdempotencyFailedRunAllowsRetry(t *testing.T) {
	mem := store.NewMemory()
	cfg := RunConfig{Store: mem, Logger: nil}
	w := New(IdempotencyKey("pay-1"))
	var calls int
	w.Step("pay", func(*Context) error {
		calls++
		if calls == 1 {
			return errors.New("declined")
		}
		return nil
	})
	if err := RunWithConfig(context.Background(), w, cfg); err == nil {
		t.Fatal("want first run error")
	}
	if err := RunWithConfig(context.Background(), w, cfg); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("want two attempts, got %d", calls)
	}
}

func TestIdempotencyRunConfigOverridesWorkflow(t *testing.T) {
	mem := store.NewMemory()
	w := New(IdempotencyKey("ignored"))
	w.Step("x", func(*Context) error { return nil })
	cfg := RunConfig{Store: mem, Logger: nil, IdempotencyKey: "real-key"}
	if err := RunWithConfig(context.Background(), w, cfg); err != nil {
		t.Fatal(err)
	}
	var n int
	w2 := New()
	w2.Step("x", func(*Context) error {
		n++
		return nil
	})
	cfg2 := RunConfig{Store: mem, Logger: nil, IdempotencyKey: "real-key"}
	if err := RunWithConfig(context.Background(), w2, cfg2); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("second workflow should skip: n=%d", n)
	}
}
