# Distributed Locks (dlock)

Distributed locks (backed by Redis) prevent race conditions across multiple running instances. Use when a critical section must run atomically.

## Why Not `sync.Mutex`?

All services run as **replicas behind a load balancer**. `sync.Mutex` only protects within a single process — it's invisible to other replicas. A distributed lock stored in Redis is the only correct primitive.

> **Default assumption:** any service will be deployed with N > 1 replicas. Never reach for `sync.Mutex` to guard shared state.

## The Interface

```go
// internal/core/port/outbound/dlock.go
type DLocker lock.DLocker

// Methods:
TryLock(ctx context.Context, id string, ttl int) error  // acquire or fail immediately
Lock(ctx context.Context, id string, ttl int) error     // block until acquired
Unlock(ctx context.Context, id string) error
```

- `id` — lock key (namespaced string, e.g. `"seq:invoice:2024-01"`)
- `ttl` — lock expiry in **seconds** — always set as safety net if `Unlock` is never reached
- `lock.ErrResourceLocked` — sentinel from `TryLock` when lock is held by another process

## TryLock vs Lock

| | `TryLock` | `Lock` |
|---|---|---|
| Behaviour | Returns `lock.ErrResourceLocked` immediately | Blocks until acquired |
| Use when | Fail-fast (skip if someone else is running) | Wait-and-proceed (must run exactly once, in order) |

## Injecting DLocker into a Service

```go
type Service struct {
    cfg     *configs.Config
    log     logger.Logger
    repo    outbound.Repository
    dlocker outbound.DLocker   // add this
}

func NewService(cfg *configs.Config, log logger.Logger, repo outbound.Repository, dlocker outbound.DLocker) *Service {
    return &Service{cfg: cfg, log: log, repo: repo, dlocker: dlocker}
}
```

Wire in `cmd/bootstrap/dependency.go` — `GetDLocker()` is already available:
```go
func (d *Dependency) GetServiceOrder(repo outbound.Repository) *order.Service {
    return d.serviceOrder.Resolve(func() *order.Service {
        return order.NewService(d.GetConfig(), d.GetLogger(), repo, d.GetDLocker())
    })
}
```

`dlocker` is a `ResourceClosable` — bootstrap manages its lifecycle. Do **not** call `Close()` manually.

## Usage Patterns

### Pattern 1: Atomic Sequence Generation (`Lock` — wait-and-proceed)

```go
func (s *Service) GenerateInvoiceNumber(ctx context.Context, period string) (string, error) {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    lockKey := fmt.Sprintf("seq:invoice:%s", period)
    const ttl = 10 // seconds

    if err := s.dlocker.Lock(ctx, lockKey, ttl); err != nil {
        return "", fail.Wrap(err)
    }
    defer s.dlocker.Unlock(ctx, lockKey) //nolint:errcheck

    seq, err := s.repo.GetSequenceRepository().NextInvoiceSequence(ctx, period)
    if err != nil {
        return "", fail.Wrap(err)
    }
    return fmt.Sprintf("INV/%s/%05d", period, seq), nil
}
```

### Pattern 2: Skip If Already Running (`TryLock` — fail-fast)

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

    return w.doWork(ctx)
}
```

## Lock Key Naming

| Purpose | Example key |
|---------|-------------|
| Sequence generation | `"seq:invoice:2024-01"` |
| Per-entity operation | `"order:process:550e8400"` |
| Singleton worker | `"worker:outbox-relay"` |

Keys are prefixed by Redis connection. Keep keys short and deterministic.

## Rules for AI Agents

1. **Always `defer Unlock`** immediately after successful `Lock`/`TryLock`
2. **Always set a reasonable TTL** — too short risks mid-operation expiry; too long holds lock on crash
3. **Use `fail.Wrap`** on all lock errors — never return raw errors
4. **Use `TryLock` for idempotent/skippable work**, `Lock` for must-complete-exactly-once operations
5. **Never call `dlocker.Close()`** — bootstrap lifecycle manages it
6. **Depend on `outbound.DLocker`** (port interface), not `*dlock.DLock` directly
