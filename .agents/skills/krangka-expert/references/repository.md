# Repository Guide

Five parts:
1. [Migration](#1-migration)
2. [Repository Port](#2-repository-port)
3. [Repository Implementation](#3-repository-implementation)
4. [qwery SDK Integration](#4-qwery-sdk-integration)
5. [Transactions](#5-transactions)

For outbox event publishing inside transactions, see [outbox.md](outbox.md).
For pagination, see [pagination.md](pagination.md).

Working examples in the codebase:
- `internal/core/port/outbound/repositories/note.go` — port interface
- `internal/adapter/outbound/mariadb/repositories/note.go` — implementation
- `internal/adapter/outbound/mariadb/repository.go` — aggregator + transaction wiring

---

## 1. Migration

### Always use make command — never create migration files manually

```bash
make migrate-new repo=mariadb name=<description>
# or: go run main.go migrate new mariadb <description>
```

Name rules: use underscores, be descriptive (`table_widgets`, `add_index_widgets_name`), one logical change per file.

### File Structure

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

### Standard Column Conventions

| Column | Type | Rule |
|--------|------|------|
| `id` | `varchar(26)` | ULID; always PRIMARY KEY |
| `created_at` | `timestamp(6)` | Microsecond precision |
| `updated_at` | `timestamp(6) ON UPDATE CURRENT_TIMESTAMP(6)` | Auto-updated |
| `deleted_at` | `int NOT NULL DEFAULT 0` | Soft delete: `0` = active, UNIX timestamp = deleted |

> **Every table must include `deleted_at int NOT NULL DEFAULT 0` — no exceptions.** Always soft-delete. Filter active rows with `WHERE deleted_at = 0`. Never use a nullable `deleted_at`; the column is `int`, not `timestamp`.

### Index and FK naming convention

Format: `{table_name}_{index_code}_{order}`

| Type | Code | Example |
|------|------|---------|
| Regular index | `ie` | `widgets_ie_1` |
| Fulltext index | `ft` | `widgets_ft_1` |
| Unique index | `uq` | `widgets_uq_1` |
| Foreign key | `fk` | `widgets_fk_1` |

### What to include in development migrations

- ✅ **Unique indexes (`uq`)** — always include
- ✅ **Foreign keys (`fk`)** — always include
- ✅ **Fulltext indexes (`ft`)** — always include; omitting them causes runtime query failures
- ❌ **Regular indexes (`ie`)** — **never add during development**; index strategy is reviewed and finalized during release preparation

---

## Repository Design Rules

- **One repository per table/domain.** Each repository file owns the queries for one table. Do not put all entity queries into a single large repository.
- **JOINs belong to the main table's repository.** If a query joins multiple tables, place it in the repository of the primary/driving table.
- **One function = one query.** A single repository method must contain exactly one SQL statement. No multi-query logic inside a repository function.
- **Cross-table atomicity belongs in the service layer.** If an operation requires multiple queries to be atomic, wire them together via `DoInTransaction` in the service. Never orchestrate multi-step writes inside a repository function.

---

## 2. Repository Port

### Define the entity interface

```go
// internal/core/port/outbound/repositories/widget.go
//go:generate mockgen -source=widget.go -destination=../../../../mocks/outbound/repositories/mock_widget.go -package=mocksrepos
package repositories

type Widget interface {
    GetWidgetByID(ctx context.Context, id string) (*domain.Widget, error)
    CreateWidget(ctx context.Context, w *domain.Widget) error
    UpdateWidget(ctx context.Context, w *domain.Widget) error
    DeleteWidget(ctx context.Context, id string) error
    ListWidgets(ctx context.Context, filter *domain.WidgetFilter, pag *pagination.Pagination) (*[]domain.Widget, error)
}
```

### Register getter in root Repository interface

```go
// internal/core/port/outbound/repository.go
type Repository interface {
    DoInTransaction(ctx context.Context, fn func(repo Repository) (any, error)) (any, error)
    PublishOutbox(ctx context.Context, target PublisherTarget, topic string, payload qwery.JSONMap) error
    RetryOutbox(ctx context.Context) error
    GetWidgetRepository() repositories.Widget  // ← add this
}
```

---

## 3. Repository Implementation

```go
// internal/adapter/outbound/mariadb/repositories/widget.go
type widgetRepository struct {
    qwery qwery.Runable  // Runable, NOT *qwery.Client — required for transaction support
}

func NewWidgetRepository(qwery qwery.Runable) *widgetRepository {
    return &widgetRepository{qwery: qwery}
}
```

**Mandatory rules for every method:**
- `tracer.Trace(ctx)` + `defer span.End()` at the top
- `fail.Wrap(err)` on every error return
- Map `sql.ErrNoRows` to typed failure from `shared/failure`

```go
func (r *widgetRepository) GetWidgetByID(ctx context.Context, id string) (*domain.Widget, error) {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    var w domain.Widget
    err := r.qwery.
        RunRaw(`SELECT id, name, created_at FROM widgets WHERE id = {{ .id }} AND deleted_at = 0`).
        WithParam("id", id).
        ScanStruct(&w).
        Query(ctx)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, fail.Wrap(err).WithFailure(failure.ErrWidgetNotFound)
        }
        return nil, fail.Wrap(err)
    }
    return &w, nil
}
```

### Wire into the aggregator

```go
// internal/adapter/outbound/mariadb/repository.go
type mariaDBRepository struct {
    // ... existing fields
    WidgetRepository portRepo.Widget
}

func NewMariaDBRepository(...) *mariaDBRepository {
    return &mariaDBRepository{
        // ...
        WidgetRepository: implRepo.NewWidgetRepository(qwery.Client),
    }
}

// Transaction-aware getter
func (r *mariaDBRepository) GetWidgetRepository() portRepo.Widget {
    if r.qweryTx != nil {
        return implRepo.NewWidgetRepository(r.qweryTx)
    }
    return r.WidgetRepository
}
```

---

## 4. qwery SDK Integration

### Query Chain Reference

| Method | When to use |
|--------|------------|
| `.RunRaw(query)` | Start any query |
| `.WithParam("key", value)` | Bind a single named parameter |
| `.WithParams(v)` | Bind all fields from struct (via `qwery` tags) or `map[string]any` |
| `.WithPagination(pag)` | Attach pagination — adds LIMIT/cursor logic automatically |
| `.WithOrderBy(cols...)` | Prefix `+`/`-` for ASC/DESC. E.g. `"-created_at", "+id"` |
| `.ScanStruct(&dest)` | Scan one row |
| `.ScanStructs(&dest)` | Scan many rows |
| `.Query(ctx)` | Execute SELECT (returns `error`) |
| `.Exec(ctx)` | Execute INSERT/UPDATE/DELETE (returns `sql.Result, error`) |

### Template Parameters

```sql
WHERE id = {{ .id }}
{{ if .search }} AND name LIKE CONCAT('%', {{ .search }}, '%') {{ end }}
{{ if .is_active }} AND active = {{ .is_active }} {{ end }}
```

> **CRITICAL — Never concatenate query strings.** Always use `WithParam`/`WithParams`. Never `fmt.Sprintf` into `RunRaw`.

### SELECT — single row
```go
var w domain.Widget
err := r.qwery.
    RunRaw(`SELECT id, name FROM widgets WHERE id = {{ .id }} AND deleted_at = 0`).
    WithParam("id", id).
    ScanStruct(&w).
    Query(ctx)
```

### SELECT — list with pagination
```go
items := make([]domain.Widget, 0)
err := r.qwery.
    RunRaw(`
        SELECT id, name, created_at
        FROM widgets
        WHERE deleted_at = 0
        {{ if .search }} AND name LIKE CONCAT('%', {{ .search }}, '%') {{ end }}
    `).
    WithParams(map[string]any{"search": filter.Search}).
    WithPagination(pag).
    WithOrderBy("-created_at", "+id").
    ScanStructs(&items).
    Query(ctx)
```

### INSERT
```go
_, err := r.qwery.
    RunRaw(`INSERT INTO widgets (id, name) VALUES ({{ .id }}, {{ .name }})`).
    WithParams(w).
    Exec(ctx)
```

### UPDATE
```go
_, err := r.qwery.
    RunRaw(`UPDATE widgets SET name = {{ .name }} WHERE id = {{ .id }} AND deleted_at = 0`).
    WithParams(w).
    Exec(ctx)
```

### Soft Delete
```go
_, err := r.qwery.
    RunRaw(`UPDATE widgets SET deleted_at = UNIX_TIMESTAMP() WHERE id = {{ .id }} AND deleted_at = 0`).
    WithParam("id", id).
    Exec(ctx)
```

---

## 5. Transactions

### When to use
- **Multiple writes**: Multiple tables — all must commit or roll back together
- **Write + outbox**: Domain write + `PublishOutbox` must be atomic
- **Read-modify-write**: Concurrent update would cause conflict

### How it works

```go
_, err := s.repo.DoInTransaction(ctx, func(repo outbound.Repository) (any, error) {
    // ⚠️ Use `repo` (lambda arg), NEVER `s.repo` — only `repo` is transactional
    repoWidget := repo.GetWidgetRepository()

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

**Critical:** `repo` (lambda arg) ≠ `s.repo`. Only `repo` is transactional.

### Lifecycle
```
DoInTransaction called
  ├─ BEGIN transaction
  ├─ fn(repo) runs — repo uses tx-scoped qwery
  ├─ On success → COMMIT
  ├─ On error   → ROLLBACK
  └─ On panic   → ROLLBACK, re-panic
```

### Transaction registry — CRITICAL

When adding a new repository to the MariaDB aggregator (`internal/adapter/outbound/mariadb/repository.go`):

- **Do NOT** add the new repository field (e.g. `NoteRepository`, `UserRepository`) to the **transaction registry** struct — i.e. the `&mariaDBRepository{ ... }` created **inside** `DoInTransaction` when `r.qweryTx == nil`.
- That struct must contain **only**: `cfg`, `log`, `qwery`, `qweryTx`, `outbox`. No repository fields.
- The **getter** (e.g. `GetUserRepository()`) already returns a tx-backed instance when `r.qweryTx != nil` by calling `implRepo.NewXxxRepository(r.qweryTx)`. Putting repository fields on the transaction registry is invalid and must never be done.

### Do not modify DoInTransaction or handleTransaction

Do **not** change anything inside the function bodies of `DoInTransaction` or `handleTransaction` in `internal/adapter/outbound/mariadb/repository.go`. Do not add, remove, or edit logic in those two functions.

### When NOT to use
- Read-only operations: use `s.repo.GetXxxRepository()` directly
- Single write with no outbox: single CRUD is already atomic

---

## Quick Checklist — Adding a New Entity

- [ ] Domain: `internal/core/domain/<entity>.go` with `qwery` tags
- [ ] Migration: `make migrate-new repo=mariadb name=table_<entity>s`
- [ ] Port interface: `internal/core/port/outbound/repositories/<entity>.go`
- [ ] Add `Get<Entity>Repository()` to root `Repository` interface
- [ ] Implementation: `internal/adapter/outbound/mariadb/repositories/<entity>.go`
  - [ ] Field `qwery qwery.Runable` (not `*qwery.Client`)
  - [ ] `tracer.Trace(ctx)` + `defer span.End()` in every method
  - [ ] `fail.Wrap` on every error; `.WithFailure(failure.ErrXxx)` for `sql.ErrNoRows`
- [ ] Wire: field + constructor + transaction-aware getter in `repository.go`
  - [ ] **Do NOT** add repository fields to the transaction registry struct inside `DoInTransaction` (only `cfg`, `log`, `qwery`, `qweryTx`, `outbox`)