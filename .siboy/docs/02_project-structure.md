# Project Structure

This document provides a detailed breakdown of the Krangka project structure, explaining the purpose and organization of each directory and file.

## Root Directory Overview

```
krangka/
├── build/                        # Build output artifacts
├── cli/                          # CLI tooling (code generator, scaffolding)
├── cmd/                          # Application entry points
├── configs/                      # Configuration management
├── deployment/                   # Deployment configuration files
├── internal/                     # Internal application code
├── openspec/                     # OpenSpec change management workflow
├── scripts/                      # Build and utility scripts
├── shared/                       # Shared utilities
├── main.go                       # Application entry point
├── go.mod                        # Go module definition
├── go.sum                        # Go module checksums
├── Makefile                      # Build and run commands
└── README.md                     # Project documentation
```

## Application Entry Points (`cmd/`)

Contains the main entry points for different application modes.

```
cmd/
├── bootstrap/                    # Bootstrap system for dependency injection
│   ├── bootstrap.go             # Bootstrap lifecycle management
│   ├── dependency.go             # Dependency resolution and wiring
│   └── interface.go              # Resource type definitions
├── http.go                       # HTTP server command
├── migrate.go                    # Database migration command
├── root.go                       # CLI root command
└── worker.go                     # Background worker command
```

### Command Files

- **`http.go`**: Starts the HTTP server with Fiber framework using bootstrap pattern
- **`migrate.go`**: Handles database migrations (up, down, new) using bootstrap pattern
- **`root.go`**: Main CLI command with subcommands and configuration flag
- **`worker.go`**: Starts background workers for job processing using bootstrap pattern

### Bootstrap System (`cmd/bootstrap/`)

The bootstrap system manages dependency injection and application lifecycle:

- **`bootstrap.go`**: Manages application lifecycle (start, stop, cleanup), signal handling, and resource cleanup
- **`dependency.go`**: Defines `Dependency` struct with all application dependencies and their getter methods
- **`interface.go`**: Defines resource types (`Resource`, `ResourceRunnable`, `ResourceExecutable`, `ResourceClosable`) and interfaces (`Runnable`, `Executable`, `Closable`)

## Configuration Management (`configs/`)

Manages application configuration across different environments.

```
configs/
├── config.go                     # Configuration structs
├── env.go                        # Environment handling
├── files/                        # Configuration files
│   ├── default.yaml              # Default configuration
│   └── example.yaml              # Example configuration template
└── version.go                    # Version information
```

### Configuration Files

- **`config.go`**: Defines configuration structs and validation
- **`env.go`**: Handles environment variable loading
- **`files/`**: YAML configuration files; `example.yaml` serves as the template for environment-specific configs
- **`version.go`**: Application version information

## CLI Tooling (`cli/`)

Contains standalone CLI tools for development productivity.

```
cli/
└── krangka/                        # Krangka CLI tool (separate Go module)
    ├── generator/                # Code generator
    │   └── generator.go          # Generator implementation
    ├── gonew/                    # Project scaffolding tool
    │   └── gonew.go              # Scaffolding logic
    ├── utils/                    # CLI utility functions
    │   └── stringx/              # String utilities
    ├── root.go                   # CLI root command
    ├── go.mod                    # Separate module definition
    └── Makefile                  # CLI build commands
```

The `cli/krangka` tool is a separate Go module providing developer tooling such as code generation and project scaffolding.

## Deployment (`deployment/`)

Deployment-related configuration files.

```
deployment/
├── development_main.yaml         # Development environment config
└── development_kafka.yaml        # Development Kafka config
```

## OpenSpec (`openspec/`)

Manages structured change workflows used by Cursor AI agent skills.

```
openspec/
├── config.yaml                   # OpenSpec configuration
└── changes/                      # Active and archived changes
```

## Internal Application Code (`internal/`)

The core application code following hexagonal architecture.

```
internal/
├── adapter/                      # Adapters (Ports & Adapters)
│   ├── inbound/                  # Inbound adapters (driving)
│   │   ├── http/                 # HTTP API adapter
│   │   ├── migrate/              # Migration adapter
│   │   └── worker/               # Background worker adapter
│   └── outbound/                 # Outbound adapters (driven)
│       ├── dlock/                # Distributed lock adapter
│       ├── kafka/                # Kafka messaging adapter
│       ├── mariadb/              # MariaDB database adapter
│       ├── redis/                # Redis cache adapter
│       └── redisstream/          # Redis Streams messaging adapter
└── core/                         # Core business logic
    ├── domain/                   # Domain entities and models
    ├── port/                     # Port interfaces
    │   ├── inbound/              # Inbound port interfaces
    │   └── outbound/             # Outbound port interfaces
    └── service/                  # Application services (use cases)
```

### Inbound Adapters (`internal/adapter/inbound/`)

Entry points to the application that drive business logic.

#### HTTP Adapter (`internal/adapter/inbound/http/`)

```
http/
├── docs/                         # API documentation
│   ├── docs.go                   # Swagger documentation setup
│   ├── swagger.json              # OpenAPI specification
│   └── swagger.yaml              # OpenAPI specification (YAML)
├── error.go                      # HTTP error handling
├── handler/                      # HTTP request handlers
│   ├── dto/                      # Data Transfer Objects
│   │   ├── note.go               # Note DTOs
│   │   └── todo.go               # Todo DTOs
│   ├── note.go                   # Note HTTP handlers
│   └── todo.go                   # Todo HTTP handlers
├── http.go                       # HTTP server setup
├── middleware/                   # HTTP middleware
│   ├── auth.go                   # Authentication middleware
│   ├── recover.go                # Panic recovery middleware
│   ├── request_id.go             # Request ID middleware
│   └── security_header.go        # Security headers middleware
├── response/                     # HTTP response utilities
│   ├── failed.go                 # Error response helpers
│   ├── metadata.go               # Response metadata
│   └── success.go                # Success response helpers
└── router.go                     # Route definitions
```

#### Migration Adapter (`internal/adapter/inbound/migrate/`)

```
migrate/
├── generate.go                   # Migration generation
├── migrate.go                    # Migration execution
└── migrator.go                   # Migration logic
```

#### Worker Adapter (`internal/adapter/inbound/worker/`)

```
worker/
├── middleware/                   # Worker middleware
│   ├── errors.go                 # Error handling middleware (SkipRetryError)
│   ├── request_id.go             # Request ID propagation middleware
│   └── retry.go                  # Retry middleware with configurable backoff
├── worker_generate_id.go         # Generate ID worker
├── worker_outbox_relay.go        # Outbox relay worker implementation
└── worker_subscriber_note.go     # Note subscriber worker
```

### Outbound Adapters (`internal/adapter/outbound/`)

External systems that the application depends on.

#### MariaDB Adapter (`internal/adapter/outbound/mariadb/`)

```
mariadb/
├── conn.go                       # Database connection (Qwery client)
├── migrator.go                   # Migrator implementation
├── migrations/                   # Database migrations
│   └── scripts/                  # Migration SQL files
│       ├── 20250126141000-ddl_outboxes.sql
│       ├── 20250126144000-table_notes.sql
│       └── 20250126144001-table_todos.sql
├── outbox.go                     # Outbox repository implementation
├── queries/                      # External SQL query files (if any)
├── repositories/                 # Repository implementations
│   ├── dto/                      # Internal DTOs for database mapping
│   │   └── note.go               # Note database DTOs
│   ├── note.go                   # Note repository
│   └── todo.go                   # Todo repository
└── repository.go                 # Main repository factory
```

#### Redis Adapter (`internal/adapter/outbound/redis/`)

```
redis/
└── conn.go                       # Redis connection (cache)
```

#### Redis Streams Adapter (`internal/adapter/outbound/redisstream/`)

```
redisstream/
└── conn.go                       # Redis Streams publisher/subscriber connection
```

Used for event-driven messaging via Redis Streams (Watermill).

#### Kafka Adapter (`internal/adapter/outbound/kafka/`)

```
kafka/
└── conn.go                       # Kafka publisher/subscriber connection
```

Used for event-driven messaging via Kafka (Watermill).

#### Distributed Lock Adapter (`internal/adapter/outbound/dlock/`)

```
dlock/
└── conn.go                       # Distributed lock connection (Redis-backed)
```

### Core Business Logic (`internal/core/`)

The heart of the application containing business logic and domain rules.

#### Domain Layer (`internal/core/domain/`)

```
domain/
├── note.go                       # Note domain entity
└── todo.go                       # Todo domain entity
```

**Domain Entities**: Pure business objects with no external dependencies.

#### Port Interfaces (`internal/core/port/`)

```
port/
├── inbound/                      # Inbound port interfaces
│   ├── note.go                   # Note service interface
│   ├── outbox.go                 # Outbox service interface
│   └── todo.go                   # Todo service interface
└── outbound/                     # Outbound port interfaces
    ├── cache.go                  # Cache interface
    ├── dlock.go                  # Distributed lock interface
    ├── publisher.go              # Message publisher interface (Watermill)
    ├── repositories/             # Repository interfaces
    │   ├── note.go               # Note repository interface
    │   └── todo.go               # Todo repository interface
    ├── repository.go             # Main repository interface
    └── subscriber.go             # Message subscriber interface (Watermill)
```

**Port Interfaces**: Define contracts between the core and external systems.

#### Service Layer (`internal/core/service/`)

```
service/
├── note/                         # Note service
│   └── service.go                # Note business logic
└── todo/                         # Todo service
    └── service.go                # Todo business logic
```

**Services**: Implement business logic and orchestrate domain operations.

## Shared Utilities (`shared/`)

Common utilities used across the application.

```
shared/
├── failure/                      # Application-level error definitions
│   └── failure.go                # Typed failure constants (uses komon/fail)
├── libctx/                       # Context utilities
│   └── libctx.go                 # Context helpers (JWT claims, account, etc.)
└── utils/                        # Utility functions
    └── debug.go                  # Debug utilities
```

### Shared Components

- **`failure/`**: Centralized typed error definitions using `komon/fail.Failure`. All domain-level errors (e.g., `ErrTodoNotFound`, `ErrNoteAlreadyExists`) are declared here with HTTP status codes and error codes
- **`libctx/`**: Context utilities for JWT claims, bearer tokens, and account information
- **`utils/`**: General utility functions

## Scripts (`scripts/`)

Build and utility scripts.

```
scripts/
└── bash/                         # Bash scripts
    └── version.sh                # Version management script
```

## Root Files

### `main.go`
The main application entry point that:
- Calls `cmd.Run()` to execute the CLI root command
- All dependency injection and lifecycle management is handled by the bootstrap system in `cmd/bootstrap/`

### `go.mod`
Go module definition containing:
- Module name and version
- Go version requirement
- External dependencies

### `go.sum`
Go module checksums for dependency verification.

### `Makefile`
Build and development commands:
- `make http`: Start HTTP server
- `make worker`: Start background worker
- `make migrate-up`: Run database migrations
- `make migrate-down`: Rollback migrations
- `make test`: Run tests
- `make build`: Build application

## Key Architectural Principles

### 1. Separation of Concerns
- **Domain**: Pure business logic
- **Ports**: Interface definitions
- **Adapters**: External system implementations

### 2. Dependency Direction
```
External Systems → Inbound Adapters → Inbound Ports → Services → Outbound Ports → Outbound Adapters → External Systems
```

### 3. Bootstrap Organization
- Dependencies are centralized in `cmd/bootstrap/dependency.go`
- Clear resource types for different lifecycle needs
- Automatic resource cleanup for closable resources
- Easy to test and maintain

### 4. Configuration Management
- Environment-based configuration
- YAML files for different environments
- Environment variable support

## File Naming Conventions

### Go Files
- **Domain entities**: `entity.go` (e.g., `todo.go`, `user.go`)
- **Services**: `service.go` (e.g., `todo/service.go`)
- **Repositories**: `repository.go` (e.g., `todo/repository.go`)
- **Handlers**: `handler.go` (e.g., `todo/handler.go`)
- **DTOs**: `dto.go` (e.g., `todo/dto.go`)

### SQL Files
- **Migrations**: `YYYYMMDDHHMMSS-description.sql` (stored in `internal/adapter/outbound/mariadb/migrations/scripts/`)
- **Queries**: Inline SQL queries written directly in repository methods using `RunRaw()` with Qwery template syntax (`{{ .field }}`)

### Configuration Files
- **Environment-specific**: `environment.yaml` (e.g., `development_main.yaml`)
- **Example**: `example.yaml`

## Best Practices

### 1. Package Organization
- Keep related functionality together
- Use clear, descriptive package names
- Follow Go package naming conventions

### 2. File Structure
- One main type per file
- Related functionality in the same directory
- Clear separation between layers

### 3. Import Organization
- Standard library imports first
- Third-party imports second
- Internal imports last

### 4. Error Handling
- Use `shared/failure` for typed application errors (via `komon/fail`)
- Wrap errors at every layer to preserve stack traces
- Consistent error propagation using the `fail` package
