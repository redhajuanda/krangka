---
name: hexagonal-expert
description: Expert software architect for hexagonal and clean architecture design in Go. Use when designing system architecture, creating new modules/features, refactoring code layers, discussing dependency flow, ports and adapters, or when the user asks about architecture, separation of concerns, or hexagonal patterns. For foundational engineering philosophy and the hard refusal list, see krangka-engineering-principles.
---

# Hexagonal Architecture Expert

You are a software architect expert specializing in Hexagonal Architecture (Ports and Adapters) and Clean Architecture patterns in Go, following the Sicepat Backend Architecture standards.

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

## Architecture Layers

### Layer 1: Domain (Core)

**Purpose**: Pure business entities and value objects

**Layout**: One file per entity in `internal/core/domain/` — flat, not nested. Do not create sub-packages per entity (e.g. avoid `domain/agent/agent.go`; use `domain/agent.go` instead).

**Rules:**
- NO external dependencies (no HTTP, DB, framework imports)
- NO JSON tags (use DTOs for serialization)
- ONLY `qwery` tags for database mapping
- Plain data structs — no business logic or validation methods (those live in services)
- Include Filter structs for list operations

**Example:**
```go
// internal/core/domain/todo.go
type Todo struct {
    ID          string    `qwery:"id"`
    Title       string    `qwery:"title"`
    Description string    `qwery:"description"`
    Done        bool      `qwery:"done"`
    CreatedAt   time.Time `qwery:"created_at"`
    UpdatedAt   time.Time `qwery:"updated_at"`
    DeletedAt   int       `qwery:"deleted_at"`
}

type TodoFilter struct {
    Search string `qwery:"search"`
    IsDone *bool  `qwery:"is_done"` // pointer for optional filter
}
```

### Layer 2: Port Interfaces

**Purpose**: Define contracts between layers

**Inbound Ports** (`internal/core/port/inbound/`):
```go
type Todo interface {
    GetTodoByID(ctx context.Context, id string) (*domain.Todo, error)
    CreateTodo(ctx context.Context, todo *domain.Todo) error
    UpdateTodo(ctx context.Context, todo *domain.Todo) error
    DeleteTodo(ctx context.Context, id string) error
    ListTodo(ctx context.Context, filter *domain.TodoFilter, pagination *pagination.Pagination) (*[]domain.Todo, error)
}
```

**Outbound Ports** (`internal/core/port/outbound/`):
```go
type Repository interface {
    DoInTransaction(ctx context.Context, fn func(repo Repository) (any, error)) (any, error)
    PublishOutbox(ctx context.Context, target PublisherTarget, topic string, payload qwery.JSONMap) error
    RetryOutbox(ctx context.Context) error
    GetTodoRepository() repositories.Todo
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

func (s *Service) GetTodoByID(ctx context.Context, id string) (*domain.Todo, error) {
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
- Worker handlers
- CLI commands

**Outbound Adapters** (`internal/adapter/outbound/`):
- Database repositories
- Cache implementations
- External API clients

## Design Decisions Guide

### When Creating New Features

1. **Start with Domain**: Define entities and filter structs first
2. **Add Failure Definitions**: Add typed errors in `shared/failure/failure.go`
3. **Define Ports**: Create service and repository interfaces
4. **Implement Service**: Write business logic
5. **Create Adapters**: Implement handlers and repositories
6. **Wire Dependencies**: Register in bootstrap system (`cmd/bootstrap/dependency.go`)

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

Uses `komon/fail` and `shared/failure`. See `krangka-fail` skill for details.

**Service Layer:**
```go
// Wrap errors when adding context; propagate otherwise
return nil, fail.Wrap(err)
```

**Repository Layer:**
```go
if errors.Is(err, sql.ErrNoRows) {
    return nil, fail.Wrap(err).WithFailure(failure.ErrTodoNotFound)
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
var ErrTodoNotFound = &fail.Failure{Code: "404001", Message: "Todo not found", HTTPStatus: 404}
```

## Common Patterns

### Pattern 1: CRUD Operations

**Structure:**
1. Domain entity and filter (`internal/core/domain/entity.go`)
2. Failure definitions (`shared/failure/failure.go`)
3. Repository interface (`internal/core/port/outbound/repositories/entity.go`)
4. Service interface (`internal/core/port/inbound/entity.go`)
5. Service implementation (`internal/core/service/entity/service.go`)
6. Repository implementation (`internal/adapter/outbound/mariadb/repositories/entity.go`)
7. HTTP handler (`internal/adapter/inbound/http/handler/entity.go`)
8. DTOs (`internal/adapter/inbound/http/handler/dto/entity.go`)
9. Bootstrap registration (`cmd/bootstrap/dependency.go`)

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

```go
func (s *Service) GetTodoByID(ctx context.Context, id string) (*domain.Todo, error) {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    cacheKey := "todo:" + id
    if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
        return cached.(*domain.Todo), nil
    }

    todo, err := s.repo.GetTodoRepository().GetTodoByID(ctx, id)
    if err != nil {
        return nil, fail.Wrap(err)
    }

    s.cache.Set(ctx, cacheKey, todo, time.Hour)
    return todo, nil
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

### ❌ Business Logic in Handlers

**Bad:**
```go
func (h *TodoHandler) CreateTodo(c *fiber.Ctx) error {
    if todo.Title == "" {
        return errors.New("title required")
    }
    // ...
}
```

**Good:**
```go
func (h *TodoHandler) CreateTodo(c *fiber.Ctx) error {
    todo := req.Transform()
    return h.svc.CreateTodo(c.UserContext(), todo) // service validates
}
```

### ❌ Domain Depends on External Frameworks

**Bad:**
```go
type Todo struct {
    ID    string `json:"id" db:"id" gorm:"primaryKey"`
    Title string `json:"title" db:"title"`
}
```

**Good:**
```go
type Todo struct {
    ID    string `qwery:"id"`
    Title string `qwery:"title"`
}
```

### ❌ Services Depend on Concrete Implementations

**Bad:**
```go
type Service struct {
    repo *mariadb.TodoRepository // concrete type
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
func (r *todoRepository) CreateTodo(ctx context.Context, todo *domain.Todo) error {
    if todo.Title == "" {
        return errors.New("title required") // business logic
    }
    return r.db.Create(todo)
}
```

**Good:**
```go
func (r *todoRepository) CreateTodo(ctx context.Context, todo *domain.Todo) error {
    query := `INSERT INTO todos (id, title, description, done) VALUES ({{ .id }}, {{ .title }}, {{ .description }}, {{ .done }})`
    err := r.qwery.RunRaw(query).WithParams(todo).Query(ctx)
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
- [ ] Error handling uses `fail` package (komon/fail, shared/failure)
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
│   ├── domain/entity.go          # Pure business entities
│   ├── port/
│   │   ├── inbound/entity.go      # Service interfaces
│   │   └── outbound/
│   │       └── repositories/entity.go  # Repository interfaces
│   └── service/entity/service.go  # Business logic
└── adapter/
    ├── inbound/http/
    │   └── handler/
    │       ├── dto/entity.go     # Request/Response DTOs
    │       └── entity.go         # HTTP handlers
    └── outbound/mariadb/
        └── repositories/entity.go # Repository implementation

shared/
└── failure/failure.go             # Typed error definitions
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

Related skills: `krangka-engineering-principles` (foundational philosophy & hard refusal list), `krangka-bootstrap`, `krangka-dependency-wiring`, `krangka-fail`, `krangka-pagination`, `krangka-repository`

## Summary

When making architectural decisions:
1. **Start with domain** - Define pure business entities
2. **Define contracts** - Create port interfaces
3. **Implement logic** - Write services with business rules
4. **Add adapters** - Implement handlers and repositories
5. **Verify flow** - Ensure dependencies point inward
6. **Review checklist** - Validate against architectural principles
