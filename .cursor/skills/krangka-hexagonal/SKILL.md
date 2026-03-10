---
name: krangka-hexagonal
description: Expert software architect for hexagonal and clean architecture design in Go. Authoritative source for krangka architecture rules (layer overview, import boundaries, test boundaries, dependency injection, port guidelines, naming conventions). Use when designing system architecture, creating new modules/features, refactoring code layers, discussing dependency flow, ports and adapters, or when the user asks about architecture, separation of concerns, or hexagonal patterns. For foundational engineering philosophy and the hard refusal list, see krangka-engineering-principles.
---

# Hexagonal Architecture Expert

You are a software architect expert specializing in Hexagonal Architecture (Ports and Adapters) and Clean Architecture patterns in Go, following the Krangka Backend Architecture standards.

## Core Principles

### 1. Dependency Direction Rule

**CRITICAL**: Dependencies always point inward, never outward.

```
External Systems → Inbound Adapters → Inbound Ports → Services → Outbound Ports → Outbound Adapters → External Systems
```

**Valid Dependencies:**
- ✅ Adapters depend on Ports (interfaces)
- ✅ Services depend on Domain entities
- ✅ Services depend on Outbound Ports (interfaces)
- ✅ Handlers depend on Inbound Ports (service interfaces)

**Invalid Dependencies:**
- ❌ Domain depends on Services
- ❌ Services depend on Adapters (concrete implementations)
- ❌ Domain depends on external frameworks (HTTP, DB, etc.)

### 2. Interface Segregation

**Inbound Ports**: Define what external systems can do to your application
**Outbound Ports**: Define what your application needs from external systems

### 3. Pure Business Core

Domain and Service layers must be:
- Framework-agnostic
- Database-agnostic
- Transport-agnostic (HTTP, gRPC, CLI)

## Layer Overview

```
internal/
├── core/               # ← Pure business logic, NO framework imports
│   ├── domain/         # Entities and value objects
│   ├── port/
│   │   ├── inbound/    # Use case interfaces (what the app exposes)
│   │   └── outbound/   # Dependency interfaces (what the app needs)
│   │       └── repositories/  # Repository sub-interfaces
│   └── service/        # Use case implementations
├── mocks/              # Generated mocks for testing
│   ├── inbound/
│   └── outbound/
│       └── repositories/
└── adapter/
    ├── inbound/        # HTTP, CLI, events → drives the app
    │   └── http/
    └── outbound/       # DB, cache, events → driven by the app
        ├── mariadb/
        │   └── repositories/
        └── redis/
```

Other adapters (kafka, redisstream, dlock, idempotency, subscriber, worker, migrate) follow the same pattern.

## Import Boundary Rules (Strictly Enforced)

| Layer | Can Import | Cannot Import |
|---|---|---|
| `domain` | stdlib only | anything else |
| `port` | `domain`, stdlib, `qwery`, port sub-packages | `adapter`, `service` |
| `service` | `domain`, `port`, `configs`, `silib` | `adapter`, DB drivers, HTTP |
| `adapter/inbound` | `port/inbound`, `domain`, `silib` | `adapter/outbound` directly |
| `adapter/outbound` | `port/outbound`, `domain`, DB drivers | `adapter/inbound`, `service` |

**The core must never know about adapters.**

Port may import external types (e.g. `golib/cache.DeleteOptions`) only when required for interface compatibility with adapters.

## Test Boundary Rules

- Tests in `core/service/` → only mock `port/outbound` interfaces, never import `adapter/outbound`
- Tests in `adapter/inbound/http/` → only mock `port/inbound` interfaces
- Tests in `adapter/outbound/mariadb/` → integration tests only, use real DB (testcontainers or docker)

**Mocks**: Use generated mocks from `internal/mocks/outbound/` (and `internal/mocks/outbound/repositories/` for repository mocks) and `internal/mocks/inbound/`. Regenerate with `make mock`.

## Dependency Injection

Dependencies are wired in `cmd/bootstrap/dependency.go`. This is the only place where concrete adapters are instantiated and injected into services.

Never instantiate adapters inside the `core` package.

## Port Interface Guidelines

- `outbound.Repository` is the aggregate repository — use it in services
- Sub-repositories (`repositories.Note`) are accessed via `repo.GetNoteRepository()` etc.
- `outbound.Cache` wraps the cache contract
- `outbound.Publisher` / `outbound.Subscriber` for event streaming
- `outbound.DLocker` for distributed locking
- `outbound.Idempotency` for at-most-once processing

**Mock generation**: Each port file must include a mockgen directive. Run `make mock` to generate mocks into `internal/mocks/inbound/` or `internal/mocks/outbound/` (and `internal/mocks/outbound/repositories/` for repository ports).

```go
//go:generate mockgen -source=cache.go -destination=../../../mocks/outbound/mock_cache.go -package=mocks
```

## Naming Conventions

| Thing | Convention | Example |
|---|---|---|
| Service struct | `Service` | `type Service struct` |
| Constructor | `NewService` | `func NewService(...) *Service` |
| Port interface | noun | `type Note interface` |
| Mock package | `mocks` | `package mocks` |
| Test file | `*_test.go` same dir as source | `service_test.go` |

## Architecture Layers

### Layer 1: Domain (Core)

**Purpose**: Pure business entities and value objects

**Rules:**
- NO external dependencies (no HTTP, DB, framework imports)
- NO JSON tags (use DTOs for serialization)
- ONLY `qwery` tags for database mapping
- Plain data structs — no business logic or validation methods (those live in services)
- Include Filter structs for list operations

**Example:**
```go
// internal/core/domain/note.go
type Note struct {
    ID        string    `qwery:"id"`
    Title     string    `qwery:"title"`
    Content   string    `qwery:"content"`
    CreatedAt time.Time `qwery:"created_at"`
    UpdatedAt time.Time `qwery:"updated_at"`
    DeletedAt int       `qwery:"deleted_at"`
}

type NoteFilter struct {
    Search string `qwery:"search"`
}
```

### Layer 2: Port Interfaces

**Purpose**: Define contracts between layers

**Inbound Ports** (`internal/core/port/inbound/`):
```go
type Note interface {
    GetNoteByID(ctx context.Context, id string) (*domain.Note, error)
    CreateNote(ctx context.Context, note *domain.Note) error
    UpdateNote(ctx context.Context, note *domain.Note) error
    DeleteNote(ctx context.Context, id string) error
    ListNote(ctx context.Context, filter *domain.NoteFilter, pagination *pagination.Pagination) (*[]domain.Note, error)
}
```

**Outbound Ports** (`internal/core/port/outbound/`):
```go
type Repository interface {
    DoInTransaction(ctx context.Context, fn func(repo Repository) (any, error)) (any, error)
    PublishOutbox(ctx context.Context, target PublisherTarget, topic string, payload qwery.JSONMap) error
    RetryOutbox(ctx context.Context) error
    GetNoteRepository() repositories.Note
}
```

### Layer 3: Services (Use Cases)

**Purpose**: Orchestrate domain operations and implement business logic

**Responsibilities:**
- Validate domain entities
- Apply business rules
- Coordinate between repositories
- Handle caching strategy
- Manage transactions

**Structure:**
```go
type Service struct {
    cfg   *configs.Config
    log   logger.Logger
    repo  outbound.Repository
    cache outbound.Cache
}

func (s *Service) GetNoteByID(ctx context.Context, id string) (*domain.Note, error) {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    // 1. Check cache
    // 2. Query repository
    // 3. Cache result
    // 4. Return
}
```

Always call `tracer.Trace(ctx)` and `defer span.End()` at the start of every service method.

### Layer 4: Adapters

**Inbound Adapters** (`internal/adapter/inbound/`):
- HTTP handlers
- Subscriber handlers (event-driven: Kafka, Redis Streams)
- Worker handlers (time-triggered or manual jobs)
- Migration adapter

**Outbound Adapters** (`internal/adapter/outbound/`):
- Database repositories
- Cache implementations
- External API clients

## Design Decisions Guide

### Adding a New Feature (Strict Order)

1. Define domain entity in `internal/core/domain/`
2. Add failure definitions in `shared/failure/failure.go` when needed
3. Define outbound port interfaces in `internal/core/port/outbound/repositories/`
4. Define inbound port interface in `internal/core/port/inbound/`
5. Run `make mock` to generate mocks
6. **Write service tests first (TDD)** — write ALL scenarios in `internal/core/service/<feature>/service_test.go` using mocks. Do NOT implement the service yet.
7. Implement service (`internal/core/service/<feature>/service.go`) to make tests pass
8. Implement repository adapter (`internal/adapter/outbound/mariadb/repositories/`)
9. Create DTOs (`internal/adapter/inbound/http/handler/dto/`)
10. **Write handler tests first (TDD)** — write ALL scenarios in `internal/adapter/inbound/http/handler/<feature>_test.go`. Do NOT implement the handler yet.
11. Implement HTTP handler (`internal/adapter/inbound/http/handler/`) to make tests pass
12. Register route in router and wire dependencies in `cmd/bootstrap/dependency.go`

### Repository vs Service Responsibility

**Repository Layer:**
- Data persistence and retrieval
- SQL query execution
- Database-specific operations
- Basic CRUD operations

**Service Layer:**
- Business logic orchestration
- Domain entity validation
- Caching strategy
- Transaction coordination
- Cross-repository operations

### DTO vs Domain Entity

**Use DTOs** (`internal/adapter/inbound/http/handler/dto/`):
- API request/response
- External system communication
- Include validation tags (e.g., `validate:"required"`)
- Include JSON tags

**Use Domain Entities** (`internal/core/domain/`):
- Internal business logic
- Service layer operations
- Repository layer operations
- No JSON tags, only `qwery` tags

### Error Handling Strategy

Uses `silib/fail` and `shared/failure`. See `krangka-fail` skill for details.

**Service Layer:**
```go
// Wrap errors when adding context; propagate otherwise
return nil, fail.Wrap(err)
```

**Repository Layer:**
```go
if errors.Is(err, sql.ErrNoRows) {
    return nil, fail.Wrap(err).WithFailure(failure.ErrNoteNotFound)
}
return nil, fail.Wrap(err) // always wrap to record stack trace
```

**Handler Layer:**
```go
if err := req.Validate(); err != nil {
    return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
}
```

**Typed Failures** (in `shared/failure/failure.go`):
```go
var ErrNoteNotFound = &fail.Failure{Code: "404002", Message: "Note not found", HTTPStatus: 404}
```

## Common Patterns

### Pattern 1: CRUD Operations

Same as "Adding a New Feature" — domain → failures → ports → mocks → **service tests (TDD)** → service → repository → DTOs → **handler tests (TDD)** → handler → router → bootstrap.

### Pattern 2: Transaction Management

The callback receives the transactional `repo` (not a separate context). Use `repo.GetXRepository()` inside the callback to get transaction-scoped repositories.

```go
func (s *Service) CreateUserWithProfile(ctx context.Context, user *domain.User) error {
    _, err := s.repo.DoInTransaction(ctx, func(repo outbound.Repository) (any, error) {
        if err := repo.GetUserRepository().Create(ctx, user); err != nil {
            return nil, err
        }
        if err := repo.GetProfileRepository().Create(ctx, user.Profile); err != nil {
            return nil, err
        }
        return nil, nil
    })
    return err
}
```

### Pattern 3: Caching Strategy

Use `GetObject` / `Set` for domain entities. `Set` expects `expiration` in seconds (int).

```go
func (s *Service) GetNoteByID(ctx context.Context, id string) (*domain.Note, error) {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    cacheKey := "note:" + id
    var cached domain.Note
    if err := s.cache.GetObject(ctx, cacheKey, &cached); err == nil {
        return &cached, nil
    }

    note, err := s.repo.GetNoteRepository().GetNoteByID(ctx, id)
    if err != nil {
        return nil, fail.Wrap(err)
    }

    _ = s.cache.Set(ctx, cacheKey, note, 3600) // 3600 seconds = 1 hour
    return note, nil
}
```

### Pattern 4: Soft Delete

Always use soft delete with Unix timestamps:

**Domain:**
```go
DeletedAt int `qwery:"deleted_at"` // 0 = active, >0 = deleted timestamp
```

**SQL Queries:**
```sql
-- All SELECT queries
WHERE deleted_at = 0

-- DELETE operation
UPDATE table SET deleted_at = UNIX_TIMESTAMP() WHERE deleted_at = 0 AND id = ?
```

## Anti-Patterns to Avoid

### ❌ Skipping TDD

**Bad:** Implementing service or handler before writing tests. Jumping straight to implementation.

**Good:** Write all test scenarios first (red), then implement to make them pass (green). Tests define expected behavior; implementation satisfies them. This applies to every new feature — see `krangka-engineering-principles` for the TDD principle.

### ❌ Business Logic in Handlers

**Bad:**
```go
func (h *NoteHandler) CreateNote(c fiber.Ctx) error {
    if note.Title == "" {
        return errors.New("title required")
    }
    // ...
}
```

**Good:**
```go
func (h *NoteHandler) CreateNote(c fiber.Ctx) error {
    note := req.Transform()
    return h.svc.CreateNote(c.Context(), note) // service validates
}
```

### ❌ Domain Depends on External Frameworks

**Bad:**
```go
type Note struct {
    ID    string `json:"id" db:"id" gorm:"primaryKey"`
    Title string `json:"title" db:"title"`
}
```

**Good:**
```go
type Note struct {
    ID    string `qwery:"id"`
    Title string `qwery:"title"`
}
```

### ❌ Services Depend on Concrete Implementations

**Bad:**
```go
type Service struct {
    repo *mariadb.NoteRepository // concrete type
}
```

**Good:**
```go
type Service struct {
    repo outbound.Repository // interface
}
```

### ❌ Repository Contains Business Logic

**Bad:**
```go
func (r *noteRepository) CreateNote(ctx context.Context, note *domain.Note) error {
    if note.Title == "" {
        return errors.New("title required") // business logic
    }
    return r.db.Create(note)
}
```

**Good:**
```go
func (r *noteRepository) CreateNote(ctx context.Context, note *domain.Note) error {
    query := `INSERT INTO notes (id, title, content) VALUES ({{ .id }}, {{ .title }}, {{ .content }})`
    err := r.qwery.RunRaw(query).WithParams(note).Query(ctx)
    return fail.Wrap(err)
}
```

## Architectural Review Checklist

When reviewing code or designing features, verify:

### Domain Layer
- [ ] No external dependencies (no HTTP, DB, framework imports)
- [ ] Only `qwery` tags for database mapping
- [ ] Plain structs (no business logic or validation methods)
- [ ] Filter struct for list operations
- [ ] Soft delete field present (`DeletedAt int`)

### Port Interfaces
- [ ] Inbound ports define service contracts
- [ ] Outbound ports define repository contracts
- [ ] All methods include `context.Context`
- [ ] Return domain entities, not DTOs

### Service Layer
- [ ] Depends on port interfaces, not concrete types
- [ ] Contains business logic orchestration
- [ ] Validates domain entities
- [ ] Handles caching appropriately
- [ ] Manages transactions when needed

### Adapter Layer
- [ ] HTTP handlers use DTOs for request/response
- [ ] Repositories implement outbound port interfaces
- [ ] Error handling uses `fail` package (silib/fail, shared/failure)
- [ ] Proper transformation between DTOs and domain entities

### Dependency Flow
- [ ] Dependencies point inward
- [ ] No circular dependencies
- [ ] Adapters depend on ports (interfaces)
- [ ] Core is independent of external systems

## Quick Reference

### File Locations
```
internal/
├── core/
│   ├── domain/entity.go              # Pure business entities
│   ├── port/
│   │   ├── inbound/entity.go         # Service interfaces
│   │   └── outbound/
│   │       └── repositories/entity.go # Repository interfaces
│   └── service/entity/service.go     # Business logic
├── mocks/
│   ├── inbound/                      # Generated mocks for inbound ports
│   └── outbound/
│       └── repositories/             # Generated mocks for repository ports
└── adapter/
    ├── inbound/http/
    │   └── handler/
    │       ├── dto/entity.go         # Request/Response DTOs
    │       └── entity.go             # HTTP handlers
    └── outbound/mariadb/
        └── repositories/entity.go   # Repository implementation

shared/
└── failure/failure.go               # Typed error definitions
```

### Key Imports by Layer

**Domain:** None (except time)
**Ports:** Domain entities only
**Services:** Domain, Ports, Config, Logger, tracer, fail
**Handlers:** DTOs, Service interfaces (inbound ports), fail
**Repositories:** Domain, qwery, fail, failure, tracer

## Additional Resources

For detailed information, read:
- `.krangka/docs/01_architecture-overview.md` - Hexagonal architecture concepts
- `.krangka/docs/02_project-structure.md` - Project organization
- `.krangka/docs/03_core-components.md` - Layer responsibilities and examples
- `.krangka/docs/04_adding-new-features.md` - Step-by-step feature guide

Related skills: `krangka-engineering-principles` (foundational philosophy & hard refusal list), `krangka-bootstrap`, `krangka-dependency-wiring`, `krangka-fail`, `krangka-pagination`, `krangka-repository`, `krangka-subscriber`, `krangka-worker`

## Summary

When making architectural decisions:
1. **Start with domain** - Define pure business entities
2. **Define contracts** - Create port interfaces
3. **Implement logic** - Write services with business rules
4. **Add adapters** - Implement handlers and repositories
5. **Verify flow** - Ensure dependencies point inward
6. **Review checklist** - Validate against architectural principles