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

#### Example: Todo Entity

```go
// internal/core/domain/todo.go
package domain

import "time"

type Todo struct {
    ID          string    `qwery:"id"`
    Title       string    `qwery:"title"`
    Description string    `qwery:"description"`
    Done        bool      `qwery:"done"`
    CreatedAt   time.Time `qwery:"created_at"`
    UpdatedAt   time.Time `qwery:"updated_at"`
    DeletedAt   int       `qwery:"deleted_at"`
}

// Filter struct for list operations
type TodoFilter struct {
    Search string `qwery:"search"`
    IsDone *bool  `qwery:"is_done"` // pointer to make boolean optional
}
```

#### Example: Note Entity

```go
// internal/core/domain/note.go
package domain

import "time"

type Note struct {
    ID        string    `qwery:"id"`
    Title     string    `qwery:"title"`
    Content   string    `qwery:"content"`
    CreatedAt time.Time `qwery:"created_at"`
    UpdatedAt time.Time `qwery:"updated_at"`
    DeletedAt int       `qwery:"deleted_at"`
}

// Filter struct for list operations
type NoteFilter struct {
    Search string `qwery:"search"`
}
```

### Domain Best Practices

#### 1. Pure Business Logic
- No external dependencies (no HTTP, database, or framework imports)
- No JSON tags (use DTOs for serialization)
- Only use `qwery` tags for database mapping

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
type Todo struct {
    ID          string    `qwery:"id"`
    Title       string    `qwery:"title"`
    Description string    `qwery:"description"`
    Done        bool      `qwery:"done"`
    CreatedAt   time.Time `qwery:"created_at"`
    UpdatedAt   time.Time `qwery:"updated_at"`
    DeletedAt   int       `qwery:"deleted_at"`  // Soft delete field (Unix timestamp)
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
SELECT id, title, description, done, created_at, updated_at
FROM todos
WHERE deleted_at = 0
AND id = {{ .id }}

-- List with filters
SELECT id, title, description, done, created_at, updated_at
FROM todos
WHERE deleted_at = 0
{{ if .search }} AND (title LIKE CONCAT('%', {{ .search }}, '%') OR description LIKE CONCAT('%', {{ .search }}, '%')) {{ end }}
{{ if .is_done }} AND done = {{ .is_done }} {{ end }}
```

**Update Queries:**
```sql
-- Update existing record
UPDATE todos 
SET title = {{ .title }}, description = {{ .description }}, done = {{ .done }}
WHERE deleted_at = 0
AND id = {{ .id }}
```

**Delete Queries:**
```sql
-- Soft delete (mark as deleted with timestamp)
UPDATE todos SET deleted_at = UNIX_TIMESTAMP() WHERE deleted_at = 0 AND id = {{ .id }}
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

#### Example: Todo Service Interface

```go
// internal/core/port/inbound/todo.go
package inbound

import (
    "context"
    "github.com/redhajuanda/krangka/internal/core/domain"
    "github.com/redhajuanda/komon/pagination"
)

type Todo interface {
    // GetTodoByID retrieves a todo item by its ID
    GetTodoByID(ctx context.Context, id string) (*domain.Todo, error)
    // CreateTodo creates a new todo item
    CreateTodo(ctx context.Context, todo *domain.Todo) error
    // UpdateTodo updates an existing todo item
    UpdateTodo(ctx context.Context, todo *domain.Todo) error
    // DeleteTodo deletes a todo item by its ID
    DeleteTodo(ctx context.Context, id string) error
    // ListTodo retrieves a list of todo items with pagination
    ListTodo(ctx context.Context, req *domain.TodoFilter, pagination *pagination.Pagination) (*[]domain.Todo, error)
}
```

### Outbound Ports

Outbound ports define what your application needs from external systems.

#### Repository Interfaces

```go
// internal/core/port/outbound/repositories/todo.go
package repositories

import (
    "context"
    "github.com/redhajuanda/krangka/internal/core/domain"
    "github.com/redhajuanda/komon/pagination"
)

type Todo interface {
    // GetTodoByID retrieves a todo item by its ID
    GetTodoByID(ctx context.Context, id string) (*domain.Todo, error)
    // CreateTodo creates a new todo item
    CreateTodo(ctx context.Context, todo *domain.Todo) error
    // UpdateTodo updates an existing todo item
    UpdateTodo(ctx context.Context, todo *domain.Todo) error
    // DeleteTodo deletes a todo item by its ID
    DeleteTodo(ctx context.Context, id string) error
    // ListTodos retrieves a list of todo items with pagination
    ListTodos(ctx context.Context, req *domain.TodoFilter, pagination *pagination.Pagination) (*[]domain.Todo, error)
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
    // DoInTransaction executes a function within a database transaction
    DoInTransaction(ctx context.Context, fn func(repo Repository) (any, error)) (any, error)
    // PublishOutbox publishes an outbox event to the given target topic
    PublishOutbox(ctx context.Context, target PublisherTarget, topic string, payload qwery.JSONMap) error
    // RetryOutbox retries failed outbox events
    RetryOutbox(ctx context.Context) error
    // GetTodoRepository returns the TodoRepository instance
    GetTodoRepository() repositories.Todo
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

import "github.com/ThreeDotsLabs/watermill/message"

// Publisher is a contract for the message publisher (Watermill)
type Publisher message.Publisher

// Publishers is a map of publishers keyed by target
type Publishers map[PublisherTarget]message.Publisher

type PublisherTarget string

const (
    PublisherTargetRedisstream PublisherTarget = "redisstream"
    PublisherTargetKafka       PublisherTarget = "kafka"
)

// internal/core/port/outbound/subscriber.go
// Subscriber is a contract for the message subscriber (Watermill)
type Subscriber message.Subscriber
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

#### Example: Todo Service

```go
// internal/core/service/todo/service.go
package todo

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

func (s *Service) GetTodoByID(ctx context.Context, id string) (*domain.Todo, error) {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    var (
        repoTodo = s.repo.GetTodoRepository()
    )

    todo, err := repoTodo.GetTodoByID(ctx, id)
    if err != nil {
        return nil, fail.Wrap(err)
    }
    return todo, nil
}

func (s *Service) CreateTodo(ctx context.Context, todo *domain.Todo) error {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    var (
        repoTodo = s.repo.GetTodoRepository()
    )

    return repoTodo.CreateTodo(ctx, todo)
}

func (s *Service) UpdateTodo(ctx context.Context, todo *domain.Todo) error {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    var (
        repoTodo = s.repo.GetTodoRepository()
    )

    return repoTodo.UpdateTodo(ctx, todo)
}

func (s *Service) DeleteTodo(ctx context.Context, id string) error {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    var (
        repoTodo = s.repo.GetTodoRepository()
    )

    return repoTodo.DeleteTodo(ctx, id)
}

func (s *Service) ListTodo(ctx context.Context, req *domain.TodoFilter, pagination *pagination.Pagination) (*[]domain.Todo, error) {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    var (
        repoTodo = s.repo.GetTodoRepository()
    )

    res, err := repoTodo.ListTodos(ctx, req, pagination)
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
// internal/adapter/inbound/http/handler/todo.go
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

type TodoHandler struct {
    cfg *configs.Config
    log logger.Logger
    svc inbound.Todo
}

func NewTodoHandler(cfg *configs.Config, log logger.Logger, svc inbound.Todo) *TodoHandler {
    return &TodoHandler{
        cfg: cfg,
        log: log,
        svc: svc,
    }
}

func (h *TodoHandler) RegisterRoutes(app *fiber.App) {
    app.Get("/todos/:id", h.GetTodoByID)
    app.Post("/todos", h.CreateTodo)
    app.Put("/todos/:id", h.UpdateTodo)
    app.Delete("/todos/:id", h.DeleteTodo)
    app.Get("/todos", h.ListTodos)
}

// GetTodoByID godoc
// @Summary      Get Todo by ID
// @Description  Retrieves a todo by its id
// @Tags         Todos
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Todo ID"
// @Success      200  {object}  response.ResponseSuccess{data=dto.ResGetTodoByID}
// @Failure      400  {object}  response.ResponseFailed{}
// @Failure      404  {object}  response.ResponseFailed{}
// @Failure      500  {object}  response.ResponseFailed{}
// @Router       /todos/{id} [get]
func (h *TodoHandler) GetTodoByID(c *fiber.Ctx) error {

    var (
        req dto.ReqGetTodoByID
        res dto.ResGetTodoByID
        ctx = c.UserContext()
    )

    if err := c.ParamsParser(&req); err != nil {
        return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
    }

    if err := req.Validate(); err != nil {
        return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
    }

    todo, err := h.svc.GetTodoByID(ctx, req.ID)
    if err != nil {
        return err
    }

    res.Transform(todo)

    return response.SuccessOK(c, res, "Todo retrieved successfully")
}

func (h *TodoHandler) CreateTodo(c *fiber.Ctx) error {

    var (
        req = dto.ReqCreateTodo{}
        res = dto.ResCreateTodo{}
        ctx = c.UserContext()
    )

    if err := c.BodyParser(&req); err != nil {
        return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
    }

    if err := req.Validate(); err != nil {
        return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
    }

    todo := req.Transform()

    err := h.svc.CreateTodo(ctx, todo)
    if err != nil {
        return err
    }

    res.Transform(todo)

    return response.SuccessCreated(c, res, "Todo created successfully")
}
```

#### HTTP DTOs

DTOs live in `internal/adapter/inbound/http/handler/dto/` and handle request parsing, validation, and transformation.

```go
// internal/adapter/inbound/http/handler/dto/todo.go
package dto

import (
    "time"
    "github.com/redhajuanda/krangka/internal/core/domain"
    "github.com/redhajuanda/komon/pagination"
    "github.com/go-playground/validator/v10"
    "github.com/oklog/ulid/v2"
)

// Request DTOs — parse input and validate
type ReqCreateTodo struct {
    Title       string `json:"title"       validate:"required"`
    Description string `json:"description" validate:"required"`
    Done        bool   `json:"done"`
}

func (r *ReqCreateTodo) Validate() error {
    return validator.New().Struct(r)
}

// Transform converts the request DTO to a domain entity
func (r *ReqCreateTodo) Transform() *domain.Todo {
    return &domain.Todo{
        ID:          ulid.Make().String(),
        Title:       r.Title,
        Description: r.Description,
        Done:        r.Done,
    }
}

// Response DTOs — shape the output, never expose domain internals like DeletedAt
type ResGetTodoByID struct {
    ID          string    `json:"id"`
    Title       string    `json:"title"`
    Description string    `json:"description"`
    Done        bool      `json:"done"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

func (r *ResGetTodoByID) Transform(todo *domain.Todo) {
    r.ID          = todo.ID
    r.Title       = todo.Title
    r.Description = todo.Description
    r.Done        = todo.Done
    r.CreatedAt   = todo.CreatedAt
    r.UpdatedAt   = todo.UpdatedAt
}

// List request — embed pagination directly
type ReqListTodo struct {
    pagination.Pagination
    Search string `query:"search" validate:"omitempty,max=100"`
    IsDone *bool  `query:"is_done" validate:"omitempty"`
}

func (r *ReqListTodo) Transform() *domain.TodoFilter {
    return &domain.TodoFilter{
        Search: r.Search,
        IsDone: r.IsDone,
    }
}
```

### Outbound Adapters

Outbound adapters handle external system communication.

#### Database Repository

```go
// internal/adapter/outbound/mariadb/repositories/todo.go
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

type todoRepository struct {
    qwery qwery.Runable
}

func NewTodoRepository(qwery qwery.Runable) *todoRepository {
    return &todoRepository{qwery: qwery}
}

func (r *todoRepository) GetTodoByID(ctx context.Context, id string) (*domain.Todo, error) {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    var todo domain.Todo

    query := `
        SELECT 
            id, 
            title, 
            description, 
            done
        FROM todos
        WHERE id = {{ .id }} 
        AND deleted_at = 0
    `

    err := r.qwery.
        RunRaw(query).
        WithParam("id", id).
        ScanStruct(&todo).
        Query(ctx)

    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, fail.Wrap(err).WithFailure(failure.ErrTodoNotFound)
        }
        return nil, fail.Wrap(err)
    }

    return &todo, nil
}

func (r *todoRepository) CreateTodo(ctx context.Context, todo *domain.Todo) error {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    query := `
        INSERT INTO todos (id, title, description, done) 
        VALUES ({{ .id }}, {{ .title }}, {{ .description }}, {{ .done }})
    `

    err := r.qwery.
        RunRaw(query).
        WithParams(todo).
        Query(ctx)

    if err != nil {
        return fail.Wrap(err)
    }

    return nil
}

func (r *todoRepository) UpdateTodo(ctx context.Context, todo *domain.Todo) error {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    query := `
        UPDATE todos 
        SET title = {{ .title }}, description = {{ .description }}, done = {{ .done }} 
        WHERE id = {{ .id }} 
        AND deleted_at = 0
    `

    err := r.qwery.
        RunRaw(query).
        WithParams(todo).
        Query(ctx)

    if err != nil {
        return fail.Wrap(err)
    }

    return nil
}

func (r *todoRepository) DeleteTodo(ctx context.Context, id string) error {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    query := `
        UPDATE todos 
        SET deleted_at = UNIX_TIMESTAMP() 
        WHERE id = {{ .id }} 
        AND deleted_at = 0
    `

    err := r.qwery.
        RunRaw(query).
        WithParam("id", id).
        Query(ctx)

    if err != nil {
        return fail.Wrap(err)
    }

    return nil
}

func (r *todoRepository) ListTodos(ctx context.Context, req *domain.TodoFilter, pagination *pagination.Pagination) (*[]domain.Todo, error) {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    todos := make([]domain.Todo, 0)

    query := `
        SELECT id, title, description, done, created_at, updated_at, deleted_at
        FROM todos
        WHERE deleted_at = 0
        {{ if .search }} AND (title LIKE CONCAT('%', {{ .search }}, '%') OR description LIKE CONCAT('%', {{ .search }}, '%')) {{ end }}
        {{ if .is_done }} AND done = {{ .is_done }} {{ end }}
    `

    err := r.qwery.
        RunRaw(query).
        WithParams(map[string]any{
            "search":  req.Search,
            "is_done": req.IsDone,
        }).
        WithPagination(pagination).
        WithOrderBy("-created_at", "id").
        ScanStructs(&todos).
        Query(ctx)

    if err != nil {
        return nil, fail.Wrap(err)
    }
    return &todos, nil
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
- Use Qwery template syntax (`{{ .field }}`) for parameterized queries
- Always call `tracer.Trace(ctx)` and `defer span.End()` at the start of every method
- Always use `WithPagination()` and `WithOrderBy()` when listing data with pagination
- Include `WHERE deleted_at = 0` in all SELECT and UPDATE queries
- Use `{{ if .field }}` for optional filter conditions

#### 5. Performance
- Use connection pooling for databases (handled by Qwery)
- Optimize queries with appropriate indexes

## Error Handling

### The `fail` Package

All errors in Krangka use `github.com/redhajuanda/komon/fail`. This package records stack traces and carries typed failure information.

```go
// Wrap an error to record its stack trace
return fail.Wrap(err)

// Map to a typed failure (changes HTTP status code and error code in response)
return fail.Wrap(err).WithFailure(failure.ErrTodoNotFound)

// Wrap with a built-in failure type (e.g., bad request)
return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
```

### The `shared/failure` Package

Application-specific typed errors are declared centrally in `shared/failure/failure.go`:

```go
package failure

import "github.com/redhajuanda/komon/fail"

var (
    ErrTodoNotFound      = &fail.Failure{Code: "404001", Message: "Todo not found",      HTTPStatus: 404}
    ErrTodoAlreadyExists = &fail.Failure{Code: "409001", Message: "Todo already exists", HTTPStatus: 409}

    ErrNoteNotFound      = &fail.Failure{Code: "404001", Message: "Note not found",      HTTPStatus: 404}
    ErrNoteAlreadyExists = &fail.Failure{Code: "409001", Message: "Note already exists", HTTPStatus: 409}
)
```

### Error Handling by Layer

| Layer | Pattern |
|---|---|
| Repository | `fail.Wrap(err)` — always wrap; map sentinel errors with `.WithFailure(failure.ErrXxx)` |
| Service | `fail.Wrap(err)` — wrap when adding context; propagate otherwise |
| HTTP Handler | `fail.Wrap(err).WithFailure(fail.ErrBadRequest)` for parse/validation errors; propagate service errors |

## Component Interaction

### Dependency Flow

```
HTTP Handler → Todo Service → Todo Repository → Database
     ↓              ↓              ↓
  DTOs         Domain Logic    SQL Queries
```

### Example Flow: Creating a Todo

1. **HTTP Handler** receives request, parses and validates DTO
2. **DTO** `Transform()` builds the domain entity (generates ID with `ulid.Make()`)
3. **Service** delegates to repository (applies business rules if any)
4. **Repository** persists the entity to database via Qwery
5. **Response DTO** maps the domain entity back for the HTTP response

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
