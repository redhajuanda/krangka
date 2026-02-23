## 1. Update Infrastructure Layer

- [x] 1.1 Update `configs/module.go` to export `Load(cfgFile string) (*Config, error)` function
- [x] 1.2 Remove FX imports from `configs/module.go`
- [x] 1.3 Update `shared/logger/module.go` to export `New(cfg LoggerConfig) Logger` function
- [x] 1.4 Remove FX imports from `shared/logger/module.go`
- [x] 1.5 Verify no FX-specific code remains in infrastructure packages
- [x] 1.6 Run tests for infrastructure packages to ensure they still work

## 2. Update Outbound Adapters - MariaDB

- [x] 2.1 Update `internal/adapter/outbound/mariadb/module.go` to export `NewMain()` constructor
- [x] 2.2 Update `internal/adapter/outbound/mariadb/module.go` to export `NewWorker()` constructor
- [x] 2.3 Remove FX imports and Module exports from mariadb package
- [x] 2.4 Update constructor parameters to accept explicit dependencies
- [x] 2.5 Verify mariadb adapter tests still pass

## 3. Update Outbound Adapters - Redis

- [x] 3.1 Update `internal/adapter/outbound/redis/module.go` to export constructor function
- [x] 3.2 Remove FX imports and Module exports from redis package
- [x] 3.3 Update constructor parameters to accept explicit dependencies
- [x] 3.4 Verify redis adapter tests still pass

## 4. Update Outbound Adapters - Redis Stream

- [x] 4.1 Update `internal/adapter/outbound/redisstream/module.go` to export `NewPublisher()` constructor
- [x] 4.2 Remove FX imports and Module exports from redisstream package
- [x] 4.3 Update constructor parameters to accept explicit dependencies
- [x] 4.4 Verify redisstream adapter tests still pass

## 5. Update Outbound Adapters - Distributed Lock

- [x] 5.1 Update `internal/adapter/outbound/dlock/module.go` to export constructor function
- [x] 5.2 Remove FX imports and Module exports from dlock package
- [x] 5.3 Update constructor parameters to accept explicit dependencies
- [x] 5.4 Verify dlock adapter tests still pass

## 6. Update Other Outbound Adapters

- [x] 6.1 Identify any remaining outbound adapters not yet updated
- [x] 6.2 Update remaining outbound adapter module files with constructors
- [x] 6.3 Remove all FX imports from outbound adapter packages
- [x] 6.4 Verify all outbound adapter tests pass

## 7. Update Service Layer

- [x] 7.1 Update `internal/core/service/module.go` to export `NewNoteService()` constructor
- [x] 7.2 Update `internal/core/service/module.go` to export `NewTodoService()` constructor
- [x] 7.3 Remove FX imports and annotations from service module
- [x] 7.4 Ensure service constructors accept port interfaces as parameters
- [x] 7.5 Verify service tests still pass with direct construction
- [x] 7.6 Update any service tests that used FX test helpers

## 8. Update Inbound Adapter - HTTP

- [x] 8.1 Update `internal/adapter/inbound/http/module.go` to remove FX Module export
- [x] 8.2 Create constructor function for HTTP server initialization
- [x] 8.3 Update `internal/adapter/inbound/http/http.go` to implement `Run(context.Context) error` method
- [x] 8.4 Remove FX lifecycle hooks (OnStart, OnStop) from HTTP adapter
- [x] 8.5 Implement signal handling in HTTP server for graceful shutdown
- [x] 8.6 Update HTTP handler module to export constructor functions
- [x] 8.7 Remove all FX imports from HTTP adapter package
- [x] 8.8 Verify HTTP adapter tests still pass

## 9. Update Inbound Adapter - Worker

- [x] 9.1 Update `internal/adapter/inbound/worker/module.go` to remove FX Module exports
- [x] 9.2 Create constructor functions for worker initialization
- [x] 9.3 Update worker interface to use `Execute(context.Context) error` pattern
- [x] 9.4 Remove `fx.Shutdowner` from worker implementations
- [x] 9.5 Update OneTime workers to use context for cancellation
- [x] 9.6 Update LongRunning workers to monitor context.Done() for shutdown
- [x] 9.7 Update worker handler constructors (GenerateID, RelayOutbox)
- [x] 9.8 Remove all FX imports from worker adapter package
- [x] 9.9 Verify worker adapter tests still pass

## 10. Update Inbound Adapter - Migrate

- [x] 10.1 Update `internal/adapter/inbound/migrate/module.go` to remove FX Module exports
- [x] 10.2 Create constructor functions for migration initialization
- [x] 10.3 Remove FX lifecycle hooks from migrate adapter
- [x] 10.4 Remove all FX imports from migrate adapter package
- [x] 10.5 Verify migrate adapter tests still pass

## 11. Update Entry Point - HTTP Command

- [x] 11.1 Update `cmd/http.go` to remove `fx.New()` call
- [x] 11.2 Create `runHTTP(cfgFile string) error` bootstrap function
- [x] 11.3 Wire dependencies manually in bootstrap function: config → logger → DB → cache → repos → services → handlers → server
- [x] 11.4 Implement signal handling (SIGINT, SIGTERM) in HTTP command
- [x] 11.5 Implement graceful shutdown with timeout
- [x] 11.6 Remove all FX imports from `cmd/http.go`
- [ ] 11.7 Test HTTP server startup and graceful shutdown manually

## 12. Update Entry Point - Worker Command

- [x] 12.1 Update `cmd/worker.go` to remove `fx.New()` calls
- [x] 12.2 Create `runWorker(cfgFile string, workerName string) error` bootstrap function
- [x] 12.3 Wire dependencies manually for each worker type
- [x] 12.4 Implement context-based cancellation for workers
- [x] 12.5 Implement signal handling for worker shutdown
- [x] 12.6 Remove all FX imports from `cmd/worker.go`
- [ ] 12.7 Test OneTime worker execution (generate-id)
- [ ] 12.8 Test LongRunning worker graceful shutdown (relay-outbox)

## 13. Update Entry Point - Migrate Command

- [x] 13.1 Update `cmd/migrate.go` migrate up command to remove `fx.New()`
- [x] 13.2 Update `cmd/migrate.go` migrate down command to remove `fx.New()`
- [x] 13.3 Update `cmd/migrate.go` migrate new command to remove `fx.New()`
- [x] 13.4 Create bootstrap functions for each migration command
- [x] 13.5 Wire dependencies manually for migration operations
- [x] 13.6 Remove all FX imports from `cmd/migrate.go`
- [ ] 13.7 Test migrate up command execution
- [ ] 13.8 Test migrate down command execution
- [ ] 13.9 Test migrate new command execution

## 14. Update Root and Other Commands

- [x] 14.1 Review `cmd/root.go` for any FX dependencies
- [x] 14.2 Remove any FX imports from root command if present
- [x] 14.3 Identify any other command files that use FX
- [x] 14.4 Update remaining command files to remove FX
- [x] 14.5 Verify all commands still function correctly

## 15. Cleanup and Verification

- [x] 15.1 Search entire codebase for any remaining `go.uber.org/fx` imports
- [x] 15.2 Remove `go.uber.org/fx` dependency from `go.mod`
- [x] 15.3 Run `go mod tidy` to clean up module dependencies
- [x] 15.4 Run full test suite to verify all tests pass
- [x] 15.5 Build application to verify no compilation errors
- [x] 15.6 Verify no FX-specific patterns remain (fx.Annotate, fx.Populate, fx.Invoke)
- [x] 15.7 Search for and remove any FX-related comments in code

## 16. Integration Testing

- [ ] 16.1 Create integration test for HTTP server bootstrap
- [ ] 16.2 Create integration test for HTTP server graceful shutdown
- [ ] 16.3 Create integration test for OneTime worker execution
- [ ] 16.4 Create integration test for LongRunning worker graceful shutdown
- [ ] 16.5 Create integration test for migration commands
- [ ] 16.6 Verify application behavior matches pre-migration behavior

## 17. Documentation Updates

- [x] 17.1 Update documentation to reflect new initialization patterns
- [x] 17.2 Remove references to FX from architecture documentation
- [x] 17.3 Add examples of manual dependency wiring
- [x] 17.4 Document new bootstrap function patterns
- [ ] 17.5 Update developer onboarding documentation
- [ ] 17.6 Add migration guide for teams using this boilerplate

## 18. Final Verification

- [ ] 18.1 Run application locally with HTTP server
- [ ] 18.2 Test all HTTP endpoints to verify correct behavior
- [ ] 18.3 Run workers to verify correct execution
- [ ] 18.4 Run migrations to verify correct operation
- [ ] 18.5 Test graceful shutdown for all entry points
- [ ] 18.6 Verify no regression in application functionality
- [ ] 18.7 Check application logs for any errors or warnings
- [ ] 18.8 Verify deployment readiness (no breaking changes to external APIs)
