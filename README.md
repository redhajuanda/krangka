# Krangka - Hexagonal Architecture Go Application

[![Go Version](https://img.shields.io/badge/Go-1.24.2-blue.svg)](https://golang.org/)

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

### Create New Project

Create new project using Krangka CLI:
```bash
# krangka new [package_name] [directory_name]
krangka new gitlab.sicepat.tech/backoffice/project-x.git project-x
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

Swagger documentation is available at `http://localhost:8080/docs/index.html`
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

## 📚 Documentation

For comprehensive documentation and in-depth guides, please consult the [docs](./.krangka/docs/README.md) directory.