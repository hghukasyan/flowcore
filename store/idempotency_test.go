package store

import (
	"context"
	"errors"
	"testing"
)

func TestMemoryIdempotencySuccessSkipsSecond(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	const key = "order-123"
	ok, err := m.TryIdempotencyStart(ctx, key, "wf-a")
	if err != nil || !ok {
		t.Fatalf("first claim: ok=%v err=%v", ok, err)
	}
	if err := m.FinishIdempotency(ctx, key, true); err != nil {
		t.Fatal(err)
	}
	ok2, err := m.TryIdempotencyStart(ctx, key, "wf-b")
	if err != nil || ok2 {
		t.Fatalf("second after success: shouldRun=%v err=%v (want false, nil)", ok2, err)
	}
}

func TestMemoryIdempotencyFailedAllowsRetry(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	const key = "k"
	if ok, err := m.TryIdempotencyStart(ctx, key, "wf1"); !ok || err != nil {
		t.Fatal(err)
	}
	if err := m.FinishIdempotency(ctx, key, false); err != nil {
		t.Fatal(err)
	}
	ok, err := m.TryIdempotencyStart(ctx, key, "wf2")
	if err != nil || !ok {
		t.Fatalf("retry after fail: ok=%v err=%v", ok, err)
	}
}

func TestMemoryIdempotencyInProgress(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	const key = "k"
	if ok, err := m.TryIdempotencyStart(ctx, key, "wf1"); !ok || err != nil {
		t.Fatal(err)
	}
	ok2, err := m.TryIdempotencyStart(ctx, key, "wf2")
	if ok2 || !errors.Is(err, ErrIdempotencyInProgress) {
		t.Fatalf("want ErrIdempotencyInProgress, ok=%v err=%v", ok2, err)
	}
}
