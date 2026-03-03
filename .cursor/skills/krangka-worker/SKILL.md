---
name: krangka-worker
description: Guide for the worker system in krangka (Execute, Schedule, Run). Use when creating or modifying workers, adding scheduled jobs, wiring Executable or Runnable workers, choosing Execute vs Schedule vs Run, or when the user asks about background jobs, cron workers, or batch processing. For event-driven consumers (Kafka/Redis), see krangka-subscriber instead.
---

# Worker in Krangka

Workers are background jobs that run **once** (Execute), **on a schedule** (Schedule), or **long-running until signal** (Run). They live in `internal/adapter/inbound/worker/` and can implement either `Executable` or `Runnable`. **Subscribers** are a separate concept: **pure event consumers** (Kafka/Redisstream message handlers) live in `internal/adapter/inbound/subscriber/`.

## Worker vs Subscriber

| Aspect | Worker | Subscriber |
|--------|--------|------------|
| Trigger | Cron, one-off, or long-running until signal | Event-driven (Kafka/Redisstream message) |
| Bootstrap | `Execute`, `Schedule`, or `Run` | `Run` |
| Interface | `Executable` or `Runnable` | `Runnable` (via Watermill router) |
| Location | `internal/adapter/inbound/worker/` | `internal/adapter/inbound/subscriber/` |
| Use case | Batch jobs, cleanup, outbox relay, ID generation, long-running non-consumer processes | Pure event consumption (subscribe to topics, process messages) |

**Split rule:** Only **pure consumers** (subscribe to topics, process messages) go in subscriber. Everything else — one-off, cron, or long-running workers — goes in worker. Workers can use `Runnable` too.

## Worker Types and Bootstrap Choice

| Type | Bootstrap method | Interface | Example |
|------|-------------------|-----------|---------|
| **One-off** | `boot.Execute(ctx, runner)` | `Executable` | `worker generate-id` — runs once and exits |
| **Scheduled (Cron)** | `boot.Schedule(pattern, runner, opts)` | `Executable` | `worker outbox-relay` — runs every N seconds until SIGINT/SIGTERM |
| **Long-running** | `boot.Run(runner, opts)` | `Runnable` | Worker that runs until signal (e.g. polling, custom loop) |

**Decision:** One-off and exit? → `Execute`. Repeat on schedule? → `Schedule`. Long-running until signal (and not a pure event consumer)? → `Run` in worker.

---

## Directory Layout

```
internal/adapter/inbound/worker/
├── worker_generate_id.go    # One-off: generate ULIDs
└── worker_outbox_relay.go   # Scheduled: retry outbox entries
```

---

## Adding a New Worker

### Checklist

1. Create worker struct in `internal/adapter/inbound/worker/worker_<name>.go`
2. Add `ResourceExecutable` or `ResourceRunnable` field and getter in `cmd/bootstrap/dependency.go`
3. Add cobra command in `cmd/worker.go`
4. Register command in `cmd/root.go` via `workerCmd.AddCommand(...)`
5. If scheduled: add cron pattern config in YAML (or use existing section)

---

### 1. Worker struct and Execute

**One-off worker (minimal dependencies):**

```go
// internal/adapter/inbound/worker/worker_generate_id.go
package worker

import (
    "context"
    "github.com/redhajuanda/krangka/configs"
    "github.com/redhajuanda/komon/logger"
)

type WorkerGenerateID struct {
    cfg *configs.Config
    log logger.Logger
}

func NewWorkerGenerateID(cfg *configs.Config, log logger.Logger) *WorkerGenerateID {
    return &WorkerGenerateID{cfg: cfg, log: log}
}

// Execute implements Executable
func (w *WorkerGenerateID) Execute(ctx context.Context) error {
    w.log.WithContext(ctx).Info("starting generate id worker")
    // ... do work
    w.log.WithContext(ctx).Info("generate id worker completed")
    return nil
}
```

**Scheduled worker (with DB/outbound):**

```go
// internal/adapter/inbound/worker/worker_outbox_relay.go
package worker

import (
    "context"
    "github.com/redhajuanda/krangka/configs"
    "github.com/redhajuanda/krangka/internal/core/port/outbound"
    "github.com/redhajuanda/komon/logger"
)

type WorkerOutboxRelay struct {
    cfg        *configs.Config
    log        logger.Logger
    repository outbound.Repository
}

func NewWorkerOutboxRelay(cfg *configs.Config, log logger.Logger, repo outbound.Repository) *WorkerOutboxRelay {
    return &WorkerOutboxRelay{cfg: cfg, log: log, repository: repo}
}

func (w *WorkerOutboxRelay) Execute(ctx context.Context) error {
    w.log.WithContext(ctx).Info("starting outbox relay execution")
    err := w.repository.RetryOutbox(ctx)
    if err != nil {
        w.log.WithContext(ctx).Error("outbox relay execution failed", "error", err)
        return err
    }
    w.log.WithContext(ctx).Info("outbox relay execution completed")
    return nil
}
```

**Long-running worker (Runnable):**

For workers that run until signal (polling, custom loop, etc.) — not pure event consumers:

```go
// internal/adapter/inbound/worker/worker_xxx.go
type WorkerXxx struct {
    cfg *configs.Config
    log logger.Logger
    // ...
}

func (w *WorkerXxx) OnStart(ctx context.Context) error {
    // Start goroutines, listeners, etc.
    return nil
}

func (w *WorkerXxx) OnStop(ctx context.Context) error {
    // Graceful shutdown
    return nil
}
```

Wire as `ResourceRunnable`; use `boot.Run(runner, opts)` in the command.

**Rules:**
- Executable: implement `Execute(ctx context.Context) error`
- Runnable: implement `OnStart(ctx)` and `OnStop(ctx)`
- Use `GetRepository(d.GetQweryWorker())` for DB — workers use the **worker** DB, not main
- Respect `ctx` for cancellation during shutdown
- Wrap errors with `fail.Wrap` / `fail.Wrapf` per krangka-fail
- Use `log.WithContext(ctx)` for logging

---

### 2. Dependency wiring

In `cmd/bootstrap/dependency.go`:

**Add field (with other worker fields):**
```go
workerGenerateID  ResourceExecutable[*worker.WorkerGenerateID]
workerOutboxRelay ResourceExecutable[*worker.WorkerOutboxRelay]
workerXxx         ResourceExecutable[*worker.WorkerXxx]   // Executable
workerYyy         ResourceRunnable[*worker.WorkerYyy]   // Runnable (long-running)
```

**Add getter:**

One-off (no repo):
```go
func (d *Dependency) GetWorkerGenerateID() *worker.WorkerGenerateID {
    return d.workerGenerateID.Resolve(func() *worker.WorkerGenerateID {
        return worker.NewWorkerGenerateID(d.GetConfig(), d.GetLogger())
    })
}
```

Scheduled (with repo):
```go
func (d *Dependency) GetWorkerOutboxRelay() *worker.WorkerOutboxRelay {
    return d.workerOutboxRelay.Resolve(func() *worker.WorkerOutboxRelay {
        repo := d.GetRepository(d.GetQweryWorker())
        return worker.NewWorkerOutboxRelay(d.GetConfig(), d.GetLogger(), repo)
    })
}
```

**Critical:** For workers that need DB, use `GetRepository(d.GetQweryWorker())` — never `GetQweryMain()`. For services, pass `repo` as parameter: `d.GetServiceXxx(repo)`.

---

### 3. Cobra command in cmd/worker.go

**One-off (Execute):**
```go
var workerGenerateIDCmd = &cobra.Command{
    Use:   "generate-id",
    Short: "Start the generate id worker",
    Run: func(c *cobra.Command, args []string) {
        var (
            ctx    = context.Background()
            dep    = bootstrap.NewDependency(cfgFile)
            logger = dep.GetLogger()
            runner = dep.GetWorkerGenerateID()
            boot   = bootstrap.New(dep)
        )
        err := boot.Execute(ctx, runner)
        if err != nil {
            logger.Fatalf("failed to execute worker generate id: %v", err)
        }
    },
}
```

**Scheduled (Schedule):**
```go
var workerOutboxRelayCmd = &cobra.Command{
    Use:   "outbox-relay",
    Short: "Start the outbox relay worker",
    Run: func(c *cobra.Command, args []string) {
        var (
            dep    = bootstrap.NewDependency(cfgFile)
            cfg    = dep.GetConfig()
            logger = dep.GetLogger()
            runner = dep.GetWorkerOutboxRelay()
            boot   = bootstrap.New(dep)
            opts   = bootstrap.ScheduleOptions{
                ShutdownTimeout: 30 * time.Second,
                SingletonMode:   true,  // Prevent overlapping executions
            }
        )
        err := boot.Schedule(cfg.Event.Outbox.RelayPattern, runner, opts)
        if err != nil {
            logger.Fatalf("failed to schedule worker outbox relay: %v", err)
        }
    },
}
```

**Long-running (Run):**
```go
var workerXxxCmd = &cobra.Command{
    Use:   "xxx",
    Short: "Start the xxx worker",
    Run: func(c *cobra.Command, args []string) {
        var (
            dep    = bootstrap.NewDependency(cfgFile)
            cfg    = dep.GetConfig()
            logger = dep.GetLogger()
            runner = dep.GetWorkerXxx()
            boot   = bootstrap.New(dep)
            opts   = bootstrap.RunOptions{
                StartTimeout: cfg.Http.StartTimeout,
                StopTimeout:  cfg.Http.StopTimeout,
            }
        )
        err := boot.Run(runner, opts)
        if err != nil {
            logger.Fatalf("failed to run worker xxx: %v", err)
        }
    },
}
```

**Rules:**
- One-off: `boot.Execute(ctx, runner)` — no opts
- Scheduled: `boot.Schedule(pattern, runner, opts)` — pattern from config or constant
- Long-running: `boot.Run(runner, opts)` — blocks until SIGINT/SIGTERM
- `ScheduleOptions.SingletonMode: true` — prevents overlapping runs (recommended for most cron jobs)
- `ScheduleOptions.ShutdownTimeout` — max wait for in-flight execution on shutdown

---

### 4. Register in cmd/root.go

```go
workerCmd.AddCommand(workerGenerateIDCmd)
workerCmd.AddCommand(workerOutboxRelayCmd)
workerCmd.AddCommand(workerXxxCmd)  // new
```

---

## Cron Pattern

- **6-field (seconds):** `"*/10 * * * * *"` — every 10 seconds
- **5-field (standard):** `"*/5 * * * *"` — every 5 minutes; `"0 * * * *"` — every hour

Store patterns in config when they vary per environment:

```yaml
# configs/files/default.yaml
event:
  outbox:
    relay_pattern: "*/10 * * * * *"
```

Add new config section for custom workers:

```yaml
event:
  workers:
    cleanup_pattern: "0 0 * * *"   # daily at midnight
```

---

## ScheduleOptions Reference

| Option | Type | Purpose |
|--------|------|---------|
| `ShutdownTimeout` | `time.Duration` | Max wait for running job on SIGINT/SIGTERM (default 30s) |
| `SingletonMode` | `bool` | If true, next run waits for current run to finish |

---

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Using `GetQweryMain()` for worker DB | Use `GetQweryWorker()` |
| Hardcoding repo in service getter | Pass `repo` from `GetRepository(GetQweryWorker())` |
| Skipping `ctx` in Execute | Use `ctx` for downstream calls and cancellation |
| Returning raw error | Wrap with `fail.Wrap` / `fail.Wrapf` |
| Forgetting to register command | Add `workerCmd.AddCommand(workerXxxCmd)` in root.go |
| Overlapping cron runs | Set `ScheduleOptions.SingletonMode: true` |

---

## Quick Reference

| Step | Action |
|------|--------|
| 1 | Create `worker_<name>.go` with struct + `Execute(ctx) error` (Executable) or `OnStart`/`OnStop` (Runnable) |
| 2 | Add `ResourceExecutable` or `ResourceRunnable` field + `GetWorkerXxx()` in dependency.go |
| 3 | Add cobra command in cmd/worker.go (`Execute`, `Schedule`, or `Run`) |
| 4 | Register with `workerCmd.AddCommand(workerXxxCmd)` in root.go |
| 5 | (Optional) Add config pattern in YAML for scheduled workers |

---

## Related Skills

- **krangka-bootstrap**: Execute vs Schedule vs Run
- **krangka-dependency-wiring**: ResourceExecutable, repo parameter for services
- **krangka-subscriber**: Event-driven consumers (different from workers)
- **krangka-fail**: Error wrapping
- **krangka-logger**: Logging with context
