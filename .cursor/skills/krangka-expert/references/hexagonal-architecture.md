# Hexagonal Architecture

> For full architecture concepts and diagrams, see [.krangka/docs/01_architecture-overview.md](../.krangka/docs/01_architecture-overview.md) and [.krangka/docs/03_core-components.md](../.krangka/docs/03_core-components.md).

## Dependency Direction Rule (Critical)

Dependencies **always point inward**, never outward:

```
External Systems → Inbound Adapters → Inbound Ports → Services → Outbound Ports → Outbound Adapters → External Systems
```

**Valid:**
- ✅ Adapters depend on Ports (interfaces)
- ✅ Services depend on Domain entities and Outbound Ports

**Invalid:**
- ❌ Domain depends on Services
- ❌ Services depend on Adapters (concrete implementations)
- ❌ Domain depends on external frameworks (HTTP, DB, etc.)

## Layer Overview

```
internal/
├── core/               # Pure business logic — NO framework imports
│   ├── domain/         # Entities and value objects
│   ├── port/
│   │   ├── inbound/    # Use case interfaces (what the app exposes)
│   │   └── outbound/   # Dependency interfaces (what the app needs)
│   │       └── repositories/
│   └── service/        # Use case implementations
├── mocks/              # Generated mocks for testing
│   ├── inbound/
│   └── outbound/
│       └── repositories/
└── adapter/
    ├── inbound/        # HTTP, CLI, events → drives the app
    │   ├── http/
    │   ├── subscriber/
    │   └── worker/
    └── outbound/       # DB, cache, events → driven by the app
        ├── mariadb/
        │   └── repositories/
        └── redis/
```

## Import Boundary Rules (Strictly Enforced)

| Layer | Can Import | Cannot Import |
|---|---|---|
| `domain` | stdlib only | anything else |
| `port` | `domain`, stdlib, `sikat`, port sub-packages | `adapter`, `service` |
| `service` | `domain`, `port`, `configs`, `silib` | `adapter`, DB drivers, HTTP |
| `adapter/inbound` | `port/inbound`, `domain`, `silib` | `adapter/outbound` directly |
| `adapter/outbound` | `port/outbound`, `domain`, DB drivers | `adapter/inbound`, `service` |

**The core must never know about adapters.**

## Test Boundary Rules

- Tests in `core/service/` → only mock `port/outbound` interfaces, never import `adapter/outbound`
- Tests in `adapter/inbound/http/` → only mock `port/inbound` interfaces
- Tests in `adapter/outbound/mariadb/` → integration tests only, use real DB

**Regenerate mocks:** `make mock`

## Layers in Detail

### Layer 1: Domain (Core)

Pure business entities — no external dependencies.

```go
// internal/core/domain/note.go
type Note struct {
    ID        string    `sikat:"id"`
    Title     string    `sikat:"title"`
    Content   string    `sikat:"content"`
    CreatedAt time.Time `sikat:"created_at"`
    UpdatedAt time.Time `sikat:"updated_at"`
    DeletedAt int       `sikat:"deleted_at"`
}

type NoteFilter struct {
    Search string `sikat:"search"`
}
```

Rules:
- NO external dependencies (no HTTP, DB, framework imports)
- NO JSON tags (use DTOs for serialization)
- ONLY `sikat` tags for database mapping
- Plain data structs — no business logic or validation methods
- Include `DeletedAt int` for soft delete
- Include `Filter` struct for list operations

### Layer 2: Port Interfaces

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

**Outbound Ports** (`internal/core/port/outbound/repository.go`):
```go
type Repository interface {
    DoInTransaction(ctx context.Context, fn func(repo Repository) (any, error)) (any, error)
    PublishOutbox(ctx context.Context, target PublisherTarget, topic string, payload sikat.JSONMap) error
    RetryOutbox(ctx context.Context) error
    GetNoteRepository() repositories.Note
}
```

Add `//go:generate mockgen` to each port file.

### Layer 3: Services (Use Cases)

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
    // ...
}
```

Always `tracer.Trace(ctx)` + `defer span.End()` in every method.

### Layer 4: Adapters

- **Inbound**: HTTP handlers, subscriber handlers, worker handlers
- **Outbound**: Database repositories, cache implementations, external API clients

## Naming Conventions

| Thing | Convention | Example |
|---|---|---|
| Service struct | `Service` | `type Service struct` |
| Constructor | `NewService` | `func NewService(...) *Service` |
| Port interface | noun | `type Note interface` |
| Mock package | `mocks` | `package mocks` |

## Common Anti-Patterns

### ❌ Business Logic in Handlers
```go
// BAD
func (h *NoteHandler) CreateNote(c fiber.Ctx) error {
    if note.Title == "" { return errors.New("title required") }
}

// GOOD
func (h *NoteHandler) CreateNote(c fiber.Ctx) error {
    note := req.Transform()
    return h.svc.CreateNote(c.Context(), note) // service validates
}
```

### ❌ Domain with External Framework Tags
```go
// BAD
type Note struct {
    ID string `json:"id" db:"id" gorm:"primaryKey"`
}

// GOOD
type Note struct {
    ID string `sikat:"id"`
}
```

### ❌ Service Depends on Concrete Implementation
```go
// BAD
type Service struct { repo *mariadb.NoteRepository }

// GOOD
type Service struct { repo outbound.Repository }
```

## Architectural Review Checklist

- [ ] Domain has no external dependencies, only `sikat` tags
- [ ] `DeletedAt int` present in all entities
- [ ] Filter struct for list operations
- [ ] Inbound ports define service contracts
- [ ] Outbound ports define repository contracts
- [ ] All methods include `context.Context`
- [ ] Service depends on port interfaces, not concrete types
- [ ] HTTP handlers use DTOs for request/response
- [ ] Error handling uses `fail` package
- [ ] Dependencies point inward, no circular dependencies
