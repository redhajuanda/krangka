package mariadb

import (
	"context"

	"github.com/redhajuanda/komon/fail"
	"github.com/redhajuanda/komon/logger"
	"gitlab.sicepat.tech/pka/sds/configs"
	implRepo "gitlab.sicepat.tech/pka/sds/internal/adapter/outbound/mariadb/repositories"
	"gitlab.sicepat.tech/pka/sds/internal/core/port/outbound"
	portRepo "gitlab.sicepat.tech/pka/sds/internal/core/port/outbound/repositories"
	"github.com/redhajuanda/qwery"
)

// mariaDBRepository is a wrapper around the MariaDB repository
type mariaDBRepository struct {
	cfg     *configs.Config
	log     logger.Logger
	qwery   *Qwery
	qweryTx qwery.Runable
	outbox  *Outbox

	NoteRepository portRepo.Note
}

// Interface compliance checks
var _ outbound.Repository = (*mariaDBRepository)(nil)

// NewMariaDBRepository creates a new MariaDBRepository instance
func NewMariaDBRepository(cfg *configs.Config, log logger.Logger, qwery *Qwery, publishers outbound.Publishers) *mariaDBRepository {
	return &mariaDBRepository{
		cfg:    cfg,
		log:    log,
		qwery:  qwery,
		outbox: NewOutbox(cfg, log, qwery.Client, false, publishers),

		NoteRepository: implRepo.NewNoteRepository(qwery.Client),
	}
}

// PublishOutbox creates an outbox entry in the database and also publishes it to the target publisher when the transaction is committed successfully.
// It will be retried by the outbox relay worker if the transaction is not committed successfully
func (r *mariaDBRepository) PublishOutbox(ctx context.Context, target outbound.PublisherTarget, topic string, payload qwery.JSONMap) error {
	return r.outbox.PublishOutbox(ctx, target, topic, payload)
}

// RetryOutbox retries the outbox entries that are not committed successfully, it should be called by the outbox relay worker
func (r *mariaDBRepository) RetryOutbox(ctx context.Context) error {
	return r.outbox.Retry(ctx)
}

// GetNoteRepository returns the NoteRepository instance
func (r *mariaDBRepository) GetNoteRepository() portRepo.Note {
	if r.qweryTx != nil {
		return implRepo.NewNoteRepository(r.qweryTx)
	}
	return r.NoteRepository
}

// DoInTransaction executes a function in a transaction
func (r *mariaDBRepository) DoInTransaction(ctx context.Context, fn func(repo outbound.Repository) (any, error)) (out any, err error) {

	var tx *qwery.Tx
	registry := r

	if r.qweryTx == nil {

		// begin a new transaction
		tx, err = r.qwery.BeginTransaction(ctx)
		if err != nil {
			return nil, fail.Wrapf(err, "failed to begin transaction")
		}

		// defer the function to handle the transaction
		defer func() {
			handleErr := r.handleTransaction(ctx, tx, &err)
			if handleErr != nil {
				err = handleErr
				return
			}

			// publish the outbox entries to the target publisher
			registry.outbox.qwery = r.qweryTx
			registry.outbox.PublishBuffered(ctx)
		}()

		registry = &mariaDBRepository{
			cfg:     r.cfg,
			log:     r.log,
			qwery:   r.qwery,
			qweryTx: tx,
			outbox:  NewOutbox(r.cfg, r.log, tx, true, r.outbox.publishers),
		}
	}

	// execute the function in the transaction
	out, err = fn(registry)
	if err != nil {
		return nil, fail.Wrapf(err, "failed to execute function in transaction")
	}

	return
}

// handleTransaction handles the transaction
func (r *mariaDBRepository) handleTransaction(ctx context.Context, tx *qwery.Tx, errIn *error) (errOut error) {

	if p := recover(); p != nil {

		r.log.WithContext(ctx).Debug("panic occurred, rolling back transaction")

		err := tx.Rollback()
		if err != nil {
			errOut = fail.Wrapf(err, "failed to rollback transaction")
		}
		panic(p) // re-throw panic after Rollback

	} else if *errIn != nil {

		r.log.WithContext(ctx).Debug("error occurred, rolling back transaction")

		err := tx.Rollback()
		if err != nil {
			errOut = fail.Wrapf(err, "failed to rollback transaction")
		}

	} else {

		r.log.WithContext(ctx).Debug("committing transaction")

		err := tx.Commit()
		if err != nil {
			errOut = fail.Wrapf(err, "failed to commit transaction")
		}

		r.log.WithContext(ctx).Debug("transaction committed")

	}
	return errOut

}