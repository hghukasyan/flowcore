package flowcore

import "github.com/hghukasyan/flowcore/store"

// ErrIdempotencyInProgress is returned when a second run uses the same idempotency key while the first is still active.
var ErrIdempotencyInProgress = store.ErrIdempotencyInProgress
