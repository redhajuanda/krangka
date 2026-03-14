---
name: krangka-install
description: "Guides the user through installing the Krangka CLI, creating a new project, and starting it step by step. Interactive: asks for project/directory name, package (GitLab/GitHub), and other options when not specified. Use when the user wants to install krangka, create a new krangka project, init/startup a krangka app, or set up krangka from scratch."
---

# Krangka Install and Startup

Guides install, project creation, configuration, and startup of a new Krangka project. **Be interactive**: ask for any value the user did not specify before running commands.

---

## Phase 1: Gather inputs (interactive)

Before running any install or project-creation commands, **resolve these values**. If the user did not provide one, **ask** (conversationally or via a single clear question):

| Input | Meaning | Example | When to ask |
|-------|---------|---------|-------------|
| **Package name** | Go module / Git remote path (GitLab or GitHub) | `github.com/backoffice/project-x` or `github.com/org/repo` | Not specified |
| **Project location** | Whether to initialize in the **current directory** or create a **new directory** | “current directory” or “new directory” | Always ask if not specified |
| **Directory name** | Local folder name — only needed when creating a new directory | `project-x` | Only when project location is “new directory” |
| **Parent path** | Where to create the new directory (directory will be created inside it) | Current dir `.` or e.g. `~/dev` | Only when project location is “new directory” and the user wants a specific parent |
| **Need Kafka** | Whether the app will use Kafka (affects `make docker-up-kafka`) | Yes / No | After project exists or when discussing run steps |

**Project location clarification:**
- **Current directory**: runs `krangka new <package> .` inside the current directory. The current directory must not contain any files that conflict with the boilerplate — if it does, `krangka new` will error and list the conflicts.
- **New directory**: runs `krangka new <package> <directory_name>` and creates a new folder. The directory must not already exist and be non-empty.

Confirm with the user before running `krangka new`. Examples:
- Current dir: “I’ll run: `krangka new <package> .` in the current directory — proceed?”
- New dir: “I’ll run: `krangka new <package> <directory>` in `<parent>` — proceed?”

---

## Phase 2: Prerequisites

Ensure the user has:

- **Go 1.24.2+**
- **Docker**
- **Make** (optional; used in steps below)

If any is missing, say what to install and pause before continuing.

---

## Phase 3: Install Krangka CLI

```bash
go install github.com/redhajuanda/krangka/cli/krangka@latest
```

Verify: `krangka --help` (or `krangka -h`).

---

## Phase 4: Create new project

**Option A — Initialize in current directory:**

```bash
# Run from inside the target directory
krangka new <package_name> .
go mod tidy
```

Example:
```bash
krangka new github.com/backoffice/project-x .
go mod tidy
```

> If any boilerplate files already exist in the current directory, `krangka new` will error and list the conflicts. Resolve them before retrying.

**Option B — Create a new directory:**

From the **parent path** (e.g. current directory or the path the user chose):

```bash
krangka new <package_name> <directory_name>
cd <directory_name>
go mod tidy
```

Example:
```bash
krangka new github.com/backoffice/project-x project-x
cd project-x
go mod tidy
```

---

## Phase 5: Configuration

1. **Copy config**
   ```bash
   make cfg
   ```
   or if no Makefile:
   ```bash
   cp configs/files/example.yaml configs/files/default.yaml
   ```

2. **Start Docker (app + deps)**
   ```bash
   make docker-up
   ```

3. **Start Kafka** (only if user said they need Kafka)
   ```bash
   make docker-up-kafka
   ```
   Then **enable the Kafka publisher** in the active config (`configs/files/default.yaml`): set `kafka.publisher.enabled: true`. (The template `example.yaml` keeps `enabled: false` by default; the running config should enable it when Kafka is used.)

4. **Run migrations**
   ```bash
   make migrate-up repo=mariadb
   ```
   or:
   ```bash
   go run main.go migrate up mariadb
   ```

---

## Phase 6: Start the application

Choose what to run and tell the user how:

- **HTTP server**
  ```bash
  make http
  ```
  or `go run main.go http`  
  Swagger: `http://localhost:8000/docs/index.html` — generate with `make swag` if needed.

- **Subscriber**
  ```bash
  make subscriber
  ```
  or `go run main.go subscriber`

- **Worker**
  ```bash
  make worker name=<worker-name>
  ```
  or `go run main.go worker <worker-name>`

Ask which of these they want to run first (e.g. “Do you want to start the HTTP server, subscriber, or a worker?”).

---

## Summary checklist

Use this to track progress and report to the user:

1. [ ] Inputs gathered (package, project location; if new dir: directory name and parent path; Kafka yes/no)
2. [ ] Prerequisites checked (Go, Docker, Make)
3. [ ] CLI installed and verified
4. [ ] Project created with `krangka new`; `go mod tidy` (+ `cd <dir>` if new directory)
5. [ ] Config copied; Docker up (and Kafka if needed); if Kafka used: set `kafka.publisher.enabled: true` in `configs/files/default.yaml`
6. [ ] Migrations run
7. [ ] User started HTTP / subscriber / worker

---

## Reference

For full details (prereqs, config paths, migration options), see [reference.md](references/reference.md).
