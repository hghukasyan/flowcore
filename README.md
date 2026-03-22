# Flowcore

Flowcore is a small workflow library for Go. You describe steps, optional dependencies, retries, and compensations. The library runs them in order, runs independent steps in parallel, and keeps a simple in-memory record of what happened.

It fits inside your process. You do not need extra servers or a database to get started.

Repository: [github.com/hghukasyan/flowcore](https://github.com/hghukasyan/flowcore)

## Features

- **Steps** — plain functions `func(ctx *flowcore.Context) error`
- **Dependencies** — `DependsOn("other_step")` so steps wait for their prerequisites
- **Parallel runs** — steps with no ordering constraint run at the same time
- **Retries** — `Retry(n)` plus optional fixed or exponential backoff
- **Timeouts** — `WithTimeout` per attempt
- **Saga-style compensation** — `WithCompensation` runs in reverse order after a failure
- **Sync and async** — `wf.Run(ctx)` or `engine.New().RunAsync(ctx, wf)`
- **Hooks** — optional `Logger` for start / success / fail
- **State** — in-memory store (workflow id, step status, retry count); swap later for Redis or SQL

**Standard library only.** No extra modules.

## Installation

```bash
git clone https://github.com/hghukasyan/flowcore.git
cd flowcore
```

Add to your module:

```bash
go get github.com/hghukasyan/flowcore
```

The main API is `package flowcore` at the module root, so you import `github.com/hghukasyan/flowcore` for workflows and `.../engine` or `.../store` for those subpackages.

## Quick example

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

`wf.Run` uses a default in-memory store and prints step lines to stdout. To turn off logs, use `flowcore.RunWithConfig(ctx, wf, flowcore.RunConfig{Logger: nil})` (and set `Store` if you want a custom store).

The same program lives in `examples/basic/basic.go`. Run it with `go run ./examples/basic` (Go needs one `main` package per folder, so examples use small subfolders).

## Advanced example (retry + compensation)

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

See `examples/saga/saga.go` — run with `go run ./examples/saga`.

### Custom engine (own store, quiet logs)

```go
import (
	"context"

	"github.com/hghukasyan/flowcore"
	"github.com/hghukasyan/flowcore/engine"
	"github.com/hghukasyan/flowcore/store"
)

e := engine.New(
	engine.WithStore(store.NewMemory()),
	engine.WithLogger(nil), // no lifecycle logs
)
err := e.Run(ctx, wf)
```

Async:

```go
errCh := e.RunAsync(ctx, wf)
err := <-errCh
```

## Why Flowcore (and not Temporal)?

Temporal is powerful for long-lived, distributed workflows. It also needs a cluster, workers, and more moving parts.

Flowcore is the opposite: a few hundred lines you embed in your app. Good for **local sagas**, **batch pipelines**, **integration tests**, or **small services** where you want structure without running another platform.

If you outgrow it, you can still move workflows to a full engine later.

## Project layout

| Path        | Role                                      |
|------------|-------------------------------------------|
| repo root (`package flowcore`) | Workflow API, context, run, retries, saga |
| `engine/`  | `Engine`, async run, `PlanParallel`       |
| `store/`   | `Store` interface + memory backend        |
| `examples/`| Runnable programs                         |

## Roadmap

- Redis (or SQL) `Store` implementation
- Optional distributed mode (lease + heartbeat), still keeping the API small
- Cron-style scheduled workflows
- Richer metrics hooks (OpenTelemetry) without pulling heavy deps by default

## License

This project is licensed under the MIT License — see [LICENSE](LICENSE).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).
