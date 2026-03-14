# Bootstrap Lifecycle

The bootstrap in `cmd/bootstrap` wires dependencies and manages the application lifecycle.

## Core Concepts

| Concept | Location | Purpose |
|--------|----------|---------|
| `Dependency` | `cmd/bootstrap/dependency.go` | Resolves and caches dependencies. Created once per command with `bootstrap.NewDependency(cfgFile)`. |
| `Bootstrap` | `cmd/bootstrap/bootstrap.go` | Runs the app, manages lifecycle, handles SIGINT/SIGTERM, ensures resource cleanup. |
| `cfgFile` | `cmd/root.go` | Set by `--config` flag. Default: `configs/files/default.yaml`. |

## Three Bootstrap Methods

| Method | Use when | Behavior |
|--------|----------|----------|
| `Run(runner, opts)` | Long-running server/process | Blocks until signal; starts runner, waits for SIGINT/SIGTERM, then stops runner and closes `Closable` resources. |
| `Execute(ctx, execute)` | One-off task (migrate, generate) | Runs once and returns. No signal handling. |
| `Schedule(pattern, execute, opts)` | Cron-style worker | Runs execute on cron schedule until signal; graceful shutdown. |

## Decision Flow

1. **Long-running until signal?** (HTTP, subscriber worker) → `Run(runner, RunOptions)`
2. **Run once and exit?** (migrate up/down, kubernetes job) → `Execute(ctx, runner)`
3. **Run on cron schedule until signal?** → `Schedule(pattern, runner, ScheduleOptions)`

## Command Patterns

### 1. Long-Running Server → `Run`

```go
var httpCmd = &cobra.Command{
    Use: "http",
    Run: func(_ *cobra.Command, _ []string) {
        var (
            dep        = bootstrap.NewDependency(cfgFile)
            cfg        = dep.GetConfig()
            logger     = dep.GetLogger()
            runnerHTTP = dep.GetHTTP()
            opts       = bootstrap.RunOptions{
                StartTimeout: cfg.Http.StartTimeout,
                StopTimeout:  cfg.Http.StopTimeout,
            }
        )
        if err := bootstrap.New(dep).Run(runnerHTTP, opts); err != nil {
            logger.Fatal(err)
        }
    },
}
```

Runner must implement `Runnable`: `OnStart(ctx)`, `OnStop(ctx)`.

### 2. One-Off Task → `Execute`

```go
func runMigrate(migrateType string, args []string) {
    var (
        ctx    = context.Background()
        dep    = bootstrap.NewDependency(cfgFile)
        logger = dep.GetLogger()
        runner = dep.GetMigrate(migrateType, maxInt, repository)
        boot   = bootstrap.New(dep)
    )
    if err := boot.Execute(ctx, runner); err != nil {
        logger.Fatalf("failed to execute migrate: %v", err)
    }
}
```

Runner must implement `Executable`: `Execute(ctx context.Context) error`.

### 3. Scheduled Worker → `Schedule`

```go
var workerOutboxRelayCmd = &cobra.Command{
    Use: "outbox-relay",
    Run: func(c *cobra.Command, args []string) {
        var (
            dep    = bootstrap.NewDependency(cfgFile)
            cfg    = dep.GetConfig()
            logger = dep.GetLogger()
            runner = dep.GetWorkerOutboxRelay()
            boot   = bootstrap.New(dep)
            opts   = bootstrap.ScheduleOptions{
                ShutdownTimeout: 30 * time.Second,
                SingletonMode:   true,
            }
        )
        if err := boot.Schedule(cfg.Event.Outbox.RelayPattern, runner, opts); err != nil {
            logger.Fatalf("failed to schedule worker outbox relay: %v", err)
        }
    },
}
```

- First arg: cron pattern string (`"*/5 * * * *"` or 6-field `"*/10 * * * * *"` with seconds)
- Runner must implement `Executable`: `Execute(ctx context.Context) error`

## Adding a New Command

1. **Add a getter** in `cmd/bootstrap/dependency.go` (if new runner/dependency needed)
2. **Add the cobra command** in the right `cmd/*.go` file
3. **Wire with bootstrap** using the correct pattern
4. **Register** in `cmd/root.go` via `rootCmd.AddCommand(...)` or subcommand `AddCommand`

## Interfaces

| Interface | Methods | Used by |
|-----------|---------|---------|
| `Runnable` | `OnStart(ctx)`, `OnStop(ctx)` | `Run` |
| `Executable` | `Execute(ctx)` | `Execute`, `Schedule` |

## Options Reference

**RunOptions:**
- `StartTimeout`: max wait for runner to start (default 30s)
- `StopTimeout`: max wait for runner to stop on shutdown (default 30s)

**ScheduleOptions:**
- `ShutdownTimeout`: max wait for running job on shutdown (default 30s)
- `SingletonMode`: `true` prevents overlapping executions
