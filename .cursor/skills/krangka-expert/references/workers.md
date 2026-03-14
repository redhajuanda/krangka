# Workers

Workers are background jobs: **one-off** (Execute), **scheduled/cron** (Schedule), or **long-running until signal** (Run). They live in `internal/adapter/inbound/worker/`.

**Subscribers** (pure event consumers from Kafka/Redis) are a separate concept — see [subscriber.md](subscriber.md).

## Worker vs Subscriber

| Aspect | Worker | Subscriber |
|--------|--------|------------|
| Trigger | Cron, one-off, or long-running | Event-driven (Kafka/Redis message) |
| Bootstrap | `Execute`, `Schedule`, or `Run` | `Run` |
| Location | `internal/adapter/inbound/worker/` | `internal/adapter/inbound/subscriber/` |

## Worker Types

| Type | Bootstrap method | Interface | Example |
|------|-------------------|-----------|---------|
| **One-off** | `boot.Execute(ctx, runner)` | `Executable` | `worker generate-id` — runs once and exits |
| **Scheduled (Cron)** | `boot.Schedule(pattern, runner, opts)` | `Executable` | `worker outbox-relay` — runs every N seconds |
| **Long-running** | `boot.Run(runner, opts)` | `Runnable` | Custom loop until SIGINT/SIGTERM |

## Adding a New Worker — Checklist

1. Create `internal/adapter/inbound/worker/worker_<name>.go`
2. Add `ResourceExecutable` or `ResourceRunnable` field + getter in `cmd/bootstrap/dependency.go`
3. Add cobra command in `cmd/worker.go`
4. Register with `workerCmd.AddCommand(workerXxxCmd)` in `cmd/root.go`
5. (Scheduled only) Add cron pattern config in YAML

## Step-by-Step

### 1. Worker Struct

**One-off (minimal):**
```go
// internal/adapter/inbound/worker/worker_generate_id.go
package worker

type WorkerGenerateID struct {
    cfg *configs.Config
    log logger.Logger
}

func NewWorkerGenerateID(cfg *configs.Config, log logger.Logger) *WorkerGenerateID {
    return &WorkerGenerateID{cfg: cfg, log: log}
}

func (w *WorkerGenerateID) Execute(ctx context.Context) error {
    w.log.WithContext(ctx).Info("starting generate id worker")
    // ... do work
    return nil
}
```

**Scheduled (with DB):**
```go
type WorkerCleanup struct {
    cfg  *configs.Config
    log  logger.Logger
    repo outbound.Repository
}

func NewWorkerCleanup(cfg *configs.Config, log logger.Logger, repo outbound.Repository) *WorkerCleanup {
    return &WorkerCleanup{cfg: cfg, log: log, repo: repo}
}

func (w *WorkerCleanup) Execute(ctx context.Context) error {
    w.log.WithContext(ctx).Info("starting cleanup worker")
    if err := w.repo.GetCleanupRepository().DeleteExpired(ctx); err != nil {
        return fail.Wrap(err)
    }
    w.log.WithContext(ctx).Info("cleanup worker completed")
    return nil
}
```

**Long-running (Runnable):**
```go
type WorkerPoller struct {
    cfg *configs.Config
    log logger.Logger
}

func (w *WorkerPoller) OnStart(ctx context.Context) error {
    // start goroutines, listeners, etc.
    return nil
}

func (w *WorkerPoller) OnStop(ctx context.Context) error {
    // graceful shutdown
    return nil
}
```

**Rules:**
- Use `GetRepository(d.GetSikatWorker())` for DB — **never** `GetSikatMain()` in workers
- Wrap errors with `fail.Wrap` / `fail.Wrapf`
- Use `log.WithContext(ctx)` for logging
- Respect `ctx` for cancellation

### 2. Dependency Wiring

```go
// Field
workerCleanup ResourceExecutable[*worker.WorkerCleanup]   // Executable
workerPoller  ResourceRunnable[*worker.WorkerPoller]       // Runnable

// Getter — one-off (no repo)
func (d *Dependency) GetWorkerGenerateID() *worker.WorkerGenerateID {
    return d.workerGenerateID.Resolve(func() *worker.WorkerGenerateID {
        return worker.NewWorkerGenerateID(d.GetConfig(), d.GetLogger())
    })
}

// Getter — with repo (always use GetSikatWorker)
func (d *Dependency) GetWorkerCleanup() *worker.WorkerCleanup {
    return d.workerCleanup.Resolve(func() *worker.WorkerCleanup {
        repo := d.GetRepository(d.GetSikatWorker())
        return worker.NewWorkerCleanup(d.GetConfig(), d.GetLogger(), repo)
    })
}
```

### 3. Cobra Command

**One-off:**
```go
var workerGenerateIDCmd = &cobra.Command{
    Use:   "generate-id",
    Short: "Run generate ID worker once",
    Run: func(c *cobra.Command, args []string) {
        var (
            ctx    = context.Background()
            dep    = bootstrap.NewDependency(cfgFile)
            logger = dep.GetLogger()
            runner = dep.GetWorkerGenerateID()
            boot   = bootstrap.New(dep)
        )
        if err := boot.Execute(ctx, runner); err != nil {
            logger.Fatalf("failed: %v", err)
        }
    },
}
```

**Scheduled:**
```go
var workerCleanupCmd = &cobra.Command{
    Use:   "cleanup",
    Short: "Start the cleanup worker",
    Run: func(c *cobra.Command, args []string) {
        var (
            dep    = bootstrap.NewDependency(cfgFile)
            cfg    = dep.GetConfig()
            logger = dep.GetLogger()
            runner = dep.GetWorkerCleanup()
            boot   = bootstrap.New(dep)
            opts   = bootstrap.ScheduleOptions{
                ShutdownTimeout: 30 * time.Second,
                SingletonMode:   true, // prevent overlapping runs
            }
        )
        if err := boot.Schedule(cfg.Workers.CleanupPattern, runner, opts); err != nil {
            logger.Fatalf("failed: %v", err)
        }
    },
}
```

### 4. Register Command

```go
// cmd/root.go
workerCmd.AddCommand(workerCleanupCmd)
```

## Cron Patterns

- **6-field (with seconds):** `"*/10 * * * * *"` — every 10 seconds
- **5-field (standard):** `"*/5 * * * *"` — every 5 minutes; `"0 * * * *"` — every hour

Store environment-varying patterns in config:
```yaml
event:
  workers:
    cleanup_pattern: "0 0 * * *"   # daily at midnight
```

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Using `GetSikatMain()` for worker DB | Use `GetSikatWorker()` |
| Hardcoding repo in service getter | Pass `repo` from `GetRepository(GetSikatWorker())` |
| Returning raw error | Wrap with `fail.Wrap` |
| Forgetting to register command | Add `workerCmd.AddCommand(...)` in root.go |
| Overlapping cron runs | Set `ScheduleOptions.SingletonMode: true` |
