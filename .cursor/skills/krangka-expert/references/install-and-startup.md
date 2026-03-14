# Install and Startup — Krangka Framework

How to install the Krangka CLI, create a project, configure it, and run HTTP server, subscriber, and workers.

---

## Prerequisites

- **Go 1.24.2 or higher**
- **Docker**
- **Make** (optional, for Makefile commands)

---

## Installation

Install Krangka CLI:

```bash
go install github.com/redhajuanda/krangka/cli/krangka@latest
```

---

## Create New Project

```bash
# krangka new [package_name] [directory_name]
krangka new github.com/backoffice/project-x.git project-x
```

---

## Configuration

1. **Update modules**
   ```bash
   go mod tidy
   ```

2. **Copy config**
   ```bash
   make cfg
   ```
   or
   ```bash
   cp configs/files/example.yaml configs/files/default.yaml
   ```

3. **Start Docker (app + deps)**
   ```bash
   make docker-up
   ```

4. **Start Kafka** (if needed)
   ```bash
   make docker-up-kafka
   ```

5. **Run migrations**
   ```bash
   make migrate-up repo=mariadb
   ```
   or
   ```bash
   go run main.go migrate up mariadb
   ```

---

## Kickstart Application

### HTTP server

```bash
make http
```
or
```bash
go run main.go http
```

- Swagger: `http://localhost:8000/docs/index.html`
- Generate swagger: `make swag`

### Subscriber

```bash
make subscriber
```
or
```bash
go run main.go subscriber
```

### Worker

```bash
make worker name=[worker-name]
```
or
```bash
go run main.go worker [worker-name]
```

---

## Summary (order of operations)

1. Install CLI → create project → `go mod tidy`
2. Copy config → `make docker-up` (and `make docker-up-kafka` if needed)
3. `make migrate-up repo=mariadb`
4. Run: `make http` / `make subscriber` / `make worker name=...`
