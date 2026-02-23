## Context

Krangka currently uses Uber FX for dependency injection across all layers (inbound adapters, outbound adapters, services, infrastructure). This introduces reflection-based "magic" that obscures the dependency graph and makes initialization flow difficult to trace. The FX framework requires learning its specific patterns (`fx.Module`, `fx.Provide`, `fx.Annotate`, `fx.Invoke`, `fx.Lifecycle`) and debugging initialization failures often involves understanding FX's internals.

**Current State:**
- All `module.go` files export `fx.Option` values
- Entry points use `fx.New()` to bootstrap the application
- Lifecycle management uses `fx.Lifecycle` hooks (OnStart, OnStop)
- Workers use `fx.Shutdowner` for graceful termination
- Service dependencies are resolved via `fx.Annotate` with interface casting

**Constraints:**
- Must maintain hexagonal architecture principles
- Must preserve existing business logic and behavior
- Must not break external APIs or deployment processes
- Must maintain or improve testability

## Goals / Non-Goals

**Goals:**
- Replace all FX dependency injection with explicit constructor-based wiring
- Make dependency graphs transparent and traceable through code
- Simplify application initialization and debugging
- Remove `go.uber.org/fx` dependency completely
- Maintain hexagonal architecture with clear layer boundaries
- Preserve existing application behavior and external interfaces
- Improve testing by removing FX test helpers in favor of direct construction

**Non-Goals:**
- Changing business logic or domain models
- Modifying external APIs or HTTP endpoints
- Introducing a different DI framework
- Altering the hexagonal architecture structure
- Performance optimization (though it may improve as a side effect)
- Changing deployment or build processes

## Decisions

### 1. Constructor Pattern

**Decision:** Replace `fx.Module` exports with standard Go constructor functions.

**Rationale:**
- Constructor functions are idiomatic Go (e.g., `New()`, `NewService()`)
- Dependencies become explicit function parameters
- No reflection or runtime magic - compile-time type safety
- Easier to trace dependencies by following function calls

**Pattern:**
```go
// Before (FX)
var Module = fx.Module(
    "service",
    fx.Provide(
        fx.Annotate(note.NewService, fx.As(new(inbound.Note))),
    ),
)

// After (Manual)
func NewService(noteRepo outbound.NoteRepository, logger logger.Logger) inbound.Note {
    return note.NewService(noteRepo, logger)
}
```

**Alternatives Considered:**
- **Keep FX but simplify**: Rejected - still maintains framework dependency and complexity
- **Use a simpler DI framework (Wire, Dig)**: Rejected - user wants no DI framework at all

### 2. Application Bootstrap Pattern

**Decision:** Create explicit bootstrap functions in each entry point that manually construct and wire dependencies.

**Rationale:**
- Full control over initialization order
- Easy to understand flow from top to bottom
- Simple to add conditional logic or environment-specific setup
- No hidden initialization through `fx.Invoke`

**Pattern:**
```go
// In cmd/http.go
func runHTTP(cfgFile string) error {
    // 1. Load config
    cfg, err := configs.Load(cfgFile)
    if err != nil {
        return err
    }
    
    // 2. Initialize infrastructure
    log := logger.New(cfg.Logger)
    db := mariadb.New(cfg.MariaDB, log)
    cache := redis.New(cfg.Redis, log)
    
    // 3. Initialize repositories (outbound adapters)
    noteRepo := repositories.NewNoteRepository(db)
    todoRepo := repositories.NewTodoRepository(db)
    
    // 4. Initialize services
    noteSvc := service.NewNoteService(noteRepo, log)
    todoSvc := service.NewTodoService(todoRepo, log)
    
    // 5. Initialize handlers (inbound adapters)
    noteHandler := handler.NewNoteHandler(noteSvc)
    todoHandler := handler.NewTodoHandler(todoSvc)
    
    // 6. Initialize HTTP server
    server := http.NewHTTP(cfg.HTTP, noteHandler, todoHandler, log)
    
    // 7. Run with graceful shutdown
    return server.Run(context.Background())
}
```

**Alternatives Considered:**
- **Single factory function**: Rejected - too monolithic, harder to test individual components
- **Builder pattern**: Rejected - adds unnecessary abstraction for this use case

### 3. Lifecycle Management

**Decision:** Replace `fx.Lifecycle` with standard Go patterns: `Start()/Stop()` methods, context cancellation, and signal handling.

**Rationale:**
- Standard Go idioms (context, errgroup)
- No framework-specific lifecycle concepts
- More flexible control over startup/shutdown sequences
- Easier to test lifecycle events

**Pattern:**
```go
// Component with lifecycle
type HTTP struct {
    app *fiber.App
    cfg Config
}

func (h *HTTP) Run(ctx context.Context) error {
    // Setup signal handling
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    
    // Start server in goroutine
    errChan := make(chan error, 1)
    go func() {
        errChan <- h.app.Listen(h.cfg.Port)
    }()
    
    // Wait for shutdown signal or error
    select {
    case <-ctx.Done():
        return h.app.Shutdown()
    case <-sigChan:
        return h.app.Shutdown()
    case err := <-errChan:
        return err
    }
}
```

**Alternatives Considered:**
- **Third-party lifecycle library**: Rejected - adds another dependency
- **Global init/shutdown functions**: Rejected - less flexible, harder to test

### 4. Worker Shutdown Pattern

**Decision:** Replace `fx.Shutdowner` with context cancellation and os.Exit() for one-time workers.

**Rationale:**
- Context cancellation is the standard Go way to signal shutdown
- OneTime workers can exit naturally after completion
- LongRunning workers use context.Done() channel for graceful stop
- Simpler than maintaining global shutdowner reference

**Pattern:**
```go
// OneTime worker
func (w *WorkerGenerateID) Execute(ctx context.Context) error {
    id := ulid.Generate()
    log.Println(id)
    return nil // Worker exits naturally
}

// LongRunning worker
func (w *RelayOutbox) Execute(ctx context.Context) error {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            if err := w.processOutbox(ctx); err != nil {
                log.Error(err)
            }
        }
    }
}
```

**Alternatives Considered:**
- **Keep global shutdowner**: Rejected - maintains FX pattern without FX
- **Channel-based shutdown**: Considered equivalent to context, but context is more idiomatic

### 5. Module Organization

**Decision:** Keep `module.go` files but export constructor functions instead of `fx.Option`.

**Rationale:**
- Maintains existing file organization
- Provides central location for constructing each module's types
- Gradual migration path - can update one module at a time
- Easy to find initialization code

**Pattern:**
```go
// internal/adapter/outbound/mariadb/module.go
func NewMain(cfg configs.MariaDB, log logger.Logger) (*qwery.MariaDB, error) {
    return qwery.NewMariaDB(cfg.DSN, qwery.WithLogger(log))
}

func NewWorker(cfg configs.MariaDB, log logger.Logger) (*qwery.MariaDB, error) {
    return qwery.NewMariaDB(cfg.DSN, qwery.WithLogger(log))
}
```

**Alternatives Considered:**
- **Remove module.go files entirely**: Rejected - loses organizational structure
- **Single NewModule() factory**: Rejected - some modules need multiple variants (Main vs Worker DB)

## Risks / Trade-offs

### Risk: Boilerplate Code Increases

**Description:** Manual wiring requires more lines of code in entry points compared to FX's automatic resolution.

**Mitigation:**
- Keep bootstrap functions focused and well-organized
- Extract common initialization patterns to helper functions
- Benefits (explicitness, debuggability) outweigh verbosity

**Trade-off:** More code to write vs. easier to understand and maintain.

### Risk: Initialization Order Bugs

**Description:** Manual wiring makes it possible to construct dependencies in the wrong order (e.g., using DB before it's initialized).

**Mitigation:**
- Use Go's type system - dependencies are function parameters
- Compile will fail if dependencies aren't provided
- Clear top-to-bottom initialization flow makes order obvious
- Add initialization tests that verify proper setup

**Trade-off:** Must be explicit about order vs. FX automatically determining order.

### Risk: Breaking Changes in Tests

**Description:** Tests that rely on FX test helpers (fx.Invoke, fx.Populate) will need updates.

**Mitigation:**
- Most tests already use direct construction patterns
- Test changes are mostly removing FX boilerplate, not rewriting logic
- Opportunity to simplify tests by removing FX abstractions

**Trade-off:** One-time test updates vs. long-term simpler test setup.

### Risk: Duplicate Initialization Logic

**Description:** Multiple entry points (http, worker, migrate) may duplicate similar bootstrap code.

**Mitigation:**
- Extract common initialization patterns to shared functions
- Keep entry points focused on their specific needs
- Some duplication is acceptable for clarity

**Trade-off:** Some code duplication vs. premature abstraction.

## Migration Plan

### Phase 1: Update Infrastructure Layer
1. Update `configs/module.go` to export `Load(cfgFile string) (*Config, error)`
2. Update `shared/logger/module.go` to export `New(cfg LoggerConfig) Logger`
3. Remove FX imports from these packages
4. Verify no FX-specific code remains

### Phase 2: Update Outbound Adapters
1. For each outbound adapter package (mariadb, redis, redisstream, dlock):
   - Replace `Module` with constructor functions (e.g., `NewMain()`, `NewWorker()`)
   - Remove FX imports and options
   - Update dependencies to be explicit constructor parameters
2. Verify adapter functionality with existing tests

### Phase 3: Update Services
1. Update `internal/core/service/module.go`:
   - Export constructors for each service (e.g., `NewNoteService()`, `NewTodoService()`)
   - Remove FX annotations
2. Ensure services receive port interfaces as parameters
3. Verify service tests still pass

### Phase 4: Update Inbound Adapters
1. For each inbound adapter (http, worker, migrate):
   - Replace module files with constructors
   - Remove FX lifecycle hooks
   - Implement `Run(context.Context)` pattern for lifecycle
2. Update worker interfaces to use context for shutdown

### Phase 5: Update Entry Points
1. Update `cmd/http.go`:
   - Remove `fx.New()`
   - Add explicit bootstrap function
   - Wire dependencies manually
   - Implement signal handling for graceful shutdown
2. Update `cmd/migrate.go` similarly
3. Update `cmd/worker.go` with worker-specific bootstrap
4. Update `cmd/root.go` if needed

### Phase 6: Cleanup
1. Remove `go.uber.org/fx` from `go.mod`
2. Run `go mod tidy`
3. Verify all tests pass
4. Update documentation to reflect new initialization patterns
5. Remove any FX-related comments or docs

### Rollback Strategy

If issues arise during migration:
1. Each phase is independent - can rollback individual packages
2. Git commits should be per-phase for easy reversion
3. Tests must pass at each phase before proceeding
4. If critical issues found, can maintain FX in go.mod temporarily while fixing

### Testing Strategy

- Run full test suite after each phase
- Add integration test for each entry point to verify bootstrap
- Test graceful shutdown behavior for HTTP and LongRunning workers
- Verify application behavior is identical before/after migration

## Open Questions

**Q1: Should we create a shared bootstrap helper package?**

Leaning toward: No initially, wait to see if patterns emerge across entry points that warrant abstraction. Premature abstraction could make entry points harder to understand.

**Q2: How to handle environment-specific initialization?**

Current approach: Keep it in entry point bootstrap functions with conditional logic based on config. This makes environment differences explicit.

**Q3: Should lifecycle management be standardized with an interface?**

Leaning toward: No. Different components have different lifecycle needs. HTTP server needs signal handling, workers need context cancellation, migrations are one-shot. Better to keep them explicit than force a common interface.
