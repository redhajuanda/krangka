package outbound

import (
	"context"

	"github.com/redhajuanda/krangka/internal/core/port/outbound/repositories"
	"github.com/redhajuanda/qwery"
)

type Repository interface {
	// DoInTransaction executes a function in a transaction
	DoInTransaction(ctx context.Context, fn func(repo Repository) (any, error)) (any, error)
	// PublishOutbox publishes an outbox event
	PublishOutbox(ctx context.Context, target PublisherTarget, topic string, payload qwery.JSONMap) error
	// RetryOutbox retries an outbox event
	RetryOutbox(ctx context.Context) error
	// GetTodoRepository returns the TodoRepository instance
	GetTodoRepository() repositories.Todo
	// GetNoteRepository returns the NoteRepository instance
	GetNoteRepository() repositories.Note
}
