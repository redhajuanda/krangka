package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/redhajuanda/komon/fail"
	"github.com/redhajuanda/komon/pagination"
	"github.com/redhajuanda/komon/tracer"
	"github.com/redhajuanda/krangka/internal/core/domain"
	"github.com/redhajuanda/krangka/shared/failure"
	"github.com/redhajuanda/qwery"
)

type todoRepository struct {
	qwery qwery.Runable
}

func NewTodoRepository(qwery qwery.Runable) *todoRepository {
	return &todoRepository{qwery: qwery}
}

// GetTodoByID retrieves a todo item by its ID
func (r *todoRepository) GetTodoByID(ctx context.Context, id string) (*domain.Todo, error) {

	ctx, span := tracer.Trace(ctx)
	defer span.End()

	var todo domain.Todo

	query := `
		SELECT 
			id, 
			title, 
			description, 
			done
		FROM todos
		WHERE id = {{ .id }} 
		AND deleted_at = 0
	`

	err := r.qwery.
		RunRaw(query).
		WithParam("id", id).
		ScanStruct(&todo).
		Query(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fail.Wrap(err).WithFailure(failure.ErrTodoNotFound)
		}
		return nil, fail.Wrap(err)
	}

	return &todo, nil

}

// CreateTodo creates a new todo item
func (r *todoRepository) CreateTodo(ctx context.Context, todo *domain.Todo) error {

	ctx, span := tracer.Trace(ctx)
	defer span.End()

	query := `
		INSERT INTO todos (id, title, description, done) 
		VALUES ({{ .id }}, {{ .title }}, {{ .description }}, {{ .done }})
	`

	err := r.qwery.
		RunRaw(query).
		WithParams(todo).
		Query(ctx)

	if err != nil {
		return fail.Wrap(err)
	}

	return nil

}

// UpdateTodo updates an existing todo item
func (r *todoRepository) UpdateTodo(ctx context.Context, todo *domain.Todo) error {

	ctx, span := tracer.Trace(ctx)
	defer span.End()

	query := `
		UPDATE todos 
		SET title = {{ .title }}, description = {{ .description }}, done = {{ .done }} 
		WHERE id = {{ .id }} 
		AND deleted_at = 0
	`

	err := r.qwery.
		RunRaw(query).
		WithParams(todo).
		Query(ctx)

	if err != nil {
		return fail.Wrap(err)
	}

	return nil

}

// DeleteTodo deletes a todo item by its ID
func (r *todoRepository) DeleteTodo(ctx context.Context, id string) error {

	ctx, span := tracer.Trace(ctx)
	defer span.End()

	query := `
		UPDATE todos 
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

// ListTodos retrieves a list of todo items with pagination
func (r *todoRepository) ListTodos(ctx context.Context, req *domain.TodoFilter, pagination *pagination.Pagination) (*[]domain.Todo, error) {

	ctx, span := tracer.Trace(ctx)
	defer span.End()

	todos := make([]domain.Todo, 0)

	query := `
		SELECT 
			id, 
			title, 
			description, 
			done, 
			created_at, 
			updated_at, 
			deleted_at
		FROM todos
		WHERE deleted_at = 0
		{{ if .search }} AND (title LIKE CONCAT('%', {{ .search }}, '%') OR description LIKE CONCAT('%', {{ .search }}, '%')) {{ end }}
		{{ if .is_done }} AND done = {{ .is_done }} {{ end }}
	`

	err := r.qwery.
		RunRaw(query).
		WithParams(map[string]any{
			"search":  req.Search,
			"is_done": req.IsDone,
		}).
		WithPagination(pagination).
		WithOrderBy("-created_at", "id").
		ScanStructs(&todos).
		Query(ctx)

	if err != nil {
		return nil, fail.Wrap(err)
	}

	return &todos, nil

}
