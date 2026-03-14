---
name: /krangka-install
id: krangka-install
category: Workflow
description: Install Krangka CLI, create a new project, and run initial setup using krangka-install skill (interactive)
---

Install the Krangka CLI, create a new project, configure it, and start the app. This command runs the **krangka-install** skill.

**Skill**: Use the **krangka-install** skill for the full workflow. This command invokes that workflow.

**Input**: Optional. User may specify:
- Package name (e.g. `github.com/backoffice/myapp` or `github.com/org/repo`)
- Project location: **current directory** (`.`) or **new directory**
- Directory name (e.g. `myapp`) — only needed when creating a new directory
- Parent path (where to create the new directory)
- Whether they need Kafka

If any of these are **not** specified, **ask** the user before running install or project-creation commands. Be interactive.

**Steps**

1. **Read the krangka-install skill**

   Load `.cursor/skills/krangka-install/SKILL.md` and follow its phases.

2. **Phase 1 — Gather inputs**

   Resolve: package name, **project location (current dir or new dir)**, directory name (if new dir), parent path, need Kafka. Ask for each value the user did not provide. Confirm before running `krangka new`.

3. **Phase 2–6**

   Follow the skill: prerequisites → install CLI → create project → config (copy config, docker up, optional Kafka + enable `kafka.publisher.enabled`, migrate) → start HTTP / subscriber / worker as the user chooses.

**Output**

- Report which step you are on.
- After each phase, briefly confirm what was done (e.g. “CLI installed”, “Project created in `./myapp`”, “Config copied; Docker up”).
- When Kafka is enabled, confirm that `kafka.publisher.enabled: true` was set in `configs/files/default.yaml`.

**Guardrails**

- Do not run `krangka new` or install commands until required inputs (package name and project location) are known — ask first.
- When project location is current directory, run `krangka new <package> .`; no `cd` needed after.
- When project location is new directory, run `krangka new <package> <dir>` then `cd <dir>`.
- When the user enables Kafka, run `make docker-up-kafka` and set `kafka.publisher.enabled: true` in `configs/files/default.yaml`.
