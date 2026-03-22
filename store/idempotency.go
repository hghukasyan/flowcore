package store

import (
	"context"
	"errors"
)

// ErrIdempotencyInProgress means another run is already using this key and has not finished yet.
var ErrIdempotencyInProgress = errors.New("store: idempotency key already in use")

// IdempotencyStore records workflow-level idempotency so duplicate triggers (retries, double-clicks)
// can be skipped after a successful run, or rejected while a run is in flight.
type IdempotencyStore interface {
	// TryIdempotencyStart reserves the key for this workflowID when the workflow should execute.
	// If it returns shouldRun=false and err=nil, a previous run already completed successfully and
	// the caller must not execute the workflow (treat as success).
	// If shouldRun=false and err=ErrIdempotencyInProgress, another execution holds the key.
	// When shouldRun=true, the caller must eventually call FinishIdempotency with the same key.
	TryIdempotencyStart(ctx context.Context, key, workflowID string) (shouldRun bool, err error)
	// FinishIdempotency marks the key completed (success) or failed (allows a later retry with the same key).
	FinishIdempotency(ctx context.Context, key string, success bool) error
}
