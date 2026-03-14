# Krangka Code Style

---

## Constants

- **Domain-specific** constants (statuses, types, values belonging to one bounded context) → put in the **domain** package (`internal/core/domain/<entity>.go`)
- **Non-domain** constants used across the project → put in **`shared/constants`**

Always use `const ()` or `var ()` blocks with doc comments in both cases.

---

## Service Struct Conventions

- **Interface assertion** after constructor: `var _ inbound.File = (*Service)(nil)`
- **Constructor comment**: `// NewService creates a new file service.`
- **Implementation comment**: `// Service implements inbound.Color.`

---

## Error Handling — `fail.Wrap` Everywhere

**This is the most important convention.** Every error must be wrapped with the `fail` package so stack traces are recorded.

```go
// ✅ Correct
return fail.Wrap(err)
return fail.Wrapf(err, "failed to load user %s", userID)
return fail.New("user record is corrupted")

// ❌ Wrong — bare errors lose stack traces
return err
return fmt.Errorf("failed: %w", err)
```

### CRITICAL: Always nil-check before wrapping

**Never pass an error to `fail.Wrap` without checking `err != nil` first.**

In Go, `error` is an interface. An interface value is only `nil` when **both** its type and value are `nil`. If a function returns a concrete error type (e.g. `*fail.Fail`) as a typed nil, the returned `error` interface is **non-nil** — passing this to `fail.Wrap` can cause nil pointer dereferences and panics.

```go
// ✅ Correct — explicit nil check before wrapping
result, err := s.repo.GetUserRepository().GetUserByID(ctx, id)
if err != nil {
    return nil, fail.Wrap(err)
}
return result, nil

// ❌ DANGEROUS — skipping nil check can wrap a typed-nil interface
result, err := s.repo.GetUserRepository().GetUserByID(ctx, id)
return result, fail.Wrap(err)
```

```go
// ❌ DANGEROUS
_, err := repo.CreateWidget(ctx, w)
return fail.Wrap(err)

// ✅ Correct
_, err := repo.CreateWidget(ctx, w)
if err != nil {
    return fail.Wrap(err)
}
return nil
```

`fail.New` is exempt — it always creates a non-nil error by definition.

### Attaching public failures

```go
return fail.Wrap(err).WithFailure(fail.ErrNotFound)
return fail.New("quota exceeded").WithFailure(fail.ErrTooManyRequest)
return fail.New("invalid input").WithFailure(fail.ErrBadRequest).WithData(validationErrs)
```

### Built-in failures

`fail.ErrBadRequest` (400), `fail.ErrUnauthorized` (401), `fail.ErrForbidden` (403), `fail.ErrNotFound` (404), `fail.ErrConflict` (409), `fail.ErrUnprocessable` (422), `fail.ErrTooManyRequest` (429), `fail.ErrInternalServer` (500).

### Custom failures

Create custom failures in `shared/failure/failure.go`. Use standard failures for generic cases.

**Create custom when** the error has a specific business meaning, clients need to handle it differently, or you want a stable error code for API consumers.

```go
// shared/failure/failure.go
var ErrMaxBankAccount = &fail.Failure{Code: "400001", Message: "Maximum bank accounts allowed per agent exceeded", HTTPStatus: 400}
var ErrUserNotFound   = &fail.Failure{Code: "404001", Message: "User not found", HTTPStatus: 404}
var ErrUsernameTaken  = &fail.Failure{Code: "409001", Message: "Username already taken", HTTPStatus: 409}
```

Convention: `HTTPSTATUS + sequential number` (e.g. `400001`, `404002`). Never define failures in handlers, services, or other packages.

### Per-layer error rules

**Repository** — map `sql.ErrNoRows` to typed failure, wrap everything else:

```go
if errors.Is(err, sql.ErrNoRows) {
    return nil, fail.Wrap(err).WithFailure(failure.ErrWidgetNotFound)
}
return nil, fail.Wrap(err)
```

**Service** — propagate (typed failure already attached by repo):

```go
if err != nil {
    return nil, fail.Wrap(err)
}
```

**Handler** — wrap parse/validation errors with `fail.ErrBadRequest`, propagate service errors directly:

```go
if err := c.Bind().Body(&req); err != nil {
    return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
}
return h.svc.CreateUser(c.Context(), req.Transform()) // propagate directly
```

---

## Comments

Every function must have a brief comment above it explaining what it does (and business logic when relevant). Prefer one short sentence; add more only for non-obvious behavior.

```go
// UploadFile uploads a file to the storage.
func (s *Service) UploadFile(ctx context.Context, req *inbound.UploadFileRequest) (*inbound.UploadFileResult, error) {
    // ...
}
```

---

## Tracer

Always use `tracer.Trace(ctx)` and `defer span.End()` in every function that takes `context.Context`.

```go
ctx, span := tracer.Trace(ctx)
defer span.End()
```

---

## Variable Grouping with `var()`

Whenever a variable **can** be initialized at the start of the function, initialize it there in a grouped `var()` block. Align `=` for visual consistency.

**Service methods — repos, config, derived values:**

```go
var (
    fileRepo   = s.repo.GetFileRepository()
    bucket     = s.cfg.Storage.BucketPrivate
    objectName = utils.GenerateStorageObjectName(req.Filename)
)
```

**Transaction callbacks — group repo getters at the start of the callback:**

```go
out, err := s.repo.DoInTransaction(ctx, func(repo outbound.Repository) (any, error) {
    var (
        agentRepo = repo.GetAgentRepository()
        roleRepo  = repo.GetRoleRepository()
        userRepo  = repo.GetUserRepository()
    )
    // ⚠️ Use `repo` (lambda arg), NEVER `s.repo` inside transactions
    // ...
})
```

### When to use `var()`

| Situation | Convention |
|-----------|-----------|
| 1 or more variables initializable at function start | Group in `var()` |
| Single variable not initializable at function start | Inline variable |
| 1 or more repo tx getters | Group in `var()` at start of transaction callback |
