package repositories

import (
	"context"

	"github.com/redhajuanda/komon/pagination"
	"github.com/redhajuanda/krangka/internal/core/domain"
)

type Todo interface {
	// GetTodoByID retrieves a todo item by its ID
	GetTodoByID(ctx context.Context, id string) (*domain.Todo, error)
	// CreateTodo creates a new todo item
	CreateTodo(ctx context.Context, todo *domain.Todo) error
	// UpdateTodo updates an existing todo item
	UpdateTodo(ctx context.Context, todo *domain.Todo) error
	// DeleteTodo deletes a todo item by its ID
	DeleteTodo(ctx context.Context, id string) error
	// ListTodos retrieves a list of todo items with pagination
	ListTodos(ctx context.Context, req *domain.TodoFilter, pagination *pagination.Pagination) (*[]domain.Todo, error)
}
