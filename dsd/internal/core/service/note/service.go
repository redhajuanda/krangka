package note

import (
	"context"
	"time"

	"github.com/redhajuanda/komon/fail"
	"github.com/redhajuanda/komon/logger"
	"github.com/redhajuanda/komon/pagination"
	"github.com/redhajuanda/komon/tracer"
	"gitlab.sicepat.tech/pka/sds/configs"
	"gitlab.sicepat.tech/pka/sds/internal/core/domain"
	"gitlab.sicepat.tech/pka/sds/internal/core/port/outbound"
	"github.com/redhajuanda/qwery"
)

type Service struct {
	cfg   *configs.Config
	log   logger.Logger
	repo  outbound.Repository
	cache outbound.Cache
}

// NewService creates a new note service
func NewService(cfg *configs.Config, log logger.Logger, repo outbound.Repository, cache outbound.Cache) *Service {
	return &Service{
		cfg:   cfg,
		log:   log,
		repo:  repo,
		cache: cache,
	}
}

// GetNoteByID retrieves a note item by its ID
func (s *Service) GetNoteByID(ctx context.Context, id string) (*domain.Note, error) {

	ctx, span := tracer.Trace(ctx)
	defer span.End()

	var (
		repoNote = s.repo.GetNoteRepository()
	)

	note, err := repoNote.GetNoteByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return note, nil

}

// CreateNote creates a new note item
func (s *Service) CreateNote(ctx context.Context, note *domain.Note) error {

	ctx, span := tracer.Trace(ctx)
	defer span.End()

	// define the function to be executed in the transaction
	var fnTx = func(repo outbound.Repository) (any, error) {

		var (
			repoNote = s.repo.GetNoteRepository()
		)

		// create note
		err := repoNote.CreateNote(ctx, note)
		if err != nil {
			return nil, err
		}

		// publish event using outbox pattern
		err = repo.PublishOutbox(ctx, outbound.PublisherTargetKafka, "note.created", qwery.JSONMap{
			"id":         note.ID,
			"title":      note.Title,
			"content":    note.Content,
			"created_at": note.CreatedAt,
			"updated_at": note.UpdatedAt,
		})
		return nil, err
	}

	// execute the function in the transaction
	_, err := s.repo.DoInTransaction(ctx, fnTx)
	if err != nil {
		return fail.Wrap(err)
	}

	return nil

}

// UpdateNote updates an existing note item
func (s *Service) UpdateNote(ctx context.Context, note *domain.Note) error {

	ctx, span := tracer.Trace(ctx)
	defer span.End()

	// define the function to be executed in the transaction
	var fnTx = func(repo outbound.Repository) (any, error) {

		var (
			repoNote = s.repo.GetNoteRepository()
		)

		// update note
		err := repoNote.UpdateNote(ctx, note)
		if err != nil {
			return nil, err
		}

		// publish event using outbox pattern
		err = repo.PublishOutbox(ctx, outbound.PublisherTargetKafka, "note.updated", qwery.JSONMap{
			"id":         note.ID,
			"title":      note.Title,
			"content":    note.Content,
			"created_at": note.CreatedAt,
			"updated_at": note.UpdatedAt,
		})
		return nil, err

	}

	// execute the function in the transaction
	_, err := s.repo.DoInTransaction(ctx, fnTx)
	if err != nil {
		return fail.Wrap(err)
	}

	return nil

}

// DeleteNote deletes a note item by its ID
func (s *Service) DeleteNote(ctx context.Context, id string) error {

	ctx, span := tracer.Trace(ctx)
	defer span.End()

	// define the function to be executed in the transaction
	var fnTx = func(repo outbound.Repository) (any, error) {
		var (
			repoNote = s.repo.GetNoteRepository()
		)

		// delete note
		err := repoNote.DeleteNote(ctx, id)
		if err != nil {
			return nil, err
		}

		// publish event using outbox pattern
		err = repo.PublishOutbox(ctx, outbound.PublisherTargetKafka, "note.deleted", qwery.JSONMap{
			"id":         id,
			"deleted_at": time.Now().Unix(),
		})
		if err != nil {
			return nil, err
		}

		return nil, err
	}

	// execute the function in the transaction
	_, err := s.repo.DoInTransaction(ctx, fnTx)
	if err != nil {
		return fail.Wrap(err)
	}

	return nil
}

// ListNote retrieves a list of note items with pagination
func (s *Service) ListNote(ctx context.Context, req *domain.NoteFilter, pagination *pagination.Pagination) (*[]domain.Note, error) {

	ctx, span := tracer.Trace(ctx)
	defer span.End()

	var (
		repoNote = s.repo.GetNoteRepository()
	)

	notes, err := repoNote.ListNote(ctx, req, pagination)
	if err != nil {
		return nil, err
	}
	return notes, nil

}