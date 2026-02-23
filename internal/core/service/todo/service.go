package todo

import (
	"context"

	"github.com/redhajuanda/komon/fail"
	"github.com/redhajuanda/komon/logger"
	"github.com/redhajuanda/komon/pagination"
	"github.com/redhajuanda/komon/tracer"
	"github.com/redhajuanda/krangka/configs"
	"github.com/redhajuanda/krangka/internal/core/domain"
	"github.com/redhajuanda/krangka/internal/core/port/outbound"
)

type Service struct {
	cfg   *configs.Config
	log   logger.Logger
	repo  outbound.Repository
	cache outbound.Cache
}

// NewService creates a new todo service
func NewService(cfg *configs.Config, log logger.Logger, repo outbound.Repository, cache outbound.Cache) *Service {
	return &Service{
		cfg:   cfg,
		log:   log,
		repo:  repo,
		cache: cache,
	}
}

// GetTodoByID retrieves a todo item by its ID
func (s *Service) GetTodoByID(ctx context.Context, id string) (*domain.Todo, error) {

	ctx, span := tracer.Trace(ctx)
	defer span.End()

	var (
		repoTodo = s.repo.GetTodoRepository()
		todo     *domain.Todo
	)

	todo, err := repoTodo.GetTodoByID(ctx, id)
	if err != nil {
		return nil, fail.Wrap(err)
	}
	return todo, nil

}

// CreateTodo creates a new item of todo
func (s *Service) CreateTodo(ctx context.Context, todo *domain.Todo) error {

	ctx, span := tracer.Trace(ctx)
	defer span.End()

	var (
		repoTodo = s.repo.GetTodoRepository()
	)

	return repoTodo.CreateTodo(ctx, todo)

}

// UpdateTodo updates an existing todo item
func (s *Service) UpdateTodo(ctx context.Context, todo *domain.Todo) error {

	ctx, span := tracer.Trace(ctx)
	defer span.End()

	var (
		repoTodo = s.repo.GetTodoRepository()
	)

	return repoTodo.UpdateTodo(ctx, todo)

}

// DeleteTodo deletes a todo item by its ID
func (s *Service) DeleteTodo(ctx context.Context, id string) error {

	ctx, span := tracer.Trace(ctx)
	defer span.End()

	var (
		repoTodo = s.repo.GetTodoRepository()
	)

	return repoTodo.DeleteTodo(ctx, id)

}

// ListTodo retrieves a list of todo items with pagination
func (s *Service) ListTodo(ctx context.Context, req *domain.TodoFilter, pagination *pagination.Pagination) (*[]domain.Todo, error) {

	ctx, span := tracer.Trace(ctx)
	defer span.End()

	var (
		repoTodo = s.repo.GetTodoRepository()
	)

	res, err := repoTodo.ListTodos(ctx, req, pagination)
	if err != nil {
		return nil, err
	}
	return res, err

}
