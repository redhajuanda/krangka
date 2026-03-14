# Pagination Guide

krangka uses `silib/pagination` with `sikat` for **offset** and **cursor** pagination. The same code path works for both — sikat generates SQL automatically based on `pagination_type`.

## Query Parameters

| Param | Type | For offset | For cursor | Description |
|-------|------|------------|------------|-------------|
| `pagination_type` | string | ✓ | ✓ | **Required.** `offset` or `cursor` |
| `page` | int | ✓ | ✗ | Page number (offset only) |
| `per_page` | int | ✓ | ✓ | Items per page |
| `cursor` | string | ✗ | ✓ | Opaque cursor — omit on first request, pass `pagination.Result.Cursor.Next` for next page |
| `count_total_data` | bool | ✓ | ✓ | Include total count (skips `COUNT(*)` if false) |

## Request Flow

1. **DTO**: Embed `pagination.Pagination` in list request
2. **Handler**: Parse query params, pass pagination to service
3. **Service**: Pass pagination reference to repository
4. **Repository**: Use `.WithPagination(pagination).WithOrderBy(...)` — sikat adds LIMIT/OFFSET or cursor logic
5. **Handler**: Return `response.SuccessOKWithPagination(c, data, pagination)`

## End-to-End Example

### DTO
```go
type ReqListNote struct {
    pagination.Pagination
    Search string `query:"search" validate:"omitempty,max=100"`
}
```

### Handler
```go
func (h *NoteHandler) ListNotes(c fiber.Ctx) error {
    var req dto.ReqListNote
    if err := c.Bind().Query(&req); err != nil {
        return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
    }

    pagination := req.Pagination
    filter := &domain.NoteFilter{Search: req.Search}
    notes, err := h.svc.ListNote(c.Context(), filter, &pagination)
    if err != nil {
        return err
    }

    res := make([]dto.RespNote, 0, len(*notes))
    for _, n := range *notes {
        res = append(res, *new(dto.RespNote).Transform(&n))
    }
    return response.SuccessOKWithPagination(c, res, pagination)
}
```

### Service
```go
func (s *Service) ListNote(ctx context.Context, filter *domain.NoteFilter, pag *pagination.Pagination) (*[]domain.Note, error) {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    notes, err := s.repo.GetNoteRepository().ListNotes(ctx, filter, pag)
    if err != nil {
        return nil, fail.Wrap(err)
    }
    return notes, nil
}
```

### Repository
```go
func (r *noteRepository) ListNotes(ctx context.Context, filter *domain.NoteFilter, pag *pagination.Pagination) (*[]domain.Note, error) {
    ctx, span := tracer.Trace(ctx)
    defer span.End()

    notes := make([]domain.Note, 0)
    err := r.sikat.
        RunRaw(`
            SELECT id, title, content, created_at
            FROM notes
            WHERE deleted_at = 0
            {{ if .search }} AND title LIKE CONCAT('%', {{ .search }}, '%') {{ end }}
        `).
        WithParams(map[string]any{"search": filter.Search}).
        WithPagination(pag).
        WithOrderBy("+id").
        ScanStructs(&notes).
        Query(ctx)
    if err != nil {
        return nil, fail.Wrap(err)
    }
    return &notes, nil
}
```

## Cursor Pagination — Next Page

1. First request: `GET /notes?pagination_type=cursor&per_page=10`
2. Response includes `metadata.pagination.cursor.next`
3. Next request: `GET /notes?pagination_type=cursor&per_page=10&cursor=<value_from_next>`

## Cursor Pagination — OrderBy Rules

Cursor pagination **requires** an `OrderBy` clause. The columns must be **guaranteed unique**.

| Pattern | Use case |
|---------|---------|
| `WithOrderBy("+id")` | Simplest — ULID is unique and time-ordered |
| `WithOrderBy("-created_at", "+id")` | Order by time (not unique alone), append `id` |
| `WithOrderBy("name nullable", "-id")` | Nullable column — suffix with `nullable` |

**Rules:**
- Combination of `OrderBy` columns must be unique
- Only **one** column can be marked `nullable`
- Nullable columns are never unique — always combine with `id`

### Prefix Reference

| Prefix | Meaning |
|--------|---------|
| `+column` or `column` | Ascending |
| `-column` | Descending |
| `column nullable` | Ascending, nullable |
| `-column nullable` | Descending, nullable |

## Response Format

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

`total_data` is omitted when `count_total_data=false`.
