package inbound

//go:generate mockgen -source=note.go -destination=../../../mocks/inbound/mock_note.go -package=mocks

import (
	"context"

	"github.com/redhajuanda/komon/pagination"
	"gitlab.sicepat.tech/pka/sds/internal/core/domain"
)

type Note interface {
	// GetNoteByID retrieves a note item by its ID
	GetNoteByID(ctx context.Context, id string) (*domain.Note, error)
	// CreateNote creates a new note item
	CreateNote(ctx context.Context, todo *domain.Note) error
	// UpdateNote updates an existing note item
	UpdateNote(ctx context.Context, todo *domain.Note) error
	// DeleteNote deletes a note item by its ID
	DeleteNote(ctx context.Context, id string) error
	// ListNote retrieves a list of note items with pagination
	ListNote(ctx context.Context, req *domain.NoteFilter, pagination *pagination.Pagination) (*[]domain.Note, error)
}