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
2. **Apply** all changes described:
   - **Added**: Create new files, add new code/config as specified
   - **Removed**: Delete files or remove code as specified
   - **Modified**: Update files, dependencies, config as specified
   - **Dependencies**: Update `go.mod` (e.g. qwery, komon versions) and run `go mod tidy`
3. **Follow** migration notes (commands to run, manual steps)
4. **Update** `.krangka/.VERSION` to the version just applied (only after successfully applying that version)

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
- Prefer applying changes exactly as described; avoid inferring beyond the reference
- After dependency changes, always run `go mod tidy`
- If the project has diverged from the boilerplate (custom files, removed files), adapt: skip or note conflicts; do not blindly overwrite user code
