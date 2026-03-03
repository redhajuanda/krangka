# Architecture Overview

## What is Hexagonal Architecture?

Hexagonal Architecture (also known as Ports and Adapters pattern) is a software design pattern that creates a clear separation between the business logic (core) and external concerns (adapters). The core application is isolated from external dependencies through well-defined interfaces (ports), and external systems interact with the core through adapters.

### Core Principles

This project follows **Hexagonal Architecture** (Ports and Adapters pattern) with these key principles:

- **Dependencies point inward**: External layers depend on internal layers
- **Core is independent**: Domain and services have no external dependencies  
- **Interfaces define contracts**: Ports define what the core needs and provides
- **Adapters implement interfaces**: Concrete implementations are in the adapter layer

### Key Benefits

- **Testability**: Business logic can be tested independently of external dependencies
- **Flexibility**: Easy to swap implementations (e.g., different databases, HTTP frameworks)
- **Maintainability**: Clear boundaries make the codebase easier to understand and modify
- **Independence**: Core business logic is not tied to specific technologies

## Architecture Layers

```
                     ┌────────────┐   ┌────────────┐   ┌────────────┐
                     │  HTTP API  │   │  Workers   │   │     CLI    │
                     └─────┬──────┘   └─────┬──────┘   └─────┬──────┘
                           │                │                │
                           └────────────────┬────────────────┘
                                            │
                                            ▼
        ┌─────────────────────────────────────────────────────────────────────────┐
        │                              INBOUND PORT                               │
        │                         (Input Port Interfaces)                         │
        └─────────────────────────────────┬───────────────────────────────────────┘
                                          │
                                          ▼
        ┌─────────────────────────────────────────────────────────────────────────┐
        │                        APPLICATION CORE                                 │
        │                                                                         │
        │    ┌─────────────────────┐         ┌─────────────────────────────────┐  │
        │    │      DOMAIN         │         │           SERVICES              │  │
        │    │    (Entities)       │         │         (Use Cases)             │  │
        │    └─────────────────────┘         └─────────────────────────────────┘  │
        └─────────────────────────────────────────────────────────────────────────┘
                                          │
                                          ▼
        ┌─────────────────────────────────────────────────────────────────────────┐
        │                            OUTBOUND PORT                                │
        │                       (Output Port Interfaces)                          │
        └─────────────────────────────────┬───────────────────────────────────────┘
                                          │
                           ┌──────────────┼──────────────┐
                           │              │              │
                           ▼              ▼              ▼
                    ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
                    │   MariaDB   │ │ PostgreSQL  │ │    Redis    │
                    │  (Adapter)  │ │  (Adapter)  │ │  (Adapter)  │
                    └─────────────┘ └─────────────┘ └─────────────┘
```

## Layer Responsibilities

### 1. Inbound Adapters (Driving)
These are the entry points to your application that drive the business logic.

**Examples:**
- HTTP API (REST endpoints)
- Background Workers
- CLI Commands
- gRPC Services
- Message Queue Consumers

**Characteristics:**
- Implement inbound port interfaces
- Handle external requests and convert them to domain operations
- Manage request/response formatting
- Handle authentication and authorization

### 2. Application Core
The heart of your application containing business logic and domain rules.

**Domain Layer:**
- Pure business entities and value objects
- Business rules and invariants
- No external dependencies
- Framework-agnostic

**Service Layer:**
- Orchestrate domain operations
- Implement use cases (business logic)
- Coordinate between different domain entities
- Handle business workflows

### 3. Outbound Adapters (Driven)
These are the external systems your application depends on.

**Examples:**
- Database repositories
- Cache systems
- External APIs
- File systems
- Message queues

**Characteristics:**
- Implement outbound port interfaces
- Handle external system communication
- Manage data persistence and retrieval
- Handle external service integration

## Ports and Interfaces

### Inbound Ports
Define contracts for external systems that drive your application.

```go
// Example: Todo service interface
type Todo interface {
    GetTodoByID(ctx context.Context, id string) (*domain.Todo, error)
    CreateTodo(ctx context.Context, todo *domain.Todo) error
    UpdateTodo(ctx context.Context, todo *domain.Todo) error
    DeleteTodo(ctx context.Context, id string) error
    ListTodo(ctx context.Context, req *domain.TodoFilter, pagination *pagination.Pagination) (*[]domain.Todo, error)
}
```

### Outbound Ports
Define contracts for external systems your application depends on.

```go
// Example: Repository interface
type Repository interface {
    DoInTransaction(ctx context.Context, fn func(ctx context.Context) error) (any, error)
    GetTodoRepository() repositories.Todo
    GetNoteRepository() repositories.Note
}

// Example: Cache interface
type Cache cache.Cache
```

## Dependency Flow

### Dependency Direction
```
External Systems → Inbound Adapters → Inbound Ports → Services → Outbound Ports → Outbound Adapters → External Systems
```

### Key Principles
1. **Dependencies point inward**: External layers depend on internal layers
2. **Core is independent**: Domain and services have no external dependencies
3. **Interfaces define contracts**: Ports define what the core needs and provides
4. **Adapters implement interfaces**: Concrete implementations are in the adapter layer

## Technology Stack

### Core Technologies
- **Language**: Go 1.24.2+
- **Architecture**: Hexagonal Architecture (Ports & Adapters)
- **Dependency Injection**: Manual wiring through bootstrap pattern
- **HTTP Framework**: Fiber
- **Database**: MariaDB/PostgreSQL with Qwery ORM (inline SQL queries)
- **Cache**: Redis
- **Logging**: Structured logging with `github.com/redhajuanda/komon/logger`
- **Tracing**: Distributed tracing with `github.com/redhajuanda/komon/tracer`

### Supporting Technologies
- **CLI**: Cobra for command-line interface
- **Configuration**: YAML-based configuration with environment support
- **Documentation**: Swagger/OpenAPI for API documentation
- **Testing**: Go testing framework with testify
- **Migration**: Custom migration system

## Design Patterns Used

### 1. Repository Pattern
Abstracts data access logic and provides a collection-like interface.

```go
type TodoRepository interface {
    GetByID(ctx context.Context, id string) (*domain.Todo, error)
    Create(ctx context.Context, todo *domain.Todo) error
    Update(ctx context.Context, todo *domain.Todo) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context, filter *domain.TodoFilter) ([]*domain.Todo, error)
}
```

### 2. Service Layer Pattern
Encapsulates business logic and orchestrates domain operations.

```go
type TodoService struct {
    repo  Repository
    cache Cache
}

func (s *TodoService) GetTodoByID(ctx context.Context, id string) (*domain.Todo, error) {
    // Check cache first
    if cached, err := s.cache.Get(ctx, "todo:"+id); err == nil {
        return cached.(*domain.Todo), nil
    }
    
    // Get from repository
    todo, err := s.repo.GetTodoRepository().GetByID(ctx, id)
    if err != nil {
        return nil, err
    }
    
    // Cache the result
    s.cache.Set(ctx, "todo:"+id, todo, time.Hour)
    
    return todo, nil
}
```

### 3. DTO Pattern
Data Transfer Objects for API request/response handling.

```go
type CreateTodoRequest struct {
    Title       string `json:"title" validate:"required"`
    Description string `json:"description"`
}

type CreateTodoResponse struct {
    ID string `json:"id"`
}
```

### 4. Factory Pattern
Creates and configures complex objects.

```go
func NewTodoService(cfg *configs.Config, repo Repository, cache Cache) *TodoService {
    return &TodoService{
        cfg:   cfg,
        repo:  repo,
        cache: cache,
    }
}
```

## Benefits of This Architecture

### 1. Testability
- Business logic can be unit tested without external dependencies
- Easy to mock interfaces for testing
- Clear separation of concerns

### 2. Maintainability
- Changes to external systems don't affect core logic
- Clear boundaries between layers
- Easy to understand and modify

### 3. Flexibility
- Easy to swap implementations (e.g., different databases)
- Can add new adapters without changing core logic
- Technology-agnostic core

### 4. Scalability
- Can scale different layers independently
- Easy to add new features without affecting existing code
- Clear separation allows for microservice extraction

## Common Anti-Patterns to Avoid

### 1. Leaky Abstractions
❌ **Don't**: Expose database-specific details in domain entities
```go
type Todo struct {
    ID        string    `json:"id" db:"id"`
    Title     string    `json:"title" db:"title"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}
```

✅ **Do**: Keep domain entities pure
```go
type Todo struct {
    ID        string    `qwery:"id"`
    Title     string    `qwery:"title"`
    CreatedAt time.Time `qwery:"created_at"`
}
```

### 2. Business Logic in Adapters
❌ **Don't**: Put business rules in HTTP handlers
```go
func (h *TodoHandler) CreateTodo(c *fiber.Ctx) error {
    // Business logic here
    if todo.Title == "" {
        return errors.New("title cannot be empty")
    }
    // ...
}
```

✅ **Do**: Keep business logic in services
```go
func (s *TodoService) CreateTodo(ctx context.Context, todo *domain.Todo) error {
    if err := todo.Validate(); err != nil {
        return err
    }
    return s.repo.Create(ctx, todo)
}
```

### 3. Direct Dependencies
❌ **Don't**: Import concrete implementations in services
```go
import "github.com/gofiber/fiber/v2"
```

✅ **Do**: Depend on interfaces
```go
type Repository interface {
    // methods
}
```

## Dependency Wiring

Krangka uses **manual dependency injection** through a bootstrap pattern. Dependencies are explicitly wired in the `Dependency` struct (`cmd/bootstrap/dependency.go`) and resolved lazily through getter methods, making the dependency graph transparent and traceable.

### Bootstrap System

The bootstrap system consists of two main components:

1. **Dependency** (`cmd/bootstrap/dependency.go`): Manages dependency resolution and caching
2. **Bootstrap** (`cmd/bootstrap/bootstrap.go`): Manages application lifecycle (start, stop, cleanup)

### Resource Types

There are 4 types of resources in the bootstrap system:

| Type | Interface | Use Case | Examples |
|------|-----------|----------|----------|
| `Resource[T]` | None | Simple resources that don't need lifecycle management | Config, logger, services, repositories, handlers |
| `ResourceRunnable[T]` | `OnStart(ctx), OnStop(ctx)` | Long-running processes that need graceful shutdown | HTTP server, subscriber workers |
| `ResourceExecutable[T]` | `Execute(ctx)` | One-time or scheduled tasks | Migrate commands, cron workers |
| `ResourceClosable[T]` | `Close()` | Resources that need cleanup | DB connections, Redis clients, Kafka publishers |

### Dependency Pattern

Dependencies are defined in the `Dependency` struct and resolved through getter methods:

```go
// cmd/bootstrap/dependency.go
type Dependency struct {
    cfg          Resource[*configs.Config]
    log          Resource[logger.Logger]
    repository   Resource[outbound.Repository]
    serviceNote  Resource[*note.Service]
    httpHandlers Resource[[]http.Handler]
    
    qweryMain    ResourceClosable[*mariadb.Qwery]
    redis        ResourceClosable[*redis.Redis]
    http         ResourceRunnable[*http.HTTP]
    migrate      ResourceExecutable[*migrate.Migrate]
}

// Getter methods resolve dependencies lazily
func (d *Dependency) GetConfig() *configs.Config {
    return d.cfg.Resolve(func() *configs.Config {
        return configs.LoadConfig(d.cfgFile)
    })
}

func (d *Dependency) GetLogger() logger.Logger {
    return d.log.Resolve(func() logger.Logger {
        cfg := d.GetConfig()
        log := logger.New(cfg.App.Name, logger.Options{
            RedactedFields: cfg.Log.RedactedFields,
        })
        return log.WithParam("service", cfg.App.Name)
    })
}
```

### Bootstrap Pattern

Entry points use the bootstrap system to wire dependencies and manage lifecycle:

```go
// cmd/http.go
var httpCmd = &cobra.Command{
    Use: "http",
    Run: func(_ *cobra.Command, _ []string) {
        var (
            dep        = bootstrap.NewDependency(cfgFile)
            cfg        = dep.GetConfig()
            logger     = dep.GetLogger()
            runnerHTTP = dep.GetHTTP()
            opts       = bootstrap.RunOptions{
                StartTimeout: cfg.Http.StartTimeout,
                StopTimeout:  cfg.Http.StopTimeout,
            }
        )

        err := bootstrap.New(dep).Run(runnerHTTP, opts)
        if err != nil {
            logger.Fatal(err)
        }
    },
}
```

### Bootstrap Methods

The bootstrap provides three methods for different use cases:

1. **`Run(runner, opts)`**: For long-running servers/processes
   - Blocks until signal (SIGINT/SIGTERM)
   - Starts runner, waits for signal, then stops runner
   - Closes all `ResourceClosable` resources automatically

2. **`Execute(ctx, execute)`**: For one-off tasks
   - Runs once and returns immediately
   - No signal handling, no blocking
   - Used for migrate commands, code generation

3. **`Schedule(pattern, execute, opts)`**: For cron-style workers
   - Runs execute on a cron schedule until signal
   - Supports graceful shutdown
   - Supports singleton mode to prevent overlapping executions

### Benefits of Bootstrap Pattern

- **Transparent**: Dependency graph is visible in `Dependency` struct
- **Type-safe**: Compiler catches missing or incorrect dependencies
- **Lazy Loading**: Dependencies are resolved only when needed
- **Thread-safe**: Uses `sync.Once` for safe concurrent access
- **Automatic Cleanup**: `ResourceClosable` resources are closed automatically
- **Lifecycle Management**: Handles start/stop/cleanup automatically
- **Testable**: Components can be constructed directly in tests