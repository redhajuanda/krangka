---
name: krangka-bootstrap
description: Use when choosing between Run/Execute/Schedule for a new command, understanding application lifecycle, or when the user asks about bootstrap, Dependency, Run, Execute, or Schedule. For wiring resource types (Resource, ResourceRunnable, ResourceExecutable, ResourceClosable), see krangka-dependency-wiring.
---

# Bootstrap Usage in Krangka

The bootstrap in `cmd/bootstrap` wires dependencies and manages application lifecycle (start, stop, cleanup). All application commands must use it consistently.

## Core Concepts

| Concept | Location | Purpose |
|--------|----------|---------|
| **Dependency** | `cmd/bootstrap/dependency.go` | Resolves and caches dependencies. Created once per command with `bootstrap.NewDependency(cfgFile)`. |
| **Bootstrap** | `cmd/bootstrap/bootstrap.go` | Runs the app, manage lifecycle, handles signals (SIGINT/SIGTERM), and ensures resource cleanup. Created with `bootstrap.New(dep)`. |
| **cfgFile** | `cmd/root.go` | Set by the `--config` flag. Pass it to `NewDependency(cfgFile)`. Default is `configs/files/default.yaml` if not set. |

## Three Bootstrap Methods

| Method | Use when | Behavior |
|--------|----------|----------|
| `Run(runner, opts)` | Long‑running server/process | Blocks until signal; starts runner, waits for SIGINT/SIGTERM, then stops runner and closes resources that implements `Closable` interface. Runner must implement `Runnable` interface. |
| `Execute(ctx, execute)` | One-off task (migrate, generate) | Runs once and returns. No signal handling, no blocking. |
| `Schedule(pattern, execute, opts)` | Cron-style worker | Runs execute on a cron schedule until signal; graceful shutdown. |

---

## Command Patterns (Copy These and modify to fit the use case)

### 1. Long‑Running Server → `Run`

```go
var httpCmd = &cobra.Command{
	Use: "http",
	Run: func(_ *cobra.Command, _ []string) {
		var (
			dep        = bootstrap.NewDependency(cfgFile) // init new dependency instance
			cfg        = dep.GetConfig() // get/resolve config dependency
			logger     = dep.GetLogger() // get/resolve logger dependency
			runnerHTTP = dep.GetHTTP() // get/resolve HTTP server dependency
			opts       = bootstrap.RunOptions{ // create run options
				StartTimeout: cfg.Http.StartTimeout, // get start timeout from config
				StopTimeout:  cfg.Http.StopTimeout, // get stop timeout from config
			}
		)

		err := bootstrap.New(dep).Run(runnerHTTP, opts) // run the HTTP server
		if err != nil {
			logger.Fatal(err)
		}
	},
}
```

Rules:
- Use `dep.GetXxx()` to get the runner (must implement `Runnable` interface: `OnStart`, `OnStop`).
- Use `bootstrap.Run(runner, opts)` to run the application.
- On error: `logger.Fatal(err)`.
- `RunOptions` is a struct that contains the options for the `Run` method: `StartTimeout`, `StopTimeout`. Default is 30 seconds if not set.

---

### 2. One‑Off Task → `Execute`

```go
func runMigrate(migrateType string, args []string) {
	var (
		ctx    = context.Background() // create new context
		dep    = bootstrap.NewDependency(cfgFile) // init new dependency instance
		logger = dep.GetLogger() // get/resolve logger dependency
		runner = dep.GetMigrate(migrateType, maxInt, repository) // get/resolve migrate dependency (must implement `Executable` interface: `Execute(ctx context.Context) error`)
		boot   = bootstrap.New(dep) // init new bootstrap instance
	)

	err := boot.Execute(ctx, runner) // execute the migrate
	if err != nil {
		logger.Fatalf("failed to execute migrate: %v", err)
	}
}
```

Rules:
- Use `bootstrap.Execute(ctx, runner)`.
- Runner must implement `Executable`: `Execute(ctx context.Context) error`.
- No `RunOptions`; task runs once and exits.

---

### 3. Scheduled Worker (Cron) → `Schedule`

```go
var workerOutboxRelayCmd = &cobra.Command{
	Use:   "outbox-relay",
	Short: "Start the outbox relay worker",
	Run: func(c *cobra.Command, args []string) {
		var (
			dep    = bootstrap.NewDependency(cfgFile) // init new dependency instance
			cfg    = dep.GetConfig() // get/resolve config dependency
			logger = dep.GetLogger() // get/resolve logger dependency
			runner = dep.GetWorkerOutboxRelay() // get/resolve worker outbox relay dependency (must implement `Executable` interface: `Execute(ctx context.Context) error`)
			boot   = bootstrap.New(dep) // init new bootstrap instance
		)

		err := boot.Schedule(cfg.Event.Outbox.RelayPattern, runner, opts) // schedule the worker outbox relay
		if err != nil {
			logger.Fatalf("failed to schedule worker outbox relay: %v", err) // log error if failed to schedule worker outbox relay
		}
	},
}
```

Rules:
- First arg: cron pattern string (e.g. `"*/5 * * * *"`). it also handle seconds pattern (e.g. `"* * * * * *"`).
- Runner must implement `Executable` interface: `Execute(ctx context.Context) error`.
- `ScheduleOptions` is a struct that contains the options for the `Schedule` method: `ShutdownTimeout`, `SingletonMode`.

---

## Decision Flow

1. **Long-running until signal?** (HTTP, subscriber worker) → `Run(runner, RunOptions)`
2. **Run once and exit?** (migrate up/down, migrate new, kubernetes scheduled job, etc.) → `Execute(ctx, runner)`
3. **Run on a cron schedule until signal?** → `Schedule(pattern, runner, ScheduleOptions)`

---

## Adding a New Command

1. **Add a getter in `cmd/bootstrap/dependency.go`** (if you need a new runner/dependency).
2. **Add the cobra command** in the right `cmd/*.go` file.
3. **Wire with bootstrap** using the correct pattern:
   - Long‑running → `Run`
   - One‑off → `Execute`
   - Cron → `Schedule`
4. **Register the command** in `cmd/root.go` via `rootCmd.AddCommand(...)` or subcommand `AddCommand`.

---

## Interfaces (cmd/bootstrap/interface.go)

| Interface | Methods | Used by |
|-----------|---------|---------|
| `Runnable` | `OnStart(ctx)`, `OnStop(ctx)` | `Run` |
| `Executable` | `Execute(ctx)` | `Execute`, `Schedule` |

---

## Options Reference

**RunOptions**
- `StartTimeout`: max wait for runner to start (default 30s).
- `StopTimeout`: max wait for runner to stop on shutdown (default 30s).

**ScheduleOptions**
- `ShutdownTimeout`: max wait for running job on shutdown (default 30s).
- `SingletonMode`: `true` prevents overlapping executions.
