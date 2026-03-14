# Krangka Testing Patterns

**Read this before writing any test.**

---

## TDD Cycle — Mandatory

Always follow **Red → Green → Refactor**. Never write implementation before a failing test exists.

1. **Red**: Write a failing test that describes the desired behavior
2. **Green**: Write the minimal implementation to make the test pass
3. **Refactor**: Clean up without changing behavior, keeping tests green

When to write tests first:
- New service method → write `service_test.go` first
- New domain behavior → write `domain_test.go` first
- New HTTP handler → write `handler_test.go` first
- Bug fix → write a test that reproduces the bug first, then fix

Test file placement:
```
internal/core/domain/<entity>_test.go
internal/core/service/<entity>/service_test.go
internal/adapter/inbound/http/handler/<entity>_test.go
```

---

## Test Naming Convention

- One test function per method: `Test{TypeName}_{MethodName}`
- Subtest name is `tt.scenario` (e.g. `"success"`, `"not found"`, `"repo error"`)
- Examples: `TestNoteService_GetNoteByID`, `TestNoteHandler_CreateNote`

---

## Table-Driven Tests — Always Required

**Always use table-driven tests.** One function per method, one row per scenario.

### Required struct fields

| Field | Purpose |
|-------|---------|
| `scenario` | Short name for `t.Run(tt.scenario, ...)` |
| `scenarioDescription` | Documents the scenario for maintainability |
| Input fields | Method args (e.g. `id`, `email`, `payload`) |
| `setup` | Configures mocks; receives typed mock pointers |
| `wantErr` | When true, expect an error |
| `wantFailure` | Optional; assert `fail.IsFailure(err, tt.wantFailure)` |
| `wantMsg` | Optional; assert `assert.Contains(t, err.Error(), tt.wantMsg)` |
| `wantX` | Optional success assertions (e.g. `wantNote`, `wantStatus`) |

### Minimal example (service)

```go
func TestNoteService_GetNoteByID(t *testing.T) {
    tests := []struct {
        scenario            string
        scenarioDescription string
        id                  string
        setup               func(*mocks_outbound.MockRepository, *mocks_outbound_repositories.MockNote)
        wantErr             bool
        wantNote            *domain.Note
    }{
        {
            scenario:            "success",
            scenarioDescription: "Existing ID returns the note.",
            id:                  "01JXYZ",
            setup: func(repo *mocks_outbound.MockRepository, noteRepo *mocks_outbound_repositories.MockNote) {
                repo.EXPECT().GetNoteRepository().Return(noteRepo)
                noteRepo.EXPECT().GetNoteByID(gomock.Any(), "01JXYZ").
                    Return(&domain.Note{ID: "01JXYZ", Title: "Test"}, nil)
            },
            wantErr:  false,
            wantNote: &domain.Note{ID: "01JXYZ", Title: "Test"},
        },
        {
            scenario:            "not found",
            scenarioDescription: "Non-existent ID returns ErrNotFound.",
            id:                  "01JXYZ",
            setup: func(repo *mocks_outbound.MockRepository, noteRepo *mocks_outbound_repositories.MockNote) {
                repo.EXPECT().GetNoteRepository().Return(noteRepo)
                noteRepo.EXPECT().GetNoteByID(gomock.Any(), "01JXYZ").
                    Return(nil, fail.New("not found").WithFailure(fail.ErrNotFound))
            },
            wantErr:  true,
            wantNote: nil,
        },
    }
    for _, tt := range tests {
        t.Run(tt.scenario, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()
            mockRepo := mocks_outbound.NewMockRepository(ctrl)
            mockNoteRepo := mocks_outbound_repositories.NewMockNote(ctrl)
            tt.setup(mockRepo, mockNoteRepo)
            svc := note.NewService(cfg, log, mockRepo)
            result, err := svc.GetNoteByID(context.Background(), tt.id)
            if tt.wantErr {
                require.Error(t, err)
                assert.Nil(t, result)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.wantNote, result)
        })
    }
}
```

### Full example with error assertions

```go
func TestService_Login(t *testing.T) {
    tests := []struct {
        scenario            string
        scenarioDescription string
        email               string
        password            string
        setup               func(*mocks_outbound.MockRepository, *mocks_outbound_repositories.MockUser)
        wantErr             bool
        wantFailure         *fail.Failure
        wantMsg             string
    }{
        {
            scenario:            "success",
            scenarioDescription: "Valid credentials; returns access and refresh tokens.",
            email:               "user@example.com",
            password:            validPassword,
            setup: func(repo *mocks_outbound.MockRepository, userRepo *mocks_outbound_repositories.MockUser) {
                repo.EXPECT().GetUserRepository().Return(userRepo).Times(1)
                userRepo.EXPECT().GetUserByEmail(gomock.Any(), "user@example.com").Return(user, nil).Times(1)
                userRepo.EXPECT().UpdateUser(gomock.Any(), gomock.Any()).Return(nil).Times(1)
            },
            wantErr: false,
        },
        {
            scenario:            "user not found",
            scenarioDescription: "Email does not exist; GetUserByEmail returns ErrNotFound.",
            email:               "nonexistent@example.com",
            password:            "any",
            setup: func(repo *mocks_outbound.MockRepository, userRepo *mocks_outbound_repositories.MockUser) {
                repo.EXPECT().GetUserRepository().Return(userRepo).Times(1)
                userRepo.EXPECT().GetUserByEmail(gomock.Any(), "nonexistent@example.com").
                    Return(nil, fail.New("user not found").WithFailure(fail.ErrNotFound)).Times(1)
            },
            wantErr:     true,
            wantFailure: fail.ErrUnauthorized,
            wantMsg:     "invalid credentials",
        },
        {
            scenario:            "repo error",
            scenarioDescription: "GetUserByEmail returns a non-NotFound error; error is propagated.",
            email:               "user@example.com",
            password:            "any",
            setup: func(repo *mocks_outbound.MockRepository, userRepo *mocks_outbound_repositories.MockUser) {
                repo.EXPECT().GetUserRepository().Return(userRepo).Times(1)
                userRepo.EXPECT().GetUserByEmail(gomock.Any(), "user@example.com").
                    Return(nil, errors.New("database connection failed")).Times(1)
            },
            wantErr:     true,
            wantFailure: fail.ErrInternalServer,
            wantMsg:     "database connection failed",
        },
    }

    for _, tt := range tests {
        t.Run(tt.scenario, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()

            mockRepo := mocks_outbound.NewMockRepository(ctrl)
            mockUserRepo := mocks_outbound_repositories.NewMockUser(ctrl)
            tt.setup(mockRepo, mockUserRepo)

            svc := NewService(cfg, log, mockRepo)
            result, err := svc.Login(context.Background(), tt.email, tt.password)

            if tt.wantErr {
                require.Error(t, err)
                assert.Nil(t, result)
                if tt.wantFailure != nil {
                    assert.True(t, fail.IsFailure(err, tt.wantFailure))
                }
                if tt.wantMsg != "" {
                    assert.Contains(t, err.Error(), tt.wantMsg)
                }
                return
            }

            require.NoError(t, err)
            require.NotNil(t, result)
            assert.NotEmpty(t, result.AccessToken)
            assert.NotEmpty(t, result.RefreshToken)
        })
    }
}
```

### HTTP handler example (Fiber)

```go
func TestNoteHandler_CreateNote(t *testing.T) {
    tests := []struct {
        scenario            string
        scenarioDescription string
        body                string
        setup               func(*inboundmocks.MockNote)
        wantStatus          int
    }{
        {
            scenario:            "success",
            scenarioDescription: "Valid payload returns 201.",
            body:                `{"title":"Test","content":"body"}`,
            setup:               func(m *inboundmocks.MockNote) { m.EXPECT().CreateNote(gomock.Any(), gomock.Any()).Return(nil) },
            wantStatus:          fiber.StatusCreated,
        },
        {
            scenario:            "invalid payload",
            scenarioDescription: "Malformed JSON returns 400.",
            body:                `{invalid`,
            setup:               func(m *inboundmocks.MockNote) {},
            wantStatus:          fiber.StatusBadRequest,
        },
    }
    for _, tt := range tests {
        t.Run(tt.scenario, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()
            mockSvc := inboundmocks.NewMockNote(ctrl)
            tt.setup(mockSvc)
            handler := http.NewHandler(mockSvc)
            app := fiber.New()
            app.Post("/notes", handler.CreateNote)
            req := httptest.NewRequest(http.MethodPost, "/notes", strings.NewReader(tt.body))
            req.Header.Set("Content-Type", "application/json")
            resp, err := app.Test(req)
            require.NoError(t, err)
            assert.Equal(t, tt.wantStatus, resp.StatusCode)
        })
    }
}
```

---

## Required Imports for Tests

```go
import (
    "context"
    "errors"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "go.uber.org/mock/gomock"

    "github.com/redhajuanda/komon/fail"

    mocks_outbound "github.com/capioteknologi/krangka/internal/mocks/outbound"
    mocks_outbound_repositories "github.com/capioteknologi/krangka/internal/mocks/outbound/repositories"
)
```

---

## Mock Rules

- **Never hand-write mocks** — always use generated ones from `internal/mocks/outbound/` and `internal/mocks/inbound/`
- Run `make mock` after any port interface change
- Use `gomock.Any()` for context parameters (OpenTelemetry wraps contexts)
- Use `gomock.InOrder(...)` when call order matters
- Use `.Times(n)` or `.AnyTimes()` explicitly when a mock is called multiple times
- Prefer `require.NoError` over `assert.NoError` when subsequent assertions depend on no error

---

## Coverage Requirements

| Layer | Minimum Coverage |
|-------|-----------------|
| Domain | 100% (pure logic, no excuses) |
| Service | ≥ 80% (all happy paths + primary error paths) |
| HTTP handlers | ≥ 70% (status codes, request validation) |

---

## Build Tags

- Unit tests (service, domain): no build tag needed, run with `make test`
- Integration tests (repository/DB): use `//go:build integration`, run with `make test-integration`

```go
//go:build integration

package mariadb_test
```

---

## Test Helpers

- Use `t.Helper()` in shared test helpers for accurate failure line numbers
- Use `t.Cleanup(func() { ... })` instead of `defer` in test helpers
- Prefer `context.Background()` in tests; only use real timeouts when testing timeout behavior

---

## What AI Must Generate

When asked to implement a method or feature:
1. Generate the `_test.go` file **before** the implementation file
2. Always use table-driven tests — one function per method, one row per scenario
3. Include at minimum: happy path + at least one error path
4. Use `scenario` and `scenarioDescription` in every table row
5. For error cases, use `wantErr`, `wantFailure`, and `wantMsg` when applicable
6. Use mockgen-generated mocks from `internal/mocks/`
7. Always import `testify/require` and `testify/assert`
