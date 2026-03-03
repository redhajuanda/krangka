---
name: krangka-outbox
description: Guide for the transactional outbox pattern in krangka: how it works, how to publish events, how the relay worker retries failed publications, and how to add or run the outbox relay worker. Use when publishing events, debugging outbox failures, adding a relay worker, or when the user asks about outbox, event publishing, or at-least-once delivery.
---

# krangka Outbox Guide

The transactional outbox guarantees **at-least-once** event delivery: the event row is written in the same database transaction as domain data. If the broker is down, the relay worker retries until the event is published.

---

## How It Works

### Flow overview

```
Service writes data + calls repo.PublishOutbox inside DoInTransaction
  └─ Same DB transaction:
       ├─ Domain row (e.g. note) INSERT/UPDATE
       └─ Outbox row INSERT (status: pending)
  └─ On commit:
       └─ PublishBuffered: publish each buffered entry to broker (Kafka/Redis)
       └─ success → outboxes.status = "success"
       └─ failure → outboxes.status = "failed" (relay worker will retry)

Relay worker (runs on cron)
  └─ Queries outboxes WHERE status = 'pending' AND attempt < max_attempts
  └─ For each entry: publish to broker → update status to success/failed
```

### Two modes of operation

| Mode | When | Behavior |
|------|------|----------|
| ** transactional** | Inside `DoInTransaction` | Insert outbox row in the same tx as domain data. Buffer entry in memory. On commit, publish buffered entries. |
| **Non-transactional** | Outside `DoInTransaction` | Insert outbox row and publish immediately. Use only when there is no domain write in the same operation. |

**Rule:** Always call `PublishOutbox` inside `DoInTransaction` when a write must be atomic with the event.

---

## How to Use

### From a service (write + event)

```go
func (s *Service) CreateWidget(ctx context.Context, w *domain.Widget) error {
    _, err := s.repo.DoInTransaction(ctx, func(repo outbound.Repository) (any, error) {
        repoWidget := repo.GetWidgetRepository()  // use repo, not s.repo
        if err := repoWidget.CreateWidget(ctx, w); err != nil {
            return nil, err
        }
        err := repo.PublishOutbox(ctx, outbound.PublisherTargetKafka, "widget.created", qwery.JSONMap{
            "id":   w.ID,
            "name": w.Name,
        })
        return nil, err
    })
    return fail.Wrap(err)
}
```

### Critical rules

1. **Use `repo` (lambda arg), never `s.repo`** inside `DoInTransaction`. Only `repo` is transactional.
2. **Topic convention**: `<entity>.<event>` — e.g. `widget.created`, `widget.updated`, `widget.deleted`.
3. **Payload**: `qwery.JSONMap` (flat `map[string]any`). Avoid deeply nested structures.

### Publisher targets

| Constant | Broker |
|----------|--------|
| `outbound.PublisherTargetKafka` | Kafka |
| `outbound.PublisherTargetRedisstream` | Redis Streams |

---

## Outbox Table Schema

Migration: `internal/adapter/outbound/mariadb/migrations/scripts/20250126141000-ddl_outboxes.sql`

| Column | Type | Purpose |
|--------|------|---------|
| `id` | varchar(26) | ULID; primary key; used as message UUID for consumers |
| `topic` | varchar(50) | e.g. `note.created` |
| `payload` | json | Event payload |
| `status` | varchar(20) | `pending`, `success`, `failed` |
| `retry_attempt` | int | Incremented on each retry attempt |
| `error_message` | text | Last error if failed |
| `target` | varchar(50) | `kafka` or `redisstream` |

---

## Relay Worker

The relay worker periodically queries pending outbox entries and publishes them to the broker. Failed entries are retried up to `max_retry_attempts`.

### Run the relay worker

```bash
go run main.go worker outbox-relay
```

Or with config:

```bash
go run main.go --config configs/files/example.yaml worker outbox-relay
```

### Config (event.outbox)

| Key | Description | Example |
|-----|-------------|---------|
| `relay_pattern` | Cron pattern for relay schedule | `*/10 * * * * *` (every 10 seconds) |
| `max_retry_attempts` | Max retry attempts per entry | `3` |
| `fetch_per_page` | Batch size per cursor page | `100` |

Example in config YAML (`event.outbox`):

```yaml
event:
  outbox:
    relay_pattern: "*/10 * * * * *"   # cron: every 10 seconds (6 fields = sec min hr day month dow)
    max_retry_attempts: 3
    fetch_per_page: 100
```

### How the relay executes

1. `WorkerOutboxRelay.Execute(ctx)` is called by the scheduler on the cron pattern.
2. It calls `repo.RetryOutbox(ctx)`, which:
   - Queries `outboxes` WHERE `status = 'pending'` AND `retry_attempt < max_retry_attempts`
   - For each entry: publish to the target broker → update `status` and `retry_attempt`

---

## Adding the Relay Worker (if not present)

If the project does not yet have an outbox relay worker, add it as follows.

### 1. Worker implementation

Create `internal/adapter/inbound/worker/worker_outbox_relay.go`:

```go
package worker

import (
	"context"

	"github.com/redhajuanda/krangka/configs"
	"github.com/redhajuanda/krangka/internal/core/port/outbound"
	"github.com/redhajuanda/komon/logger"
)

type WorkerOutboxRelay struct {
	cfg        *configs.Config
	log        logger.Logger
	repository outbound.Repository
}

func NewWorkerOutboxRelay(cfg *configs.Config, log logger.Logger, repository outbound.Repository) *WorkerOutboxRelay {
	return &WorkerOutboxRelay{
		cfg:        cfg,
		log:        log,
		repository: repository,
	}
}

func (w *WorkerOutboxRelay) Execute(ctx context.Context) error {
	w.log.WithContext(ctx).Info("starting outbox relay execution")
	err := w.repository.RetryOutbox(ctx)
	if err != nil {
		w.log.WithContext(ctx).Error("outbox relay execution failed", "error", err)
		return err
	}
	w.log.WithContext(ctx).Info("outbox relay execution completed")
	return nil
}
```

### 2. Dependency wiring

In `cmd/bootstrap/dependency.go`:

- Add field: `workerOutboxRelay ResourceExecutable[*worker.WorkerOutboxRelay]`
- Add getter:

```go
func (d *Dependency) GetWorkerOutboxRelay() *worker.WorkerOutboxRelay {
	return d.workerOutboxRelay.Resolve(func() *worker.WorkerOutboxRelay {
		repo := d.GetRepository(d.GetQweryWorker())
		return worker.NewWorkerOutboxRelay(d.GetConfig(), d.GetLogger(), repo)
	})
}
```

**Note:** Use `GetQweryWorker()` so the relay uses the worker DB pool, not the main HTTP pool.

### 3. Cobra command

In `cmd/worker.go`:

```go
var workerOutboxRelayCmd = &cobra.Command{
	Use:   "outbox-relay",
	Short: "Start the outbox relay worker",
	Run: func(c *cobra.Command, args []string) {
		var (
			dep    = bootstrap.NewDependency(cfgFile)
			cfg    = dep.GetConfig()
			logger = dep.GetLogger()
			runner = dep.GetWorkerOutboxRelay()
			boot   = bootstrap.New(dep)
			opts   = bootstrap.ScheduleOptions{
				ShutdownTimeout: 30 * time.Second,
				SingletonMode:   true,
			}
		)
		err := boot.Schedule(cfg.Event.Outbox.RelayPattern, runner, opts)
		if err != nil {
			logger.Fatalf("failed to schedule worker outbox relay: %v", err)
		}
	},
}
```

### 4. Register command

In `cmd/root.go`:

```go
workerCmd.AddCommand(workerOutboxRelayCmd)
```

---

## Key Implementation Details

### Repository wiring

- **Default outbox**: `NewOutbox(cfg, log, qwery.Client, false, publishers)` — used for `PublishOutbox` when not in a transaction.
- **Transactional outbox**: Inside `DoInTransaction`, a new outbox is created with `tx` and `isTransaction: true`. It buffers entries until commit.

### Commit flow (simplified)

```go
// repository.go DoInTransaction
defer func() {
    handleTransaction(ctx, tx, &err)  // Commit or Rollback
    registry.outbox.qwery = r.qwery   // Switch to main client for publish
    registry.outbox.PublishBuffered(ctx)  // Publish buffered entries
}()
```

### Message metadata

Published messages include:
- `msg.UUID` = outbox `id` (ULID) — use for consumer idempotence
- Metadata: `target`, `topic`, `id`

---

## Troubleshooting

| Symptom | Cause | Action |
|---------|-------|--------|
| Events never published | Broker down / publisher misconfigured | Check broker connectivity; relay will retry |
| `status = 'failed'` after max retry attempts | Broker error, wrong topic, serialization failure | Check logs for `error_message`; fix config or payload |
| Duplicate events | Consumer not idempotent | Use `msg.UUID` (outbox id) for idempotency |
| Relay not running | Config wrong or command not started | Ensure `relay_pattern` is set; run `worker outbox-relay` |
