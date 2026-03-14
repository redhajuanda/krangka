# Transactional Outbox

The transactional outbox guarantees **at-least-once** event delivery: the event row is written in the same DB transaction as domain data. If the broker is down, the relay worker retries until published.

## How It Works

```
Service writes data + calls repo.PublishOutbox inside DoInTransaction
  └─ Same DB transaction:
       ├─ Domain row INSERT/UPDATE
       └─ Outbox row INSERT (status: pending)
  └─ On commit:
       └─ PublishBuffered: publish each buffered entry to broker (Kafka/Redis)
       └─ success → outboxes.status = "success"
       └─ failure → outboxes.status = "failed" (relay worker will retry)

Relay worker (cron)
  └─ Queries outboxes WHERE status = 'pending' AND attempt < max_attempts
  └─ For each entry: publish to broker → update status
```

## Two Modes

| Mode | When | Behavior |
|------|------|----------|
| **Transactional** | Inside `DoInTransaction` | Insert outbox row in same tx. Buffer entry. On commit, publish buffered entries. |
| **Non-transactional** | Outside `DoInTransaction` | Insert + publish immediately. Only when no domain write is co-located. |

**Rule:** Always call `PublishOutbox` inside `DoInTransaction` when a write must be atomic with the event.

## How to Use

```go
func (s *Service) CreateWidget(ctx context.Context, w *domain.Widget) error {
    _, err := s.repo.DoInTransaction(ctx, func(repo outbound.Repository) (any, error) {
        // ⚠️ Use `repo` (lambda arg), never `s.repo`
        repoWidget := repo.GetWidgetRepository()
        if err := repoWidget.CreateWidget(ctx, w); err != nil {
            return nil, err
        }
        err := repo.PublishOutbox(ctx, outbound.PublisherTargetKafka, "widget.created", sikat.JSONMap{
            "id":   w.ID,
            "name": w.Name,
        })
        return nil, err
    })
    return fail.Wrap(err)
}
```

### Critical Rules

1. **Use `repo` (lambda arg), never `s.repo`** inside `DoInTransaction`.
2. **Topic convention**: `<entity>.<event>` — e.g. `widget.created`, `widget.updated`, `widget.deleted`.
3. **Payload**: `sikat.JSONMap` (flat `map[string]any`). Avoid deeply nested structures.

## Publisher Targets

| Constant | Broker |
|----------|--------|
| `outbound.PublisherTargetKafka` | Kafka |
| `outbound.PublisherTargetRedisstream` | Redis Streams |

## Outbox Table Schema

| Column | Type | Purpose |
|--------|------|---------|
| `id` | varchar(26) | ULID; primary key; used as message UUID for consumers |
| `topic` | varchar(50) | e.g. `widget.created` |
| `payload` | json | Event payload |
| `status` | varchar(20) | `pending`, `success`, `failed` |
| `retry_attempt` | int | Incremented on each retry |
| `error_message` | text | Last error if failed |
| `target` | varchar(50) | `kafka` or `redisstream` |

## Relay Worker

Periodically queries pending outbox entries and publishes them. Failed entries retried up to `max_retry_attempts`.

```bash
go run main.go worker outbox-relay
# or: make worker name=outbox-relay
```

### Config

```yaml
event:
  outbox:
    relay_pattern: "*/10 * * * * *"  # every 10 seconds (6-field with seconds)
    max_retry_attempts: 3
    fetch_per_page: 100
```

## Adding the Relay Worker (if not present)

### 1. Worker implementation

```go
// internal/adapter/inbound/worker/worker_outbox_relay.go
type WorkerOutboxRelay struct {
    cfg        *configs.Config
    log        logger.Logger
    repository outbound.Repository
}

func NewWorkerOutboxRelay(cfg *configs.Config, log logger.Logger, repository outbound.Repository) *WorkerOutboxRelay {
    return &WorkerOutboxRelay{cfg: cfg, log: log, repository: repository}
}

func (w *WorkerOutboxRelay) Execute(ctx context.Context) error {
    w.log.WithContext(ctx).Info("starting outbox relay execution")
    if err := w.repository.RetryOutbox(ctx); err != nil {
        w.log.WithContext(ctx).WithParam("err", err).Error("outbox relay execution failed")
        return err
    }
    w.log.WithContext(ctx).Info("outbox relay execution completed")
    return nil
}
```

### 2. Dependency wiring

```go
// Field
workerOutboxRelay ResourceExecutable[*worker.WorkerOutboxRelay]

// Getter — use GetSikatWorker() for worker DB pool
func (d *Dependency) GetWorkerOutboxRelay() *worker.WorkerOutboxRelay {
    return d.workerOutboxRelay.Resolve(func() *worker.WorkerOutboxRelay {
        repo := d.GetRepository(d.GetSikatWorker())
        return worker.NewWorkerOutboxRelay(d.GetConfig(), d.GetLogger(), repo)
    })
}
```

### 3. Cobra command (`cmd/worker.go`)

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

```go
// cmd/root.go
workerCmd.AddCommand(workerOutboxRelayCmd)
```

## Troubleshooting

| Symptom | Cause | Action |
|---------|-------|--------|
| Events never published | Broker down / publisher misconfigured | Check broker connectivity; relay will retry |
| `status = 'failed'` after max retries | Broker error, wrong topic, serialization failure | Check `error_message`; fix config or payload |
| Duplicate events | Consumer not idempotent | Use `msg.UUID` (outbox id) for idempotency |
| Relay not running | Config wrong or command not started | Ensure `relay_pattern` is set; run `worker outbox-relay` |
