# Logger Usage

Use `github.com/redhajuanda/komon/logger` for all logging.

## Rules

### 1. WithContext — Always use when `ctx` is available

```go
// ✅ Correct
logger.WithContext(ctx).Info("user created")

// ❌ Wrong — request_id/correlation_id will not appear in logs
logger.Info("user created")
```

### 2. Message — Clear, human-readable

For simple identifiers (id, code, name), in-message interpolation is acceptable.

```go
// ✅ Correct
logger.WithContext(ctx).Infof("note %s fetched", noteID)
logger.WithContext(ctx).Infof("user %s created", userID)
```

### 3. Structured Data — Use WithParam or WithParams

Put counts, status, maps, and multi-field context in `WithParam` or `WithParams`.

```go
// ✅ Correct — structured/multiple fields in params
logger.WithContext(ctx).WithParams(logger.Params{
    "order_id": orderID,
    "status":   "paid",
}).Info("order payment completed")

logger.WithContext(ctx).WithParam("tenant_id", tenantID).Warn("rate limit approaching")

// ❌ Wrong — complex data in message
logger.WithContext(ctx).Infof("tenant %s count %d", tenantID, count)
```

### 4. Errors — Use WithStack when stack trace helps

```go
// ✅ For expected/operational errors (no stack needed)
logger.WithContext(ctx).WithParam("err", err).Error("failed to save note")

// ✅ For unexpected/panic-like errors (stack helps debugging)
logger.WithContext(ctx).WithStack(err).Error("unexpected database error")
```

### 5. Chaining Order

`WithContext` → `WithStack` (if used) → `WithParam`/`WithParams` → level method

```go
logger.WithContext(ctx).WithStack(err).WithParam("query", q).Error("query failed")
```

## Quick Reference

| When | Do |
|------|-----|
| `ctx` available | Start with `.WithContext(ctx)` |
| Add key-value data | `.WithParam(key, value)` or `.WithParams(logger.Params{...})` |
| Unexpected error (stack trace needed) | `.WithStack(err)` before level |
| Simple id/name in message | OK to use `Infof("user %s created", id)` |

## Log Levels

| Level | When |
|-------|------|
| `Info` | Normal operations, expected events |
| `Warn` | Unexpected but non-critical situations |
| `Error` | Errors that need attention |
| `Fatal` | Unrecoverable errors (calls `os.Exit`) |
| `Debug` | Development/debugging (should not appear in production) |
