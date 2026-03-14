# Adding New Features in Krangka

Follow the canonical order below. **TDD is mandatory**: write all tests first, then **pause for user review** before implementing.

## TDD Review Gate (Critical)

After writing service tests (Step 6):

1. **Stop** — do not proceed to implementation
2. **Present** the test scenarios to the user (list scenarios covered per method/endpoint)
3. **Ask**: "Please review the test scenarios above. Are they correct and complete? Reply to confirm or request changes."
4. **Wait** for user confirmation before implementing
5. **Proceed** only after user confirms

**Never skip.** Implementation without confirmed test scenarios is an anti-pattern.

## Canonical Order

```
- [ ] 1. Database Migration
- [ ] 2. Domain (entity + filter)
- [ ] 3. Failure definitions
- [ ] 4. Port interfaces (inbound + outbound)
- [ ] 5. Generate mocks (make mock)
- [ ] 6. Service tests (TDD) — all scenarios first
- [ ] 6b. ⏸️ REVIEW GATE — present scenarios, wait for user confirmation
- [ ] 7. Implement service
- [ ] 8. Register service in bootstrap
- [ ] 9. Database repository
- [ ] 10. Register repository
- [ ] 11. DTOs
- [ ] 12. HTTP handler
- [ ] 13. Register handler
- [ ] 14. Swagger (godoc + make swag)
- [ ] 15. Run migration
```

## Commands

```bash
make migrate-new repo=mariadb name=create_table_users
make mock
make swag
make migrate-up repo=mariadb
```

## Step-by-Step

### 1. Migration

```bash
make migrate-new repo=mariadb name=table_users
```

Template:
```sql
-- +migrate Up
CREATE TABLE IF NOT EXISTS users (
  id         varchar(26)  PRIMARY KEY NOT NULL,
  username   varchar(255) NOT NULL,
  created_at timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  deleted_at int          NOT NULL DEFAULT 0
);

-- +migrate Down
DROP TABLE IF EXISTS users;
```

Standard columns: `id` (varchar 26 ULID), `created_at`, `updated_at`, `deleted_at int DEFAULT 0` (soft delete).

### 2. Domain

```go
// internal/core/domain/user.go
type User struct {
    ID        string    `sikat:"id"`
    Username  string    `sikat:"username"`
    CreatedAt time.Time `sikat:"created_at"`
    UpdatedAt time.Time `sikat:"updated_at"`
    DeletedAt int       `sikat:"deleted_at"`
}

type UserFilter struct {
    Search *string `sikat:"search"` // optional filters use *bool or *string
}
```

Rules: plain structs, `sikat` tags only, no JSON tags, include `DeletedAt int`.

### 3. Failures

```go
// shared/failure/failure.go
var ErrUserNotFound    = fail.Register("404003", "User not found", 404)
var ErrUsernameTaken   = fail.Register("409003", "Username already taken", 409)
```

Convention: `HTTPSTATUS + sequential number` (e.g. `404003`). Always register in `shared/failure/failure.go`.

### 4. Ports

**Inbound** (`internal/core/port/inbound/user.go`):
```go
//go:generate mockgen -source=user.go -destination=../../../mocks/inbound/mock_user.go -package=mocks
package inbound

type User interface {
    GetUserByID(ctx context.Context, id string) (*domain.User, error)
    CreateUser(ctx context.Context, user *domain.User) error
    UpdateUser(ctx context.Context, user *domain.User) error
    DeleteUser(ctx context.Context, id string) error
    ListUsers(ctx context.Context, filter *domain.UserFilter, pag *pagination.Pagination) (*[]domain.User, error)
}
```

**Outbound** (`internal/core/port/outbound/repositories/user.go`):
```go
//go:generate mockgen -source=user.go -destination=../../../../mocks/outbound/repositories/mock_user.go -package=mocksrepos
package repositories

type User interface {
    GetUserByID(ctx context.Context, id string) (*domain.User, error)
    CreateUser(ctx context.Context, user *domain.User) error
    UpdateUser(ctx context.Context, user *domain.User) error
    DeleteUser(ctx context.Context, id string) error
    ListUsers(ctx context.Context, filter *domain.UserFilter, pag *pagination.Pagination) (*[]domain.User, error)
}
```

Add `GetUserRepository() repositories.User` to the root `Repository` interface in `outbound/repository.go`.

### 5. Mocks

```bash
make mock
```

Mocks go to `internal/mocks/inbound/` and `internal/mocks/outbound/repositories/`.

### 6–7. Service (TDD)

**6. Write tests first** — table-driven tests for every method:

```go
// internal/core/service/user/service_test.go
func TestUserService_GetUserByID(t *testing.T) {
    tests := []struct {
        scenario    string
        id          string
        setup       func(*mocks.MockRepository, *mocksrepos.MockUser)
        wantErr     bool
        wantFailure *fail.Failure
    }{
        {
            scenario: "success",
            id:       "01JKM1234",
            setup: func(repo *mocks.MockRepository, userRepo *mocksrepos.MockUser) {
                user := &domain.User{ID: "01JKM1234", Username: "testuser"}
                repo.EXPECT().GetUserRepository().Return(userRepo).Times(1)
                userRepo.EXPECT().GetUserByID(gomock.Any(), "01JKM1234").Return(user, nil).Times(1)
            },
            wantErr: false,
        },
        {
            scenario: "not found",
            id:       "01JKM1234",
            setup: func(repo *mocks.MockRepository, userRepo *mocksrepos.MockUser) {
                repo.EXPECT().GetUserRepository().Return(userRepo).Times(1)
                userRepo.EXPECT().GetUserByID(gomock.Any(), "01JKM1234").Return(nil, sql.ErrNoRows).Times(1)
            },
            wantErr:     true,
            wantFailure: failure.ErrUserNotFound,
        },
        {
            scenario: "repo error",
            id:       "01JKM1234",
            setup: func(repo *mocks.MockRepository, userRepo *mocksrepos.MockUser) {
                repo.EXPECT().GetUserRepository().Return(userRepo).Times(1)
                userRepo.EXPECT().GetUserByID(gomock.Any(), "01JKM1234").
                    Return(nil, errors.New("db error")).Times(1)
            },
            wantErr:     true,
            wantFailure: fail.ErrInternalServer,
        },
    }

    for _, tt := range tests {
        t.Run(tt.scenario, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()
            mockRepo := mocks.NewMockRepository(ctrl)
            mockUserRepo := mocksrepos.NewMockUser(ctrl)
            tt.setup(mockRepo, mockUserRepo)

            svc := NewService(nil, nil, mockRepo, nil)
            result, err := svc.GetUserByID(context.Background(), tt.id)

            if tt.wantErr {
                require.Error(t, err)
                assert.Nil(t, result)
                if tt.wantFailure != nil {
                    assert.True(t, fail.IsFailure(err, tt.wantFailure))
                }
                return
            }
            require.NoError(t, err)
            require.NotNil(t, result)
            assert.Equal(t, tt.id, result.ID)
        })
    }
}
```

**⏸️ STOP HERE — present scenarios to user, wait for confirmation.**

**7. Implement service** (only after user confirms):

```go
// internal/core/service/user/service.go
type Service struct {
    cfg  *configs.Config
    log  logger.Logger
    repo outbound.Repository
}

func NewService(cfg *configs.Config, log logger.Logger, repo outbound.Repository) *Service {
    return &Service{cfg: cfg, log: log, repo: repo}
}

func (s *Service) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    user, err := s.repo.GetUserRepository().GetUserByID(ctx, id)
    if err != nil {
        return nil, fail.Wrap(err)
    }
    return user, nil
}
```

### 8. Register Service

In `cmd/bootstrap/dependency.go`:
```go
// Field
serviceUser Resource[*user.Service]

// Getter — always accept repo as parameter
func (d *Dependency) GetServiceUser(repo outbound.Repository) *user.Service {
    return d.serviceUser.Resolve(func() *user.Service {
        return user.NewService(d.GetConfig(), d.GetLogger(), repo)
    })
}
```

See [dependency-wiring.md](dependency-wiring.md) for full wiring details.

### 9. Repository

See [repository.md](repository.md) for full implementation.

### 10. Register Repository

In `internal/adapter/outbound/mariadb/repository.go`:
```go
// Add to struct
UserRepository portRepo.User

// Add to constructor
UserRepository: implRepo.NewUserRepository(sikat.Client),

// Add getter — transaction-aware
func (r *mariaDBRepository) GetUserRepository() portRepo.User {
    if r.sikatTx != nil {
        return implRepo.NewUserRepository(r.sikatTx)
    }
    return r.UserRepository
}
```

**CRITICAL:** Do **NOT** add `UserRepository` (or any repository field) to the `&mariaDBRepository{ ... }` struct that is created **inside** `DoInTransaction`. That transaction registry must contain only `cfg`, `log`, `sikat`, `sikatTx`, `outbox`. The getter already returns a tx-backed instance when `sikatTx != nil`.

### 11. DTOs

```go
// internal/adapter/inbound/http/handler/dto/user.go

// Request
type ReqGetUser struct {
    ID string `uri:"id" swaggerignore:"true"`
}

type ReqCreateUser struct {
    Username string `json:"username" validate:"required,max=100"`
}

func (r *ReqCreateUser) Validate() error {
    return validator.Validate(r)
}

func (r *ReqCreateUser) Transform() *domain.User {
    return &domain.User{
        ID:       ulid.Make().String(),
        Username: r.Username,
    }
}

// Response
type RespUser struct {
    ID        string    `json:"id"`
    Username  string    `json:"username"`
    CreatedAt time.Time `json:"created_at"`
}

func (r *RespUser) Transform(u *domain.User) *RespUser {
    r.ID = u.ID
    r.Username = u.Username
    r.CreatedAt = u.CreatedAt
    return r
}
```

Rules:
- `uri` for path params, `query` for query params, `json` for body (Fiber v3)
- `swaggerignore:"true"` on path params
- Never expose `DeletedAt` in responses
- `ulid.Make().String()` for IDs on create

### 12. HTTP Handler

```go
// internal/adapter/inbound/http/handler/user.go
type UserHandler struct {
    cfg *configs.Config
    log logger.Logger
    svc inbound.User
}

func NewUserHandler(cfg *configs.Config, log logger.Logger, svc inbound.User) *UserHandler {
    return &UserHandler{cfg: cfg, log: log, svc: svc}
}

// @Summary Get user by ID
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} response.Response{data=dto.RespUser}
// @Failure 404 {object} response.Response
// @Router /users/{id} [get]
func (h *UserHandler) GetUserByID(c fiber.Ctx) error {
    ctx := c.Context()

    var req dto.ReqGetUser
    if err := c.Bind().URI(&req); err != nil {
        return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
    }

    user, err := h.svc.GetUserByID(ctx, req.ID)
    if err != nil {
        return err
    }

    return response.SuccessOK(c, new(dto.RespUser).Transform(user))
}
```

Binding: `c.Bind().URI()` for path params, `c.Bind().Query()` for query, `c.Bind().Body()` for body.
Errors: wrap parse/validation errors with `fail.ErrBadRequest`, propagate service errors directly.

### 13. Register Handler

In `cmd/bootstrap/dependency.go`:
```go
func (d *Dependency) GetHTTPHandlers() []http.Handler {
    return d.httpHandlers.Resolve(func() []http.Handler {
        repo := d.GetRepository(d.GetSikatMain())
        return []http.Handler{
            httpHandler.NewUserHandler(d.GetConfig(), d.GetLogger(), d.GetServiceUser(repo)),
        }
    })
}
```

### 14. Swagger

Run `make swag` after adding godoc annotations to all handler methods.

### 15. Run Migration

```bash
make migrate-up repo=mariadb
```
