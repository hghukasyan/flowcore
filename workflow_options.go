package flowcore

// WorkflowOption configures a workflow when created with [New].
type WorkflowOption func(*Workflow)

// IdempotencyKey ties runs of this workflow to a stable business identifier (for example an order id).
// When the store implements [store.IdempotencyStore], a second successful run with the same key is skipped
// (returns nil without re-executing steps). A failed run releases the key so you can retry.
// Use [RunConfig.IdempotencyKey] to override per run.
func IdempotencyKey(key string) WorkflowOption {
	return func(w *Workflow) {
		w.idempotencyKey = key
	}
}
