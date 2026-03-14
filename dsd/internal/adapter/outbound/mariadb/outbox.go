package mariadb

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/oklog/ulid/v2"
	"github.com/redhajuanda/komon/common"
	"github.com/redhajuanda/komon/fail"
	"github.com/redhajuanda/komon/logger"
	"github.com/redhajuanda/komon/pagination"
	"github.com/redhajuanda/komon/tracer"
	"gitlab.sicepat.tech/pka/sds/configs"
	"gitlab.sicepat.tech/pka/sds/internal/core/port/outbound"
	"github.com/redhajuanda/qwery"
)

// Outbox implements the transactional outbox pattern
type Outbox struct {
	cfg           *configs.Config
	log           logger.Logger
	qwery         qwery.Runable
	publishers    outbound.Publishers
	outboxBuffer  []DomainOutboxInsert // Track outbox entries created in this transaction
	mu            sync.Mutex           // Protects outboxBuffer for concurrent access
	isTransaction bool
}

const (
	OutboxStatusPending = "pending"
	OutboxStatusSuccess = "success"
	OutboxStatusFailed  = "failed"
)

// NewOutbox creates a new Outbox instance
func NewOutbox(cfg *configs.Config, log logger.Logger, qwery qwery.Runable, isTransaction bool, publishers outbound.Publishers) *Outbox {
	return &Outbox{
		cfg:           cfg,
		log:           log,
		qwery:         qwery,
		isTransaction: isTransaction,
		publishers:    publishers,
		outboxBuffer:  make([]DomainOutboxInsert, 0),
	}
}

// Retry retries the outbox entries that failed to publish
func (r *Outbox) Retry(ctx context.Context) error {

	var (
		cursor  string
		perPage = r.cfg.Event.Outbox.FetchPerPage
	)

	for {
		pag := &pagination.Pagination{
			Type:    "cursor",
			PerPage: perPage,
			Cursor:  cursor,
		}

		outboxItems, err := r.GetOutbox(ctx, pag)
		if err != nil {
			r.log.WithContext(ctx).Error("failed to get outbox", err)
			return fail.Wrap(err)
		}

		// Early exit if no items
		if len(outboxItems) == 0 {
			break
		}

		for _, outboxItem := range outboxItems {
			// map to DomainOutboxInsert
			entry := DomainOutboxInsert{
				ID:           outboxItem.ID,
				Target:       outbound.PublisherTarget(outboxItem.Target),
				Topic:        outboxItem.Topic,
				RetryAttempt: outboxItem.RetryAttempt,
				Payload:      outboxItem.Payload,
			}

			// publish outbox entry
			if err := r.publishReal(ctx, entry); err != nil {
				r.log.WithContext(ctx).Errorf("failed to publish outbox, error: %v, id: %s", err, outboxItem.ID)
			} else {
				r.log.WithContext(ctx).Debugf("outbox published successfully, id: %s", outboxItem.ID)
			}
		}

		// get next cursor
		cursor = pag.Result.Cursor.Next
		if cursor == "" {
			break
		}
	}

	return nil
}

// GetOutbox retrieves outbox entries from the database
func (r *Outbox) GetOutbox(ctx context.Context, pagination *pagination.Pagination) ([]DomainOutboxInsert, error) {

	ctx, span := tracer.Trace(ctx)
	defer span.End()

	var outbox []DomainOutboxInsert

	// Query optimized for index on (status, deleted_at, retry_attempt, id)
	query := `
		SELECT 
			id, topic, payload, target, retry_attempt
		FROM outboxes
		WHERE status = {{ .status }}
			AND deleted_at = 0
			AND retry_attempt < {{ .retry_attempt }}
	`

	err := r.qwery.
		RunRaw(query).
		WithParams(map[string]any{
			"status":        OutboxStatusPending,
			"retry_attempt": r.cfg.Event.Outbox.MaxRetryAttempts,
		}).
		WithPagination(pagination).
		WithOrderBy("+id").
		ScanStructs(&outbox).
		Query(ctx)

	if err != nil {
		return nil, fail.Wrap(err)
	}

	return outbox, nil
}

// UpdateOutbox updates an outbox entry in the database
func (r *Outbox) UpdateOutbox(ctx context.Context, params DomainOutboxUpdate) error {
	ctx, span := tracer.Trace(ctx)
	defer span.End()

	query := `
		UPDATE outboxes
		SET
			retry_attempt = {{ .retry_attempt }}
			{{ if .error_message }}, error_message = {{ .error_message }}{{ end }}
			{{ if .status }}, status = {{ .status }}{{ end }} 
		WHERE id = {{ .id }}
	`

	_, err := r.qwery.
		RunRaw(query).
		WithParams(params).
		Exec(ctx)

	if err != nil {
		return fail.Wrap(err)
	}

	return nil
}

// InsertOutbox inserts a new outbox entry into the database
func (r *Outbox) InsertOutbox(ctx context.Context, outbox DomainOutboxInsert) error {
	ctx, span := tracer.Trace(ctx)
	defer span.End()

	query := `
		INSERT INTO outboxes (id, topic, payload, status, retry_attempt, error_message, target) 
		VALUES ({{ .id }}, {{ .topic }}, {{ .payload }}, 'pending', 0, NULL, {{ .target }})
	`

	_, err := r.qwery.
		RunRaw(query).
		WithParams(outbox).
		Exec(ctx)

	if err != nil {
		return fail.Wrap(err)
	}

	return nil
}

// PublishOutbox inserts a new outbox entry into the database and buffers it if it is in a transaction
func (r *Outbox) PublishOutbox(ctx context.Context, target outbound.PublisherTarget, topic string, payload qwery.JSONMap) error {
	outboxEntry := DomainOutboxInsert{
		ID:      ulid.Make().String(),
		Target:  target,
		Topic:   topic,
		Payload: payload,
	}

	r.log.WithContext(ctx).Debugf("publishing outbox, transaction mode: %t, topic: %s", r.isTransaction, topic)

	// insert outbox entry into database
	if err := r.InsertOutbox(ctx, outboxEntry); err != nil {
		return fail.Wrap(fmt.Errorf("failed to insert outbox: %w", err))
	}

	if r.isTransaction {
		// if in transaction, buffer the outbox entry
		r.mu.Lock()
		r.outboxBuffer = append(r.outboxBuffer, outboxEntry)
		r.mu.Unlock()
		r.log.WithContext(ctx).Debugf("buffered outbox entry, topic: %s, buffer size: %d", topic, len(r.outboxBuffer))
	} else {
		// otherwise, publish immediately
		if err := r.publishReal(ctx, outboxEntry); err != nil {
			return fail.Wrap(fmt.Errorf("failed to publish outbox: %w", err))
		}
	}

	return nil
}

// PublishBuffered publishes all buffered outbox entries immediately
// Returns the number of successful and failed publications
func (r *Outbox) PublishBuffered(ctx context.Context) (success, failed int) {
	r.mu.Lock()
	bufferCopy := make([]DomainOutboxInsert, len(r.outboxBuffer))
	copy(bufferCopy, r.outboxBuffer)
	r.outboxBuffer = r.outboxBuffer[:0] // Clear buffer
	r.mu.Unlock()

	if len(bufferCopy) == 0 {
		return 0, 0
	}

	r.log.WithContext(ctx).Infof("publishing %d buffered outbox entries", len(bufferCopy))

	for _, entry := range bufferCopy {
		if err := r.publishReal(ctx, entry); err != nil {
			r.log.WithContext(ctx).Errorf("failed to publish buffered outbox, id: %s, error: %v", entry.ID, err)
			failed++
		} else {
			success++
		}
	}

	r.log.WithContext(ctx).Infof("published buffered outbox entries, success: %d, failed: %d", success, failed)
	return success, failed
}

// publishReal publishes the outbox entry to the target publisher, it will update the outbox status to success or failed in the database
func (r *Outbox) publishReal(ctx context.Context, entry DomainOutboxInsert) (err error) {

	defer func() {
		status := OutboxStatusSuccess
		var errorMessage *string

		if err != nil {
			entry.RetryAttempt++
			if entry.RetryAttempt < r.cfg.Event.Outbox.MaxRetryAttempts {
				status = OutboxStatusPending
			} else {
				status = OutboxStatusFailed
			}
			errorMessage = common.ToPointer(err.Error())
		} else { // if no error, update the outbox status to success
			status = OutboxStatusSuccess
		}

		// Update outbox status in database
		params := DomainOutboxUpdate{
			ID:           entry.ID,
			RetryAttempt: entry.RetryAttempt,
			ErrorMessage: errorMessage,
			Status:       status,
		}

		if errUpdate := r.UpdateOutbox(ctx, params); errUpdate != nil {
			r.log.WithContext(ctx).Errorf("failed to update outbox status, error: %v, id: %s", errUpdate, entry.ID)
			// Don't return here as we still want the original error
		} else if err == nil {
			r.log.WithContext(ctx).Debugf("outbox published successfully, id: %s, topic: %s", entry.ID, entry.Topic)
		}
	}()

	// Get publisher by target
	publisher, ok := r.publishers[entry.Target]
	if !ok {
		err = fail.Newf("publisher not found: %s", entry.Target)
		r.log.WithContext(ctx).Error(err)
		return fail.Wrap(err)
	}

	// Marshal payload
	payload, err := json.Marshal(entry.Payload)
	if err != nil {
		r.log.WithContext(ctx).Errorf("failed to marshal outbox payload, error: %v, id: %s", err, entry.ID)
		return fail.Wrap(err)
	}

	// Publish to target publisher
	msg := message.NewMessage(entry.ID, payload)
	msg.SetContext(ctx)
	msg.Metadata.Set("target", string(entry.Target))
	msg.Metadata.Set("topic", entry.Topic)
	msg.Metadata.Set("id", entry.ID)

	if err = publisher.Publish(entry.Topic, msg); err != nil {
		r.log.WithContext(ctx).Errorf("failed to publish outbox immediately, will be retried by relay worker, error: %v, id: %s, topic: %s", err, entry.ID, entry.Topic)
		return fail.Wrap(err)
	}

	return nil
}

type DomainOutboxInsert struct {
	ID           string                   `qwery:"id"`
	Topic        string                   `qwery:"topic"`
	Payload      qwery.JSONMap            `qwery:"payload"`
	RetryAttempt int                      `qwery:"retry_attempt"`
	Target       outbound.PublisherTarget `qwery:"target"`
}

type DomainOutboxUpdate struct {
	ID           string  `qwery:"id"`
	RetryAttempt int     `qwery:"retry_attempt"`
	ErrorMessage *string `qwery:"error_message"`
	Status       string  `qwery:"status"`
}