# qwery Query Append Rules (from qwery.md)

## How Pagination and Ordering Append

`WithPagination` and `WithOrderBy` **append** clauses to the parsed query at build time. They do not modify the SQL file.

## Offset Pagination (`Type: "offset"`)

Appends:
1. **ORDER BY** — before LIMIT/OFFSET
2. **LIMIT** — `PerPage`
3. **OFFSET** — `(Page - 1) * PerPage`

**Example:** `SELECT * FROM users WHERE status = ?` becomes:

```sql
SELECT * FROM users WHERE status = ? ORDER BY created_at DESC LIMIT ? OFFSET ?
```

Params: `[status, limit, offset]`

## Cursor Pagination (`Type: "cursor"`)

**First page:**
1. **ORDER BY** — required
2. **LIMIT** — `PerPage + 1`

```sql
SELECT * FROM users WHERE status = ? ORDER BY id ASC LIMIT ?
```

**Next page:** Adds WHERE condition before ORDER BY:

```sql
SELECT * FROM users WHERE status = ? AND (id > ?) ORDER BY id ASC LIMIT ?
```

For multi-column: `(col1, col2) > (?, ?)` or `(col1 < ? OR (col1 = ? AND col2 < ?))` depending on order.

## OrderBy Prefix

- `+column` or `column` — ASC
- `-column` — DESC

## CTE Queries

With `WithCTETarget("cte")`:
- ORDER BY: both CTE body and main query
- LIMIT/OFFSET: inside CTE only
- Cursor WHERE: inside CTE only