# Krangka Install and Startup — Reference

Quick reference for prerequisites, commands, and run options. The main flow is in [SKILL.md](SKILL.md).

## Prerequisites

- **Go 1.24.2 or higher**
- **Docker**
- **Make** (optional)

## Commands summary

| Step | Command |
|------|--------|
| Install CLI | `go install github.com/redhajuanda/krangka/cli/krangka@latest` |
| New project | `krangka new <package_name> <directory_name>` |
| Update modules | `go mod tidy` |
| Copy config | `make cfg` or `cp configs/files/example.yaml configs/files/default.yaml` |
| Docker up | `make docker-up` |
| Kafka (if needed) | `make docker-up-kafka`; then set `kafka.publisher.enabled: true` in `configs/files/default.yaml` |
| Migrate | `make migrate-up repo=mariadb` or `go run main.go migrate up mariadb` |
| HTTP | `make http` or `go run main.go http` |
| Subscriber | `make subscriber` or `go run main.go subscriber` |
| Worker | `make worker name=<name>` or `go run main.go worker <name>` |

## Config and defaults

- Config file: set via `--config`; default `configs/files/default.yaml`.
- Kafka publisher is disabled by default (`kafka.publisher.enabled: false` in example.yaml). When the user enables Kafka, set `kafka.publisher.enabled: true` in `configs/files/default.yaml` after `make docker-up-kafka`.
- Swagger: `http://localhost:8000/docs/index.html`; generate with `make swag`.

## Order of operations

1. Install CLI → create project → `go mod tidy`
2. Copy config → `make docker-up` (and `make docker-up-kafka` if needed)
3. `make migrate-up repo=mariadb`
4. Run: `make http` / `make subscriber` / `make worker name=...`
