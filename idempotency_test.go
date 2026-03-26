package flowcore

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/hghukasyan/flowcore/store"
	redisstore "github.com/hghukasyan/flowcore/store/redis"
	goredis "github.com/redis/go-redis/v9"
)

// withEachIdempotencyBackend runs f with store.Memory and with Redis (miniredis).
func withEachIdempotencyBackend(t *testing.T, f func(t *testing.T, st store.Store)) {
	t.Helper()
	t.Run("memory", func(t *testing.T) {
		f(t, store.NewMemory())
	})
	t.Run("redis", func(t *testing.T) {
		mr, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { mr.Close() })
		cl := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
		t.Cleanup(func() { _ = cl.Close() })
		st, err := redisstore.New(cl)
		if err != nil {
			t.Fatal(err)
		}
		f(t, st)
	})
}

func TestIdempotencySecondRunSkipsSteps(t *testing.T) {
	withEachIdempotencyBackend(t, func(t *testing.T, st store.Store) {
		cfg := RunConfig{Store: st, Logger: nil}
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
	})
}

func TestIdempotencyFailedRunAllowsRetry(t *testing.T) {
	withEachIdempotencyBackend(t, func(t *testing.T, st store.Store) {
		cfg := RunConfig{Store: st, Logger: nil}
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
	})
}

func TestIdempotencyRunConfigOverridesWorkflow(t *testing.T) {
	withEachIdempotencyBackend(t, func(t *testing.T, st store.Store) {
		w := New(IdempotencyKey("ignored"))
		w.Step("x", func(*Context) error { return nil })
		cfg := RunConfig{Store: st, Logger: nil, IdempotencyKey: "real-key"}
		if err := RunWithConfig(context.Background(), w, cfg); err != nil {
			t.Fatal(err)
		}
		var n int
		w2 := New()
		w2.Step("x", func(*Context) error {
			n++
			return nil
		})
		cfg2 := RunConfig{Store: st, Logger: nil, IdempotencyKey: "real-key"}
		if err := RunWithConfig(context.Background(), w2, cfg2); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatalf("second workflow should skip: n=%d", n)
		}
	})
}
