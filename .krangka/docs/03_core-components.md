# Core Components

This document explains the core components of the Krangka application following hexagonal architecture principles. Understanding these components is essential for building maintainable and testable applications.

## Overview

The core components are organized into four main layers:

1. **Domain Layer** - Pure business entities and logic
2. **Port Interfaces** - Contracts between layers
3. **Service Layer** - Business logic orchestration
4. **Adapter Layer** - External system implementations

## 1. Domain Layer

The domain layer contains the heart of your business logic - the entities, value objects, and business rules that define what your application does.

### Domain Entities

Domain entities represent the core business objects in your application. They are pure Go structs with no external dependencies.

#### Example: Note Entity

```go
// internal/core/domain/note.go
package domain

import "time"

type Note struct {
    ID        string    `sikat:"id"`
    Title     string    `sikat:"title"`
    Content   string    `sikat:"content"`
    CreatedAt time.Time `sikat:"created_at"`
    UpdatedAt time.Time `sikat:"updated_at"`
    DeletedAt int       `sikat:"deleted_at"`
}

// Filter struct for list operations
type NoteFilter struct {
    Search string `sikat:"search"`
}
```

### Domain Best Practices

#### 1. Pure Business Logic
- No external dependencies (no HTTP, database, or framework imports)
- No JSON tags (use DTOs for serialization)
- Only use `sikat` tags for database mapping

#### 2. Keep It Simple
- Domain entities are plain data structs
- Avoid embedding heavy business logic methods in entities — that belongs in the service layer
- Business rules and validation live in services, not entities

#### 3. Immutability
- Keep entities immutable where possible
- Prefer creating new instances rather than mutating existing ones

#### 4. Value Objects
- Use value objects for complex business concepts
- Implement proper equality and comparison methods

#### 5. Timestamp Handling
- **Check database schema first**: Determine if timestamps are handled by database
- **Database-managed timestamps**: Don't set `CreatedAt`/`UpdatedAt` in application code
- **Application-managed timestamps**: Set timestamps manually when database doesn't handle them
- **Consistency**: Use the same approach across all entities in the application

**Guideline**: Always check the database schema first before deciding whether to set created_at and updated_at in application code:
- ✅ Database handles created_at and updated_at → Don't set in code
- ❌ Database doesn't handle created_at and updated_at → Set manually in code

#### 6. Error Handling
- Domain entities should not return errors directly
- Validation and domain error handling belongs in the service layer
- Use `shared/failure` for typed application-level errors (defined centrally)

### Soft Delete Implementation

The application implements soft delete functionality to preserve data integrity and enable data recovery. This is implemented consistently across all entities.

All domain entities include a `DeletedAt` field that stores Unix timestamps:

```go
type Note struct {
    ID        string    `sikat:"id"`
    Title     string    `sikat:"title"`
    Content   string    `sikat:"content"`
    CreatedAt time.Time `sikat:"created_at"`
    UpdatedAt time.Time `sikat:"updated_at"`
    DeletedAt int       `sikat:"deleted_at"`  // Soft delete field (Unix timestamp)
}
```

The `deleted_at` field is implemented as an integer storing Unix timestamps:
- `0`: Record is active (not deleted)
- `> 0`: Unix timestamp when the record was soft deleted

**Why Unix Timestamps?**
- **Precise Timing**: Exact deletion time for audit trails
- **Time Zone Independent**: No timezone conversion issues
- **Easy Comparison**: Simple integer comparisons for queries
- **Storage Efficient**: Smaller than datetime fields
- **Debugging**: Easy to convert to readable dates when needed

All queries automatically filter out soft-deleted records:

**Select Queries:**
```sql
-- Get by ID
SELECT id, title, content, created_at, updated_at
FROM notes
WHERE deleted_at = 0
AND id = {{ .id }}

-- List with filters
SELECT id, title, content, created_at, updated_at
FROM notes
WHERE deleted_at = 0
{{ if .search }} AND (title LIKE CONCAT('%', {{ .search }}, '%') OR content LIKE CONCAT('%', {{ .search }}, '%')) {{ end }}
```

**Update Queries:**
```sql
-- Update existing record
UPDATE notes 
SET title = {{ .title }}, content = {{ .content }}
WHERE deleted_at = 0
AND id = {{ .id }}
```

**Delete Queries:**
```sql
-- Soft delete (mark as deleted with timestamp)
UPDATE notes SET deleted_at = UNIX_TIMESTAMP() WHERE deleted_at = 0 AND id = {{ .id }}
```

#### Benefits of Soft Delete

1. **Data Recovery**: Deleted records can be restored if needed
2. **Audit Trail**: Maintains history of all operations with precise deletion timestamps
3. **Referential Integrity**: Preserves relationships between entities
4. **Compliance**: Meets regulatory requirements for data retention
5. **Analytics**: Enables historical data analysis
6. **Timing Information**: Unix timestamps provide exact deletion timing for debugging and compliance

#### Implementation Guidelines

1. **Always include `WHERE deleted_at = 0`** in SELECT and UPDATE queries
2. **Use UPDATE with `deleted_at = UNIX_TIMESTAMP()`** instead of DELETE statements
3. **Never expose `deleted_at` field** in API responses (use DTOs)
4. **Consider adding indexes** on `deleted_at` for large tables

## 2. Port Interfaces

Port interfaces define the contracts between the core application and external systems. They are the "ports" in hexagonal architecture.

### Inbound Ports

Inbound ports define what external systems can do to drive your application.

#### Example: Note Service Interface

```go
// internal/core/port/inbound/note.go
package inbound

import (
    "context"
    "github.com/redhajuanda/krangka/internal/core/domain"
    "github.com/redhajuanda/komon/pagination"
)

type Note interface {
    // GetNoteByID retrieves a note item by its ID
    GetNoteByID(ctx context.Context, id string) (*domain.Note, error)
    // CreateNote creates a new note item
    CreateNote(ctx context.Context, note *domain.Note) error
    // UpdateNote updates an existing note item
    UpdateNote(ctx context.Context, note *domain.Note) error
    // DeleteNote deletes a note item by its ID
    DeleteNote(ctx context.Context, id string) error
    // ListNote retrieves a list of note items with pagination
    ListNote(ctx context.Context, req *domain.NoteFilter, pagination *pagination.Pagination) (*[]domain.Note, error)
}
```

### Outbound Ports

Outbound ports define what your application needs from external systems.

#### Repository Interfaces

```go
// internal/core/port/outbound/repositories/note.go
package repositories

import (
    "context"
    "github.com/redhajuanda/krangka/internal/core/domain"
    "github.com/redhajuanda/komon/pagination"
)

type Note interface {
    // GetNoteByID retrieves a note item by its ID
    GetNoteByID(ctx context.Context, id string) (*domain.Note, error)
    // CreateNote creates a new note item
    CreateNote(ctx context.Context, note *domain.Note) error
    // UpdateNote updates an existing note item
    UpdateNote(ctx context.Context, note *domain.Note) error
    // DeleteNote deletes a note item by its ID
    DeleteNote(ctx context.Context, id string) error
    // ListNote retrieves a list of note items with pagination
    ListNote(ctx context.Context, req *domain.NoteFilter, pagination *pagination.Pagination) (*[]domain.Note, error)
}
```

#### Main Repository Interface

```go
// internal/core/port/outbound/repository.go
package outbound

import (
    "context"
    "github.com/redhajuanda/krangka/internal/core/port/outbound/repositories"
    "github.com/redhajuanda/qwery"
)

type Repository interface {
    // DoInTransaction executes a function in a transaction
    DoInTransaction(ctx context.Context, fn func(repo Repository) (any, error)) (any, error)
    // PublishOutbox publishes an outbox event
    PublishOutbox(ctx context.Context, target PublisherTarget, topic string, payload sikat.JSONMap) error
    // RetryOutbox retries an outbox event
    RetryOutbox(ctx context.Context) error
    // GetNoteRepository returns the NoteRepository instance
    GetNoteRepository() repositories.Note
}
```

#### Cache Interface

```go
// internal/core/port/outbound/cache.go
package outbound

import "github.com/redhajuanda/komon/cache"

type Cache cache.Cache
```

#### Messaging Interfaces

```go
// internal/core/port/outbound/publisher.go
package outbound

import (
    "context"
    "github.com/ThreeDotsLabs/watermill/message"
)

// Publisher is the emitting part of a Pub/Sub (Watermill contract)
type Publisher interface {
    Publish(topic string, messages ...*message.Message) error
    Close() error
}

// Publishers is a map of publishers by target
type Publishers map[PublisherTarget]Publisher

type PublisherTarget string

const (
    PublisherTargetRedisstream PublisherTarget = "redisstream"
    PublisherTargetKafka       PublisherTarget = "kafka"
)

// internal/core/port/outbound/subscriber.go
// Subscriber is the consuming part of the Pub/Sub (Watermill contract)
type Subscriber interface {
    Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error)
    Close() error
}
```

#### Idempotency Interface

Used by the subscriber middleware to ensure at-most-once processing of events.

```go
// internal/core/port/outbound/idempotency.go
package outbound

import (
    "context"
    "time"
)

type Idempotency interface {
    // TryClaim atomically claims the idempotency key for the given topic and message ID.
    // Returns true if claimed (caller should process), false if already processed (caller should skip/ACK).
    TryClaim(ctx context.Context, topic, messageID string, ttl time.Duration) (claimed bool, err error)
}
```

#### Distributed Lock Interface

```go
// internal/core/port/outbound/dlock.go
package outbound

import "github.com/redhajuanda/komon/lock"

// DLocker is a contract for distributed locking
type DLocker lock.DLocker
```

### Port Interface Best Practices

#### 1. Clear Contracts
- Define exactly what each method should do
- Use descriptive method names
- Include proper error handling

#### 2. Context Support
- Always include `context.Context` as the first parameter
- Support cancellation and timeouts

#### 3. Domain-Centric
- Use domain entities in method signatures
- Avoid exposing implementation details

#### 4. Consistent Patterns
- Follow consistent naming conventions
- Use similar patterns across different interfaces

## 3. Service Layer

The service layer implements the business logic and orchestrates domain operations. Services are the "use cases" of your application.

### Service Implementation

#### Example: Note Service

```go
// internal/core/service/note/service.go
package note

import (
    "context"

    "github.com/redhajuanda/krangka/configs"
    "github.com/redhajuanda/krangka/internal/core/domain"
    "github.com/redhajuanda/krangka/internal/core/port/outbound"
    "github.com/redhajuanda/komon/fail"
    "github.com/redhajuanda/komon/logger"
    "github.com/redhajuanda/komon/pagination"
    "github.com/redhajuanda/komon/tracer"
)

type Service struct {
    cfg   *configs.Config
    log   logger.Logger
    repo  outbound.Repository
    cache outbound.Cache
}

func NewService(cfg *configs.Config, log logger.Logger, repo outbound.Repository, cache outbound.Cache) *Service {
    return &Service{
        cfg:   cfg,
        log:   log,
        repo:  repo,
        cache: cache,
    }
}

func (s *Service) GetNoteByID(ctx context.Context, id string) (*domain.Note, error) {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    var (
        repoNote = s.repo.GetNoteRepository()
    )

    note, err := repoNote.GetNoteByID(ctx, id)
    if err != nil {
        return nil, fail.Wrap(err)
    }
    return note, nil
}

func (s *Service) CreateNote(ctx context.Context, note *domain.Note) error {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    var (
        repoNote = s.repo.GetNoteRepository()
    )

    return repoNote.CreateNote(ctx, note)
}

func (s *Service) UpdateNote(ctx context.Context, note *domain.Note) error {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    var (
        repoNote = s.repo.GetNoteRepository()
    )

    return repoNote.UpdateNote(ctx, note)
}

func (s *Service) DeleteNote(ctx context.Context, id string) error {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    var (
        repoNote = s.repo.GetNoteRepository()
    )

    return repoNote.DeleteNote(ctx, id)
}

func (s *Service) ListNote(ctx context.Context, req *domain.NoteFilter, pagination *pagination.Pagination) (*[]domain.Note, error) {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    var (
        repoNote = s.repo.GetNoteRepository()
    )

    res, err := repoNote.ListNote(ctx, req, pagination)
    if err != nil {
        return nil, err
    }
    return res, err
}
```

### Service Best Practices

#### 1. Always Use Distributed Tracing
- Call `tracer.Trace(ctx)` at the start of every service method
- Always `defer span.End()` immediately after

#### 2. Business Logic Orchestration
- Coordinate between different domain entities
- Implement complex business workflows
- Keep methods focused on a single use case

#### 3. Error Handling
- Wrap errors with `fail.Wrap(err)` when you need to add stack trace context
- Propagate errors from repositories without re-wrapping unless adding context
- Do not map domain errors here; mapping is done in the repository layer

#### 4. Transaction Management
- Use `repo.DoInTransaction()` when multiple writes must succeed together
- Ensure data consistency
- Rollback on errors

#### 5. No Direct External Calls
- Services must only call outbound ports (interfaces)
- Never import adapter packages directly
- This keeps services independently testable

## 4. Adapter Layer

Adapters implement the port interfaces and handle communication with external systems.

### Inbound Adapters

Inbound adapters are the entry points to your application.

#### HTTP Adapter

```go
// internal/adapter/inbound/http/handler/note.go
package handler

import (
    "github.com/redhajuanda/krangka/configs"
    "github.com/redhajuanda/krangka/internal/adapter/inbound/http/handler/dto"
    "github.com/redhajuanda/krangka/internal/adapter/inbound/http/response"
    "github.com/redhajuanda/krangka/internal/core/port/inbound"
    "github.com/redhajuanda/komon/fail"
    "github.com/redhajuanda/komon/logger"
    "github.com/gofiber/fiber/v2"
)

type NoteHandler struct {
    cfg *configs.Config
    log logger.Logger
    svc inbound.Note
}

func NewNoteHandler(cfg *configs.Config, log logger.Logger, svc inbound.Note) *NoteHandler {
    return &NoteHandler{
        cfg: cfg,
        log: log,
        svc: svc,
    }
}

func (h *NoteHandler) RegisterRoutes(app *fiber.App) {
    app.Get("/notes/:id", h.GetNoteByID)
    app.Post("/notes", h.CreateNote)
    app.Put("/notes/:id", h.UpdateNote)
    app.Delete("/notes/:id", h.DeleteNote)
    app.Get("/notes", h.ListNotes)
}

// GetNoteByID godoc
// @Summary      Get Note by ID
// @Description  Retrieves a note by its id
// @Tags         Notes
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Note ID"
// @Success      200  {object}  response.ResponseSuccess{data=dto.ResGetNoteByID}
// @Failure      400  {object}  response.ResponseFailed{}
// @Failure      404  {object}  response.ResponseFailed{}
// @Failure      500  {object}  response.ResponseFailed{}
// @Router       /notes/{id} [get]
func (h *NoteHandler) GetNoteByID(c *fiber.Ctx) error {

    var (
        req dto.ReqGetNoteByID
        res dto.ResGetNoteByID
        ctx = c.UserContext()
    )

    if err := c.ParamsParser(&req); err != nil {
        return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
    }

    if err := req.Validate(); err != nil {
        return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
    }

    note, err := h.svc.GetNoteByID(ctx, req.ID)
    if err != nil {
        return err
    }

    res.Transform(note)

    return response.SuccessOK(c, res, "Note retrieved successfully")
}

func (h *NoteHandler) CreateNote(c *fiber.Ctx) error {

    var (
        req = dto.ReqCreateNote{}
        res = dto.ResCreateNote{}
        ctx = c.UserContext()
    )

    if err := c.BodyParser(&req); err != nil {
        return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
    }

    if err := req.Validate(); err != nil {
        return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
    }

    note := req.Transform()

    err := h.svc.CreateNote(ctx, note)
    if err != nil {
        return err
    }

    res.Transform(note)

    return response.SuccessCreated(c, res, "Note created successfully")
}
```

#### Subscriber Adapter

The subscriber adapter is an event-driven inbound adapter that consumes messages from Kafka or Redis Streams via Watermill. Handlers register routes for topics; middleware (idempotence, retry, request ID) runs before handlers.

```go
// internal/adapter/inbound/subscriber/handler/note.go
package handler

import (
    "github.com/ThreeDotsLabs/watermill/message"
    "github.com/redhajuanda/krangka/configs"
    "github.com/redhajuanda/komon/logger"
)

type NoteHandler struct {
    cfg        *configs.Config
    log        logger.Logger
    subscriber message.Subscriber
}

func NewNoteHandler(cfg *configs.Config, log logger.Logger, subscriber message.Subscriber) *NoteHandler {
    return &NoteHandler{cfg: cfg, log: log, subscriber: subscriber}
}

// RegisterRoutes registers event handlers for note topics
func (h *NoteHandler) RegisterRoutes(router *message.Router) {
    router.AddConsumerHandler("NOTE_CREATED", "note.created", h.subscriber, h.HandleNoteCreated)
    router.AddConsumerHandler("NOTE_UPDATED", "note.updated", h.subscriber, h.HandleNoteUpdated)
    router.AddConsumerHandler("NOTE_DELETED", "note.deleted", h.subscriber, h.HandleNoteDeleted)
}

func (h *NoteHandler) HandleNoteCreated(msg *message.Message) error {
    // Process event; call inbound.Note service if needed
    return nil
}
```

**Subscriber vs Worker**: The subscriber is event-driven (Kafka/Redis Streams); the worker runs time-triggered or manual jobs (Execute, Schedule, Run).

#### HTTP DTOs

DTOs live in `internal/adapter/inbound/http/handler/dto/` and handle request parsing, validation, and transformation.

```go
// internal/adapter/inbound/http/handler/dto/note.go
package dto

import (
    "time"
    "github.com/redhajuanda/krangka/internal/core/domain"
    "github.com/redhajuanda/komon/pagination"
    "github.com/go-playground/validator/v10"
    "github.com/oklog/ulid/v2"
)

// Request DTOs — parse input and validate
type ReqCreateNote struct {
    Title   string `json:"title"   validate:"required"`
    Content string `json:"content" validate:"required"`
}

func (r *ReqCreateNote) Validate() error {
    return validator.New().Struct(r)
}

// Transform converts the request DTO to a domain entity
func (r *ReqCreateNote) Transform() *domain.Note {
    return &domain.Note{
        ID:      ulid.Make().String(),
        Title:   r.Title,
        Content: r.Content,
    }
}

// Response DTOs — shape the output, never expose domain internals like DeletedAt
type ResGetNoteByID struct {
    ID        string    `json:"id"`
    Title     string    `json:"title"`
    Content   string    `json:"content"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

func (r *ResGetNoteByID) Transform(note *domain.Note) {
    r.ID        = note.ID
    r.Title     = note.Title
    r.Content   = note.Content
    r.CreatedAt = note.CreatedAt
    r.UpdatedAt = note.UpdatedAt
}

// List request — embed pagination directly
type ReqListNote struct {
    pagination.Pagination
    Search string `query:"search" validate:"omitempty,max=100"`
}

func (r *ReqListNote) Transform() *domain.NoteFilter {
    return &domain.NoteFilter{
        Search: r.Search,
    }
}
```

### Outbound Adapters

Outbound adapters handle external system communication.

#### Database Repository

```go
// internal/adapter/outbound/mariadb/repositories/note.go
package repositories

import (
    "context"
    "database/sql"
    "errors"

    "github.com/redhajuanda/krangka/internal/core/domain"
    "github.com/redhajuanda/krangka/shared/failure"
    "github.com/redhajuanda/qwery"
    "github.com/redhajuanda/komon/fail"
    "github.com/redhajuanda/komon/pagination"
    "github.com/redhajuanda/komon/tracer"
)

type noteRepository struct {
    sikat sikat.Runable
}

func NewNoteRepository(sikat sikat.Runable) *noteRepository {
    return &noteRepository{sikat: sikat}
}

func (r *noteRepository) GetNoteByID(ctx context.Context, id string) (*domain.Note, error) {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    var note domain.Note

    query := `
        SELECT 
            id, 
            title, 
            content
        FROM notes
        WHERE id = {{ .id }} 
        AND deleted_at = 0
    `

    err := r.sikat.
        RunRaw(query).
        WithParam("id", id).
        ScanStruct(&note).
        Query(ctx)

    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, fail.Wrap(err).WithFailure(failure.ErrNoteNotFound)
        }
        return nil, fail.Wrap(err)
    }

    return &note, nil
}

func (r *noteRepository) CreateNote(ctx context.Context, note *domain.Note) error {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    query := `
        INSERT INTO notes (id, title, content) 
        VALUES ({{ .id }}, {{ .title }}, {{ .content }})
    `

    err := r.sikat.
        RunRaw(query).
        WithParams(note).
        Query(ctx)

    if err != nil {
        return fail.Wrap(err)
    }

    return nil
}

func (r *noteRepository) UpdateNote(ctx context.Context, note *domain.Note) error {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    query := `
        UPDATE notes 
        SET title = {{ .title }}, content = {{ .content }} 
        WHERE id = {{ .id }} 
        AND deleted_at = 0
    `

    err := r.sikat.
        RunRaw(query).
        WithParams(note).
        Query(ctx)

    if err != nil {
        return fail.Wrap(err)
    }

    return nil
}

func (r *noteRepository) DeleteNote(ctx context.Context, id string) error {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    query := `
        UPDATE notes 
        SET deleted_at = UNIX_TIMESTAMP() 
        WHERE id = {{ .id }} 
        AND deleted_at = 0
    `

    err := r.sikat.
        RunRaw(query).
        WithParam("id", id).
        Query(ctx)

    if err != nil {
        return fail.Wrap(err)
    }

    return nil
}

func (r *noteRepository) ListNote(ctx context.Context, req *domain.NoteFilter, pagination *pagination.Pagination) (*[]domain.Note, error) {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    notes := make([]domain.Note, 0)

    query := `
        SELECT id, title, content, created_at, updated_at, deleted_at
        FROM notes
        WHERE deleted_at = 0
        {{ if .search }} AND (title LIKE CONCAT('%', {{ .search }}, '%') OR content LIKE CONCAT('%', {{ .search }}, '%')) {{ end }}
    `

    err := r.sikat.
        RunRaw(query).
        WithParams(map[string]any{
            "search": req.Search,
        }).
        WithPagination(pagination).
        WithOrderBy("-created_at", "id").
        ScanStructs(&notes).
        Query(ctx)

    if err != nil {
        return nil, fail.Wrap(err)
    }
    return &notes, nil
}
```

### Adapter Best Practices

#### 1. Interface Implementation
- Implement port interfaces exactly
- Don't add methods that aren't in the interface
- Handle all error cases properly

#### 2. Data Transformation
- Convert between external formats and domain entities
- Use DTOs for API request/response handling
- Validate data at the HTTP boundary (handlers), not in domain entities

#### 3. Error Handling
- Always use `fail.Wrap(err)` to wrap errors in the repository layer — this records the stack trace
- Map `sql.ErrNoRows` and similar sentinel errors to typed `failure.*` errors using `.WithFailure(failure.ErrXxx)`
- At the HTTP handler level, wrap parse/validation errors with `fail.Wrap(err).WithFailure(fail.ErrBadRequest)`
- Propagate service errors directly without re-wrapping

#### 4. Repository Best Practices
- Use `RunRaw()` with inline SQL queries
- Write SQL queries directly in repository methods using multi-line strings
- Use Sikat template syntax (`{{ .field }}`) for parameterized queries
- Always call `tracer.Trace(ctx)` and `defer span.End()` at the start of every method
- Always use `WithPagination()` and `WithOrderBy()` when listing data with pagination
- Include `WHERE deleted_at = 0` in all SELECT and UPDATE queries
- Use `{{ if .field }}` for optional filter conditions

#### 5. Performance
- Use connection pooling for databases (handled by Sikat)
- Optimize queries with appropriate indexes

## Error Handling

### The `fail` Package

All errors in Krangka use `github.com/redhajuanda/komon/fail`. This package records stack traces and carries typed failure information.

```go
// Wrap an error to record its stack trace
return fail.Wrap(err)

// Map to a typed failure (changes HTTP status code and error code in response)
return fail.Wrap(err).WithFailure(failure.ErrNoteNotFound)

// Wrap with a built-in failure type (e.g., bad request)
return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
```

### The `shared/failure` Package

Application-specific typed errors are declared centrally in `shared/failure/failure.go`:

```go
package failure

import "github.com/redhajuanda/komon/fail"

var (
    ErrNoteNotFound      = &fail.Failure{Code: "404002", Message: "Note not found",      HTTPStatus: 404}
    ErrNoteAlreadyExists = &fail.Failure{Code: "409002", Message: "Note already exists", HTTPStatus: 409}
)
```

### Error Handling by Layer

| Layer | Pattern |
|---|---|
| Repository | `fail.Wrap(err)` — always wrap; map sentinel errors with `.WithFailure(failure.ErrXxx)` |
| Service | `fail.Wrap(err)` — wrap when adding context; propagate otherwise |
| HTTP Handler | `fail.Wrap(err).WithFailure(fail.ErrBadRequest)` for parse/validation errors; propagate service errors |
| Subscriber Handler | Return error to trigger retry (Nack); return nil to ACK. Use `fail.Wrap(err)` when wrapping |

## Component Interaction

### Dependency Flow

**HTTP (request-driven):**
```
HTTP Handler → Note Service → Note Repository → Database
     ↓              ↓              ↓
  DTOs         Domain Logic    SQL Queries
```

**Subscriber (event-driven):**
```
Event (Kafka/Redis) → Subscriber Handler → [Note Service] → [Repository] → Database
         ↓                    ↓
   Idempotency          Domain Logic
   (middleware)
```

### Example Flow: Creating a Note (HTTP)

1. **HTTP Handler** receives request, parses and validates DTO
2. **DTO** `Transform()` builds the domain entity (generates ID with `ulid.Make()`)
3. **Service** delegates to repository (applies business rules if any)
4. **Repository** persists the entity to database via Sikat
5. **Response DTO** maps the domain entity back for the HTTP response

### Example Flow: Processing a Note Event (Subscriber)

1. **Subscriber** receives message from topic (e.g. `note.created`)
2. **Idempotency middleware** ensures at-most-once processing via `TryClaim`
3. **Handler** processes the event (optionally calls inbound port services)
4. **Retry middleware** handles transient failures with backoff

### Testing Strategy

#### Unit Tests
- Test services with mocked repositories
- Test business logic without external dependencies

#### Integration Tests
- Test adapters with real external systems
- Test complete workflows end-to-end
- Verify data persistence and retrieval

#### Contract Tests
- Ensure adapters implement interfaces correctly
- Verify error handling and edge cases
- Test performance characteristics