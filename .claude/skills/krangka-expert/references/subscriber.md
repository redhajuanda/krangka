# Subscriber (Event Consumer)

The subscriber is a long-running Watermill-based worker consuming events from Kafka or Redis Stream. It runs via `bootstrap.Run`, implements `Runnable`, and uses handlers that register topic consumers.

For time-triggered or manual background jobs, see [workers.md](workers.md).

## Directory Structure

```
internal/adapter/inbound/subscriber/
├── subscriber.go       # Subscriber struct, NewSubscriber, OnStart/OnStop
├── router.go           # RegisterRoutes — adds middleware, delegates to handlers
├── middleware/
│   ├── request_id.go   # Sets request/correlation ID from message metadata
│   ├── idempotence.go  # Ensures at-most-once processing per message UUID
│   ├── retry.go        # Optional retry with SkipRetryError support
│   └── errors.go       # SkipRetryError type
└── handler/
    └── note.go         # Example handler
```

## Handler Contract

Every handler must implement:
```go
type Handler interface {
    RegisterRoutes(router *message.Router)
}
```

## Creating a New Handler

### 1. Handler struct and constructor

```go
// internal/adapter/inbound/subscriber/handler/xxx.go
package handler

type XxxHandler struct {
    cfg        *configs.Config
    log        logger.Logger
    subscriber message.Subscriber
}

func NewXxxHandler(cfg *configs.Config, log logger.Logger, subscriber message.Subscriber) *XxxHandler {
    return &XxxHandler{cfg: cfg, log: log, subscriber: subscriber}
}
```

**Rules:**
- Accept `cfg`, `log`, and `subscriber message.Subscriber`
- `subscriber` comes from `d.GetSubscriberKafka(id)` or `d.GetSubscriberRedisstream(id)`
- No business logic in handler — call services from the core layer
- If handler needs a service, inject it the same way as HTTP handlers but use `GetSikatWorker()`

### 2. RegisterRoutes

```go
func (h *XxxHandler) RegisterRoutes(router *message.Router) {
    router.AddConsumerHandler("XXX_CREATED", "xxx.created", h.subscriber, h.HandleXxxCreated)
    router.AddConsumerHandler("XXX_UPDATED", "xxx.updated", h.subscriber, h.HandleXxxUpdated)
}
```

Rules:
- First arg: unique handler name (use `UPPER_SNAKE_CASE` for logging)
- Second arg: topic (must match publisher topic convention `<entity>.<event>`)
- Third arg: `message.Subscriber` — same instance can handle multiple topics
- Fourth arg: `func(msg *message.Message) error`

### 3. Handler function

```go
func (h *XxxHandler) HandleXxxCreated(msg *message.Message) error {
    ctx := msg.Context()
    payload := msg.Payload
    uuid := msg.UUID

    h.log.WithContext(ctx).Infof("handling xxx.created: %s", uuid)

    // process payload...
    // call service if needed

    return nil  // ACK
    // return err  // NACK (message redelivered)
}
```

**Rules:**
- Use `msg.Context()` for all downstream calls (tracing, logging)
- Wrap errors with `fail.Wrap` / `fail.Wrapf`
- Return `nil` to ACK; return error to NACK (redelivered unless Retry middleware suppresses)
- Never use `fmt.Println` — use `logger.WithContext(ctx)`

## Middleware Pipeline (in router.go)

1. `middleware.RequestID()` — sets request/correlation ID in context
2. `middleware.Idempotence(idempotency, topic, ttl)` — claims message by UUID; duplicates ACKed without processing
3. Handler routes

### Idempotence
```go
middleware.Idempotence(w.idempotency, "", w.cfg.Event.Idempotency.TTL)
// topic: empty → reads from msg.Metadata.Get("topic")
// ttl: from config event.idempotency.ttl (e.g. "24h")
```

### Retry (optional — not in pipeline by default)
```go
middleware.Retry(maxRetries, delay, func(msg *message.Message, err error, attempt int, maxReached bool) {
    // log, metric, etc.
})
```

### SkipRetryError

Use for non-retryable errors (e.g. validation failures):
```go
if validationErr != nil {
    return middleware.NewSkipRetryError(fail.Wrap(validationErr))
}
```

## Dependency Wiring

### 1. Add subscriber config (YAML)

**Default:** Use `id: general` — shared consumer group for most handlers.

```yaml
event:
  kafka:
    subscribers:
      - id: general
        brokers: ["localhost:9092"]
        consumer_group: general
        debug_enabled: true
        trace_enabled: true
  idempotency:
    ttl: 24h
```

Add a new config entry **only when** a handler needs its own consumer group (different scaling, isolation).

### 2. Register handler in GetSubscriberHandlers

```go
func (d *Dependency) GetSubscriberHandlers() []subscriber.Handler {
    return d.subscriberHandlers.Resolve(func() []subscriber.Handler {
        return []subscriber.Handler{
            subscriberHandler.NewNoteHandler(d.GetConfig(), d.GetLogger(), d.GetSubscriberKafka("general")),
            subscriberHandler.NewXxxHandler(d.GetConfig(), d.GetLogger(), d.GetSubscriberKafka("general")),
        }
    })
}
```

**Handler with service** (use worker DB):
```go
repo := d.GetRepository(d.GetSikatWorker())
subscriberHandler.NewOrderHandler(d.GetConfig(), d.GetLogger(), d.GetSubscriberKafka("general"), d.GetServiceOrder(repo))
```

## Subscriber Lifecycle

1. `OnStart(ctx)`: calls `RegisterRoutes()`, starts `router.Run()` in goroutine
2. Router: applies middleware, invokes handler per message
3. `OnStop(ctx)`: calls `router.Close()` — graceful shutdown

## Rules Summary

| Rule | Action |
|------|--------|
| Handler interface | Implement `RegisterRoutes(router *message.Router)` |
| Default subscriber | `GetSubscriberKafka("general")` |
| New consumer needed | Add new config entry + use its `id` |
| Handler with service | Use `GetSikatWorker()` + `GetRepository` + `GetServiceXxx(repo)` |
| Errors | Wrap with `fail.Wrap` / `fail.Wrapf` |
| Logging | Use `logger.WithContext(ctx)` |
| Non-retryable errors | Return `middleware.NewSkipRetryError(err)` |
| Idempotence | Always enabled; uses `msg.UUID` |

## Anti-Patterns

| Avoid | Use instead |
|-------|-------------|
| `fmt.Println` in handlers | `logger.WithContext(ctx).Info(...)` |
| Returning raw `err` | `return fail.Wrap(err)` |
| Business logic in handler | Call core service/use-case |
| Skipping `msg.Context()` | Pass `msg.Context()` for tracing |
