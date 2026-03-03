---
name: krangka-subscriber
description: Guide for the Watermill-based event subscriber in krangka. Use when creating or modifying subscriber handlers, adding event consumers, wiring Kafka/Redisstream subscribers, adding middleware (idempotence, retry, request ID), or when the user asks about event-driven consumers, message handlers, or subscriber patterns. For time-triggered or manual background jobs, see krangka-worker instead.
---

# Subscriber in Krangka

The subscriber is a long-running Watermill-based worker that consumes events from Kafka or Redis Stream. It runs via `bootstrap.Run`, implements `Runnable`, and uses handlers that register topic consumers.

Messages can come from this service's publisher (e.g. outbox) or from another service publishing to the same topic — the subscriber is source-agnostic; it only cares about the topic.

## Directory Structure

```
internal/adapter/inbound/subscriber/
├── subscriber.go       # Subscriber struct, NewSubscriber, OnStart/OnStop
├── router.go           # RegisterRoutes — adds middleware, delegates to handlers
├── middleware/
│   ├── request_id.go   # Sets request/correlation ID from message metadata
│   ├── idempotence.go  # Ensures at-most-once processing per message UUID
│   ├── retry.go        # Optional retry with SkipRetryError support
│   └── errors.go       # SkipRetryError type and helpers
└── handler/
    └── note.go         # Example: NoteHandler
```

## Core Interfaces

| Type | Location | Contract |
|------|----------|----------|
| `subscriber.Handler` | `subscriber/subscriber.go` | `RegisterRoutes(router *message.Router)` |
| `message.Subscriber` | Watermill | Kafka or Redisstream consumer; passed into handler constructor |
| `message.NoPublishHandlerFunc` | Watermill | `func(msg *message.Message) error` — used by `AddConsumerHandler` |
| `message.HandlerFunc` | Watermill | `func(msg *message.Message) ([]*message.Message, error)` — used by `AddHandler` and middleware |

## Handler Contract

Every handler must implement `subscriber.Handler`:

```go
type Handler interface {
    RegisterRoutes(router *message.Router)
}
```

A handler receives `message.Subscriber` in its constructor and registers consumer handlers via `router.AddConsumerHandler`.

## Creating a New Handler

### 1. Handler struct and constructor

```go
// internal/adapter/inbound/subscriber/handler/xxx.go
package handler

import (
    "github.com/ThreeDotsLabs/watermill/message"
    "github.com/redhajuanda/krangka/configs"
    "github.com/redhajuanda/komon/logger"
)

type XxxHandler struct {
    cfg        *configs.Config
    log        logger.Logger
    subscriber message.Subscriber
}

func NewXxxHandler(cfg *configs.Config, log logger.Logger, subscriber message.Subscriber) *XxxHandler {
    return &XxxHandler{
        cfg:        cfg,
        log:        log,
        subscriber: subscriber,
    }
}
```

**Rules:**
- Accept `cfg *configs.Config`, `log logger.Logger`, and `subscriber message.Subscriber`.
- The `subscriber` is either `*kafka.Subscriber` or `*redisstream.Subscriber` — obtained via `d.GetSubscriberKafka(id)` or `d.GetSubscriberRedisstream(id)` in dependency wiring.
- Handler must not implement business logic directly; call services/use-cases from the core layer.
- **Handler needs a service?** Same pattern as HTTP handler — inject in constructor. Subscriber is a worker, so use `GetQweryWorker()` and `GetRepository(d.GetQweryWorker())` when resolving the service. See krangka-dependency-wiring.

### 2. Implement RegisterRoutes

```go
func (h *XxxHandler) RegisterRoutes(router *message.Router) {
    topicCreated := "xxx.created"
    topicUpdated := "xxx.updated"

    router.AddConsumerHandler("XXX_CREATED", topicCreated, h.subscriber, h.HandleXxxCreated)
    router.AddConsumerHandler("XXX_UPDATED", topicUpdated, h.subscriber, h.HandleXxxUpdated)
}
```

**Rules:**
- First arg to `AddConsumerHandler`: unique handler name (used for logging; prefer `UPPER_SNAKE_CASE`).
- Second arg: topic name (must match publisher topic).
- Third arg: `message.Subscriber` — same instance can be used for multiple topics.
- Fourth arg: handler function matching `message.NoPublishHandlerFunc` (`func(msg *message.Message) error`).

### 3. Handler function signature

```go
func (h *XxxHandler) HandleXxxCreated(msg *message.Message) error {
    ctx := msg.Context()
    payload := msg.Payload
    uuid := msg.UUID
    metadata := msg.Metadata

    // Use ctx for tracing, logging, and downstream calls
    // Use komon/fail for errors — never return raw errors
    // Return nil on success; return error to trigger NACK/retry (if Retry middleware is used)
    return nil
}
```

**Rules:**
- Use `msg.Context()` for all downstream calls (tracing, logging).
- Wrap errors with `fail.Wrap` / `fail.Wrapf` per krangka-fail skill.
- Return `nil` to ACK; return error to NACK (message redelivered unless Retry middleware suppresses).
- Do not use `fmt.Println`; use `logger.WithContext(ctx).*` per krangka-logger skill.

---

## Middleware

Middleware is added in `router.go` before handlers register routes. Order matters: first added runs first (outermost).

### Current pipeline (router.go)

1. `middleware.RequestID()` — sets request ID and correlation ID from message metadata into context.
2. `middleware.Idempotence(idempotency, topic, ttl)` — claims message by UUID; duplicates are ACKed without processing.
3. Handler routes (via `handler.RegisterRoutes(router)`).

### RequestID

- Reads `tracer.RequestIDHeader` and `tracer.CorrelationIDHeader` from `msg.Metadata`.
- Sets them in context and back into metadata.
- No parameters. Always add when tracing is used.

### Idempotence

```go
middleware.Idempotence(w.idempotency, "", w.cfg.Event.Idempotency.TTL)
```

- `topic`: empty string → uses `msg.Metadata.Get("topic")` per message.
- `ttl`: from config `event.idempotency.ttl` (e.g. `24h`).
- Uses `msg.UUID` (or `msg.Metadata.Get("id")`) as idempotency key.
- If `TryClaim` returns false, handler is skipped and message is ACKed.

### Retry (optional)

`middleware.Retry` exists but is **not** currently in the pipeline. To add:

```go
middleware.Retry(maxRetries, delay, func(msg *message.Message, err error, attempt int, maxReached bool) {
    // Log, metric, etc.
})
```

Behavior:
- Retries handler up to `maxRetries` times with `delay` between attempts.
- On `context.Canceled` or `context.DeadlineExceeded`: stops immediately, ACKs (no redelivery).
- On `SkipRetryError`: stops immediately, ACKs. Use when error is non-retryable (e.g. validation failure).

### SkipRetryError

Use when a handler error should **not** trigger retries:

```go
if validationErr != nil {
    return middleware.NewSkipRetryError(fail.Wrap(validationErr))
}
```

- Check: `middleware.IsSkipRetryError(err)`.
- Wrapped error is available via `errors.As(err, &skipErr)` then `skipErr.Err`.

---

## Dependency Wiring

### 1. Add subscriber config

**Default:** One shared config `id: general`. Most handlers use `GetSubscriberKafka("general")` — they share the same consumer connection and consumer group.

**Add a new config** only when a handler needs its own consumer (e.g. different scaling, isolation, dedicated consumer group). Then create a new entry and use `GetSubscriberKafka("that-id")` for that handler.

In YAML (`configs/files/default.yaml` or `example.yaml`):

**Kafka:**
```yaml
event:
  kafka:
    subscribers:
      - id: general      # Default — shared by most handlers
        brokers: ["localhost:9092"]
        consumer_group: general
        debug_enabled: true
        trace_enabled: true
      # Add new entry only when a handler needs its own consumer
      - id: order
        brokers: ["localhost:9092"]
        consumer_group: order
        debug_enabled: true
        trace_enabled: true
```

**Redisstream:**
```yaml
event:
  redisstream:
    subscribers:
      - id: general
        consumer_group: general
```

### 2. Add handler to GetSubscriberHandlers

In `cmd/bootstrap/dependency.go`:

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

- **Default:** Use `GetSubscriberKafka("general")` — shared consumer.
- **Handler needs own consumer:** Use `GetSubscriberKafka("order")` (or other id) — requires matching config entry.
- `id` must match an entry in `event.kafka.subscribers` or `event.redisstream.subscribers`.

**Handler with service:** Use worker DB and repo:

```go
repo := d.GetRepository(d.GetQweryWorker())
subscriberHandler.NewOrderHandler(d.GetConfig(), d.GetLogger(), d.GetSubscriberKafka("general"), d.GetServiceOrder(repo)),
```

### 3. Subscriber is already wired

`GetSubscriber(closeTimeout)` builds the Subscriber with all handlers. Command usage:

```go
// cmd/subscriber.go
runner = dep.GetSubscriber(opts.StopTimeout)
boot.Run(runner, opts)
```

---

## Config Reference

| Config path | Type | Purpose |
|-------------|------|---------|
| `event.idempotency.ttl` | `time.Duration` | Idempotency key TTL (e.g. `24h`) |
| `event.kafka.subscribers` | `[]KafkaSubscriber` | Kafka consumer configs |
| `event.redisstream.subscribers` | `[]RedisstreamSubscriber` | Redis stream consumer configs |
| `event.kafka.subscribers[].id` | string | Used in `GetSubscriberKafka(id)`. Default: `general` (shared). Add new id only when handler needs dedicated consumer |
| `event.redisstream.subscribers[].id` | string | Same as Kafka |

---

## Subscriber Lifecycle

1. `OnStart(ctx)`: calls `RegisterRoutes()`, starts `router.Run()` in goroutine.
2. Router: applies middleware, then invokes handler per message.
3. `OnStop(ctx)`: calls `router.Close()` — graceful shutdown.

`closeTimeout` (from `RunOptions.StopTimeout`) is passed to `message.NewRouter` and limits how long router waits on close.

---

## Rules Summary

| Rule | Action |
|------|--------|
| Handler interface | Implement `RegisterRoutes(router *message.Router)` |
| Subscriber dependency | Default: `GetSubscriberKafka("general")`. Add new config + use its `id` only when handler needs its own consumer |
| Handler with service | Use `GetQweryWorker()` + `GetRepository` + `GetServiceXxx(repo)` — same as HTTP handler but worker DB (krangka-dependency-wiring) |
| Errors | Wrap with `fail.Wrap` / `fail.Wrapf` (krangka-fail) |
| Logging | Use `logger.WithContext(ctx)` (krangka-logger) |
| Non-retryable errors | Return `middleware.NewSkipRetryError(err)` |
| One subscriber per consumer group | One config entry can serve multiple topics in one handler |
| Idempotence | Always enabled; uses `msg.UUID` and `event.idempotency.ttl` |

---

## Anti-Patterns

| Avoid | Use instead |
|-------|-------------|
| `fmt.Println` in handlers | `logger.WithContext(ctx).Info(...)` |
| Returning raw `err` | `return fail.Wrap(err)` |
| Business logic in handler | Call core service/use-case |
| Multiple handlers sharing same consumer group with different logic | Each handler registers its own topics; same subscriber instance can consume multiple topics |
| Skipping `msg.Context()` in calls | Pass `msg.Context()` for tracing and cancellation |

---

## Related Skills

- **krangka-bootstrap**: `Run` pattern for long-running subscriber.
- **krangka-dependency-wiring**: Resource types, getter patterns.
- **krangka-fail**: Error wrapping.
- **krangka-logger**: Logging in handlers.
