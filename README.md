# Krangka - Hexagonal Architecture Go Application

[![Go Version](https://img.shields.io/badge/Go-1.25.0-blue.svg)](https://golang.org/)

Krangka is a Go boilerplate designed to kickstart Go applications using the **Hexagonal Architecture** (Ports and Adapters pattern). It provides a clean separation of concerns, making your business logic independent from external dependencies, and helps ensure your codebase is testable, maintainable, and easy to extend.

## Quick Start

Get up and running with Krangka in minutes! This guide will walk you through the essential steps to create and run your first Krangka application.

### Prerequisites

Before you begin, ensure you have the following installed:

- **Go 1.24.2 or higher**
- **Docker**
- **Make** (optional, for using Makefile commands)

### Installation

First, install Krangka CLI:
```bash
go install github.com/redhajuanda/krangka/cli/krangka@latest
```

### Upgrade CLI

Upgrade Krangka CLI to the latest version:
```bash
krangka upgrade
```

Check the latest available version without installing:
```bash
krangka upgrade --check
```

### Create New Project

Create new project using Krangka CLI:
```bash
# krangka new [package_name] [directory_name]
krangka new github.com/redhajuanda/project-x project-x
```

### Configuration

Run `go mod tidy` to update go.mod and go.sum:
```bash
go mod tidy
```

Copy example configuration file:
```bash 
make cfg 
```
or
```bash
cp configs/files/example.yaml configs/files/default.yaml
```

Run docker compose up:
```bash
make docker-up
```

Run docker compose for kafka:
```bash
make docker-up-kafka
```

Run database migration:
```bash
make migrate-up repo=mariadb
```
or
```bash
go run main.go migrate up mariadb
```

### Kickstart Application

Run HTTP server:
```bash
make http
```
or
```bash
go run main.go http
```

Swagger documentation is available at `http://localhost:8000/docs/index.html`
Run `make swag` to generate swagger documentation.

Run Subscriber:
```bash
make subscriber
```
or
```bash
go run main.go subscriber
```

Run Worker:
```bash
make worker name=[worker-name]
```
or
```bash
go run main.go worker [worker-name]
```

## Tooling

Krangka projects ship with two tools wired in by default to make AI-assisted development effective and safe:

### code-review-graph

[code-review-graph](https://github.com/tirth8205/code-review-graph) is an MCP server that builds a persistent, incremental knowledge graph of your codebase using Tree-sitter. The boilerplate ships with MCP wiring already configured (`.mcp.json`, `.cursor/mcp.json`, `.opencode.json`).

**Why we use it:**
- **Token-efficient code exploration** — agents query a structural graph instead of grepping and reading whole files, which is faster and cheaper.
- **Structural context** — gives callers, dependents, imports, and test coverage that plain file scanning cannot.
- **Smarter code review** — `detect_changes` produces risk-scored analysis and `get_impact_radius` surfaces the blast radius of a change before it ships.

### OpenSpec

[OpenSpec](https://github.com/Fission-AI/OpenSpec) is a spec-driven change-management workflow. Proposals, deltas, and archives live alongside the code so every non-trivial change is documented and reviewable before implementation. The `opsx-propose`, `opsx-apply`, and `opsx-archive` skills are built on top of it.

**Why we use it:**
- **Alignment before implementation** — design and specs are agreed upon up front, reducing rework.
- **Auditable change history** — every change has a proposal, tasks, and an archive entry, making the project's evolution traceable.
- **Better AI collaboration** — agents work from explicit specs and task lists instead of inferring intent, which leads to more predictable results.

### RTK AI

[RTK AI](https://github.com/rtk-ai/rtk) is included as part of the Krangka AI workflow. This boilerplate already embeds local RTK integration, so setup focuses on installing the CLI and running global initialization (`rtk init -g`) without local project init.

**Why we use it:**
- **Consistent AI runtime setup** — global initialization provides a shared baseline across projects and developer machines.
- **Boilerplate-friendly onboarding** — Krangka ships local RTK integration already, reducing manual per-project setup.
- **Improved developer productivity** — RTK complements spec and graph workflows for day-to-day agent usage.

For installation and setup, the `krangka-install` skill walks through all tools step by step.

## 📚 Documentation

For comprehensive documentation and in-depth guides, please consult the [docs](./.krangka/docs/README.md) directory.