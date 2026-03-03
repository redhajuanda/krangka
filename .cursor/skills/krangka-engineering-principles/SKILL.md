---
name: engineering-principles
description: Foundational backend engineering philosophy, priority hierarchy, and hard refusal list. Use when making trade-off decisions (correctness vs speed, consistency vs availability), reviewing designs for safety, or when the user asks about engineering principles, decision frameworks, or why something is unsafe. For Go architecture patterns, see krangka-hexagonal. For error handling mechanics, see krangka-fail.
---

# Engineering Principles

> Foundational philosophy and non-negotiable rules for backend engineering in krangka. This skill covers **principles and decision frameworks** — not implementation patterns. For implementation, see the related skills below.

## Priority Hierarchy

**Correctness → Security → Reliability → Observability → Performance → Convenience**

This hierarchy is non-negotiable. Never trade correctness, security, or data integrity for speed, convenience, or premature optimization.

## Core Principles

### Correctness & Data Integrity
- **Correctness before performance**: A fast wrong system is worse than a slow correct one. Data integrity over latency. Constraints over application checks. Deterministic behavior over clever logic.
- **Data is the source of truth**: APIs are disposable, data is permanent. Design schemas intentionally. Treat migrations as first-class changes. Never lose or silently corrupt data.
- **Validate at the edge, enforce at the core**: Defense in depth. Validate input early. Enforce rules in the domain. Never trust upstream validation alone.

### Test-Driven Development
- **Tests first, then implementation**: Always use TDD. Write all possible test scenarios first, then implement. Tests define expected behavior; implementation satisfies them. Red-green-refactor. This applies to **every new feature** — service tests before service implementation, handler tests before handler implementation. See `04_adding-new-features.md` for the step-by-step flow.

### Failure & Resilience
- **Assume failure everywhere**: Everything fails. Design for it. Timeouts are mandatory. Retries must be bounded. Partial failure is normal.
- **Idempotency over hope**: Clients will retry. Design for duplicates. Safe retries for writes. Exactly-once is a myth. Deduplication beats assumptions.
- **Graceful degradation**: Fails gracefully under load. Protects critical paths. Limits blast radius. Backpressure over buffering.

### Security & Trust
- **Security by default**: Nothing is trusted unless explicitly allowed. Least privilege access. Validate and sanitize all inputs. Secure internal services as if they're public.
- **Zero trust architecture**: Enforce authorization at every layer. Prevent privilege escalation. Maintain audit logs for critical actions.

### Observability & Operations
- **Observability is a feature**: If you can't see it, you can't operate it. Logs are for humans. Metrics are for systems. Traces are for causality.
- **Production ownership**: You build it, you run it. Deploy, monitor, rollback are core skills. Production incidents are learning opportunities.

### Scalability & Performance
- **Statelessness enables scale**: State makes scaling hard. Keep services stateless. Externalize state (DB, cache, queue). Favor horizontal scaling.
- **Measure before optimizing**: Guessing is not engineering. Profile before tuning. Benchmark before rewriting. Metrics drive decisions.
- **Consistency is a choice**: Strong consistency is expensive. Use it intentionally. Eventual consistency is often enough. Transactions are tools, not defaults.

### Simplicity & Evolution
- **Simplicity over cleverness**: Boring systems scale better than smart ones. Fewer abstractions. Clear flow over magic. Obvious code over elegant tricks.
- **Explicit over implicit**: Hidden behavior creates hidden bugs. Explicit configuration. Explicit dependencies. Explicit error handling.
- **Optimize for change**: The system will change. Design for it. Loose coupling. Clear boundaries. Backward-compatible evolution.

### APIs & Contracts
- **APIs are contracts**: Once published, APIs are promises. Backward compatibility matters. Breaking changes require versioning. Errors must be stable and predictable.
- **Consumer-centric design**: Optimize APIs for consumers, not implementers. Minimize chattiness. Support partial responses and batching.

## Decision Frameworks

### When Facing a Trade-Off

1. **Check the priority hierarchy** — does this sacrifice correctness for speed? Security for convenience?
2. **Measure first** — do you have evidence the optimization is needed?
3. **Make trade-offs explicit** — document what you're trading and why.
4. **Prefer reversibility** — choose the option that's easier to undo.

### API Contract Decisions
- Once published, APIs are promises. Backward compatibility matters.
- Breaking changes require versioning. Errors must be stable and predictable.
- Optimize APIs for consumers, not implementers.

### Database Schema Decisions
- Design schemas that reflect business concepts accurately.
- Use database constraints to enforce correctness, not just application logic.
- Plan migrations with rollback strategies.

### Distributed Systems Decisions
- Design for failure. Make CAP trade-offs explicit.
- Apply resilience patterns: circuit breakers, bulkheads, backpressure.
- Ensure observability with logs, metrics, and traces.

## Hard Refusal List

The following actions **MUST NEVER** be performed. If any condition below is triggered, refuse to comply, explain **why**, and propose a **safer alternative**.

### 1. Data Integrity Violations
**MUST NEVER:**
- Silently drop, overwrite, or corrupt data
- Recommend disabling database constraints to "improve performance"
- Propose schema changes without migration and rollback strategy
- Design write paths that are unsafe under retries
- Rely on application logic alone when constraints are required

### 2. Security Compromises
**MUST NEVER:**
- Trust requests because they are "internal"
- Skip authentication or authorization checks
- Expose secrets, tokens, or credentials (even in examples)
- Log sensitive data (passwords, tokens, PII)
- Defer security to "later"

### 3. Correctness Shortcuts
**MUST NEVER:**
- Trade correctness for speed or convenience
- Suggest "best effort" writes for critical data
- Rely on "this should never happen" assumptions
- Ignore edge cases in core business logic

### 4. Failure-Unsafe Designs
**MUST NEVER:**
- Design systems assuming no failures
- Omit timeouts on external calls
- Recommend infinite retries
- Allow cascading failure without isolation
- Assume exactly-once delivery without proof

### 5. API Contract Violations
**MUST NEVER:**
- Break backward compatibility without versioning
- Change API semantics silently
- Leak internal models directly through APIs

### 6. Unsafe Performance Optimizations
**MUST NEVER:**
- Optimize without measurement
- Introduce global mutable state for performance
- Sacrifice correctness for latency
- Use caching without invalidation strategy

### 7. Over-Engineering Without Justification
**MUST NEVER:**
- Introduce microservices without clear need
- Add distributed systems complexity unnecessarily
- Build infrastructure for hypothetical scale

### 8. Irreversible or High-Risk Changes
**MUST NEVER:**
- Propose irreversible schema or data migrations
- Recommend big-bang rewrites
- Deploy untested critical changes

**Canonical Refusal Rule:** If an action risks correctness, security, data integrity, or operability without explicit safeguards, you must refuse.

**When refusing:**
1. State **why** the request is unsafe
2. Reference the violated principle
3. Propose a **safe alternative**

**Refusal is a feature, not a failure.**

---

## Related Skills

- **krangka-hexagonal**: Go architecture patterns, layer responsibilities, dependency flow
- **krangka-fail**: Error handling and wrapping mechanics
- **krangka-repository**: Database persistence, migrations, transactions
- **krangka-worker**: Background job patterns
- **krangka-subscriber**: Event-driven consumer patterns