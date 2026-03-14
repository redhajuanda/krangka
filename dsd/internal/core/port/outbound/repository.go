package outbound

//go:generate mockgen -source=repository.go -destination=../../../mocks/outbound/mock_repository.go -package=mocks

import (
	"context"

	"gitlab.sicepat.tech/pka/sds/internal/core/port/outbound/repositories"
	"github.com/redhajuanda/qwery"
)

type Repository interface {
	// DoInTransaction executes a function in a transaction
	DoInTransaction(ctx context.Context, fn func(repo Repository) (any, error)) (any, error)
	// PublishOutbox publishes an outbox event
	PublishOutbox(ctx context.Context, target PublisherTarget, topic string, payload qwery.JSONMap) error
	// RetryOutbox retries an outbox event
	RetryOutbox(ctx context.Context) error
	// GetNoteRepository returns the NoteRepository instance
	GetNoteRepository() repositories.Note
}