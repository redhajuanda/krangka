---
name: /krangka-upgrade
id: krangka-upgrade
category: Workflow
description: Upgrade krangka framework from current version to latest using krangka-upgrader skill
---

Upgrade the krangka framework (boilerplate) to the latest version by reading version reference files and applying migrations in order.

**Skill**: Use the **krangka-upgrader** skill for the full workflow. This command invokes that workflow.

**Input**: None required. Optionally specify a target version (e.g., `/krangka-upgrade v1.0.5`) to upgrade only up to that version instead of latest.

**Steps**

1. **Read the krangka-upgrader skill**

   Load `.cursor/skills/krangka-upgrader/SKILL.md` and follow its Upgrade Workflow.

2. **Determine current version**

   - Read `.krangka/.VERSION` — contains the krangka framework version (e.g. `v1.0.1`)
   - If the file does not exist, assume **v1.0.0**

3. **Discover available versions**

   - List files in `.cursor/skills/krangka-upgrader/references/`
   - Parse version numbers from filenames (e.g. `v1.0.1.md` → `1.0.1`)
   - Sort semantically and identify the **latest** version
   - If user specified a target version, use that as the upper bound instead of latest

4. **Build upgrade path**

   - Versions to apply = all versions **strictly greater** than current, up to and including target (latest or user-specified)
   - If current ≥ target: report "Already at or past target version" and stop

5. **Apply each version (in order)**

   For each version in the upgrade path:
   - Read `.cursor/skills/krangka-upgrader/references/vX.Y.Z.md`
   - Apply Added, Removed, Modified, Dependencies as described
   - Follow migration notes
   - Update `.krangka/.VERSION` after successfully applying
   - Run `go mod tidy` after dependency changes

6. **Finalize**

   - Ensure `.krangka/.VERSION` reflects the final applied version
   - Run any remaining commands from migration notes (e.g. `go build ./...`)

**Output During Upgrade**

```
## krangka Upgrade: v1.0.1 → v1.0.5

Current: v1.0.1
Target: v1.0.5
Upgrade path: v1.0.2, v1.0.3, v1.0.4, v1.0.5

### Applying v1.0.2
[...changes...]
✓ v1.0.2 applied

### Applying v1.0.3
...
```

**Output On Completion**

```
## Upgrade Complete

**From:** v1.0.1
**To:** v1.0.5

All migrations applied. Run `go build ./...` to verify.
```

**Output When Already Up-to-Date**

```
## Already Up-to-Date

Current: v1.0.5
Latest: v1.0.5
No upgrade needed.
```

**Guardrails**

- Apply versions in order — do not skip
- If a reference file is missing for a version in the path, stop and report the gap
- Do not blindly overwrite user code; adapt if project has diverged
- Always run `go mod tidy` after dependency changes
