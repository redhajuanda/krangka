## ADDED Requirements

### Requirement: Constructor functions for all components

All components (services, adapters, infrastructure) SHALL be initialized through explicit constructor functions that take dependencies as typed parameters.

#### Scenario: Service construction with explicit dependencies
- **WHEN** a service is created
- **THEN** it MUST use a constructor function (e.g., `NewNoteService()`) that accepts all dependencies as typed parameters
- **AND** the constructor MUST return the service instance or interface and an error

#### Scenario: No FX imports in constructors
- **WHEN** any constructor function is defined
- **THEN** it MUST NOT import `go.uber.org/fx`
- **AND** it MUST NOT use FX-specific types or annotations

#### Scenario: Compile-time dependency validation
- **WHEN** a component is constructed with missing or incorrect dependencies
- **THEN** the Go compiler MUST catch the error at compile time
- **AND** no runtime reflection or discovery SHALL be used

### Requirement: Module organization with constructor exports

Module files (`module.go`) SHALL export constructor functions instead of FX options, maintaining the existing organizational structure.

#### Scenario: Module file exports constructor
- **WHEN** a `module.go` file is defined
- **THEN** it MUST export one or more constructor functions (e.g., `NewMain()`, `NewWorker()`)
- **AND** it MUST NOT export `fx.Option` or `fx.Module` values

#### Scenario: Multiple variant constructors
- **WHEN** a module needs different configurations (e.g., main DB vs worker DB)
- **THEN** it SHALL provide multiple named constructors (e.g., `NewMain()`, `NewWorker()`)
- **AND** each constructor MUST clearly document its intended use case

### Requirement: Explicit dependency wiring

Dependencies SHALL be wired explicitly in entry points, making the dependency graph visible and traceable through code.

#### Scenario: Dependencies constructed in order
- **WHEN** an application is bootstrapped
- **THEN** dependencies MUST be constructed in a clear top-to-bottom order
- **AND** each dependency MUST be passed explicitly to consumers that need it

#### Scenario: No hidden initialization
- **WHEN** a component is initialized
- **THEN** all initialization logic MUST be visible in the entry point or explicit setup functions
- **AND** no automatic discovery or invocation through reflection SHALL occur

#### Scenario: Traceable dependency graph
- **WHEN** a developer needs to understand what dependencies a component uses
- **THEN** they MUST be able to trace dependencies by reading constructor function signatures
- **AND** no framework-specific knowledge SHALL be required to understand the wiring

### Requirement: Interface-based dependency injection

Components SHALL depend on interfaces (ports) rather than concrete implementations, maintaining hexagonal architecture principles.

#### Scenario: Services depend on port interfaces
- **WHEN** a service is constructed
- **THEN** it MUST accept port interfaces as parameters (e.g., `outbound.NoteRepository`)
- **AND** it MUST NOT directly depend on adapter implementations

#### Scenario: Adapters implement port interfaces
- **WHEN** an adapter is constructed
- **THEN** it MUST implement a port interface
- **AND** it SHALL be passed to services as the interface type

### Requirement: Error handling in construction

Constructor functions SHALL return errors when initialization fails, enabling proper error propagation and handling.

#### Scenario: Constructor returns error on failure
- **WHEN** a constructor fails to initialize (e.g., invalid config, connection failure)
- **THEN** it MUST return a non-nil error
- **AND** the error MUST provide sufficient context for debugging

#### Scenario: Bootstrap handles construction errors
- **WHEN** any component construction fails during bootstrap
- **THEN** the application MUST terminate with a clear error message
- **AND** it MUST NOT start with partial or invalid dependencies

### Requirement: Type safety without reflection

All dependency resolution SHALL occur at compile time through typed parameters, eliminating runtime reflection.

#### Scenario: Type mismatches caught at compile time
- **WHEN** a constructor is called with incorrect parameter types
- **THEN** the Go compiler MUST reject the code
- **AND** no type assertions or interface conversions SHALL be needed

#### Scenario: No reflection-based wiring
- **WHEN** dependencies are wired together
- **THEN** no reflection API (e.g., `reflect` package) SHALL be used for dependency resolution
- **AND** all type information MUST be statically known
