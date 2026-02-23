package worker

import (
	"context"

	"github.com/redhajuanda/komon/logger"
	"github.com/redhajuanda/krangka/configs"
	"github.com/redhajuanda/krangka/internal/core/port/outbound"
)

// WorkerOutboxRelay is a worker that retries outbox entries
type WorkerOutboxRelay struct {
	cfg        *configs.Config
	log        logger.Logger
	repository outbound.Repository
}

// NewWorkerOutboxRelay creates a new WorkerOutboxRelay instance
func NewWorkerOutboxRelay(cfg *configs.Config, log logger.Logger, repository outbound.Repository) *WorkerOutboxRelay {
	return &WorkerOutboxRelay{
		cfg:        cfg,
		log:        log,
		repository: repository,
	}
}

// Execute runs a single iteration of the outbox relay
// It respects context cancellation for graceful shutdown
func (w *WorkerOutboxRelay) Execute(ctx context.Context) error {

	w.log.WithContext(ctx).Info("starting outbox relay execution")

	// Use the provided context so the task can be cancelled during shutdown
	err := w.repository.RetryOutbox(ctx)
	if err != nil {
		w.log.WithContext(ctx).Error("outbox relay execution failed", "error", err)
		return err
	}

	w.log.WithContext(ctx).Info("outbox relay execution completed")
	return nil

}
