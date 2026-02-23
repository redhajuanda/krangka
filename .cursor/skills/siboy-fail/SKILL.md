---
name: krangka-fail
description: Explains how to use the komon/fail package for error handling in Go. Use when writing, reading, or reviewing code that returns errors, creates errors, wraps errors, or handles errors at every layer. In this application errors must always be wrapped with this package so stack traces are recorded.
---

# fail package — Error Handling Guide

## Critical: always wrap errors (stack trace)

**In this application, errors must always be wrapped or created with this package.** The package records a stack trace at the call site. If you return a raw error (e.g. `return err`) or use `fmt.Errorf`, no stack trace is recorded and debugging is much harder.

- **When you create an error** → use `fail.New` / `fail.Newf`.
- **When you return an error from a callee** → use `fail.Wrap(err)` or `fail.Wrapf(err, "context")`. Never `return err` alone.
- **Every layer** (repository, service, handler) should wrap errors before returning so the stack points to where the error was produced or first propagated.

`fail.Wrap(err)` and `fail.Wrapf(err, ...)` return `nil` when `err` is `nil`, so they are safe to use directly in return statements.

---

## Two types — understand the distinction first

| Type | What it is | Audience |
|------|-----------|----------|
| `*fail.Fail` | Internal error with stack trace and cause. Implements `error`. | **Internal only** — logs, traces. Never send to end users. |
| `*fail.Failure` | Public-safe error definition: `Code`, `Message`, `HTTPStatus`. | **End users** — returned in HTTP/gRPC responses. |

A `*fail.Fail` optionally carries a `*fail.Failure` that determines what the end user sees.
If no `*fail.Failure` is attached, it falls back to `fail.ErrInternalServer` automatically.

---

## Creating a Fail (internal error)

Use these constructors so a **stack trace is recorded** at the call site:

```go
// From a string
return fail.New("user record is corrupted")

// From a format string
return fail.Newf("user %d not found", userID)

// Wrapping an existing error — use this whenever returning an error from another call
return fail.Wrap(err)

// Wrapping with added context
return fail.Wrapf(err, "failed to load user %d", userID)
```

All of these capture a stack trace. **Never** `return err` or `return fmt.Errorf(...)` for application errors — use `fail.Wrap` / `fail.Wrapf` so the stack is recorded.
`Wrap` and `Wrapf` return `nil` when `err` is `nil` — safe to use directly.

---

## Attaching a public Failure to a Fail

Use `.WithFailure(pf)` to specify what the end user sees:

```go
return fail.Wrap(err).WithFailure(fail.ErrNotFound)

return fail.New("quota exceeded").WithFailure(fail.ErrTooManyRequest)

// With extra data (e.g. validation errors)
return fail.New("invalid input").
    WithFailure(fail.ErrUnprocessable).
    WithData(validationErrs)
```

Without `.WithFailure(...)`, the end user always gets `fail.ErrInternalServer`.

---

## Built-in Failure definitions

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

---

## Registering custom Failure definitions

**Centralize all custom failure definitions in `shared/failure`.** Do not define them in handlers, services, or other packages. Define once at package level (never inside functions) in that folder:

```go
// shared/failure/failure.go
var ErrUserSuspended = fail.Register("403001", "Your account has been suspended", 403)
var ErrOrderNotFound = fail.Register("404001", "Order not found", 404)
```

Use them from other packages by importing the failure package and attaching the var:

```go
import "your-module/shared/failure"

return fail.Wrap(err).WithFailure(failure.ErrOrderNotFound)
```

---

## Extracting at the handler layer (HTTP / gRPC)

Use `fail.MustExtract` — always returns a usable `*fail.Fail`, even for plain errors:

```go
func handleErr(w http.ResponseWriter, err error) {
    f := fail.MustExtract(err)

    // Log the internal error with full stack trace
    logger.Error(f.OriginalError())

    // Respond to the user using the public Failure
    pf := f.GetFailure() // never nil
    respondJSON(w, pf.HTTPStatus, pf.Code, pf.Message, f.Data())
}
```

Use `fail.Extract` when you want to distinguish "is this a *Fail?" vs "is this a plain error?":

```go
f, ok := fail.Extract(err)
if !ok {
    // err is a plain Go error, not a *fail.Fail
}
```

---

## Checking for a specific Failure

```go
if fail.IsFailure(err, fail.ErrNotFound) {
    // handle not found specifically
}

// Or get the Failure directly
pf := fail.FailureOf(err) // returns *fail.Failure, never nil (falls back to ErrInternalServer)
w.WriteHeader(pf.HTTPStatus)
```

---

## Tweaking a Failure for a one-off response

Use `Temper*` methods to create a modified copy without altering the global var:

```go
return fail.Wrap(err).WithFailure(
    fail.ErrNotFound.TemperMessage("The order you requested does not exist"),
)
```

> **Warning**: `Temper*` mutates in place and returns `self`. If modifying a shared global, clone it first or use a dedicated `fail.Register(...)` var instead.

---

## Rules for AI agents

1. **Always wrap or create errors with this package so stack traces are recorded.** Never `return err`; use `return fail.Wrap(err)` or `fail.Wrapf(err, "…")`. Never use `fmt.Errorf` for application errors. At every layer (repo, service, handler), wrap before returning.
2. **Always use `fail.New` / `fail.Wrap` / `fail.Wrapf`** to create errors — never construct `fail.Fail{}` directly.
3. **Never log `f.Error()`** at the handler layer. Use `f.OriginalError()` for logging — it has the full stack.
4. **Never send `*fail.Fail` to end users.** Only send `*fail.Failure` fields (`Code`, `Message`, `HTTPStatus`).
5. **Register custom `*fail.Failure` vars in `shared/failure` only**, at package level (never inside functions). Do not define custom failures in handlers, services, or other packages.
6. **Use `fail.MustExtract` at handler boundaries**, not `fail.Extract`, unless you need the ok-check.
7. **Always chain `.WithFailure(...)` when the error has a user-visible meaning** other than internal server error.

**Anti-patterns (avoid):** `return err`; `return fmt.Errorf(...)`; logging `f.Error()` instead of `f.OriginalError()`; sending `*fail.Fail` or stack traces to end users.
