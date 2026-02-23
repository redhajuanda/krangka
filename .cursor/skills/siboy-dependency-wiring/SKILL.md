---
name: krangka-dependency-wiring
description: Guide for wiring dependencies in krangka. Use when choosing resource types (Resource, ResourceRunnable, ResourceExecutable, ResourceClosable), adding service/repository/worker getters, integrating external clients (DB, Redis, Kafka), or when the user asks about dependency injection or the service repo parameter pattern. For choosing between Run/Execute/Schedule lifecycle, see krangka-bootstrap.
---

# Dependency Wiring Guide

Wire all dependencies through `Dependency` in `cmd/bootstrap/dependency.go`. Add when creating commands, handlers, workers, or external clients. Skip when resource is local-only, a simple value, or needs different instances per call.

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

1. **Add field** to `Dependency` struct in `cmd/bootstrap/dependency.go`
2. **Create getter** `Get*()` that calls `Resolve(func() T { ... })`
3. **Use getter** in command/handler/worker

**Getter rules:**
- Method name starts with `Get`
- Inside `Resolve()`, call other `Get*()` methods (lazy, thread-safe)
- Init = pure setup only, no side effects (no Start/Stop)
- **Service getters MUST accept `repo outbound.Repository`** — handlers use `GetQweryMain()`, workers use `GetQweryWorker()`

## Critical: Service + Repository

Services are used by both HTTP (main DB) and workers (worker DB). **Never hardcode repo inside the service getter.**

```go
// ✅ CORRECT — repo is a parameter
func (d *Dependency) GetServiceUser(repo outbound.Repository) *user.Service {
    return d.serviceUser.Resolve(func() *user.Service {
        return user.NewService(d.GetConfig(), d.GetLogger(), repo)
    })
}

// ❌ WRONG — hardcodes main DB; worker can't use it
func (d *Dependency) GetServiceUser() *user.Service {
    return d.serviceUser.Resolve(func() *user.Service {
        repo := d.GetRepository(d.GetQweryMain())  // ❌
        return user.NewService(d.GetConfig(), d.GetLogger(), repo)
    })
}
```

**Usage:**
```go
// HTTP handler — main DB
repo := d.GetRepository(d.GetQweryMain())
httpHandler.NewUserHandler(d.GetConfig(), d.GetLogger(), d.GetServiceUser(repo))

// Worker — worker DB
repo := d.GetRepository(d.GetQweryWorker())
worker.NewWorkerUser(d.GetConfig(), d.GetLogger(), d.GetServiceUser(repo))
```

## Recipes

### Service (with repo parameter)
```go
// 1. Field
serviceUser Resource[*user.Service]

// 2. Getter
func (d *Dependency) GetServiceUser(repo outbound.Repository) *user.Service {
    return d.serviceUser.Resolve(func() *user.Service {
        return user.NewService(d.GetConfig(), d.GetLogger(), repo)
    })
}
```

### Handler
```go
func (d *Dependency) GetHTTPHandlers() []http.Handler {
    return d.httpHandlers.Resolve(func() []http.Handler {
        repo := d.GetRepository(d.GetQweryMain())
        return []http.Handler{
            httpHandler.NewUserHandler(d.GetConfig(), d.GetLogger(), d.GetServiceUser(repo)),
        }
    })
}
```

### Database connection (ResourceClosable)
```go
qweryAnalytics ResourceClosable[*mariadb.Qwery]

func (d *Dependency) GetQweryAnalytics() *mariadb.Qwery {
    return d.qweryAnalytics.Resolve(func() *mariadb.Qwery {
        cfg := d.GetConfig()
        return mariadb.NewQwery(mariadb.ParamQwery{...}, d.GetLogger())
    })
}
// Cleanup automatic — implement Close() on the type
```

### External client (Redis, Kafka)
```go
redis ResourceClosable[*redis.Redis]

func (d *Dependency) GetRedis() *redis.Redis {
    return d.redis.Resolve(func() *redis.Redis {
        return redis.New(d.GetConfig(), d.GetLogger())
    })
}
```

### Cron worker (ResourceExecutable)
```go
workerCleanup ResourceExecutable[*worker.WorkerCleanup]

func (d *Dependency) GetWorkerCleanup() *worker.WorkerCleanup {
    return d.workerCleanup.Resolve(func() *worker.WorkerCleanup {
        repo := d.GetRepository(d.GetQweryWorker())
        return worker.NewWorkerCleanup(d.GetConfig(), d.GetLogger(), repo)
    })
}
// Usage: boot.Schedule(pattern, dep.GetWorkerCleanup(), opts)
```

### Subscriber worker (ResourceRunnable)
```go
workerSubscriber ResourceRunnable[*worker.WorkerSubscriberUser]

func (d *Dependency) GetWorkerSubscriberUser() *worker.WorkerSubscriberUser {
    return d.workerSubscriber.Resolve(func() *worker.WorkerSubscriberUser {
        return worker.NewWorkerSubscriberUser(d.GetConfig(), d.GetLogger(), d.GetSubscriberKafka("topic"))
    })
}
// Usage: boot.Run(dep.GetWorkerSubscriberUser(), opts)
```

### Getter with parameters
```go
func (d *Dependency) GetRepository(qwery *mariadb.Qwery) outbound.Repository {
    return d.repository.Resolve(func() outbound.Repository {
        return mariadb.NewMariaDBRepository(d.GetConfig(), d.GetLogger(), qwery, d.GetPublishers())
    })
}
```

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| DB/Redis/Kafka as `Resource` | Use `ResourceClosable`; implement `Close()` |
| Start/stop in init | Use `ResourceRunnable`; keep init pure |
| `NewService()` directly | Use `dep.GetServiceXxx(repo)` via getter |
| Repo hardcoded in service getter | Accept `repo` as parameter; caller passes `GetRepository(GetQweryMain())` or `GetQweryWorker()` |
| Client without `Close()` | Implement `Closable` so bootstrap can clean up |
