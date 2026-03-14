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
├── subscriber.go                 # Event subscriber command
└── worker.go                     # Background worker command
```

### Command Files

- **`http.go`**: Starts the HTTP server with Fiber framework using bootstrap pattern
- **`migrate.go`**: Handles database migrations (up, down, new) using bootstrap pattern
- **`root.go`**: Main CLI command with subcommands and configuration flag
- **`subscriber.go`**: Starts the event subscriber (Watermill) for Kafka/Redis Streams using bootstrap pattern
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
└── files/                        # Configuration files
    ├── default.yaml              # Default configuration
    └── example.yaml              # Example configuration template
```

### Configuration Files

- **`config.go`**: Defines configuration structs and validation
- **`env.go`**: Handles environment variable loading
- **`files/`**: YAML configuration files; `example.yaml` serves as the template for environment-specific configs

## CLI Tooling (`cli/`)

Contains standalone CLI tools for development productivity.

```
cli/
└── krangka/                        # Krangka CLI tool (separate Go module)
    ├── generator/                # Code generator
    │   └── generator.go          # Generator implementation
    ├── gonew/                    # Project scaffolding tool
    │   └── gonew.go              # Scaffolding logic
    ├── openspec/                 # OpenSpec for CLI-specific changes (optional)
    │   └── config.yaml           # CLI-focused context and rules
    ├── utils/                    # CLI utility functions
    │   └── stringx/              # String utilities
    ├── root.go                   # CLI root command
    ├── go.mod                    # Separate module definition
    └── Makefile                  # CLI build commands
```

The `cli/krangka` tool is a separate Go module providing developer tooling such as code generation and project scaffolding. It can use its own `openspec/` directory for CLI-specific proposals, specs, and tasks (e.g. new commands, template changes).

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
├── config.yaml                   # OpenSpec configuration (main app)
└── changes/                      # Active and archived changes
```

The **main app** uses this root `openspec/`. The **CLI** can use a separate OpenSpec at `cli/krangka/openspec/` (with its own `config.yaml` and optional `changes/`) for CLI-only workflows.

## Internal Application Code (`internal/`)

The core application code following hexagonal architecture.

```
internal/
├── core/                         # Core business logic
│   ├── domain/                   # Domain entities and models
│   ├── port/                     # Port interfaces
│   │   ├── inbound/              # Inbound port interfaces
│   │   └── outbound/             # Outbound port interfaces
│   │       └── repositories/     # Repository sub-interfaces
│   └── service/                  # Application services (use cases)
├── mocks/                        # Generated mocks (run `make mock`)
│   ├── inbound/                  # Mocks for inbound ports
│   └── outbound/                 # Mocks for outbound ports
│       ├── mock_cache.go         # Cache mock
│       ├── mock_dlock.go         # Distributed lock mock
│       ├── mock_idempotency.go   # Idempotency mock
│       ├── mock_repository.go    # Repository mock
│       └── repositories/        # Mocks for repository ports
│           └── mock_note_repository.go
└── adapter/                      # Adapters (Ports & Adapters)
    ├── inbound/                  # Inbound adapters (driving)
    │   ├── http/                 # HTTP API adapter
    │   ├── migrate/              # Migration adapter
    │   ├── subscriber/           # Event subscriber adapter
    │   └── worker/               # Background worker adapter
    └── outbound/                 # Outbound adapters (driven)
        ├── dlock/                # Distributed lock adapter
        ├── idempotency/          # Idempotency adapter
        ├── kafka/                # Kafka messaging adapter
        ├── mariadb/              # MariaDB database adapter
        │   └── repositories/     # Repository implementations
        ├── redis/                # Redis cache adapter
        └── redisstream/          # Redis Streams messaging adapter
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
│   │   └── note.go               # Note DTOs
│   └── note.go                   # Note HTTP handlers
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

#### Subscriber Adapter (`internal/adapter/inbound/subscriber/`)

Event-driven entry point using Watermill for Kafka and Redis Streams.

```
subscriber/
├── handler/                      # Event handlers
│   └── note.go                  # Note event handler
├── middleware/                  # Subscriber middleware
│   ├── errors.go                # Error handling middleware (SkipRetryError)
│   ├── idempotence.go           # Idempotency middleware
│   ├── request_id.go            # Request ID propagation middleware
│   └── retry.go                 # Retry middleware with configurable backoff
├── router.go                    # Event route definitions
└── subscriber.go                # Subscriber setup and lifecycle
```

#### Worker Adapter (`internal/adapter/inbound/worker/`)

Time-triggered or manual background jobs (Execute, Schedule, Run).

```
worker/
├── worker_generate_id.go        # Generate ID worker
└── worker_outbox_relay.go       # Outbox relay worker implementation
```

### Outbound Adapters (`internal/adapter/outbound/`)

External systems that the application depends on.

#### MariaDB Adapter (`internal/adapter/outbound/mariadb/`)

```
mariadb/
├── conn.go                       # Database connection (Sikat client)
├── migrator.go                   # Migrator implementation
├── migrations/                   # Database migrations
│   └── scripts/                  # Migration SQL files
│       ├── 20250126141000-ddl_outboxes.sql
│       └── 20250126144000-table_notes.sql
├── outbox.go                     # Outbox repository implementation
├── queries/                      # External SQL query files (if any)
├── repositories/                 # Repository implementations
│   ├── dto/                      # Internal DTOs for database mapping
│   │   └── note.go               # Note database DTOs
│   └── note.go                   # Note repository
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

#### Idempotency Adapter (`internal/adapter/outbound/idempotency/`)

```
idempotency/
└── conn.go                       # Idempotency store connection (Redis-backed)
```

Used by the subscriber to prevent duplicate event processing.

### Core Business Logic (`internal/core/`)

The heart of the application containing business logic and domain rules.

#### Domain Layer (`internal/core/domain/`)

```
domain/
└── note.go                       # Note domain entity
```

**Domain Entities**: Pure business objects with no external dependencies.

#### Port Interfaces (`internal/core/port/`)

```
port/
├── inbound/                      # Inbound port interfaces
│   └── note.go                   # Note service interface
└── outbound/                     # Outbound port interfaces
    ├── cache.go                  # Cache interface
    ├── dlock.go                  # Distributed lock interface
    ├── idempotency.go            # Idempotency interface
    ├── publisher.go              # Message publisher interface (Watermill)
    ├── repositories/             # Repository interfaces
    │   └── note.go               # Note repository interface
    ├── repository.go             # Main repository interface (includes outbox)
    └── subscriber.go             # Message subscriber interface (Watermill)
```

**Port Interfaces**: Define contracts between the core and external systems.

#### Service Layer (`internal/core/service/`)

```
service/
└── note/                         # Note service
    └── service.go                # Note business logic
```

**Services**: Implement business logic and orchestrate domain operations.

## Shared Utilities (`shared/`)

Common utilities used across the application.

```
shared/
├── failure/                      # Application-level error definitions
│   └── failure.go                # Typed failure constants (uses silib/fail)
├── libctx/                       # Context utilities
│   ├── libctx.go                 # Context helpers (JWT claims, account, etc.)
│   └── role.go                   # Role definitions
└── utils/                        # Utility functions
    └── debug.go                  # Debug utilities
```

### Shared Components

- **`failure/`**: Centralized typed error definitions using `silib/fail.Failure`. All domain-level errors (e.g., `ErrNoteNotFound`, `ErrNoteAlreadyExists`) are declared here with HTTP status codes and error codes
- **`libctx/`**: Context utilities for JWT claims, bearer tokens, account information, and role definitions
- **`utils/`**: General utility functions

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
- `make subscriber`: Start event subscriber
- `make worker`: Start background worker (e.g. `make worker name=cleaning-todo`)
- `make migrate-up`: Run database migrations
- `make migrate-down`: Rollback migrations
- `make migrate-new`: Create new migration
- `make swag`: Generate Swagger documentation
- `make test`: Run tests
- `make build`: Build application
- `make mock`: Generate mocks
- `make docker-up` / `make docker-down`: Docker Compose for development

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
- **Domain entities**: `entity.go` (e.g., `note.go`, `user.go`)
- **Services**: `service.go` (e.g., `note/service.go`)
- **Repositories**: `repository.go` (e.g., `note/repository.go`)
- **Handlers**: `handler.go` (e.g., `note/handler.go`)
- **DTOs**: `dto.go` (e.g., `note/dto.go`)

### SQL Files
- **Migrations**: `YYYYMMDDHHMMSS-description.sql` (stored in `internal/adapter/outbound/mariadb/migrations/scripts/`)
- **Queries**: Inline SQL queries written directly in repository methods using `RunRaw()` with Sikat template syntax (`{{ .field }}`)

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
- Use `shared/failure` for typed application errors (via `silib/fail`)
- Wrap errors at every layer to preserve stack traces
- Consistent error propagation using the `fail` package