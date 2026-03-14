---
name: krangka-dlock
description: Guide for using distributed locks (dlock) in krangka for atomic operations. Use when writing code that needs to prevent race conditions across multiple app instances, generate custom sequence numbers, ensure only one process runs a critical section, or when the user mentions distributed lock, mutex, atomic operation, sequence numbering, or dlock.
---

# krangka-dlock — Distributed Lock Guide

Distributed locks (backed by Redis) prevent race conditions across multiple running instances of the app. Use when a critical section must run atomically, such as generating sequential IDs, deduplicating concurrent requests, or coordinating one-at-a-time background tasks.

## Why distributed locks, not `sync.Mutex`

All services in this project are assumed to **always run as replicas behind a load balancer**. A `sync.Mutex` only protects a critical section within a single process — it is completely invisible to other replicas. When multiple instances race on the same shared resource (a database row, a sequence counter, a one-at-a-time background job), in-process mutexes offer zero protection. A distributed lock stored in Redis is the only correct primitive here.

> **Default assumption:** any service you write will be deployed with N > 1 replicas. Never reach for `sync.Mutex` to guard shared state — use `dlocker`.

## The interface

```go
// internal/core/port/outbound/dlock.go
type DLocker lock.DLocker

// Methods on DLocker:
TryLock(ctx context.Context, id string, ttl int) error  // acquire or fail immediately
Lock(ctx context.Context, id string, ttl int) error     // block until acquired
Unlock(ctx context.Context, id string) error
```

- `id` — the lock key (string). Use a descriptive, namespaced key, e.g. `"seq:invoice:2024-01"`.
- `ttl` — lock expiry in **seconds**. Always set a TTL as a safety net in case `Unlock` is never reached.
- `lock.ErrResourceLocked` — sentinel error returned by `TryLock` when the lock is held by another process.

## TryLock vs Lock

| | `TryLock` | `Lock` |
|---|---|---|
| Behaviour | Returns `lock.ErrResourceLocked` immediately if locked | Blocks until lock is acquired |
| Use when | Fail-fast (e.g. "skip if someone else is processing") | Wait-and-proceed (e.g. sequential ID generation) |

## Injecting DLocker into a service

Add `outbound.DLocker` as a field and constructor parameter:

```go
import (
    "github.com/redhajuanda/komon/lock"
    "github.com/redhajuanda/krangka/internal/core/port/outbound"
)

type Service struct {
    cfg    *configs.Config
    log    logger.Logger
    repo   outbound.Repository
    dlocker outbound.DLocker   // <-- add this
}

func NewService(cfg *configs.Config, log logger.Logger, repo outbound.Repository, dlocker outbound.DLocker) *Service {
    return &Service{cfg: cfg, log: log, repo: repo, dlocker: dlocker}
}
```

Wire it in `cmd/bootstrap/dependency.go` — `GetDLocker()` is already available:

```go
func (d *Dependency) GetServiceOrder(repo outbound.Repository) *order.Service {
    return d.serviceOrder.Resolve(func() *order.Service {
        return order.NewService(d.GetConfig(), d.GetLogger(), repo, d.GetDLocker())
    })
}
```

`dlocker` is a `ResourceClosable` — it is already registered for `Close()` in the bootstrap lifecycle. Do **not** call `Close()` manually.

## Usage pattern — atomic sequence generation

```go
func (s *Service) GenerateInvoiceNumber(ctx context.Context, period string) (string, error) {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    lockKey := fmt.Sprintf("seq:invoice:%s", period)
    const ttl = 10 // seconds

    // Block until we hold the lock
    if err := s.dlocker.Lock(ctx, lockKey, ttl); err != nil {
        return "", fail.Wrap(err)
    }
    defer s.dlocker.Unlock(ctx, lockKey) //nolint:errcheck

    // Safe to read-increment-write without races
    seq, err := s.repo.GetSequenceRepository().NextInvoiceSequence(ctx, period)
    if err != nil {
        return "", fail.Wrap(err)
    }

    return fmt.Sprintf("INV/%s/%05d", period, seq), nil
}
```

## Usage pattern — skip if already running (TryLock)

```go
func (w *Worker) Execute(ctx context.Context) error {
    const lockKey = "worker:outbox-relay"
    const ttl = 60

    if err := w.dlocker.TryLock(ctx, lockKey, ttl); err != nil {
        if errors.Is(err, lock.ErrResourceLocked) {
            w.log.WithContext(ctx).Info("skipping: another instance is running")
            return nil
        }
        return fail.Wrap(err)
    }
    defer w.dlocker.Unlock(ctx, lockKey) //nolint:errcheck

    // only one instance reaches here at a time
    return w.doWork(ctx)
}
```

## Lock key naming conventions

Use a clear, namespaced key that describes the resource being protected:

| Purpose | Example key |
|---------|-------------|
| Sequence generation | `"seq:invoice:2024-01"` |
| Per-entity operation | `"order:process:550e8400"` |
| Singleton worker | `"worker:outbox-relay"` |

Keys are prefixed by the Redis connection (empty prefix by default). Keep keys short and deterministic.

## Rules for AI agents

1. **Always `defer Unlock`** immediately after a successful `Lock`/`TryLock`. Never rely on logic-path returns to unlock.
2. **Always set a reasonable TTL** (a few seconds to a minute). Too short risks expiry mid-operation; too long holds the lock if the process crashes.
3. **Use `fail.Wrap`** on all lock errors — never return raw errors (see `krangka-fail` skill).
4. **Use `TryLock` for idempotent/skippable work**, `Lock` for operations that must complete exactly once and in order.
5. **Never call `dlocker.Close()`** — the bootstrap lifecycle manages it.
6. **Depend on `outbound.DLocker`** (the port interface), not on `*dlock.DLock` directly, in services/workers.