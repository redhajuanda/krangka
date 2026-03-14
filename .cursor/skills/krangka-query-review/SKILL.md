---
name: krangka-query-review
description: Gathers Sikat RunRaw and Run queries from the repository and generates a review document. Use when the user asks to generate a query review document, review database queries, or document SQL queries for review.
---

# Query Review Document Generator

Generates a timestamped markdown document that lists all Sikat queries in the repository with their **real** generated form (not templates), explanations, and possible Sikat appends. Supports both `RunRaw()` (inline SQL) and `Run()` (SQL from `.sql` files).

## Trigger

- **Command**: `/krangka-query-review` (see `.cursor/commands/krangka-query-review.md`)
- **Phrases**: "generate document review query", "generate query review", "review queries", or similar.

## Output Path

Write the document to `docs/query-review/YYYYMMDDHHMMSS_query.md` (timestamp = current time).

## Workflow

### 0. Ask the User: Full, Single, or Scoped Review?

**Do not assume.** Before running gather, ask the user:

> Do you want a **full review**, a **single-query review**, or a **scoped review** (e.g. by repository)?
>
> - **Full review** — no filter; documents every RunRaw and Run query.
> - **Single query** — add `--filter <queryName>` (e.g. `ticket.ListTicket`).
> - **Scoped (multiple, not all)** — add `--filter <repository>` to include all queries in that repository (e.g. `ticket` → ticket.ListTicket, ticket.SimpleListTicket, ticket.CountTicketFCR, etc.).

Proceed only after the user confirms. Do not infer from phrases like "for ListTicket change" — that may mean "document the changes" (full review), "only ListTicket" (single query), or "ticket repository" (scoped).

### 1. Gather Queries

Run the gather script from the project root. Use `last_hashes.json` if it exists for automatic New/Updated/Unchanged:

**Full review (all queries):**
```bash
go run .cursor/skills/krangka-query-review/scripts/gather.go . docs/query-review/.hash/last_hashes.json --write-hashes docs/query-review/.hash/last_hashes.json
```

**Single query only** (add `--filter <queryName>`):
```bash
go run .cursor/skills/krangka-query-review/scripts/gather.go . docs/query-review/.hash/last_hashes.json --write-hashes docs/query-review/.hash/last_hashes.json --filter ticket.ListTicket
```

**Scoped by repository** (add `--filter <repository>` to include all queries in that repo):
```bash
go run .cursor/skills/krangka-query-review/scripts/gather.go . docs/query-review/.hash/last_hashes.json --write-hashes docs/query-review/.hash/last_hashes.json --filter ticket
```

If no previous review exists (no `last_hashes.json`), omit the second arg for full review, or use `--filter` without the second arg for single/scoped query.

**If the gather script fails:**

1. **Notify the user** — explicitly state that the gather step failed.

2. **Find out why it failed** — capture stderr and exit code. Common causes:
   - **Network/sandbox:** `no secure protocol found for repository` — Go cannot fetch private GitLab deps (`sikat`, `silib`). Run with `network` or `full_network` permission, or ask the user to run the gather locally.
   - **Build errors:** Missing dependencies, wrong Go version, syntax errors in the repo.
   - **Runtime errors:** Panics in gather.go or sikat.Build() failures.

3. **Report the failure** — tell the user the exact error message and suggested fix (e.g. "Run the gather command locally with network access" or "Check that `go mod tidy` resolves dependencies").

4. **Never continue** — do not generate the document or proceed with any step until gather succeeds.

The script outputs a JSON object with:
- **`suggested_filename`** — use this for the output path (e.g. `20260310170103_query.md`)
- **`queries`** — array of query objects with: `file`, `method`, `source`, `query`, `params`, `optional_params`, `pagination`, `order_by`, **`built_variants`**, `has_pagination`, `has_order_by`, `purpose_hint`, `query_hash`, `change` (when last_hashes provided)

**Trust `has_pagination` and `has_order_by`** — only queries with these true have pagination variants in `built_variants`. Use `purpose_hint` as the starting point for the Purpose field.

### 2. Compare with Last Review

If you passed `last_hashes.json`, each query already has `change` set (New/Updated/Unchanged). Otherwise, if previous `*_query.md` files exist, read the most recent and assign manually.

### 3. Generate Document

Create `docs/query-review/{suggested_filename}` using the suggested filename from the gather output.

**Exclude Unchanged queries:** When `last_hashes.json` was provided and queries have `change` set, **exclude** queries where `change == "Unchanged"` from the document. Include only **New** and **Updated**. Unchanged queries have already been reviewed and need not be repeated. If all gathered queries are Unchanged, generate a minimal doc stating "No changes to document" and list the excluded count.

### 4. Persist Hashes for Next Run

When generating the document, run gather with `--write-hashes` so the next run can diff. You can combine with last_hashes in one run:

```bash
go run .cursor/skills/krangka-query-review/scripts/gather.go . docs/query-review/.hash/last_hashes.json --write-hashes docs/query-review/.hash/last_hashes.json
```

This reads last hashes (for change status), outputs the full JSON, and writes the new hashes for the next run.

## Document Format

Use **one two-column table per query** (Field | Value). Each field is a row; content goes in the Value column. For SQL, use `<pre><code>...</code></pre>` inside the cell.

**Summary table:** `Total` = queries included in this doc (New + Updated only). When Unchanged queries were excluded, add an `Excluded (unchanged)` row.

```markdown
# Query Review — YYYY-MM-DD HH:MM

## Summary

| Metric | Count |
|--------|-------|
| Total | N |
| New | X |
| Updated | Y |
| Excluded (unchanged) | Z |

## Query Overview

| # | Method | Location | Purpose | Change |
|---|--------|----------|---------|--------|
| 1 | outbox.GetOutbox | outbox.go:113 | Fetch pending outbox entries for relay | New |
| 2 | ... | ... | ... | ... |

## Queries

### 1. [Repository].[Method] — [Brief purpose]

| Field | Value |
|-------|-------|
| Source | `RunRaw` |
| Location | `path/to/file.go:Lines` |
| Purpose | One-sentence explanation. |
| Parameters | - `status`: `"pending"` (mandatory)<br>- `retry_attempt`: `0` (mandatory)<br>- `search`: `"term"` (optional) |
| Template | <pre><code>SELECT id, title, content<br>FROM notes<br>WHERE deleted_at = 0<br>  {{ if .search }}AND (title LIKE ...){{ end }}</code></pre> |
| Real query | **Offset:**<br><pre><code>SELECT ... FROM ...<br>WHERE status = ?  -- mandatory<br>  AND deleted_at = 0  -- mandatory (soft-delete filter)<br>ORDER BY id ASC<br>LIMIT ? OFFSET ?  -- limit, offset: mandatory</code></pre><br>**Cursor first:**<br><pre><code>...</code></pre><br>**Cursor next:**<br><pre><code>...</code></pre> |
| AI Recommended Indexes | Explain why needed; use naming `{table}_ie_{order}`. E.g. Needed because query filters by deleted_at and orders by id for pagination.<br><pre><code>CREATE INDEX notes_ie_1 ON notes (deleted_at, id);</code></pre> |
| AI Recommended Query Change | Steps outside code; DDL/query in <code>. Use naming `{table}_ie_{order}` for FULLTEXT. E.g. 1. Add FULLTEXT index:<br><pre><code>ALTER TABLE notes ADD FULLTEXT INDEX notes_ie_2 (title, content);</code></pre><br>2. Change from <code>AND (title LIKE ...)</code> to <code>AND MATCH(...) AGAINST(? IN NATURAL LANGUAGE MODE)</code> |
| Change | New |

---

### 2. ...
```

## Full Example (GetOutbox)

```markdown
### 1. outbox.GetOutbox — Fetch pending outbox entries for relay

| Field | Value |
|-------|-------|
| Source | `RunRaw` |
| Location | `internal/adapter/outbound/mariadb/outbox.go:113` |
| Purpose | Fetch pending outbox entries for relay worker, filtered by status, deleted_at, and retry_attempt. |
| Parameters | - `status`: `"pending"` (mandatory)<br>- `retry_attempt`: `0` (mandatory) |
| Template | <pre><code>SELECT<br>  id, topic, payload, target, retry_attempt<br>FROM outboxes<br>WHERE status = {{ .status }}<br>  AND deleted_at = 0<br>  AND retry_attempt &lt; {{ .retry_attempt }}</code></pre> |
| Real query | **Offset:**<br><pre><code>SELECT id, topic, payload, target, retry_attempt<br>FROM outboxes<br>WHERE status = ?  -- mandatory<br>  AND deleted_at = 0  -- mandatory (soft-delete filter)<br>  AND retry_attempt &lt; ?  -- mandatory<br>ORDER BY id ASC<br>LIMIT ? OFFSET ?  -- limit, offset: mandatory</code></pre><br>**Cursor first:**<br><pre><code>... LIMIT ?  -- limit: PerPage+1 for has-more check</code></pre><br>**Cursor next:**<br><pre><code>... AND (id &gt; ?)  -- cursor: last id from previous page<br>ORDER BY id ASC LIMIT ?  -- limit: PerPage+1</code></pre> |
| AI Recommended Indexes | Needed because the query filters by status, deleted_at, retry_attempt and orders by id for pagination.<br><pre><code>CREATE INDEX outboxes_ie_1 ON outboxes (status, deleted_at, retry_attempt, id);</code></pre> |
| AI Recommended Query Change | Omit — index works with existing query. |
| Change | New |
```

**When Change is "Updated", add Changes (for DBE) and Changes (for developer) after AI Recommended Query Change:**

```markdown
| **Changes (for DBE)** | No changes to the real query. Generated SQL is identical to the previous version. |
| **Changes (for developer)** | <ul><li><strong>Template:</strong> Removed erroneous branch_id from CTE (doc fix).</li><li><strong>Parameters:</strong> Added cursor param.</li><li><strong>Query Hash:</strong> 65a314cc → 489edca6</li></ul> |
| Change | Updated |
```

When the real query *did* change, list the SQL-level differences in Changes (for DBE), e.g.:
```markdown
| **Changes (for DBE)** | <ul><li><strong>CTE WHERE:</strong> Added <code>AND t.stage IN (?)</code></li><li><strong>Main query:</strong> New JOIN <code>left join activity_log al on al.object_instance = t.id</code></li></ul> |
```

## Rules for Document Fields

- **Summary:** Use a table with columns Metric | Count. `Total` = queries included in this doc (New + Updated). When `last_hashes` was used, exclude Unchanged from the doc and show `Excluded (unchanged)` count.
- **Query Overview:** Use a table with columns # | Method | Location | Purpose | Change. One row per query.
- **Per-query:** One table with columns Field | Value. Field order: Source, Location, Query Hash, Purpose, Parameters, Template, Real query, AI Recommended Indexes, AI Recommended Query Change, **Changes (for DBE)** (only when Change is "Updated"), **Changes (for developer)** (only when Change is "Updated"), Change. Include `Query Hash` from gather output (`query_hash`). For multi-line SQL, use `<pre><code>...</code></pre>` in the Value cell. For parameters: bullet list with `<br>` between items.
- **Changes (for DBE):** When `Change` is "Updated", add a **Changes (for DBE)** row — for database engineers who only care about the real query. Compare the **Real query** section with the previous doc. If identical: "No changes to the real query. Generated SQL is identical to the previous version." If different: list what changed in the executed SQL (e.g. "CTE WHERE: added `AND t.stage IN (?)`", "Main query: new JOIN `left join activity_log al on ...`", "Removed LIMIT/OFFSET"). Place after **AI Recommended Query Change**.
- **Changes (for developer):** When `Change` is "Updated", add a **Changes (for developer)** row. Read the most recent prior query review doc for the same query (match by method/query name in `docs/query-review/`), diff the Template, Parameters, AI Recommended Indexes, AI Recommended Query Change, and Real query sections. Summarize the differences in a bullet list (e.g. "Template: removed X", "Parameters: added Y", "Query Hash: old → new"). Use `<ul><li>...</li></ul>` for the Value cell. Place after **Changes (for DBE)**.
- **NO TRUNCATION:** Never abbreviate or truncate SQL or template content. Always include the **full** query — no `...`, `... (many optional filters)`, `select t.id, t.code, ...`, or similar placeholders. Every line of the template and every variant of the real query must be complete.
- **Template:** Format the gather `query` field in `<pre><code>...</code></pre>` inside the Value cell. **Use `<br>` for line breaks** — never use actual newlines, or the Markdown table will break (newlines end table rows). Preserve `{{ .param }}` and `{{ if .param }}` placeholders. Use consistent 2-space indentation. **Include the full template** — never truncate.
- **AI Recommended Indexes:** Always explain **why** the index is needed (or why none is needed). Put explanation outside code; put only the real SQL inside `<pre><code>...</code>`. Use index naming format `{table_name}_ie_{order}` (e.g. `outboxes_ie_1`, `notes_ie_1`). For "no extra index needed", explain why (e.g. "Primary key suffices because this is a single-row lookup by PK").
- **AI Recommended Query Change:** Put step labels and explanations (e.g. "1. Add FULLTEXT index:", "2. Change query from") outside code. Put only real DDL and query snippets inside `<code>` or `<pre><code>`.

## Rules for "Real Query"

**CRITICAL — Inline comments only:** Annotate mandatory/optional **inside the SQL** with `-- mandatory` or `-- optional` next to each `?` or clause. **Never** use a separate "Params:" line outside the code block.

**CRITICAL — No truncation:** Always show the **full** real query for each variant (offset, cursor first, cursor next). Never use `...`, `select t.id, t.code, ...`, or similar placeholders. Every SELECT, FROM, JOIN, WHERE, GROUP BY, ORDER BY, LIMIT must be complete.

**CRITICAL — Use `<br>` for line breaks in table cells:** Never use actual newlines inside table cells. Markdown treats newlines as row boundaries, so newlines in the Value column will break the table. Use `<br>` instead.

1. **Trust `has_pagination`** — only include pagination variants (offset, cursor first, cursor next) when `has_pagination` is true. Queries without pagination show a single base query only.

2. **Use `built_variants`** — the gather script generates every query possibility. Pick one representative per variant type. **Show the full SQL for each variant** — no truncation.

3. **Simplify when many variants:** For queries with both optional params and pagination (e.g. ListNote), show: (a) base list without search — full offset, full cursor first, full cursor next; (b) with search — full example (e.g. offset only) and note "cursor variants follow same pattern with search param". Each variant must still be complete SQL.

4. **Sample values:** Use realistic examples: `"01HXXX0000000000000000000"` for IDs, `"pending"` for status, `0` for deleted_at.

5. **Mandatory/optional — inline only, never separate:** Annotate every `?` placeholder and fixed filter (e.g. `deleted_at = 0`) **inline in the SQL** using `-- mandatory` or `-- optional` comments. **Never** use a separate "Params:" line. Example: `WHERE status = ?  -- mandatory` and `LIMIT ? OFFSET ?  -- limit, offset: mandatory`. Every clause that uses a param or is always present must have an inline comment.

## Index Recommendation Rules (MySQL/MariaDB)

For each query, recommend indexes based on the query structure. This project uses **MySQL/MariaDB**.

**General rules:**
- **Primary key / unique lookup:** `WHERE id = ?` — primary key usually suffices; no extra index needed.
- **Filter columns:** Index columns in WHERE (equality first, then range). Order: equality (`=`, `IN`) before range (`<`, `>`, `LIKE`).
- **Soft-delete:** `deleted_at = 0` — include in composite indexes for filtered tables.
- **ORDER BY:** For pagination, index should support ORDER BY columns (avoid filesort).
- **Covering index:** If SELECT columns are few, consider including them in the index to avoid table lookups.
- **LIKE prefix:** `LIKE 'value%'` can use index; `LIKE '%value%'` cannot (full scan). For `LIKE '%term%'` on text columns, recommend FULLTEXT index and add **AI Recommended Query Change** with `MATCH ... AGAINST`.

**Format:** Index names use `{table_name}_ie_{order}` (e.g. `outboxes_ie_1`, `notes_ie_1`). Always explain why the index is needed. Put explanation outside code; real SQL inside `<code>`. Example: "Needed because the query filters by status, deleted_at, retry_attempt and orders by id for pagination; without this index the relay worker would do a full table scan." followed by `<pre><code>CREATE INDEX outboxes_ie_1 ON outboxes (status, deleted_at, retry_attempt, id);</code></pre>`. For "no extra index needed", explain why (e.g. "Primary key on id suffices because this is a single-row lookup by primary key").

**Avoid:** Redundant indexes (e.g. duplicate of primary key), indexes on low-cardinality columns alone, too many indexes on write-heavy tables.

## Query Change Recommendation Rules

When recommending an index that requires a different query pattern, include **AI Recommended Query Change** with:
- **FULLTEXT search:** For `LIKE '%term%'` on text columns, recommend `FULLTEXT(title, content)` and show `MATCH(col1, col2) AGAINST(? IN NATURAL LANGUAGE MODE)` as the replacement. Use index naming `{table_name}_ie_{order}` (e.g. `notes_ie_2`). Put step labels outside code; put only the real DDL and query snippets inside `<code>`.
- **Other cases:** If an index suggests a different WHERE/JOIN pattern (e.g. avoiding OR, using IN subquery), show the alternative.
- **Omit** when the index works with the existing query (no change needed).

## Sikat Append Reference

Use this to build the **full real query** when the chain has `WithPagination`/`WithOrderBy`:

| Chain | Append |
|-------|--------|
| `WithPagination(offset)` | `ORDER BY [orderBy] LIMIT ? OFFSET ?` |
| `WithPagination(cursor)` first page | `ORDER BY [orderBy] LIMIT ?` |
| `WithPagination(cursor)` next page | `AND (col < ? OR ...) ORDER BY ... LIMIT ?` |
| `WithOrderBy("id")` | `ORDER BY id` |
| `WithOrderBy("+id")` | `ORDER BY id ASC` |
| `WithOrderBy("-created_at")` | `ORDER BY created_at DESC` |

See [reference.md](references/reference.md) for full append rules from sikat.md.

## Ordering Queries

Order by: file path, then by method appearance (top to bottom). Group by repository/module (e.g. note, outbox).
