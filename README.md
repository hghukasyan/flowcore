<div align="center">

# Flowcore

### Workflows in Go — without the platform tax

**Compose steps. Add dependencies. Run in parallel. Retry, compensate, and stay idempotent — all inside your process.**

[![Go](https://img.shields.io/github/go-mod/go-version/hghukasyan/flowcore?logo=go&label=Go)](go.mod)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/hghukasyan/flowcore.svg)](https://pkg.go.dev/github.com/hghukasyan/flowcore)

[Repository](https://github.com/hghukasyan/flowcore) · [Examples](examples/) · [Contributing](CONTRIBUTING.md)

</div>

---

Flowcore is a **small, embeddable workflow library** for Go. You describe what should happen step by step; the library handles ordering, concurrency where it is safe, retries, saga-style rollbacks, and optional idempotency keys for real-world APIs and payments.

No workers to deploy. No broker to babysit. **Standard library only** — zero third-party dependencies.

> *“I want Temporal’s ideas — dependencies, sagas, retries — but I’m shipping a service or a batch job, not a second infrastructure stack.”*

---

## Why teams reach for Flowcore

| You need… | Flowcore gives you… |
|-----------|---------------------|
| Clear multi-step flows in code | Named steps + `DependsOn` + automatic parallel layers |
| Safer money or inventory paths | `WithCompensation` in **reverse** order on failure |
| Production-ish resilience | Retries, fixed / exponential backoff, per-step timeouts |
| Duplicate-safe HTTP or jobs | `IdempotencyKey` when your store supports it |
| Something you can read in an afternoon | A compact codebase you can fork or extend |

---

## Features at a glance

**Core**

- Plain Go functions: `func(ctx *flowcore.Context) error`
- Shared, thread-safe context for passing data between steps
- Dependencies so steps wait for the right predecessors
- Independent steps run **in parallel** automatically

**Reliability**

- Configurable retries + backoff
- Saga-style **compensation** after failures
- Optional **idempotency keys** for “run once per business id” semantics

**Operations**

- Sync `Run` or async `RunAsync` via the `engine` package
- Lifecycle hooks with a small `Logger` interface
- Pluggable `Store` (memory today; Redis / SQL on the roadmap)

---

## Installation

```bash
go get github.com/hghukasyan/flowcore
```

Clone for examples and development:

```bash
git clone https://github.com/hghukasyan/flowcore.git
cd flowcore
```

Import the root package for workflows; use `github.com/hghukasyan/flowcore/engine` and `.../store` when you need them.

---

## Quick start

```go
package main

import (
	"context"
	"fmt"

	"github.com/hghukasyan/flowcore"
)

func main() {
	wf := flowcore.New()

	wf.Step("create_order", func(ctx *flowcore.Context) error {
		ctx.Set("id", "1001")
		return nil
	})

	wf.Step("charge", func(ctx *flowcore.Context) error {
		fmt.Println("charge", ctx.Get("id"))
		return nil
	}, flowcore.DependsOn("create_order"))

	if err := wf.Run(context.Background()); err != nil {
		panic(err)
	}
}
```

`wf.Run` uses an in-memory store and prints step lifecycle lines. For silent runs:

```go
flowcore.RunWithConfig(ctx, wf, flowcore.RunConfig{Logger: nil})
```

Runnable copy: `examples/basic` → `go run ./examples/basic`

---

## Advanced: retries, backoff, compensation

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/hghukasyan/flowcore"
)

func main() {
	wf := flowcore.New()

	wf.Step("book", func(ctx *flowcore.Context) error {
		fmt.Println("book seat")
		return nil
	}, flowcore.WithCompensation(func(ctx *flowcore.Context) error {
		fmt.Println("cancel booking")
		return nil
	}))

	wf.Step("pay", func(ctx *flowcore.Context) error {
		return fmt.Errorf("card error")
	},
		flowcore.DependsOn("book"),
		flowcore.RetryWithBackoff(3, flowcore.Backoff{
			Kind:      flowcore.BackoffExponential,
			BaseDelay: 50 * time.Millisecond,
		}),
		flowcore.WithCompensation(func(ctx *flowcore.Context) error {
			fmt.Println("refund")
			return nil
		}),
	)

	_ = wf.Run(context.Background())
}
```

Saga demo: `go run ./examples/saga`

---

## Idempotency (payments, webhooks, retried requests)

```go
wf := flowcore.New(flowcore.IdempotencyKey("order-" + orderID))
```

With the default **[store.Memory](store/memory.go)**:

- After a **successful** run, another run with the same key returns **`nil`** and **does not** re-execute steps.
- After a **failed** run, the key is released so you can **retry**.
- Two overlapping runs: the second gets **`flowcore.ErrIdempotencyInProgress`** until the first finishes.

Custom stores implement **[store.IdempotencyStore](store/idempotency.go)**. If you set a key but the store does not support it, `RunWithConfig` returns an error.

**Heads-up:** a hard crash mid-run can leave a key stuck in “running” until you use a fresh store or add operational reset/TTL (not built in yet).

Override per run: `RunConfig{ IdempotencyKey: "…" }`.

---

## Engine: custom store, quiet logs, async

```go
import (
	"context"

	"github.com/hghukasyan/flowcore"
	"github.com/hghukasyan/flowcore/engine"
	"github.com/hghukasyan/flowcore/store"
)

e := engine.New(
	engine.WithStore(store.NewMemory()),
	engine.WithLogger(nil),
)
err := e.Run(ctx, wf)

// or
errCh := e.RunAsync(ctx, wf)
err = <-errCh
```

---

## Flowcore and Temporal

**Temporal** excels at long-lived, distributed workflows — and expects a cluster, workers, and operational maturity.

**Flowcore** is intentionally narrow: **embeddable**, **readable**, **stdlib-only**. It shines for local sagas, batch pipelines, integration tests, and services where you want structure **without** running another platform. If you outgrow it, you can still migrate orchestration to a full engine later.

---

## Project layout

| Location | What lives there |
|----------|------------------|
| Repo root (`package flowcore`) | Workflow API, execution, retries, saga, idempotency |
| [`engine/`](engine/) | `Engine`, `RunAsync`, `PlanParallel` |
| [`store/`](store/) | `Store`, in-memory backend, idempotency hooks |
| [`examples/`](examples/) | Runnable programs |

---

## Roadmap

- Redis or SQL `Store` implementation
- Optional distributed mode (leases / heartbeat) without bloating the core API
- Cron-style scheduled workflows
- Richer observability hooks (e.g. OpenTelemetry) as optional paths

---

## License & community

Licensed under the [MIT License](LICENSE).

Contributions are welcome — see [CONTRIBUTING.md](CONTRIBUTING.md).
