---
name: krangka-pagination
description: Guide for automatic pagination in krangka using komon pagination and qwery SDK. Covers offset vs cursor pagination, query params, response format, and the required OrderBy rules for cursor pagination. Use when implementing list endpoints, pagination, or when the user asks about offset/cursor pagination.
---

# krangka Pagination Guide

krangka uses `komon/pagination` with `qwery` to support **offset** and **cursor** pagination. The same code path works for both — qwery generates and modifies SQL automatically based on `pagination_type`.

---

## Query Parameters

| Param | Type | For offset | For cursor | Description |
|-------|------|------------|------------|-------------|
| `pagination_type` | string | ✓ | ✓ | **Required.** `offset` or `cursor` |
| `page` | int | ✓ | ✗ | Page number (offset only). Ignored for cursor |
| `per_page` | int | ✓ | ✓ | Items per page. Used by both |
| `cursor` | string | ✗ | ✓ | Opaque cursor for next/prev page. Omit on first request; pass `pagination.Result.Cursor.Next` for next page |
| `count_total_data` | bool | ✓ | ✓ | Include total count in response. User can omit to skip `COUNT(*)` |

---

## Request Flow

1. **Handler**: Embed `pagination.Pagination` in list request DTO; parse query params
2. **Service**: Pass pagination as a refference to repository
3. **Repository**: Use `.WithPagination(pagination).WithOrderBy(...)` — qwery adds LIMIT/OFFSET or cursor logic
4. **Handler**: Return `response.SuccessOKWithPagination(c, data, pagination)` — `pagination.Result` is populated by qwery

---

## End-to-End Example

### DTO (embed Pagination)

```go
type ReqListNote struct {
    pagination.Pagination
    Search string `query:"search" validate:"omitempty,max=100" form:"search"`
}
```

### Handler

```go
pagination := req.Pagination
notes, err := h.svc.ListNote(ctx, filter, &pagination)
// ...
return response.SuccessOKWithPagination(c, res, pagination)
```

### Repository

```go
err := r.qwery.
    RunRaw(`SELECT ... FROM notes WHERE deleted_at = 0 {{ if .search }} AND ... {{ end }}`).
    WithParams(map[string]any{"search": req.Search}).
    WithPagination(pagination).
    WithOrderBy("+id").
    ScanStructs(&notes).
    Query(ctx)
```

---

## Cursor Pagination — Next Page

1. First request: `GET /notes?pagination_type=cursor&per_page=10`
2. Response includes `metadata.pagination.cursor.next`
3. Next request: `GET /notes?pagination_type=cursor&per_page=10&cursor=<value_from_next>`
4. Pass the exact `cursor` string; qwery decodes it and applies the correct `WHERE` / `ORDER BY` for the next page

---

## Cursor Pagination — OrderBy Details

### Required OrderBy

Cursor pagination **requires** an `OrderBy` clause. The column names must match a struct field tag in the result — the column value is used as the cursor pointer to track the last query position.

### Unique Order Combination

The combination of columns in `OrderBy` must be **guaranteed unique**. Otherwise pagination becomes inconsistent and may skip or duplicate records.

- **Simplest**: `WithOrderBy("+id")` or `WithOrderBy("-id")`. ULID is unique and time-ordered.
- **Other column**: If not unique (e.g. `created_at`), append `id`: `WithOrderBy("-created_at", "+id")`
- **Never** use a non-unique column alone when using pagination.

### Handling Nullable Columns

If the order column is nullable, suffix it with `nullable`:
- `name nullable` — ascending, nullable
- `-name nullable` — descending, nullable

**Rules:**
- Only **one** column can be marked `nullable` in the OrderBy clause (cursor pagination requires deterministic ordering).
- Nullable columns are never unique — **always** combine with another unique column (e.g. `id`).

### OrderBy Prefix

| Prefix | Meaning | Example |
|--------|---------|---------|
| `+column` or `column` | Ascending | `"name"` or `"+name"` |
| `-column` | Descending | `"-created_at"` |

Example — order by name ascending, id descending:

```go
err := r.qwery.
    RunRaw(`SELECT ... FROM users WHERE deleted_at = 0`).
    WithPagination(pagination).
    WithOrderBy("name", "-id").
    ScanStructs(&users).
    Query(ctx)
```

---

## Response Format

`qwery` populates `pagination.Result` based on type:

**Offset** (`metadata.pagination`):
```json
{
  "type": "offset",
  "offset": {
    "total_data": 100,
    "total_page": 10,
    "current_page": 1,
    "per_page": 10
  }
}
```

**Cursor** (`metadata.pagination`):
```json
{
  "type": "cursor",
  "cursor": {
    "total_data": 100,
    "next": "<opaque_cursor>",
    "prev": "<opaque_cursor>"
  }
}
```

`total_data` is omitted when `count_total_data=false` (avoids `COUNT(*)` query).
