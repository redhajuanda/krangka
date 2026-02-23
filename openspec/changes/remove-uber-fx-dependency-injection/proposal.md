## Why

Uber FX adds complexity and "magic" to the application bootstrap process, making it harder to trace dependency flow and debug initialization issues. Moving to explicit manual dependency wiring will improve code readability, make the dependency graph more transparent, and reduce reliance on reflection-based frameworks.

## What Changes

- Remove `go.uber.org/fx` dependency from the project
- Replace all `fx.Module`, `fx.Provide`, `fx.Annotate`, and `fx.Invoke` declarations with explicit constructor functions
- Replace `fx.Lifecycle` hooks with standard initialization and shutdown logic
- Update all `module.go` files to export simple constructor functions instead of FX modules
- Update `cmd/` entry points to manually wire dependencies instead of using `fx.New()`
- Remove FX-specific patterns (populate, shutdowner) from worker implementations
- Maintain hexagonal architecture principles with explicit dependency injection through constructors
- **BREAKING**: Application initialization signatures will change; any external tooling or tests depending on FX lifecycle may need updates

## Capabilities

### New Capabilities
- `manual-dependency-wiring`: Explicit dependency construction and wiring without framework assistance, using constructor functions and composition
- `application-bootstrap`: New application initialization approach for HTTP servers, workers, and CLI commands without FX lifecycle management

### Modified Capabilities
<!-- No existing specs are being modified - this is a pure refactoring of the initialization mechanism -->

## Impact

**Affected Code:**
- All `cmd/` files (http.go, migrate.go, worker.go)
- All `module.go` files across the codebase:
  - `internal/core/service/module.go`
  - `internal/adapter/inbound/http/module.go`
  - `internal/adapter/inbound/worker/module.go`
  - `internal/adapter/inbound/migrate/module.go`
  - `internal/adapter/outbound/*/module.go` (mariadb, redis, redisstream, dlock, etc.)
  - `configs/module.go`
  - `shared/logger/module.go`

**Affected Layers (Hexagonal Architecture):**
- **Inbound Adapters**: HTTP server, workers, and CLI initialization will change
- **Outbound Adapters**: Repository and external service initialization will change
- **Application Layer**: Service wiring will become explicit
- **Infrastructure**: Configuration and logger initialization will change

**Dependencies:**
- Remove: `go.uber.org/fx` from go.mod
- No new dependencies required

**APIs:**
- No external API changes - this is purely internal refactoring
- Internal initialization APIs will change significantly

**Systems:**
- Build and deployment processes remain unchanged
- Application runtime behavior should be identical after refactoring
