---
name: /krangka-query-review
id: krangka-query-review
category: Workflow
description: Gather and Generate a query review document listing all qwery RunRaw and Run queries with real generated form
---

Generate a timestamped query review document that lists all qwery queries in the repository.

**Skill**: Use the **krangka-query-review** skill for the full workflow. This command invokes that workflow.

**Input**: None required.

**Steps**

1. **Read the query-review skill**

   Load `.cursor/skills/krangka-query-review/SKILL.md` and follow its workflow.

2. **Ask the user: full, single, or scoped review?**

   Do not assume. Ask: "Full review (all queries), single-query review (e.g. only ListTicket), or scoped review (e.g. all ticket.* queries)?" Proceed only after the user confirms.

3. **Gather queries**

   Run (use `last_hashes.json` if it exists for automatic New/Updated/Unchanged). **Always** use `--write-hashes` so hashes are persisted for the next run. Use `--filter` when the user chose single or scoped: `--filter ticket.ListTicket` (single) or `--filter ticket` (scoped, all ticket.* queries).
   ```bash
   go run .cursor/skills/krangka-query-review/scripts/gather.go . docs/query-review/.hash/last_hashes.json --write-hashes docs/query-review/.hash/last_hashes.json
   ```
   For single query: `--filter ticket.ListTicket`. For scoped (e.g. all ticket.*): `--filter ticket`.
   ```bash
   go run .cursor/skills/krangka-query-review/scripts/gather.go . docs/query-review/.hash/last_hashes.json --write-hashes docs/query-review/.hash/last_hashes.json --filter ticket.ListTicket
   go run .cursor/skills/krangka-query-review/scripts/gather.go . docs/query-review/.hash/last_hashes.json --write-hashes docs/query-review/.hash/last_hashes.json --filter ticket
   ```
   If no previous review (no `last_hashes.json` yet): `go run .cursor/skills/krangka-query-review/scripts/gather.go . --write-hashes docs/query-review/.hash/last_hashes.json`

   **If gather fails:** Notify the user immediately. Capture stderr and exit code, then diagnose why (e.g. network/sandbox blocking `go get` for private deps, build errors). Report the exact error and suggested fix. **Never continue** — do not generate the document or proceed with any step until gather succeeds.

4. **Parse output**

   The gather outputs `{ "suggested_filename": "...", "queries": [...] }`. Use `suggested_filename` for the document path. Each query has `has_pagination`, `has_order_by`, `purpose_hint`, `change` (when last_hashes provided).

5. **Generate document**

   Create `docs/query-review/{suggested_filename}` following the skill's document format. **Exclude Unchanged** queries (already reviewed); include only New and Updated. Trust `has_pagination` — only show pagination variants when true.

**Output**

- Document path: `docs/query-review/{suggested_filename}`
- Summary: total queries, new/updated/unchanged counts (from `change` field when available)
- Each query: location, purpose (use `purpose_hint`), parameters, **template**, **Real query** (one per variant type when `has_pagination`), **AI Recommended Indexes**, **AI Recommended Query Change** when needed