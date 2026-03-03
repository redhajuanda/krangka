package dto

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/oklog/ulid/v2"
	"github.com/redhajuanda/komon/pagination"
	"github.com/redhajuanda/krangka/internal/core/domain"
)

type ReqGetNoteByID struct {
	ID string `uri:"id" validate:"required"`
}

func (r *ReqGetNoteByID) Validate() error {
	var validate = validator.New()
	return validate.Struct(r)
}

type ResGetNoteByID struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (r *ResGetNoteByID) Transform(note *domain.Note) {
	r.ID = note.ID
	r.Title = note.Title
	r.Content = note.Content
	r.CreatedAt = note.CreatedAt
	r.UpdatedAt = note.UpdatedAt
}

type ReqCreateNote struct {
	Title   string `json:"title" validate:"required"`
	Content string `json:"content" validate:"required"`
}

func (r *ReqCreateNote) Validate() error {
	var validate = validator.New()
	return validate.Struct(r)
}

func (r *ReqCreateNote) Transform() *domain.Note {
	return &domain.Note{
		ID:      ulid.Make().String(),
		Title:   r.Title,
		Content: r.Content,
	}
}

type ResCreateNote struct {
	ID string `json:"id"`
}

func (r *ResCreateNote) Transform(note *domain.Note) {
	r.ID = note.ID
}

type ReqUpdateNote struct {
	ID      string `uri:"id" validate:"required" swaggerignore:"true"` // ignore in swagger because it's in the path not in the body
	Title   string `json:"title" validate:"required"`
	Content string `json:"content" validate:"required"`
}

func (r *ReqUpdateNote) Validate() error {
	var validate = validator.New()
	return validate.Struct(r)
}

func (r *ReqUpdateNote) Transform() *domain.Note {
	return &domain.Note{
		ID:      r.ID,
		Title:   r.Title,
		Content: r.Content,
	}
}

type ReqDeleteNote struct {
	ID string `uri:"id" validate:"required" swaggerignore:"true"` // ignore in swagger because it's in the path not in the body
}

func (r *ReqDeleteNote) Validate() error {
	var validate = validator.New()
	return validate.Struct(r)
}

type ReqListNote struct {
	pagination.Pagination
	Search string `query:"search" validate:"omitempty,max=100" form:"search"`
}

func (r *ReqListNote) Validate() error {
	var validate = validator.New()
	return validate.Struct(r)
}

func (r *ReqListNote) Transform() *domain.NoteFilter {
	return &domain.NoteFilter{
		Search: r.Search,
	}
}

type ListNote struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ResListNote []ListNote

func (r *ResListNote) Transform(notes *[]domain.Note) {
	for _, note := range *notes {
		*r = append(*r, ListNote{
			ID:        note.ID,
			Title:     note.Title,
			Content:   note.Content,
			CreatedAt: note.CreatedAt,
			UpdatedAt: note.UpdatedAt,
		})
	}
}
