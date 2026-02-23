package dto

import (
	"time"

	"github.com/redhajuanda/komon/pagination"
	"github.com/redhajuanda/krangka/internal/core/domain"

	"github.com/go-playground/validator/v10"
	"github.com/oklog/ulid/v2"
)

type ReqGetTodoByID struct {
	ID string `json:"id" validate:"required"`
}

func (r *ReqGetTodoByID) Validate() error {
	var validate = validator.New()
	return validate.Struct(r)
}

type ResGetTodoByID struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Done        bool      `json:"done"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (r *ResGetTodoByID) Transform(todo *domain.Todo) {
	r.ID = todo.ID
	r.Title = todo.Title
	r.Description = todo.Description
	r.Done = todo.Done
	r.CreatedAt = todo.CreatedAt
	r.UpdatedAt = todo.UpdatedAt
}

type ReqCreateTodo struct {
	Title       string `json:"title" validate:"required"`
	Description string `json:"description" validate:"required"`
	Done        bool   `json:"done"`
}

func (r *ReqCreateTodo) Validate() error {
	var validate = validator.New()
	return validate.Struct(r)
}

func (r *ReqCreateTodo) Transform() *domain.Todo {
	return &domain.Todo{
		ID:          ulid.Make().String(),
		Title:       r.Title,
		Description: r.Description,
		Done:        r.Done,
	}
}

type ResCreateTodo struct {
	ID string `json:"id"`
}

func (r *ResCreateTodo) Transform(todo *domain.Todo) {
	r.ID = todo.ID
}

type ReqUpdateTodo struct {
	ID          string `params:"id" validate:"required" swaggerignore:"true"` // ignore in swagger because it's in the path not in the body
	Title       string `json:"title" validate:"required"`
	Description string `json:"description" validate:"required"`
	Done        bool   `json:"done"`
}

func (r *ReqUpdateTodo) Validate() error {
	var validate = validator.New()
	return validate.Struct(r)
}

func (r *ReqUpdateTodo) Transform() *domain.Todo {
	return &domain.Todo{
		ID:          r.ID,
		Title:       r.Title,
		Description: r.Description,
		Done:        r.Done,
	}
}

type ReqDeleteTodo struct {
	ID string `params:"id" validate:"required" swaggerignore:"true"` // ignore in swagger because it's in the path not in the body
}

func (r *ReqDeleteTodo) Validate() error {
	var validate = validator.New()
	return validate.Struct(r)
}

type ReqListTodo struct {
	pagination.Pagination
	Search string `query:"search" validate:"omitempty,max=100" form:"search"`
	IsDone *bool  `query:"is_done" validate:"omitempty" form:"is_done"` // Optional filter for done status
}

func (r *ReqListTodo) Validate() error {
	var validate = validator.New()
	return validate.Struct(r)
}

func (r *ReqListTodo) Transform() *domain.TodoFilter {
	return &domain.TodoFilter{
		Search: r.Search,
		IsDone: r.IsDone,
	}
}

type ListTodo struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Done        bool      `json:"done"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ResListTodo []ListTodo

func (r *ResListTodo) Transform(todos *[]domain.Todo) {
	for _, todo := range *todos {
		*r = append(*r, ListTodo{
			ID:          todo.ID,
			Title:       todo.Title,
			Description: todo.Description,
			Done:        todo.Done,
			CreatedAt:   todo.CreatedAt,
			UpdatedAt:   todo.UpdatedAt,
		})
	}
}
