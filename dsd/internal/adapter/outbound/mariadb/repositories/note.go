package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/redhajuanda/komon/fail"
	"github.com/redhajuanda/komon/pagination"
	"github.com/redhajuanda/komon/tracer"
	"gitlab.sicepat.tech/pka/sds/internal/core/domain"
	"github.com/redhajuanda/qwery"
)

// noteRepository is a wrapper around the NoteRepository
type noteRepository struct {
	qwery qwery.Runable
}

// NewNoteRepository creates a new NoteRepository instance
func NewNoteRepository(qwery qwery.Runable) *noteRepository {
	return &noteRepository{qwery: qwery}
}

// GetNoteByID retrieves a note item by its ID
func (r *noteRepository) GetNoteByID(ctx context.Context, id string) (*domain.Note, error) {

	ctx, span := tracer.Trace(ctx)
	defer span.End()

	var note domain.Note

	query := `
		SELECT 
			id, 
			title, 
			content, 
			created_at, 
			updated_at, 
			deleted_at
		FROM notes
		WHERE id = {{ .id }} 
		AND deleted_at = 0
	`

	err := r.qwery.
		RunRaw(query).
		WithParam("id", id).
		ScanStruct(&note).
		Query(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fail.Wrap(err).WithFailure(fail.ErrNotFound)
		}
		return nil, fail.Wrap(err)
	}

	return &note, nil

}

// CreateNote creates a new note item
func (r *noteRepository) CreateNote(ctx context.Context, note *domain.Note) error {

	ctx, span := tracer.Trace(ctx)
	defer span.End()

	query := `
		INSERT INTO notes (id, title, content) VALUES 
		({{ .id }}, {{ .title }}, {{ .content }})
	`

	err := r.qwery.
		RunRaw(query).
		WithParams(note).
		Query(ctx)

	if err != nil {
		return fail.Wrap(err)
	}

	return nil

}

// UpdateNote updates an existing note item
func (r *noteRepository) UpdateNote(ctx context.Context, note *domain.Note) error {

	ctx, span := tracer.Trace(ctx)
	defer span.End()

	query := `
		UPDATE notes 
		SET title = {{ .title }}, content = {{ .content }} 
		WHERE id = {{ .id }} 
		AND deleted_at = 0
	`

	err := r.qwery.
		RunRaw(query).
		WithParams(note).
		Query(ctx)

	if err != nil {
		return fail.Wrap(err)
	}

	return nil

}

// DeleteNote deletes a note item by its ID
func (r *noteRepository) DeleteNote(ctx context.Context, id string) error {

	ctx, span := tracer.Trace(ctx)
	defer span.End()

	query := `
		UPDATE notes 
		SET deleted_at = UNIX_TIMESTAMP() 
		WHERE id = {{ .id }} 
		AND deleted_at = 0
	`

	err := r.qwery.
		RunRaw(query).
		WithParam("id", id).
		Query(ctx)

	if err != nil {
		return fail.Wrap(err)
	}

	return nil

}

// ListNote retrieves a list of note items with pagination
func (r *noteRepository) ListNote(ctx context.Context, req *domain.NoteFilter, pagination *pagination.Pagination) (*[]domain.Note, error) {

	ctx, span := tracer.Trace(ctx)
	defer span.End()

	notes := make([]domain.Note, 0)

	query := `
		SELECT 
			id, 
			title, 
			content, 
			created_at, 
			updated_at, 
			deleted_at
		FROM notes
		WHERE deleted_at = 0
		{{ if .search }} AND (title LIKE CONCAT('%', {{ .search }}, '%') OR content LIKE CONCAT('%', {{ .search }}, '%')) {{ end }}
	`

	err := r.qwery.
		RunRaw(query).
		WithParams(map[string]any{
			"search": req.Search,
		}).
		WithPagination(pagination).
		WithOrderBy("id"). // order by id is enough because the id is in ulid format (can be ordered)
		ScanStructs(&notes).
		Query(ctx)
	if err != nil {
		return nil, fail.Wrap(err)
	}
	return &notes, nil

}