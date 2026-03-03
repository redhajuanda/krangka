---
name: krangka-logger
description: Guide for logging with github.com/redhajuanda/komon/logger. Use when adding or modifying log statements, writing handlers, services, or workers that log.
---

# Logger Usage

Use `github.com/redhajuanda/komon/logger` for all logging. Follow these rules strictly.

## Rules

### 1. WithContext — Always use when `ctx` is available

```go
// ✅ Correct
logger.WithContext(ctx).Info("user created")

// ❌ Wrong — request_id/correlation_id will not appear
logger.Info("user created")
```

### 2. Message — Clear, human-readable

Write a descriptive message. For simple identifiers (id, code, name), in-message interpolation is acceptable.

```go
// ✅ Correct — identifier in message is fine
logger.WithContext(ctx).Infof("note %s fetched", noteID)
logger.WithContext(ctx).Infof("user %s created", userID)

// ✅ Also correct — structured fields in WithParam/WithParams
logger.WithContext(ctx).WithParams(logger.Params{"order_id": id, "status": "paid"}).Info("order payment completed")
```

### 3. Parameters — Use WithParam or WithParams for structured data

Put counts, status, maps, and multi-field context in `WithParam` or `WithParams`. Simple id/code/name may go in the message or in params.

```go
// ✅ Correct — identifier in message
logger.WithContext(ctx).Infof("note %s fetched", noteID)

// ✅ Correct — structured/multiple fields in params
logger.WithContext(ctx).WithParams(logger.Params{
    "tenant_id": tenantID,
    "count":     count,
}).Warn("rate limit approaching")

// ❌ Wrong — complex data in message instead of params
logger.WithContext(ctx).Infof("tenant %s count %d", tenantID, count)
```

### 4. Errors — Use WithStack when stack trace helps

```go
// ✅ For operational errors (expected, no stack needed)
logger.WithContext(ctx).WithParam("err", err).Error("failed to save note")

// ✅ For unexpected/panic-like errors (stack helps debugging)
logger.WithContext(ctx).WithStack(err).Error("unexpected database error")
```

### 5. Chaining order

Chain in this order: `WithContext` → `WithStack` (if used) → `WithParam`/`WithParams` → level method.

```go
logger.WithContext(ctx).WithStack(err).WithParam("query", q).Error("query failed")
```

## Quick Reference

| When | Do |
|------|-----|
| `ctx` available | Start with `.WithContext(ctx)` |
| Add key-value data | `.WithParam(key, value)` or `.WithParams(logger.Params{...})` |
| Unexpected error | `.WithStack(err)` before level |
| Message | Clear string; id/code/name OK in message; complex data → WithParam/WithParams |
