## ADDED Requirements

### Requirement: HTTP server initialization and lifecycle

The HTTP server SHALL be initialized and run through explicit bootstrap functions that handle startup, graceful shutdown, and error handling.

#### Scenario: HTTP server starts successfully
- **WHEN** `go run main.go http` command is executed
- **THEN** the application MUST load configuration, construct dependencies, and start the HTTP server
- **AND** the server MUST listen on the configured port

#### Scenario: HTTP server handles graceful shutdown
- **WHEN** the HTTP server receives an interrupt signal (SIGINT or SIGTERM)
- **THEN** it MUST gracefully shut down, completing in-flight requests
- **AND** it MUST close all connections cleanly before exiting

#### Scenario: HTTP server reports initialization errors
- **WHEN** HTTP server initialization fails (e.g., port already in use, invalid config)
- **THEN** the application MUST log a clear error message
- **AND** it MUST exit with a non-zero exit code

### Requirement: Worker initialization and execution

Workers SHALL be initialized with explicit dependencies and SHALL support both OneTime and LongRunning execution patterns.

#### Scenario: OneTime worker executes and exits
- **WHEN** a OneTime worker (e.g., `generate-id`) is executed
- **THEN** it MUST run its task to completion
- **AND** it MUST exit naturally with exit code 0 on success

#### Scenario: LongRunning worker handles context cancellation
- **WHEN** a LongRunning worker (e.g., `relay-outbox`) is running
- **AND** it receives a cancellation signal
- **THEN** it MUST stop processing gracefully within a reasonable timeout
- **AND** it MUST clean up resources before exiting

#### Scenario: Worker reports execution errors
- **WHEN** a worker encounters an error during execution
- **THEN** it MUST log the error with sufficient context
- **AND** it MUST exit with a non-zero exit code for OneTime workers
- **AND** it MUST continue running and retry for LongRunning workers (based on worker logic)

### Requirement: Migration command execution

Database migration commands SHALL execute without FX lifecycle management, supporting up, down, and new migration operations.

#### Scenario: Migration up executes successfully
- **WHEN** `go run main.go migrate up [repository]` command is executed
- **THEN** the application MUST load configuration and database connection
- **AND** it MUST run pending migrations for the specified repository
- **AND** it MUST exit with code 0 on success

#### Scenario: Migration down rolls back migrations
- **WHEN** `go run main.go migrate down [repository] [max]` command is executed
- **THEN** the application MUST rollback the specified number of migrations
- **AND** it MUST exit cleanly after rollback completes

#### Scenario: Migration new creates migration file
- **WHEN** `go run main.go migrate new [repository] [name]` command is executed
- **THEN** the application MUST generate a new migration file with timestamp
- **AND** it MUST NOT require database connection

### Requirement: Context-based lifecycle management

Component lifecycle (startup, running, shutdown) SHALL be managed using standard Go context patterns.

#### Scenario: Components receive context for cancellation
- **WHEN** a long-running component (server, worker) is started
- **THEN** it MUST receive a `context.Context` parameter
- **AND** it MUST monitor `ctx.Done()` channel for shutdown signals

#### Scenario: Context cancellation propagates to children
- **WHEN** a parent context is cancelled
- **THEN** all child components MUST receive the cancellation signal
- **AND** they MUST initiate their shutdown sequences

#### Scenario: Graceful shutdown timeout
- **WHEN** a component is shutting down
- **THEN** it SHOULD complete within a configured timeout (e.g., 30 seconds)
- **AND** it MUST force-close if the timeout is exceeded

### Requirement: Signal handling for graceful shutdown

Applications SHALL handle OS signals (SIGINT, SIGTERM) to trigger graceful shutdown.

#### Scenario: Application handles SIGINT
- **WHEN** the application receives SIGINT (Ctrl+C)
- **THEN** it MUST initiate graceful shutdown
- **AND** it MUST exit with code 0 if shutdown completes successfully

#### Scenario: Application handles SIGTERM
- **WHEN** the application receives SIGTERM (from Kubernetes, systemd, etc.)
- **THEN** it MUST initiate graceful shutdown
- **AND** it MUST exit cleanly for container orchestration

#### Scenario: Multiple signals force immediate exit
- **WHEN** the application receives a second signal during shutdown
- **THEN** it MAY force immediate termination
- **AND** it MUST exit with a non-zero code to indicate unclean shutdown

### Requirement: Bootstrap function organization

Each entry point (HTTP, worker, migrate) SHALL have a dedicated bootstrap function that constructs and wires dependencies.

#### Scenario: Bootstrap function is entry point
- **WHEN** a command (http, worker, migrate) is invoked
- **THEN** it MUST call a bootstrap function (e.g., `runHTTP()`, `runWorker()`)
- **AND** the bootstrap function MUST handle all setup, execution, and teardown

#### Scenario: Bootstrap function order is clear
- **WHEN** a bootstrap function executes
- **THEN** it MUST construct dependencies in a clear, top-to-bottom order:
  1. Load configuration
  2. Initialize infrastructure (logger, DB, cache)
  3. Initialize outbound adapters (repositories)
  4. Initialize services
  5. Initialize inbound adapters (handlers, workers)
  6. Run the application

#### Scenario: Bootstrap function handles all errors
- **WHEN** any step in the bootstrap fails
- **THEN** the bootstrap function MUST return an error
- **AND** the main command handler MUST log the error and exit

### Requirement: No FX dependency in application code

Application code SHALL NOT import or use any FX types, functions, or patterns.

#### Scenario: No FX imports in codebase
- **WHEN** the codebase is scanned for imports
- **THEN** no file SHALL import `go.uber.org/fx`
- **AND** `go.mod` MUST NOT list `go.uber.org/fx` as a dependency

#### Scenario: No FX lifecycle hooks
- **WHEN** components define lifecycle methods
- **THEN** they MUST NOT use `fx.Lifecycle`, `OnStart`, or `OnStop`
- **AND** they SHALL use standard patterns like `Run(context.Context) error` or `Start()/Stop()`

#### Scenario: No FX patterns in code
- **WHEN** code is reviewed for DI patterns
- **THEN** it MUST NOT use FX-specific patterns:
  - `fx.Annotate` for interface casting
  - `fx.Populate` for extracting values
  - `fx.Shutdowner` for programmatic shutdown
  - `fx.Invoke` for initialization hooks

### Requirement: Application behavior preservation

The application behavior after removing FX SHALL be identical to the behavior before removal.

#### Scenario: HTTP endpoints remain unchanged
- **WHEN** HTTP requests are made to any endpoint
- **THEN** the responses MUST match the previous FX-based implementation
- **AND** no endpoint behavior SHALL change

#### Scenario: Worker behavior remains unchanged
- **WHEN** workers are executed
- **THEN** they MUST perform the same tasks as before
- **AND** no worker logic SHALL change

#### Scenario: Database operations remain unchanged
- **WHEN** database operations are performed
- **THEN** they MUST produce the same results as before
- **AND** no data access patterns SHALL change
