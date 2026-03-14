# Dependency Wiring

Wire all dependencies through `Dependency` in `cmd/bootstrap/dependency.go`.

## Decision Tree: Which Resource Type?

```
Needs cleanup (Close())? → ResourceClosable[T]  (DB, Redis, Kafka)
Needs start/stop?        → ResourceRunnable[T]  (HTTP server, subscriber)
One-time or scheduled?   → ResourceExecutable[T] (migrate, cron)
Otherwise                → Resource[T]           (config, logger, services, handlers)
```

| Type | Interface | Examples |
|------|-----------|----------|
| `Resource[T]` | None | Config, logger, services, repositories, handlers |
| `ResourceRunnable[T]` | `OnStart(ctx), OnStop(ctx)` | HTTP server, subscriber workers |
| `ResourceExecutable[T]` | `Execute(ctx)` | Migrate, cron workers |
| `ResourceClosable[T]` | `Close()` | DB, Redis, Kafka clients |

## Add Dependency: 3 Steps

1. **Add field** to `Dependency` struct
2. **Create getter** `Get*()` that calls `Resolve(func() T { ... })`
3. **Use getter** in command/handler/worker

**Getter rules:**
- Method name starts with `Get`
- Inside `Resolve()`, call other `Get*()` methods (lazy, thread-safe)
- Init = pure setup only, no side effects (no Start/Stop)
- **Service getters MUST accept `repo outbound.Repository`** as parameter

## Critical: Service + Repository Pattern

Services are used by both HTTP (main DB) and workers (worker DB). **Never hardcode repo inside service getter.**

```go
// ✅ CORRECT — repo is a parameter
func (d *Dependency) GetServiceUser(repo outbound.Repository) *user.Service {
    return d.serviceUser.Resolve(func() *user.Service {
        return user.NewService(d.GetConfig(), d.GetLogger(), repo)
    })
}

// ❌ WRONG — hardcodes main DB; workers can't use it
func (d *Dependency) GetServiceUser() *user.Service {
    return d.serviceUser.Resolve(func() *user.Service {
        repo := d.GetRepository(d.GetSikatMain())  // ❌
        return user.NewService(d.GetConfig(), d.GetLogger(), repo)
    })
}
```

**Usage in HTTP context:**
```go
repo := d.GetRepository(d.GetSikatMain())
httpHandler.NewUserHandler(d.GetConfig(), d.GetLogger(), d.GetServiceUser(repo))
```

**Usage in Worker context:**
```go
repo := d.GetRepository(d.GetSikatWorker())
worker.NewWorkerUser(d.GetConfig(), d.GetLogger(), d.GetServiceUser(repo))
```

## Recipes

### Service (with repo parameter)
```go
serviceUser Resource[*user.Service]

func (d *Dependency) GetServiceUser(repo outbound.Repository) *user.Service {
    return d.serviceUser.Resolve(func() *user.Service {
        return user.NewService(d.GetConfig(), d.GetLogger(), repo)
    })
}
```

### HTTP Handler
```go
func (d *Dependency) GetHTTPHandlers() []http.Handler {
    return d.httpHandlers.Resolve(func() []http.Handler {
        repo := d.GetRepository(d.GetSikatMain())
        return []http.Handler{
            httpHandler.NewUserHandler(d.GetConfig(), d.GetLogger(), d.GetServiceUser(repo)),
        }
    })
}
```

### Database Connection (ResourceClosable)
```go
sikatAnalytics ResourceClosable[*mariadb.Sikat]

func (d *Dependency) GetSikatAnalytics() *mariadb.Sikat {
    return d.sikatAnalytics.Resolve(func() *mariadb.Sikat {
        cfg := d.GetConfig()
        return mariadb.NewSikat(mariadb.ParamSikat{...}, d.GetLogger())
    })
}
// Cleanup automatic — implement Close() on the type
```

### External Client (Redis, Kafka)
```go
redis ResourceClosable[*redis.Redis]

func (d *Dependency) GetRedis() *redis.Redis {
    return d.redis.Resolve(func() *redis.Redis {
        return redis.New(d.GetConfig(), d.GetLogger())
    })
}
```

### Cron Worker (ResourceExecutable)
```go
workerCleanup ResourceExecutable[*worker.WorkerCleanup]

func (d *Dependency) GetWorkerCleanup() *worker.WorkerCleanup {
    return d.workerCleanup.Resolve(func() *worker.WorkerCleanup {
        repo := d.GetRepository(d.GetSikatWorker())
        return worker.NewWorkerCleanup(d.GetConfig(), d.GetLogger(), repo)
    })
}
// Usage: boot.Schedule(pattern, dep.GetWorkerCleanup(), opts)
```

### Subscriber Worker (ResourceRunnable)
```go
workerSubscriber ResourceRunnable[*worker.WorkerSubscriberUser]

func (d *Dependency) GetWorkerSubscriberUser() *worker.WorkerSubscriberUser {
    return d.workerSubscriber.Resolve(func() *worker.WorkerSubscriberUser {
        return worker.NewWorkerSubscriberUser(d.GetConfig(), d.GetLogger(), d.GetSubscriberKafka("topic"))
    })
}
// Usage: boot.Run(dep.GetWorkerSubscriberUser(), opts)
```

### Getter with Parameters
```go
func (d *Dependency) GetRepository(sikat *mariadb.Sikat) outbound.Repository {
    return d.repository.Resolve(func() outbound.Repository {
        return mariadb.NewMariaDBRepository(d.GetConfig(), d.GetLogger(), sikat, d.GetPublishers())
    })
}
```

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| DB/Redis/Kafka as `Resource` | Use `ResourceClosable`; implement `Close()` |
| Start/stop in init | Use `ResourceRunnable`; keep init pure |
| `NewService()` directly | Use `dep.GetServiceXxx(repo)` via getter |
| Repo hardcoded in service getter | Accept `repo` as parameter |
| Client without `Close()` | Implement `Closable` so bootstrap can clean up |
