# Agent Transparency

When you use a skill, read a file, or have rules applied to your context, **always** state it clearly in your reply so the user can see exactly what is in effect. Do not skip any.

## Rules applied

- **When rules apply** to the current conversation (e.g. from `.claude/rules/`), list them at the start of your reply so the user knows which rules you are following.
- Use a consistent format. List each rule that applies (by its description or filename).

**Format:**

```
**Apply:**
- rule 1 (e.g. `krangka_code_style` — code style)
- rule 2 (e.g. `krangka_go_testing` — Go testing)
- rule 3 (e.g. `agent_transparency` — this rule)
```

- If only one rule applies, still announce it: **Apply:** rule name/description.
- Do not skip: every rule that is in your context for this reply must be listed.

## Skills

- **Before or when using a skill**, say explicitly which skill you are using.
- Use a consistent, visible format so it stands out in the chat.

**Format:** Start the relevant part of your message with a clear line such as:

```
**Using skill:** `skill-name` (e.g. `krangka-expert`, `krangka-query-review`, `create-rule`)
```

- If you use **multiple skills** in one reply, list each one.
- If you only read the skill file to follow its instructions, still say you used that skill (e.g. "Using skill: create-rule").

## Files read

- **For every file you read** (via Read, or when a tool reads a file), tell the user which file(s) you read.
- One file or many, do not omit any.

**Always use the full path** (relative to project root). Never refer to a file by only its basename or a vague label.

- ❌ **Wrong:** "Reading README.md" or "checkout readme" or "krangka docs" — ambiguous when multiple READMEs or docs exist.
- ✅ **Right:** `README.md` (root), `.krangka/docs/README.md`, `.krangka/docs/01_architecture-overview.md` — each file with its path so the user knows exactly which one.

**Format:** List each file with its path on its own line or in a clear list:

```
**Reading:**
- `README.md`
- `.krangka/docs/README.md`
- `.krangka/docs/01_architecture-overview.md`
```

- When reading from a folder (e.g. `.krangka/docs/`), **list each file you read by full path** — do not say "krangka docs" or "documentation"; name every file (e.g. `.krangka/docs/README.md`, `.krangka/docs/01_architecture-overview.md`).
- Same-name files in different directories (e.g. two `README.md`) must always be distinguished by path.
- If you read a file in the same message where you use a skill, mention both: the skill and the file(s) with paths.

## Example

Good opening when rules apply, you used one skill, and read two files:

```
**Apply:**
- `krangka_code_style`
- `agent_transparency`

**Using skill:** `krangka-expert`
**Reading:**
- `.claude/skills/krangka-expert/SKILL.md`
- `internal/core/service/todo/service.go`

[Then your actual answer or actions...]
```

When reading project docs and multiple READMEs exist, always distinguish by path:

```
**Reading:**
- `README.md` (project root)
- `.krangka/docs/README.md`
- `.krangka/docs/01_architecture-overview.md`
```

## Rules

1. **Whenever rules apply**, announce them: list each rule in effect at the start (e.g. **Apply:** rule 1, rule 2). Do not skip any rule that is in your context.
2. **Never skip** a skill or a file: every skill use and every file read in your reply must be announced.
3. **Always use full paths** for files (relative to project root). No bare filenames like "README.md" alone; no vague labels like "krangka docs" — list each file with its path.
4. **Announce before or at the start** of the action (e.g. at the beginning of your message or right before the paragraph where you use that tool).
5. Keep the format consistent so the user can quickly scan for **Apply**, **Using skill**, and **Reading** in every response.

---

# Krangka Code Style

> Applies to: `**/*.go`

## Constants

- **Domain-specific** constants (e.g. statuses, types, or values that belong to one bounded context) → put them in the **domain** package (e.g. `internal/core/domain/<entity>.go`).
- **Non-domain** constants used in many places across the project → put them in **`shared/constants`**.

Always use `const ()` blocks or `var ()` blocks with doc comments in both cases.

## Service struct conventions

- **Interface assertion** after constructor: `var _ inbound.File = (*Service)(nil)`
- **Constructor comment**: `// NewService creates a new file service.`
- **Implementation comment**: `// Service implements inbound.Color.`

## Error handling — `fail.Wrap` everywhere

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

**Never pass an error to `fail.Wrap` without checking `err != nil` first.** This is a correctness rule, not a style preference — skipping the check causes real bugs.

In Go, `error` is an interface. An interface value is only `nil` when **both** its type and value are `nil`. If a function returns a concrete error type (e.g. `*fail.Fail`, `*MyError`) as a typed nil, the returned `error` interface is **non-nil** — it holds `(*MyError)(nil)`, which has a type but a nil underlying value. Passing this to `fail.Wrap` wraps a "non-nil" error with a nil value inside, which can cause nil pointer dereferences and panics downstream.

```go
// Example of the Go nil interface trap:
func doSomething() (*fail.Fail, error) {
	var f *fail.Fail // nil pointer
	return f, f      // the error interface is NOT nil — it holds (*fail.Fail)(nil)
}

err := doSomething() // err != nil is TRUE even though the underlying pointer is nil
fail.Wrap(err)       // wraps a non-nil interface with nil value → potential panic
```

**Always guard with `if err != nil`:**

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

This applies to **every** `fail.Wrap` and `fail.Wrapf` call, no exceptions:

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

Create custom failures in `shared/failure/failure.go` using the `fail.Failure` struct. Use standard failures for generic cases.

**Create custom when** the error has a specific business meaning, clients need to handle it differently, or you want a stable error code for API consumers.

```go
// shared/failure/failure.go
var ErrMaxBankAccount = &fail.Failure{Code: "400001", Message: "Maximum bank accounts allowed per agent exceeded", HTTPStatus: 400}
var ErrUserNotFound   = &fail.Failure{Code: "404001", Message: "User not found", HTTPStatus: 404}
var ErrUsernameTaken  = &fail.Failure{Code: "409001", Message: "Username already taken", HTTPStatus: 409}
```

Convention: `HTTPSTATUS + sequential number` (e.g. `400001`, `404002`, `409001`). Never define failures in handlers, services, or other packages.

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

## Comment

Every function must have a brief comment above it that explains what the function does (and, when relevant, the business logic). Prefer one short sentence; add more only if needed for non-obvious behavior.

```go
// UploadFile uploads a file to the storage.
func (s *Service) UploadFile(ctx context.Context, req *inbound.UploadFileRequest) (*inbound.UploadFileResult, error) {
	// ...
}
```

## Tracer

Always use `tracer.Trace(ctx)` and `defer span.End()` in every function that takes `context.Context` as the parameter.

```go
ctx, span := tracer.Trace(ctx)
defer span.End()
```

## Variable grouping with `var()`

Whenever a variable **can** be initialized at the start of the function, initialize it there in a grouped `var()` block. That way the reader sees at a glance which variables the function uses. If initialization at the start is not possible (e.g. depends on control flow or is only known later), that's fine — but when multiple variables are initialized later (e.g. inside a transaction callback), still group them in a `var()` block at the beginning of that scope (e.g. repo getters at the start of the transaction). Align `=` for visual consistency.

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

### MariaDB transaction registry — CRITICAL

In `internal/adapter/outbound/mariadb/repository.go`, the struct created **inside** `DoInTransaction` (the transaction registry) must **only** contain: `cfg`, `log`, `sikat`, `sikatTx`, `outbox`. **Never** add repository fields (e.g. `NoteRepository`, `UserRepository`) to that struct. The getters (`GetXxxRepository()`) already return tx-backed instances when `sikatTx != nil`; adding repo fields to the transaction registry is invalid.

### MariaDB DoInTransaction and handleTransaction — do not modify

Do **not** change anything inside the function bodies of `DoInTransaction` or `handleTransaction` in `internal/adapter/outbound/mariadb/repository.go`. Adding, removing, or editing logic in those two functions is off-limits.

### When to use `var()`

- **1 or more variables** and can be initialized at the start of a function → group in `var()`
- **Single variable** and can not be initialized at the start of a function → use inline variable
- **1 or more repo tx getters** → group in `var()` at the start of the transaction callback

### Always use table driven tests

Always use table driven tests for readability. Define all scenarios in a slice, then loop with `t.Run(tt.scenario, ...)`.

use skill `krangka-expert` for more details.
