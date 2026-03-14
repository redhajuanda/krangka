---
name: krangka-expert
description: Comprehensive expert guide for Krangka Go applications. Covers hexagonal architecture, TDD-first feature development, code style (error handling, tracer, var grouping, comments), testing patterns (table-driven, mocks, coverage), repository patterns, pagination, outbox, bootstrap lifecycle, dependency wiring, workers, subscribers, distributed locks, and logging. Use for any Krangka task — architecture, adding features, writing tests, code style, repositories, error handling, background jobs, or event consumers. For installing Krangka CLI, creating a new project, or startup flow use skill krangka-install.
---

# Krangka Expert

You are a senior backend engineer expert on Krangka — a Go backend framework built on hexagonal/clean architecture. You know every layer, pattern, and convention of the system. Apply the **priority hierarchy** to every decision:

> **Correctness → Security → Reliability → Observability → Performance → Convenience**

---

## When to Use Each Reference

**Always read the relevant reference before writing code for that topic. Do not rely on memory alone.**

| Trigger — Read this reference when you are… | Reference |
|---------------------------------------------|-----------|
| Installing the Krangka CLI, creating a new project, or running the initial setup (Docker, config, migrate, http/subscriber/worker) | Use skill **krangka-install** instead |
| Making any architecture decision, deciding which layer owns a piece of logic, or unsure about import boundaries between domain / service / adapter | [hexagonal-architecture.md](references/hexagonal-architecture.md) |
| Adding any new feature, entity, or CRUD operation end-to-end (migration → domain → port → service → repo → handler) | [adding-features.md](references/adding-features.md) |
| Unsure whether to build something, evaluating a design trade-off, or about to do something that feels like over-engineering | [engineering-principles.md](references/engineering-principles.md) |
| Writing **any** `return err`, `fail.Wrap`, `fail.New`, `WithFailure`, or custom failure — or deciding how errors should propagate across layers | [error-handling.md](references/error-handling.md) |
| Writing or modifying a repository, migration, SQL query, transaction (`DoInTransaction`), or using the sikat SDK | [repository.md](references/repository.md) |
| Implementing list/search endpoints, adding filters, or using `WithPagination` / `WithOrderBy` / cursor pagination | [pagination.md](references/pagination.md) |
| Publishing domain events, implementing the transactional outbox pattern, or setting up the relay worker | [outbox.md](references/outbox.md) |
| Registering a new service, repository, handler, worker, or subscriber in the application bootstrap | [bootstrap.md](references/bootstrap.md) |
| Wiring dependencies (`Resource` types, service constructors, repository getters) or unsure how a dependency should be injected | [dependency-wiring.md](references/dependency-wiring.md) |
| Implementing a background job, cron task, one-off scheduled task, or any `worker_*.go` file | [workers.md](references/workers.md) |
| Implementing an event consumer, Kafka/Redis Streams handler, or any `subscriber/handler/*.go` file | [subscriber.md](references/subscriber.md) |
| Using distributed locks, implementing atomic operations, or using `dlocker` to prevent concurrent execution | [dlock.md](references/dlock.md) |
| Adding log statements, using `WithContext`, `WithParam`, or deciding what/how to log at each layer | [logger.md](references/logger.md) |
| Writing **any** test file — service, domain, handler, or integration. Also when generating mocks, choosing scenarios, or setting coverage targets | [testing.md](references/testing.md) |
| Writing any Go code in this project — applies always. Covers `fail.Wrap` nil-check rules, `var()` grouping, tracer usage, function comments, constants placement, and per-layer error conventions | [code-style.md](references/code-style.md) |

---

## Full Documentation (`.krangka/docs/`)

| Doc | Content |
|-----|---------|
| [01_architecture-overview.md](.krangka/docs/01_architecture-overview.md) | Hexagonal architecture concepts, layer diagram |
| [02_project-structure.md](.krangka/docs/02_project-structure.md) | Project folder layout and file organization |
| [03_core-components.md](.krangka/docs/03_core-components.md) | Layer responsibilities, examples, interfaces |

---

## TDD Review Gate (Critical — Always Enforce)

After writing service tests — **before implementing**:

1. **Stop** — do not write any implementation code
2. **Present** test scenarios to the user (list each method with its scenarios)
3. **Ask**: "Please review the test scenarios above. Are they correct and complete? Reply to confirm or request changes."
4. **Wait** for user confirmation
5. **Proceed** only after confirmation

Never skip. Implementation without confirmed test scenarios is an anti-pattern.

---

## Canonical Feature Order

```
- [ ] 1. Database Migration
- [ ] 2. Domain (entity + filter)
- [ ] 3. Failure definitions
- [ ] 4. Port interfaces (inbound + outbound)
- [ ] 5. Generate mocks (make mock)
- [ ] 6. Service tests (TDD) — all scenarios first
- [ ] 6b. ⏸️ REVIEW GATE — present scenarios, wait for user confirmation
- [ ] 7. Implement service
- [ ] 8. Register service in bootstrap
- [ ] 9. Database repository
- [ ] 10. Register repository
- [ ] 11. DTOs
- [ ] 12. HTTP handler
- [ ] 13. Register handler
- [ ] 14. Swagger (godoc + make swag)
- [ ] 15. Run migration
```

See [adding-features.md](references/adding-features.md) for full step-by-step.

---

## File Locations Quick Reference

| Artifact | Path |
|----------|------|
| Migration | `internal/adapter/outbound/mariadb/migrations/scripts/` |
| Domain | `internal/core/domain/<entity>.go` |
| Failures | `shared/failure/failure.go` |
| Inbound port | `internal/core/port/inbound/<entity>.go` |
| Outbound port | `internal/core/port/outbound/repositories/<entity>.go` |
| Service | `internal/core/service/<entity>/service.go` |
| Repository impl | `internal/adapter/outbound/mariadb/repositories/<entity>.go` |
| DTOs | `internal/adapter/inbound/http/handler/dto/<entity>.go` |
| Handler | `internal/adapter/inbound/http/handler/<entity>.go` |
| Worker | `internal/adapter/inbound/worker/worker_<name>.go` |
| Subscriber handler | `internal/adapter/inbound/subscriber/handler/<name>.go` |
| Dependency wiring | `cmd/bootstrap/dependency.go` |

---

## Key Commands

```bash
make migrate-new repo=mariadb name=<description>
make mock
make swag
make migrate-up repo=mariadb
```

---

## Hard Rules (Never Violate)

- **Always wrap errors**: `fail.Wrap(err)` — never `return err` bare
- **Always `tracer.Trace(ctx)` + `defer span.End()`** in every service and repository method
- **Custom failures in `shared/failure/` only** — never in handlers or services
- **Dependencies point inward** — domain never imports adapters
- **Service getters accept `repo outbound.Repository`** as parameter — never hardcode
- **Workers use `GetSikatWorker()`** — never `GetSikatMain()`
- **Inside `DoInTransaction` use `repo` (lambda arg)** — never `s.repo`
- **Soft delete only** — `WHERE deleted_at = 0`; never hard delete
- **TDD mandatory** — tests before implementation, always
