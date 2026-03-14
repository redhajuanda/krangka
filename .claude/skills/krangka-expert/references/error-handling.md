# Error Handling (fail package)

## Critical: Always Wrap Errors

**Every error must be wrapped with this package so stack traces are recorded.**

- **Create an error** → `fail.New` / `fail.Newf`
- **Return a callee's error** → `fail.Wrap(err)` or `fail.Wrapf(err, "context")`
- **Never** `return err` bare or use `fmt.Errorf` for application errors

`fail.Wrap(err)` returns `nil` when `err` is `nil` — safe to use directly in return statements.

## Two Types

| Type | What it is | Audience |
|------|-----------|----------|
| `*fail.Fail` | Internal error with stack trace. Implements `error`. | Internal only — logs, traces. Never send to users. |
| `*fail.Failure` | Public-safe: `Code`, `Message`, `HTTPStatus`. | End users — HTTP/gRPC responses. |

A `*fail.Fail` optionally carries a `*fail.Failure`. Without one, falls back to `fail.ErrInternalServer`.

## Creating Errors

```go
return fail.New("user record is corrupted")
return fail.Newf("user %d not found", userID)
return fail.Wrap(err)                           // wrap callee error
return fail.Wrapf(err, "failed to load user %d", userID)
```

## Attaching a Public Failure

```go
return fail.Wrap(err).WithFailure(fail.ErrNotFound)
return fail.New("quota exceeded").WithFailure(fail.ErrTooManyRequest)

// With extra data (e.g. validation errors)
return fail.New("invalid input").
    WithFailure(fail.ErrUnprocessable).
    WithData(validationErrs)
```

## Built-in Failure Definitions

```go
fail.ErrInternalServer  // 500 — default fallback
fail.ErrBadRequest      // 400
fail.ErrUnauthorized    // 401
fail.ErrForbidden       // 403
fail.ErrNotFound        // 404
fail.ErrConflict        // 409
fail.ErrUnprocessable   // 422
fail.ErrTooManyRequest  // 429
```

## Custom Failure Definitions

**Centralize ALL custom failures in `shared/failure/failure.go`.** Never define in handlers, services, or other packages.

```go
// shared/failure/failure.go
var ErrUserSuspended = fail.Register("403001", "Your account has been suspended", 403)
var ErrOrderNotFound = fail.Register("404001", "Order not found", 404)
```

Convention: `HTTPSTATUS + sequential number` per entity (e.g. `404001`, `404002`, `409001`).

Using custom failures:
```go
import "your-module/shared/failure"

return fail.Wrap(err).WithFailure(failure.ErrOrderNotFound)
```

## Extracting at Handler Layer

```go
// MustExtract — always returns usable *fail.Fail, even for plain errors
f := fail.MustExtract(err)
logger.Error(f.OriginalError()) // full stack trace — NEVER log f.Error()
pf := f.GetFailure()           // never nil
respondJSON(w, pf.HTTPStatus, pf.Code, pf.Message, f.Data())
```

## Checking for Specific Failure

```go
if fail.IsFailure(err, fail.ErrNotFound) {
    // handle not found
}

pf := fail.FailureOf(err) // returns *fail.Failure, never nil
w.WriteHeader(pf.HTTPStatus)
```

## Tweaking a Failure (One-Off)

```go
return fail.Wrap(err).WithFailure(
    fail.ErrNotFound.TemperMessage("The order you requested does not exist"),
)
```

> Warning: `Temper*` mutates in place. Use a dedicated `fail.Register(...)` var instead for shared use.

## Per-Layer Rules

### Repository Layer
```go
func (r *widgetRepository) GetWidgetByID(ctx context.Context, id string) (*domain.Widget, error) {
    // ...
    if errors.Is(err, sql.ErrNoRows) {
        return nil, fail.Wrap(err).WithFailure(failure.ErrWidgetNotFound) // typed failure
    }
    return nil, fail.Wrap(err) // always wrap for stack trace
}
```

### Service Layer
```go
func (s *Service) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    user, err := s.repo.GetUserRepository().GetUserByID(ctx, id)
    if err != nil {
        return nil, fail.Wrap(err) // propagate (typed failure already attached by repo)
    }
    return user, nil
}
```

### Handler Layer
```go
func (h *UserHandler) CreateUser(c fiber.Ctx) error {
    var req dto.ReqCreateUser
    if err := c.Bind().Body(&req); err != nil {
        return fail.Wrap(err).WithFailure(fail.ErrBadRequest) // parse error
    }
    if err := req.Validate(); err != nil {
        return fail.Wrap(err).WithFailure(fail.ErrBadRequest) // validation error
    }
    return h.svc.CreateUser(c.Context(), req.Transform()) // propagate service errors directly
}
```

## Rules for AI Agents

1. **Always wrap or create errors with this package.** Never `return err` bare. Never `fmt.Errorf` for application errors.
2. **Always use `fail.New` / `fail.Wrap` / `fail.Wrapf`** — never construct `fail.Fail{}` directly.
3. **Never log `f.Error()`** — use `f.OriginalError()` for full stack trace.
4. **Never send `*fail.Fail` to end users.** Only `*fail.Failure` fields.
5. **Register custom failures in `shared/failure` only**, at package level.
6. **Use `fail.MustExtract` at handler boundaries** unless ok-check is needed.
7. **Always chain `.WithFailure(...)` when error has user-visible meaning** other than internal server error.

**Anti-patterns:** `return err`; `return fmt.Errorf(...)`; logging `f.Error()`; sending `*fail.Fail` to users.
