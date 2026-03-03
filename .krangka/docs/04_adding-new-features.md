# Adding New Features

This guide walks you through adding a new feature to your Krangka application following hexagonal architecture principles. We'll use a **User management** feature as an example to demonstrate the complete process.

## Overview

When adding a new feature to Krangka, you'll follow these steps:

1. **Database Migration** - Create table structure with proper indexes and constraints
2. **Domain Layer** - Define entities and filter structs
3. **Failure Definitions** - Add typed errors for the new feature
4. **Port Interfaces** - Create inbound and outbound contracts for dependency inversion
5. **Service Layer** - Implement business logic
6. **Register Service** - Add service to bootstrap dependency system
7. **Database Repository** - Implement repository with Qwery using inline SQL queries (`RunRaw()`)
8. **Register Repository** - Add repository to the MariaDB repository implementation
9. **DTOs** - Create request/response data transfer objects with validation
10. **HTTP Handler** - Build REST API endpoints
11. **Register Handler** - Add handler to bootstrap dependency system
12. **Swagger Documentation** - Add API documentation comments
13. **Run Migration** - Execute database migration to create tables

## Step 1: Create Database Migration

First, create a database migration for the new table:

```bash
# Create migration file using Makefile
make migrate-new repo=mariadb name=create_table_users

# Or directly
go run main.go migrate new mariadb create_table_users
```

This creates a new migration file in `internal/adapter/outbound/mariadb/migrations/scripts/`.

Add the migration SQL:

```sql
-- internal/adapter/outbound/mariadb/migrations/scripts/YYYYMMDDHHMMSS-create_table_users.sql
-- +migrate Up
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) NOT NULL PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    deleted_at int NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_active ON users(active);

-- +migrate Down
DROP INDEX IF EXISTS idx_users_active ON users;
DROP INDEX IF EXISTS idx_users_email ON users;
DROP INDEX IF EXISTS idx_users_username ON users;
DROP TABLE IF EXISTS users;
```

## Step 2: Create Domain Entity and Filter

Create the domain entity and filter struct. Domain entities are **plain structs** with no external dependencies and no business methods — keep them simple:

```go
// internal/core/domain/user.go
package domain

import "time"

type User struct {
    ID        string    `qwery:"id"`
    Username  string    `qwery:"username"`
    Email     string    `qwery:"email"`
    Active    bool      `qwery:"active"`
    CreatedAt time.Time `qwery:"created_at"`
    UpdatedAt time.Time `qwery:"updated_at"`
    DeletedAt int       `qwery:"deleted_at"`
}

// Filter struct for list operations
type UserFilter struct {
    Search string `qwery:"search"`
    Active *bool  `qwery:"active"` // pointer to make boolean optional
}
```

### Domain Best Practices

- **Pure Data Structs**: No external dependencies, no methods, no validation logic
- **Qwery Tags**: Use `qwery` tags for database mapping
- **Soft Delete**: Always include `DeletedAt int` field
- **No JSON Tags**: JSON serialization is handled by DTOs

## Step 3: Add Failure Definitions

Add typed errors for the new feature to `shared/failure/failure.go`:

```go
// shared/failure/failure.go
package failure

import "github.com/redhajuanda/komon/fail"

var (
    ErrTodoNotFound      = &fail.Failure{Code: "404001", Message: "Todo not found",      HTTPStatus: 404}
    ErrTodoAlreadyExists = &fail.Failure{Code: "409001", Message: "Todo already exists", HTTPStatus: 409}

    ErrNoteNotFound      = &fail.Failure{Code: "404002", Message: "Note not found",      HTTPStatus: 404}
    ErrNoteAlreadyExists = &fail.Failure{Code: "409002", Message: "Note already exists", HTTPStatus: 409}

    // Add new errors for the User feature
    ErrUserNotFound      = &fail.Failure{Code: "404003", Message: "User not found",      HTTPStatus: 404}
    ErrUserAlreadyExists = &fail.Failure{Code: "409003", Message: "User already exists", HTTPStatus: 409}
)
```

Pick error codes that don't conflict with existing ones. Convention: `HTTPSTATUS + sequential number`.

## Step 4: Define Port Interfaces

Create both inbound and outbound port interfaces.

### Inbound Port Interface

```go
// internal/core/port/inbound/user.go
package inbound

import (
    "context"
    "github.com/redhajuanda/krangka/internal/core/domain"
    "github.com/redhajuanda/komon/pagination"
)

type User interface {
    // GetUserByID retrieves a user by its ID
    GetUserByID(ctx context.Context, id string) (*domain.User, error)
    // CreateUser creates a new user
    CreateUser(ctx context.Context, user *domain.User) error
    // UpdateUser updates an existing user
    UpdateUser(ctx context.Context, user *domain.User) error
    // DeleteUser deletes a user by its ID
    DeleteUser(ctx context.Context, id string) error
    // ListUsers retrieves a list of users with pagination
    ListUsers(ctx context.Context, req *domain.UserFilter, pagination *pagination.Pagination) (*[]domain.User, error)
}
```

### Outbound Port Interface

```go
// internal/core/port/outbound/repositories/user.go
package repositories

import (
    "context"
    "github.com/redhajuanda/krangka/internal/core/domain"
    "github.com/redhajuanda/komon/pagination"
)

type User interface {
    // GetUserByID retrieves a user by its ID
    GetUserByID(ctx context.Context, id string) (*domain.User, error)
    // CreateUser creates a new user
    CreateUser(ctx context.Context, user *domain.User) error
    // UpdateUser updates an existing user
    UpdateUser(ctx context.Context, user *domain.User) error
    // DeleteUser deletes a user by its ID
    DeleteUser(ctx context.Context, id string) error
    // ListUsers retrieves a list of users with pagination
    ListUsers(ctx context.Context, req *domain.UserFilter, pagination *pagination.Pagination) (*[]domain.User, error)
}
```

### Update Main Repository Interface

Add the new repository getter to the main repository interface:

```go
// internal/core/port/outbound/repository.go
type Repository interface {
    DoInTransaction(ctx context.Context, fn func(repo Repository) (any, error)) (any, error)
    PublishOutbox(ctx context.Context, target PublisherTarget, topic string, payload qwery.JSONMap) error
    RetryOutbox(ctx context.Context) error
    GetTodoRepository() repositories.Todo
    GetNoteRepository() repositories.Note
    GetUserRepository() repositories.User  // Add this line
}
```

## Step 5: Implement Service

Create the service implementation. Every method must call `tracer.Trace(ctx)` and use `fail.Wrap(err)` for error wrapping:

```go
// internal/core/service/user/service.go
package user

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

func (s *Service) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    var (
        repoUser = s.repo.GetUserRepository()
    )

    user, err := repoUser.GetUserByID(ctx, id)
    if err != nil {
        return nil, fail.Wrap(err)
    }
    return user, nil
}

func (s *Service) CreateUser(ctx context.Context, user *domain.User) error {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    var (
        repoUser = s.repo.GetUserRepository()
    )

    return repoUser.CreateUser(ctx, user)
}

func (s *Service) UpdateUser(ctx context.Context, user *domain.User) error {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    var (
        repoUser = s.repo.GetUserRepository()
    )

    return repoUser.UpdateUser(ctx, user)
}

func (s *Service) DeleteUser(ctx context.Context, id string) error {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    var (
        repoUser = s.repo.GetUserRepository()
    )

    return repoUser.DeleteUser(ctx, id)
}

func (s *Service) ListUsers(ctx context.Context, req *domain.UserFilter, pagination *pagination.Pagination) (*[]domain.User, error) {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    var (
        repoUser = s.repo.GetUserRepository()
    )

    res, err := repoUser.ListUsers(ctx, req, pagination)
    if err != nil {
        return nil, err
    }
    return res, err
}
```

### Service Best Practices

- **Always trace**: Call `tracer.Trace(ctx)` + `defer span.End()` at the start of every method
- **Single responsibility**: Each method implements one use case
- **Error wrapping**: Use `fail.Wrap(err)` when you need to add context; propagate otherwise
- **No direct external calls**: Only call outbound port interfaces, never import adapters

## Step 6: Register Service in Bootstrap

Add the service field to the `Dependency` struct and create a getter method in `cmd/bootstrap/dependency.go`:

```go
// cmd/bootstrap/dependency.go

import (
    // ... existing imports
    "github.com/redhajuanda/krangka/internal/core/service/user"
)

type Dependency struct {
    // ... existing fields
    serviceUser Resource[*user.Service]  // Add this line
}

// GetServiceUser resolves and returns the service user dependency
func (d *Dependency) GetServiceUser(repo outbound.Repository) *user.Service {
    return d.serviceUser.Resolve(func() *user.Service {
        return user.NewService(d.GetConfig(), d.GetLogger(), repo, d.GetRedis())
    })
}
```

**Important**: Service getters accept `repo outbound.Repository` as a parameter because:
- HTTP handlers use `GetQweryMain()` (main database connection)
- Workers use `GetQweryWorker()` (worker database connection)
- This allows the same service to work with different database connections without re-initializing

## Step 7: Implement Database Repository

Create the repository implementation with inline SQL queries using `RunRaw()`. Always use `fail.Wrap(err)` and map known sentinel errors (like `sql.ErrNoRows`) to typed failures from `shared/failure`:

```go
// internal/adapter/outbound/mariadb/repositories/user.go
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

type userRepository struct {
    qwery qwery.Runable
}

func NewUserRepository(qwery qwery.Runable) *userRepository {
    return &userRepository{qwery: qwery}
}

func (r *userRepository) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    var user domain.User

    query := `
        SELECT id, username, email, active, created_at, updated_at
        FROM users
        WHERE deleted_at = 0
        AND id = {{ .id }}
    `

    err := r.qwery.
        RunRaw(query).
        WithParam("id", id).
        ScanStruct(&user).
        Query(ctx)

    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, fail.Wrap(err).WithFailure(failure.ErrUserNotFound)
        }
        return nil, fail.Wrap(err)
    }
    return &user, nil
}

func (r *userRepository) CreateUser(ctx context.Context, user *domain.User) error {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    query := `
        INSERT INTO users (id, username, email, active) 
        VALUES ({{ .id }}, {{ .username }}, {{ .email }}, {{ .active }})
    `

    err := r.qwery.
        RunRaw(query).
        WithParams(user).
        Query(ctx)

    if err != nil {
        return fail.Wrap(err)
    }
    return nil
}

func (r *userRepository) UpdateUser(ctx context.Context, user *domain.User) error {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    query := `
        UPDATE users 
        SET username = {{ .username }}, email = {{ .email }}, active = {{ .active }}
        WHERE deleted_at = 0
        AND id = {{ .id }}
    `

    err := r.qwery.
        RunRaw(query).
        WithParams(user).
        Query(ctx)

    if err != nil {
        return fail.Wrap(err)
    }
    return nil
}

func (r *userRepository) DeleteUser(ctx context.Context, id string) error {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    query := `
        UPDATE users 
        SET deleted_at = UNIX_TIMESTAMP() 
        WHERE deleted_at = 0 
        AND id = {{ .id }}
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

func (r *userRepository) ListUsers(ctx context.Context, req *domain.UserFilter, pagination *pagination.Pagination) (*[]domain.User, error) {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    users := make([]domain.User, 0)

    query := `
        SELECT id, username, email, active, created_at, updated_at
        FROM users
        WHERE deleted_at = 0
        {{ if .search }} AND (username LIKE CONCAT('%', {{ .search }}, '%') OR email LIKE CONCAT('%', {{ .search }}, '%')) {{ end }}
        {{ if .active }} AND active = {{ .active }} {{ end }}
    `

    err := r.qwery.
        RunRaw(query).
        WithParams(map[string]any{
            "search": req.Search,
            "active": req.Active,
        }).
        WithPagination(pagination).
        WithOrderBy("-created_at", "id").
        ScanStructs(&users).
        Query(ctx)

    if err != nil {
        return nil, fail.Wrap(err)
    }
    return &users, nil
}
```

### Repository Best Practices

- Use `RunRaw()` with **inline SQL** — no external `.sql` files for queries
- Use Qwery template syntax (`{{ .field }}`) for parameterized queries
- Always call `tracer.Trace(ctx)` + `defer span.End()` at the start of every method
- Always use `fail.Wrap(err)` — never return raw errors; this captures the stack trace
- Map `sql.ErrNoRows` to typed failures using `errors.Is(err, sql.ErrNoRows)` + `.WithFailure(failure.ErrXxx)`
- Use `WithPagination()` + `WithOrderBy()` for all list queries — no need to add `LIMIT`/`OFFSET`/`ORDER BY` manually
- Always use `WHERE deleted_at = 0` in SELECT and UPDATE queries
- Use `{{ if .field }}` for optional filter conditions

## Step 8: Register Repository in MariaDB Implementation

Add the new repository to the `mariaDBRepository` struct in `internal/adapter/outbound/mariadb/repository.go`:

```go
// internal/adapter/outbound/mariadb/repository.go

type mariaDBRepository struct {
    cfg     *configs.Config
    log     logger.Logger
    qwery   *Qwery
    qweryTx qwery.Runable
    outbox  *Outbox

    TodoRepository portRepo.Todo
    NoteRepository portRepo.Note
    UserRepository portRepo.User  // Add this line
}

func NewMariaDBRepository(cfg *configs.Config, log logger.Logger, qwery *Qwery, publishers outbound.Publishers) *mariaDBRepository {
    return &mariaDBRepository{
        cfg:    cfg,
        log:    log,
        qwery:  qwery,
        outbox: NewOutbox(cfg, log, qwery.Client, false, publishers),

        TodoRepository: implRepo.NewTodoRepository(qwery.Client),
        NoteRepository: implRepo.NewNoteRepository(qwery.Client),
        UserRepository: implRepo.NewUserRepository(qwery.Client),  // Add this line
    }
}

// Add the getter method
func (r *mariaDBRepository) GetUserRepository() portRepo.User {
    if r.qweryTx != nil {
        return implRepo.NewUserRepository(r.qweryTx)
    }
    return r.UserRepository
}
```

Note the transaction-aware pattern: if `qweryTx` is set (we are inside a `DoInTransaction` call), create a new repository instance bound to the transaction connection instead of returning the cached one.

## Step 9: Create DTOs

Create request/response DTOs in `internal/adapter/inbound/http/handler/dto/user.go`:

```go
// internal/adapter/inbound/http/handler/dto/user.go
package dto

import (
    "time"
    "github.com/redhajuanda/krangka/internal/core/domain"
    "github.com/redhajuanda/komon/pagination"
    "github.com/go-playground/validator/v10"
    "github.com/oklog/ulid/v2"
)

type ReqGetUserByID struct {
    ID string `params:"id" validate:"required"`
}

func (r *ReqGetUserByID) Validate() error {
    return validator.New().Struct(r)
}

type ResGetUserByID struct {
    ID        string    `json:"id"`
    Username  string    `json:"username"`
    Email     string    `json:"email"`
    Active    bool      `json:"active"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

func (r *ResGetUserByID) Transform(user *domain.User) {
    r.ID        = user.ID
    r.Username  = user.Username
    r.Email     = user.Email
    r.Active    = user.Active
    r.CreatedAt = user.CreatedAt
    r.UpdatedAt = user.UpdatedAt
}

type ReqCreateUser struct {
    Username string `json:"username" validate:"required"`
    Email    string `json:"email"    validate:"required,email"`
    Active   bool   `json:"active"`
}

func (r *ReqCreateUser) Validate() error {
    return validator.New().Struct(r)
}

func (r *ReqCreateUser) Transform() *domain.User {
    return &domain.User{
        ID:       ulid.Make().String(),
        Username: r.Username,
        Email:    r.Email,
        Active:   r.Active,
        // CreatedAt and UpdatedAt are handled by the database
    }
}

type ResCreateUser struct {
    ID string `json:"id"`
}

func (r *ResCreateUser) Transform(user *domain.User) {
    r.ID = user.ID
}

type ReqUpdateUser struct {
    ID       string `params:"id" validate:"required" swaggerignore:"true"` // path param, ignored in swagger body
    Username string `json:"username" validate:"required"`
    Email    string `json:"email"    validate:"required,email"`
    Active   bool   `json:"active"`
}

func (r *ReqUpdateUser) Validate() error {
    return validator.New().Struct(r)
}

func (r *ReqUpdateUser) Transform() *domain.User {
    return &domain.User{
        ID:       r.ID,
        Username: r.Username,
        Email:    r.Email,
        Active:   r.Active,
        // UpdatedAt is handled by the database
    }
}

type ReqDeleteUser struct {
    ID string `params:"id" validate:"required" swaggerignore:"true"`
}

func (r *ReqDeleteUser) Validate() error {
    return validator.New().Struct(r)
}

type ReqListUser struct {
    pagination.Pagination
    Search string `query:"search" validate:"omitempty,max=100"`
    Active *bool  `query:"active" validate:"omitempty"`
}

func (r *ReqListUser) Validate() error {
    return validator.New().Struct(r)
}

func (r *ReqListUser) Transform() *domain.UserFilter {
    return &domain.UserFilter{
        Search: r.Search,
        Active: r.Active,
    }
}

type ListUser struct {
    ID        string    `json:"id"`
    Username  string    `json:"username"`
    Email     string    `json:"email"`
    Active    bool      `json:"active"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type ResListUser []ListUser

func (r *ResListUser) Transform(users *[]domain.User) {
    for _, user := range *users {
        *r = append(*r, ListUser{
            ID:        user.ID,
            Username:  user.Username,
            Email:     user.Email,
            Active:    user.Active,
            CreatedAt: user.CreatedAt,
            UpdatedAt: user.UpdatedAt,
        })
    }
}
```

### DTO Best Practices

- Use `validate` tag for input validation (powered by `github.com/go-playground/validator/v10`)
- Use `json` tag for JSON marshalling/unmarshalling
- Use Fiber struct tags (`params`, `query`, `form`) for request parsing
- Keep request and response DTOs separate — response DTOs are sometimes not needed (e.g., update/delete)
- Use `Transform()` to convert between DTOs and domain entities
- Use `Validate()` to validate the request
- Use `ulid.Make().String()` to generate the entity ID in `ReqCreate*.Transform()`
- Use `swaggerignore:"true"` on path parameter fields (`params` tag) to exclude them from the swagger body schema
- **Never expose** `DeletedAt` in response DTOs

## Step 10: Implement HTTP Handler

Create the HTTP handler in `internal/adapter/inbound/http/handler/user.go`:

```go
// internal/adapter/inbound/http/handler/user.go
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

type UserHandler struct {
    cfg *configs.Config
    log logger.Logger
    svc inbound.User
}

func NewUserHandler(cfg *configs.Config, log logger.Logger, svc inbound.User) *UserHandler {
    return &UserHandler{cfg: cfg, log: log, svc: svc}
}

func (h *UserHandler) RegisterRoutes(app *fiber.App) {
    app.Get("/users/:id", h.GetUserByID)
    app.Post("/users", h.CreateUser)
    app.Put("/users/:id", h.UpdateUser)
    app.Delete("/users/:id", h.DeleteUser)
    app.Get("/users", h.ListUsers)
}

// GetUserByID godoc
// @Summary      Get User by ID
// @Description  Retrieves a user by its id
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "User ID"
// @Success      200  {object}  response.ResponseSuccess{data=dto.ResGetUserByID}
// @Failure      400  {object}  response.ResponseFailed{}
// @Failure      404  {object}  response.ResponseFailed{}
// @Failure      500  {object}  response.ResponseFailed{}
// @Router       /users/{id} [get]
func (h *UserHandler) GetUserByID(c *fiber.Ctx) error {
    var (
        req dto.ReqGetUserByID
        res dto.ResGetUserByID
        ctx = c.UserContext()
    )

    if err := c.ParamsParser(&req); err != nil {
        return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
    }

    if err := req.Validate(); err != nil {
        return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
    }

    user, err := h.svc.GetUserByID(ctx, req.ID)
    if err != nil {
        return err
    }

    res.Transform(user)
    return response.SuccessOK(c, res, "User retrieved successfully")
}

// CreateUser godoc
// @Summary      Create User
// @Description  Creates a new user
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        user  body      dto.ReqCreateUser  true  "User data"
// @Success      201   {object}  response.ResponseSuccess{data=dto.ResCreateUser}
// @Failure      400   {object}  response.ResponseFailed{}
// @Failure      500   {object}  response.ResponseFailed{}
// @Router       /users [post]
func (h *UserHandler) CreateUser(c *fiber.Ctx) error {
    var (
        req = dto.ReqCreateUser{}
        res = dto.ResCreateUser{}
        ctx = c.UserContext()
    )

    if err := c.BodyParser(&req); err != nil {
        return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
    }

    if err := req.Validate(); err != nil {
        return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
    }

    user := req.Transform()
    if err := h.svc.CreateUser(ctx, user); err != nil {
        return err
    }

    res.Transform(user)
    return response.SuccessCreated(c, res, "User created successfully")
}

// UpdateUser godoc
// @Summary      Update User
// @Description  Updates an existing user by ID
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        id    path      string            true  "User ID"
// @Param        user  body      dto.ReqUpdateUser true  "User data"
// @Success      200   {object}  response.ResponseSuccess{}
// @Failure      400   {object}  response.ResponseFailed{}
// @Failure      404   {object}  response.ResponseFailed{}
// @Failure      500   {object}  response.ResponseFailed{}
// @Router       /users/{id} [put]
func (h *UserHandler) UpdateUser(c *fiber.Ctx) error {
    var (
        req dto.ReqUpdateUser
        ctx = c.UserContext()
    )

    if err := c.ParamsParser(&req); err != nil {
        return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
    }

    if err := c.BodyParser(&req); err != nil {
        return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
    }

    if err := req.Validate(); err != nil {
        return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
    }

    if err := h.svc.UpdateUser(ctx, req.Transform()); err != nil {
        return err
    }

    return response.SuccessOK(c, nil, "User updated successfully")
}

// DeleteUser godoc
// @Summary      Delete User
// @Description  Deletes a user by ID
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "User ID"
// @Success      200  {object}  response.ResponseSuccess{}
// @Failure      400  {object}  response.ResponseFailed{}
// @Failure      404  {object}  response.ResponseFailed{}
// @Failure      500  {object}  response.ResponseFailed{}
// @Router       /users/{id} [delete]
func (h *UserHandler) DeleteUser(c *fiber.Ctx) error {
    var (
        req dto.ReqDeleteUser
        ctx = c.UserContext()
    )

    if err := c.ParamsParser(&req); err != nil {
        return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
    }

    if err := req.Validate(); err != nil {
        return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
    }

    if err := h.svc.DeleteUser(ctx, req.ID); err != nil {
        return err
    }

    return response.SuccessOK(c, nil, "User deleted successfully")
}

// ListUsers godoc
// @Summary      List Users
// @Description  Retrieves a list of users with optional filtering and pagination
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        request  query     dto.ReqListUser  false  "Request"
// @Success      200      {object}  response.ResponseSuccess{data=dto.ResListUser}
// @Failure      400      {object}  response.ResponseFailed{}
// @Failure      500      {object}  response.ResponseFailed{}
// @Router       /users [get]
func (h *UserHandler) ListUsers(c *fiber.Ctx) error {
    var (
        req dto.ReqListUser
        res dto.ResListUser
        ctx = c.UserContext()
    )

    if err := c.QueryParser(&req); err != nil {
        return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
    }

    if err := req.Validate(); err != nil {
        return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
    }

    filter := req.Transform()
    pagination := req.Pagination

    users, err := h.svc.ListUsers(ctx, filter, &pagination)
    if err != nil {
        return err
    }

    res.Transform(users)
    return response.SuccessOKWithPagination(c, res, pagination)
}
```

### HTTP Handler Best Practices

- Use `c.UserContext()` to extract the context from the request
- Use `c.ParamsParser()` for path params, `c.QueryParser()` for query params, `c.BodyParser()` for body
- For update endpoints, call `c.ParamsParser()` first then `c.BodyParser()` to fill the same DTO
- Wrap parse/validation errors with `fail.Wrap(err).WithFailure(fail.ErrBadRequest)`
- Propagate service errors directly — never re-wrap them
- Use `response.SuccessCreated()` for POST (201), `response.SuccessOK()` for GET/PUT/DELETE (200)
- Use `response.SuccessOKWithPagination()` for list endpoints

## Step 11: Register Handler in Bootstrap

Add the handler to `GetHTTPHandlers()` in `cmd/bootstrap/dependency.go`:

```go
// cmd/bootstrap/dependency.go

import (
    // ... existing imports
    "github.com/redhajuanda/krangka/internal/core/service/user"
)

type Dependency struct {
    // ... existing fields
    serviceUser Resource[*user.Service]  // already added in Step 6
}

// GetServiceUser already added in Step 6

// GetHTTPHandlers resolves and returns the http handlers dependency
func (d *Dependency) GetHTTPHandlers() []http.Handler {
    return d.httpHandlers.Resolve(func() []http.Handler {
        repo := d.GetRepository(d.GetQweryMain())
        return []http.Handler{
            httpHandler.NewNoteHandler(d.GetConfig(), d.GetLogger(), d.GetServiceNote(repo)),
            httpHandler.NewTodoHandler(d.GetConfig(), d.GetLogger(), d.GetServiceTodo(repo)),
            httpHandler.NewUserHandler(d.GetConfig(), d.GetLogger(), d.GetServiceUser(repo)), // Add this line
        }
    })
}
```

The handler will automatically register its routes when the HTTP server calls `RegisterRoutes()` during startup.

## Step 12: Generate Swagger Documentation

After adding Swagger comments to your handlers, regenerate the OpenAPI docs:

```bash
make swag
# This runs: swag init --output internal/adapter/inbound/http/docs --parseDependency
```

Generated files land in `internal/adapter/inbound/http/docs/`:
- `docs.go` — Swagger configuration
- `swagger.json` — OpenAPI spec (JSON)
- `swagger.yaml` — OpenAPI spec (YAML)

### Swagger Notes

- `swaggerignore:"true"` on path parameter fields (those with `params` tag) prevents them from appearing in the request body schema
- Use sentence-case tag names (e.g., `@Tags Users` not `@Tags users` or `@Tags user-management`)

## Step 13: Run Database Migration

Finally, run the migration:

```bash
make migrate-up repo=mariadb

# Or directly
go run main.go migrate up mariadb
```

## Testing Your New Feature

### Unit Tests

```go
// internal/core/service/user/service_test.go
package user

import (
    "context"
    "testing"
    "github.com/redhajuanda/krangka/internal/core/domain"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestUserService_GetUserByID(t *testing.T) {
    mockRepo := &MockRepository{}
    mockUserRepo := &MockUserRepository{}
    service := NewService(nil, nil, mockRepo, nil)

    expected := &domain.User{ID: "01JKM1234", Username: "testuser"}

    mockRepo.On("GetUserRepository").Return(mockUserRepo)
    mockUserRepo.On("GetUserByID", mock.Anything, "01JKM1234").Return(expected, nil)

    result, err := service.GetUserByID(context.Background(), "01JKM1234")

    assert.NoError(t, err)
    assert.Equal(t, expected, result)
    mockRepo.AssertExpectations(t)
}
```

### Integration Tests

```go
// internal/adapter/inbound/http/handler/user_test.go
package handler

import (
    "bytes"
    "encoding/json"
    "net/http/httptest"
    "testing"
    "github.com/gofiber/fiber/v2"
    "github.com/stretchr/testify/assert"
)

func TestUserHandler_CreateUser(t *testing.T) {
    app := fiber.New()
    handler := NewUserHandler(nil, nil, &MockUserService{})
    handler.RegisterRoutes(app)

    reqBody := dto.ReqCreateUser{
        Username: "testuser",
        Email:    "test@example.com",
    }

    body, _ := json.Marshal(reqBody)
    req := httptest.NewRequest("POST", "/users", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    resp, _ := app.Test(req)
    assert.Equal(t, 201, resp.StatusCode)
}
```
