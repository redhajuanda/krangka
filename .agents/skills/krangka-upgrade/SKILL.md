---
name: krangka-upgrade
description: Tracks krangka framework changes and upgrades projects from their current version to the latest. Reads version reference files and applies migrations (add, remove, modify). Use when upgrading krangka, checking for framework updates, or when the user asks about krangka version, changelog, or migration.
---

# krangka Upgrade

Upgrade krangka framework (boilerplate) from the current version to the latest by reading version reference files and applying changes in order.

## When to Use

- User wants to upgrade krangka to the latest version
- User asks about krangka version, changelog, or migration
- User mentions "upgrade krangka", "update framework", "krangka version"

## Upgrade Workflow

### 1. Determine Current Version

- Read `.krangka/.VERSION` — contains the krangka framework version (e.g. `v1.0.1`)
- If the file does not exist, assume **v1.0.0** (first version)

### 2. Discover Available Versions

- Resolve the references directory — use the **first path that exists** in the project:
  1. `.agents/skills/krangka-upgrade/references/`
  2. `.cursor/skills/krangka-upgrade/references/`
  3. `.claude/skills/krangka-upgrade/references/`
- List all `v*.md` files in that directory (if several trees exist, they should carry the same set of files; prefer the first path above)
- Parse version numbers from filenames (e.g. `v1.0.1.md` → `1.0.1`)
- Sort versions semantically (1.0.0 < 1.0.1 < 1.0.2 < 1.1.0)
- Identify the **latest** version in references

### 3. Build Upgrade Path

- Versions to apply = all versions **strictly greater** than current, up to and including latest
- Example: current `v1.0.1`, latest `v1.0.5` → apply v1.0.2, v1.0.3, v1.0.4, v1.0.5 in order

### 4. Apply Each Version

For each version in the upgrade path (in order):

1. **Read** the reference file: `<references-dir>/vX.Y.Z.md` (same directory resolved in step 2)
2. **CRITICAL — Clone the exact upstream version before applying any code change.** Reference files describe changes at a high level (e.g. "add function `Blabla()`", "modify `Bootstrap()` to register X"). The actual implementation can vary in signature, body, imports, ordering, and surrounding context. **Never** assume or infer how to implement the change — always read the real source from the upstream boilerplate at that exact version:
   - Clone (or `git fetch`) the boilerplate at the target tag into a scratch directory:
     ```bash
     git clone --depth 1 --branch vX.Y.Z https://github.com/redhajuanda/krangka /tmp/krangka-vX.Y.Z
     ```
     (or reuse an existing clone and `git checkout vX.Y.Z`)
   - For every **Added** / **Modified** item in the reference file, open the corresponding file in `/tmp/krangka-vX.Y.Z/...` and copy the **exact** code as it appears in that version (function bodies, imports, comments, ordering).
   - Also diff against the **previous** tag (`vPREV`) when the reference says "modify": `git -C /tmp/krangka-vX.Y.Z diff vPREV vX.Y.Z -- <path>` to see the precise edits, not just the final state.
   - If the project has diverged (renames, custom code), reconcile manually — but the source of truth for what krangka itself changed is **always** the upstream tag, never the reference summary alone.
3. **Apply** all changes described, using the upstream source as the authoritative implementation:
   - **Added**: Create new files / code with the exact contents from the upstream tag
   - **Removed**: Delete files or remove code as specified
   - **Modified**: Apply the upstream diff, adapting only where the project legitimately diverges
   - **Dependencies**: Update `go.mod` (e.g. qwery, komon versions) to match upstream `go.mod` at that tag and run `go mod tidy`
4. **Follow** migration notes (commands to run, manual steps)
5. **Update** `.krangka/.VERSION` to the version just applied (only after successfully applying that version)

### 5. Finalize

- After applying the **latest** version, set `.krangka/.VERSION` to that version
- Run any final commands from migration notes (e.g. `go mod tidy`, `go build ./...`)

## Reference File Format

Each `references/vX.Y.Z.md` should describe changes from the **previous** version. Use sections:

- **Dependencies**: Package upgrades (e.g. qwery v1.0.0 → v1.0.1)
- **Added**: New files, new code, new config
- **Removed**: Deleted files, removed interfaces/code
- **Modified**: Changed files, refactors, breaking changes
- **Migration notes**: Commands to run, manual steps, breaking change guidance

## Rules

- Apply versions **in order** — do not skip versions
- If a reference file is missing for a version in the path, stop and report the gap
- **Never invent or paraphrase code from a reference file.** Always clone `github.com/redhajuanda/krangka` at the target tag and copy the exact implementation from there.
- Prefer applying changes exactly as described; avoid inferring beyond the reference
- After dependency changes, always run `go mod tidy`
- If the project has diverged from the boilerplate (custom files, removed files), adapt: skip or note conflicts; do not blindly overwrite user code
