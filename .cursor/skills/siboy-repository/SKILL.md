---
name: krangka-repository
description: Guide for working with repositories in krangka: creating migration files, defining repository port interfaces, implementing repositories with the qwery SDK, and using transactions. Use when adding a new repository, writing SQL queries, creating a migration, or using DoInTransaction. For outbox event publishing, see krangka-outbox. For pagination, see krangka-pagination.
---

# krangka Repository Guide

Five parts in order:
1. [Migration](#1-migration) — create and name SQL migration files
2. [Repository Port](#2-repository-port) — define the interface in the core layer
3. [Repository Implementation](#3-repository-implementation) — wire the implementation in the adapter layer
4. [Integration with qwery](#4-integration-with-qwery) — write queries using the qwery SDK
5. [Transactions](#5-transactions) — run multiple operations atomically

> **Outbox Pattern**: For publishing events atomically with a DB write, see **krangka-outbox** skill.
> **Pagination**: For offset/cursor pagination details, see **krangka-pagination** skill.

Working examples to read when needed:
- `internal/core/port/outbound/repositories/note.go` — port interface
- `internal/adapter/outbound/mariadb/repositories/note.go` — implementation
- `internal/adapter/outbound/mariadb/repository.go` — aggregator + transaction wiring
- `internal/core/service/note/service.go` — service using repo, transaction, and outbox

---

## 1. Migration

### Location

```
internal/adapter/outbound/mariadb/migrations/scripts/
```

### Creating a new migration file

**Always use the make command — never create migration files manually.** The command generates the file with the correct timestamp-based name automatically.

```bash
make migrate-new repo=mariadb name=<description>
```

Or equivalently:

```bash
go run main.go migrate new mariadb <description>
```

Example:

```bash
make migrate-new repo=mariadb name=table_widgets
# generates: internal/adapter/outbound/mariadb/migrations/scripts/20250219120000-table_widgets.sql
```

Rules for the `name` argument:
- Use underscores, not spaces or hyphens
- Be descriptive: `table_widgets`, `add_index_widgets_name`, `alter_widgets_add_status`
- One logical change per file

### File structure

Every migration file must have both markers:

```sql
-- +migrate Up

CREATE TABLE IF NOT EXISTS widgets (
  id          varchar(26)  PRIMARY KEY NOT NULL,
  name        varchar(255) NOT NULL,
  created_at  timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at  timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  deleted_at  int          NOT NULL DEFAULT 0
);

-- +migrate Down

DROP TABLE IF EXISTS widgets;
```

### Standard column conventions

| Column | Type | Rule |
|--------|------|------|
| `id` | `varchar(26)` | ULID; always the PRIMARY KEY |
| `created_at` | `timestamp(6)` | Microsecond precision; set on insert |
| `updated_at` | `timestamp(6) ON UPDATE CURRENT_TIMESTAMP(6)` | Auto-updated on every change |
| `deleted_at` | `int NOT NULL DEFAULT 0` | Soft delete: `0` = active, UNIX timestamp = deleted |

> Always soft-delete. Never use `deleted_at IS NULL` — the column is `int`, filter with `WHERE deleted_at = 0`.

---

## 2. Repository Port

The port is a Go interface that lives in the **core layer**. It has no knowledge of SQL or any SDK.

### 2a — Define the entity interface

Create `internal/core/port/outbound/repositories/<entity>.go`:

```go
package repositories

import (
    "context"

    "github.com/redhajuanda/krangka/internal/core/domain"
    "github.com/redhajuanda/komon/pagination"
)

type Widget interface {
    GetWidgetByID(ctx context.Context, id string) (*domain.Widget, error)
    CreateWidget(ctx context.Context, w *domain.Widget) error
    UpdateWidget(ctx context.Context, w *domain.Widget) error
    DeleteWidget(ctx context.Context, id string) error
    ListWidgets(ctx context.Context, filter *domain.WidgetFilter, pag *pagination.Pagination) (*[]domain.Widget, error)
}
```

### 2b — Register a getter in the root Repository interface

Add to `internal/core/port/outbound/repository.go`:

```go
type Repository interface {
    DoInTransaction(ctx context.Context, fn func(repo Repository) (any, error)) (any, error)
    PublishOutbox(ctx context.Context, target PublisherTarget, topic string, payload qwery.JSONMap) error
    RetryOutbox(ctx context.Context) error
    GetNoteRepository() repositories.Note
    GetTodoRepository() repositories.Todo
    GetWidgetRepository() repositories.Widget  // ← add this
}
```

### 2c — Define the domain struct

Create `internal/core/domain/<entity>.go`. All fields that map to DB columns must carry the `qwery` struct tag:

```go
package domain

import "time"

type Widget struct {
    ID        string    `qwery:"id"`
    Name      string    `qwery:"name"`
    CreatedAt time.Time `qwery:"created_at"`
    UpdatedAt time.Time `qwery:"updated_at"`
    DeletedAt int       `qwery:"deleted_at"`
}

type WidgetFilter struct {
    Search string
}
```

---

## 3. Repository Implementation

The implementation lives in the **adapter layer** and satisfies the port interface from Part 2.

### 3a — Create the implementation file

Create `internal/adapter/outbound/mariadb/repositories/<entity>.go`:

```go
package repositories

import (
    "context"
    "database/sql"
    "errors"

    "github.com/redhajuanda/krangka/internal/core/domain"
    "github.com/redhajuanda/qwery"
    "github.com/redhajuanda/komon/fail"
    "github.com/redhajuanda/komon/pagination"
    "github.com/redhajuanda/komon/tracer"
)

type widgetRepository struct {
    qwery qwery.Runable  // Runable, NOT *qwery.Client — required for transaction support
}

func NewWidgetRepository(qwery qwery.Runable) *widgetRepository {
    return &widgetRepository{qwery: qwery}
}
```

**Mandatory rules for every method:**
- Call `tracer.Trace(ctx)` + `defer span.End()` at the top
- Always `fail.Wrap(err)` on every error return
- For `sql.ErrNoRows`, additionally chain `.WithFailure(fail.ErrNotFound)`

```go
func (r *widgetRepository) GetWidgetByID(ctx context.Context, id string) (*domain.Widget, error) {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    var w domain.Widget
    err := r.qwery.
        RunRaw(`SELECT id, name, created_at, updated_at FROM widgets WHERE id = {{ .id }} AND deleted_at = 0`).
        WithParam("id", id).
        ScanStruct(&w).
        Query(ctx)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, fail.Wrap(err).WithFailure(fail.ErrNotFound)
        }
        return nil, fail.Wrap(err)
    }
    return &w, nil
}
```

### 3b — Wire into the aggregator

In `internal/adapter/outbound/mariadb/repository.go`, add the field, include it in the constructor, and add the getter:

```go
type mariaDBRepository struct {
    // ... existing fields
    WidgetRepository portRepo.Widget  // ← add field
}

func NewMariaDBRepository(...) *mariaDBRepository {
    return &mariaDBRepository{
        // ... existing fields
        WidgetRepository: implRepo.NewWidgetRepository(qwery.Client),  // ← add
    }
}

// GetWidgetRepository returns the Widget repository, using the active transaction when inside DoInTransaction.
func (r *mariaDBRepository) GetWidgetRepository() portRepo.Widget {
    if r.qweryTx != nil {
        return implRepo.NewWidgetRepository(r.qweryTx)
    }
    return r.WidgetRepository
}
```

> The `GetXxxRepository()` getter pattern is what makes transactions work: when `qweryTx` is set (inside `DoInTransaction`), the getter returns a fresh instance using the transaction client. Do not pre-populate sub-repositories in the transactional `registry` struct inside `DoInTransaction`; let the getter handle it.

---

## 4. Integration with qwery

qwery is the SQL SDK. It provides a fluent builder chain over raw SQL with template parameters.

### Query chain reference

| Method | When to use |
|--------|------------|
| `.RunRaw(query string)` | Start any query |
| `.WithParam("key", value)` | Bind a single named parameter |
| `.WithParams(v)` | Bind all fields from a struct (uses `qwery` tags) or `map[string]any` |
| `.WithPagination(pag)` | Attach a `*pagination.Pagination` — adds LIMIT, cursor logic automatically |
| `.WithOrderBy(cols...)` | Prefix `+` for ASC, `-` for DESC. E.g. `"-created_at", "+id"` — **see krangka-pagination for cursor pagination OrderBy rules** |
| `.ScanStruct(&dest)` | Scan one row into a struct |
| `.ScanStructs(&dest)` | Scan many rows into a slice |
| `.Query(ctx)` | Execute — use for **SELECT** (returns `error`) |
| `.Exec(ctx)` | Execute — use for **INSERT / UPDATE / DELETE** (returns `sql.Result, error`) |

> **OrderBy + cursor pagination**: When using `WithPagination()` + `WithOrderBy()`, the sort order must be deterministic. See **krangka-pagination** for full rules and examples.

### Template parameters

Placeholders in SQL use Go template syntax. qwery compiles them into safe parameterized queries — no string interpolation, no SQL injection risk.

```sql
-- single param
WHERE id = {{ .id }}

-- conditional block
{{ if .search }} AND name LIKE CONCAT('%', {{ .search }}, '%') {{ end }}

-- multiple conditions
WHERE deleted_at = 0
{{ if .is_active }} AND active = {{ .is_active }} {{ end }}
```

### SELECT — single row

```go
var w domain.Widget
err := r.qwery.
    RunRaw(`SELECT id, name FROM widgets WHERE id = {{ .id }} AND deleted_at = 0`).
    WithParam("id", id).
    ScanStruct(&w).
    Query(ctx)
```

### SELECT — list with pagination and optional filter

```go
items := make([]domain.Widget, 0)
err := r.qwery.
    RunRaw(`
        SELECT id, name, created_at
        FROM widgets
        WHERE deleted_at = 0
        {{ if .search }} AND name LIKE CONCAT('%', {{ .search }}, '%') {{ end }}
    `).
    WithParams(map[string]any{
        "search": filter.Search,
    }).
    WithPagination(pag).
    WithOrderBy("-created_at", "+id").
    ScanStructs(&items).
    Query(ctx)
```

### INSERT

```go
_, err := r.qwery.
    RunRaw(`INSERT INTO widgets (id, name) VALUES ({{ .id }}, {{ .name }})`).
    WithParams(w). // w is a *domain.Widget; fields mapped via qwery tags
    Exec(ctx)
```

### UPDATE

```go
_, err := r.qwery.
    RunRaw(`UPDATE widgets SET name = {{ .name }} WHERE id = {{ .id }} AND deleted_at = 0`).
    WithParams(w).
    Exec(ctx)
```

### Soft delete

```go
_, err := r.qwery.
    RunRaw(`UPDATE widgets SET deleted_at = UNIX_TIMESTAMP() WHERE id = {{ .id }} AND deleted_at = 0`).
    WithParam("id", id).
    Exec(ctx)
```

### `WithParams` — struct vs map

- **Struct**: qwery reads the `qwery` tag on each field. Only exported fields with a `qwery` tag are bound.
- **Map**: keys are the parameter names used in the template.

Use a map when you need to bind a subset of fields or combine values from multiple sources:

```go
WithParams(map[string]any{
    "search":    filter.Search,    // optional — nil/zero values are still bound but template {{ if }} can skip them
    "is_active": filter.IsActive,
})
```

---

## 5. Transactions

Transactions ensure multiple database operations succeed or fail together. Use `DoInTransaction` whenever a service performs more than one write, or when a write must be atomic with an outbox event.

### When to use

- **Multiple writes**: Create/update/delete across one or more tables — all must commit or all must roll back.
- **Write + outbox**: Domain write + `PublishOutbox` — both must be in the same transaction (see [Outbox Pattern](#6-outbox-pattern)).
- **Read-modify-write**: Load → modify → save, where another concurrent update would cause a conflict.

### How it works

```go
_, err := s.repo.DoInTransaction(ctx, func(repo outbound.Repository) (any, error) {
    // repo is the transactional copy — s.repo does NOT support transactions
    var (
        repoWidget = repo.GetWidgetRepository()  // assign to variable for reuse
    )
    
    if err := repoWidget.CreateWidget(ctx, w); err != nil {
        return nil, fail.Wrap(err)
    }
    
    if err := repo.PublishOutbox(ctx, outbound.PublisherTargetKafka, "widget.created", payload); err != nil {
        return nil, fail.Wrap(err)
    }
    return nil, nil
})
return fail.Wrap(err)
```

### Critical rule: `repo` (lambda arg) ≠ `s.repo` — only `repo` supports transactions

`repo` (the lambda argument) and `s.repo` (the service field) are different objects. `s.repo` does **not** support transactions; it uses the default connection. Only `repo` is transactional.

| Use (transactional) | Do not use (non-transactional) |
|---------------------|-------------------------------|
| `repo.GetWidgetRepository()` | `s.repo.GetWidgetRepository()` |
| `repo.PublishOutbox(...)` | `s.repo.PublishOutbox(...)` |

Inside the lambda, all repository and outbox calls must go through `repo`. Storing `repo.GetXxxRepository()` in a variable is fine and avoids repeated getter calls.

### Lifecycle

```
DoInTransaction(ctx, fn) called
  ├─ BEGIN transaction
  ├─ fn(repo) runs — repo uses tx-scoped qwery
  │    └─ All repo.GetXxxRepository() return tx-scoped instances
  ├─ On fn success → COMMIT
  ├─ On fn error   → ROLLBACK
  └─ On panic      → ROLLBACK, then re-panic
```

### Getters provide tx-scoped repositories

The aggregator (`mariaDBRepository`) holds `qweryTx`. When `qweryTx != nil`, getters return repositories that use the transaction:

```go
// GetWidgetRepository returns the Widget repository, using the active transaction when inside DoInTransaction.
func (r *mariaDBRepository) GetWidgetRepository() portRepo.Widget {
    if r.qweryTx != nil {
        return implRepo.NewWidgetRepository(r.qweryTx)  // tx-scoped
    }
    return r.WidgetRepository  // default connection
}
```

Repository implementations must accept `qwery.Runable` (not `*qwery.Client`) so they can receive either the default client or a `*qwery.Tx`.

### Nested calls

If `DoInTransaction` is called while already inside a transaction, the inner call reuses the same transaction — no nested BEGIN. The inner `fn` receives the same transactional `repo`.

### Error handling

- Return an error from `fn` → transaction rolls back, `DoInTransaction` returns that error.
- Panic inside `fn` → transaction rolls back, panic is rethrown.
- Success (no error) → transaction commits.

### When not to use

- **Read-only operations**: No need for a transaction. Use `s.repo.GetXxxRepository()` directly.
- **Single write with no outbox**: A single `Create`/`Update`/`Delete` is already atomic; use a transaction only if you also need `PublishOutbox` or multiple writes.

---

---

## Quick Checklist — Adding a New Entity End-to-End

- [ ] **Domain**: create `internal/core/domain/<entity>.go` with `qwery` struct tags
- [ ] **Migration**: run `make migrate-new repo=mariadb name=table_<entity>s`, then fill in the generated Up/Down SQL
- [ ] **Port interface**: create `internal/core/port/outbound/repositories/<entity>.go`
- [ ] **Root Repository interface**: add `Get<Entity>Repository()` getter to `internal/core/port/outbound/repository.go`
- [ ] **Implementation**: create `internal/adapter/outbound/mariadb/repositories/<entity>.go`
  - [ ] Field `qwery qwery.Runable` (not `*qwery.Client`)
  - [ ] `tracer.Trace(ctx)` + `defer span.End()` in every method
  - [ ] `fail.Wrap` on every error; `.WithFailure(fail.ErrNotFound)` for `sql.ErrNoRows`
- [ ] **Wire**: add field, constructor init, and getter to `internal/adapter/outbound/mariadb/repository.go`
  - [ ] Getter checks `r.qweryTx != nil` and returns tx-scoped instance
- [ ] **Service**: use [DoInTransaction](#5-transactions) for mutating operations; always use `repo` (lambda arg) inside the lambda, never `s.repo`. For outbox events, see **krangka-outbox**
