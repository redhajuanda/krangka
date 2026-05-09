---
inclusion: always
---

# Krangka project rules

This project's canonical rules live in **`AGENTS.md`** at the repository root. Read that file for all coding conventions, architecture guidance, and the code-review-graph MCP usage policy.

## Skills

Reusable, model-invoked skills live under **`.agents/skills/`**. Each skill is its own directory containing a `SKILL.md` (YAML frontmatter + markdown body). Read the `SKILL.md` of the relevant skill before performing the corresponding task:

- **`.agents/skills/krangka-install/SKILL.md`** — installing the Krangka CLI, scaffolding a new project, initial setup.
- **`.agents/skills/krangka-upgrade/SKILL.md`** — tracking framework changes and upgrading a project to the latest krangka version.
- **`.agents/skills/krangka-query-review/SKILL.md`** — gathering Qwery `RunRaw`/`Run` queries and generating a review document.
- **`.agents/skills/krangka-expert/SKILL.md`** — expert guide covering hexagonal architecture, TDD, error handling, testing patterns, repositories, workers, and dependency wiring. Use this for any general Krangka task that isn't covered by a more specific skill.

When a user request matches one of these skills, open the matching `SKILL.md` and follow it. Some skills reference additional files under their own `references/` or `scripts/` subdirectories — read those when the `SKILL.md` points to them.
