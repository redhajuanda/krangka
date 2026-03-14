# Engineering Principles

> Foundational philosophy and non-negotiable rules for krangka backend engineering.

## Priority Hierarchy

**Correctness → Security → Reliability → Observability → Performance → Convenience**

This hierarchy is non-negotiable. Never trade correctness, security, or data integrity for speed, convenience, or premature optimization.

## Core Principles

### Correctness & Data Integrity
- **Correctness before performance**: A fast wrong system is worse than a slow correct one.
- **Data is the source of truth**: APIs are disposable, data is permanent. Treat migrations as first-class changes. Never lose or silently corrupt data.
- **Validate at the edge, enforce at the core**: Validate input early. Enforce rules in the domain. Never trust upstream validation alone.

### Test-Driven Development
- **Tests first, then implementation**: Always use TDD. Write all possible test scenarios first, then implement. Red-green-refactor. This applies to **every new feature** — service tests before service implementation, handler tests before handler implementation.

### Failure & Resilience
- **Assume failure everywhere**: Everything fails. Design for it. Timeouts are mandatory. Retries must be bounded.
- **Idempotency over hope**: Clients will retry. Design for duplicates. Safe retries for writes.
- **Graceful degradation**: Fails gracefully under load. Protects critical paths. Limits blast radius.

### Security & Trust
- **Security by default**: Nothing is trusted unless explicitly allowed. Least privilege. Validate and sanitize all inputs.
- **Zero trust architecture**: Enforce authorization at every layer. Prevent privilege escalation.

### Observability & Operations
- **Observability is a feature**: If you can't see it, you can't operate it. Logs for humans. Metrics for systems. Traces for causality.

### Simplicity & Evolution
- **Simplicity over cleverness**: Boring systems scale better than smart ones. Fewer abstractions. Obvious code over elegant tricks.
- **Explicit over implicit**: Hidden behavior creates hidden bugs. Explicit configuration. Explicit dependencies.
- **Optimize for change**: Loose coupling. Clear boundaries. Backward-compatible evolution.

## Decision Frameworks

### When Facing a Trade-Off
1. **Check the priority hierarchy** — does this sacrifice correctness for speed?
2. **Measure first** — do you have evidence the optimization is needed?
3. **Make trade-offs explicit** — document what you're trading and why.
4. **Prefer reversibility** — choose the option that's easier to undo.

## Hard Refusal List

The following **MUST NEVER** be performed. If triggered, refuse, explain **why**, and propose a **safer alternative**.

### 1. Data Integrity Violations
- Silently drop, overwrite, or corrupt data
- Recommend disabling database constraints to "improve performance"
- Propose schema changes without migration and rollback strategy
- Design write paths that are unsafe under retries

### 2. Security Compromises
- Trust requests because they are "internal"
- Skip authentication or authorization checks
- Expose secrets, tokens, or credentials (even in examples)
- Log sensitive data (passwords, tokens, PII)

### 3. Correctness Shortcuts
- Trade correctness for speed or convenience
- Suggest "best effort" writes for critical data
- Ignore edge cases in core business logic

### 4. Failure-Unsafe Designs
- Design systems assuming no failures
- Omit timeouts on external calls
- Recommend infinite retries
- Allow cascading failure without isolation

### 5. API Contract Violations
- Break backward compatibility without versioning
- Change API semantics silently
- Leak internal models directly through APIs

### 6. Unsafe Performance Optimizations
- Optimize without measurement
- Sacrifice correctness for latency
- Use caching without invalidation strategy

### 7. Irreversible or High-Risk Changes
- Propose irreversible schema or data migrations without rollback
- Recommend big-bang rewrites
- Deploy untested critical changes

**Canonical Refusal Rule:** If an action risks correctness, security, data integrity, or operability without explicit safeguards, refuse.

**When refusing:**
1. State **why** the request is unsafe
2. Reference the violated principle
3. Propose a **safe alternative**
