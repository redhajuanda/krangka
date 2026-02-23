package handler

import (
	"github.com/redhajuanda/komon/fail"
	"github.com/redhajuanda/komon/logger"
	"github.com/redhajuanda/krangka/configs"
	"github.com/redhajuanda/krangka/internal/adapter/inbound/http/handler/dto"
	"github.com/redhajuanda/krangka/internal/adapter/inbound/http/response"
	"github.com/redhajuanda/krangka/internal/core/port/inbound"

	"github.com/gofiber/fiber/v2"
)

// TodoHandler is a struct that encapsulates the configuration, logger, and service
// for handling HTTP requests related to todos in the application. It provides methods to interact with the
// application's todo services, such as creating, updating, deleting, and retrieving todos.
type TodoHandler struct {
	cfg *configs.Config
	log logger.Logger
	svc inbound.Todo
}

// NewTodoHandler creates a new instance of TodoHandler with the provided configuration, logger, and service.
// It initializes the handler with the necessary dependencies for handling HTTP requests.
func NewTodoHandler(cfg *configs.Config, log logger.Logger, svc inbound.Todo) *TodoHandler {
	return &TodoHandler{
		cfg: cfg,
		log: log,
		svc: svc,
	}
}

func (h *TodoHandler) RegisterRoutes(app *fiber.App) {

	app.Get("/todos/:id", h.GetTodoByID)
	app.Post("/todos", h.CreateTodo)
	app.Put("/todos/:id", h.UpdateTodo)
	app.Delete("/todos/:id", h.DeleteTodo)
	app.Get("/todos", h.ListTodos)

}

// GetTodoByID godoc
// @Summary      Get Todo by ID
// @Description  Retrieves a todo by its id
// @Tags         Todos
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Todo ID"
// @Success      200  {object}  response.ResponseSuccess{data=dto.ResGetTodoByID}  "Todo retrieved successfully"
// @Failure      400  {object}  response.ResponseFailed{}   "Bad Request"
// @Failure      404  {object}  response.ResponseFailed{}   "Not Found"
// @Failure      500  {object}  response.ResponseFailed{}   "Internal Server Error"
// @Router       /todos/{id} [get]
func (h *TodoHandler) GetTodoByID(c *fiber.Ctx) error {

	var (
		req dto.ReqGetTodoByID
		res dto.ResGetTodoByID
		ctx = c.UserContext()
	)

	if err := c.ParamsParser(&req); err != nil {
		return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
	}

	if err := req.Validate(); err != nil {
		return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
	}

	todo, err := h.svc.GetTodoByID(ctx, req.ID)
	if err != nil {
		return err
	}

	res.Transform(todo)

	return response.SuccessOK(c, res, "Todo retrieved successfully")

}

// CreateTodo godoc
// @Summary      Create Todo
// @Description  Creates a new todo
// @Tags         Todos
// @Accept       json
// @Produce      json
// @Param        todo  body      dto.ReqCreateTodo  true  "Todo data"
// @Success      201   {object}  response.ResponseSuccess{data=dto.ResCreateTodo}  "Todo created successfully"
// @Failure      400   {object}  response.ResponseFailed{}   "Bad Request"
// @Failure      500   {object}  response.ResponseFailed{}   "Internal Server Error"
// @Router       /todos [post]
func (h *TodoHandler) CreateTodo(c *fiber.Ctx) error {

	var (
		req = dto.ReqCreateTodo{}
		res = dto.ResCreateTodo{}
		ctx = c.UserContext()
	)

	if err := c.BodyParser(&req); err != nil {
		return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
	}

	if err := req.Validate(); err != nil {
		return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
	}

	todo := req.Transform()

	err := h.svc.CreateTodo(ctx, todo)
	if err != nil {
		return err
	}

	res.Transform(todo)

	return response.SuccessCreated(c, res, "Todo created successfully")

}

// UpdateTodo godoc
// @Summary      Update Todo
// @Description  Updates an existing todo by ID
// @Tags         Todos
// @Accept       json
// @Produce      json
// @Param        id    path      string            true  "Todo ID"
// @Param        todo  body      dto.ReqUpdateTodo true  "Todo data"
// @Success      200   {object}  response.ResponseSuccess{}  "Todo updated successfully"
// @Failure      400   {object}  response.ResponseFailed{}   "Bad Request"
// @Failure      404   {object}  response.ResponseFailed{}   "Not Found"
// @Failure      500   {object}  response.ResponseFailed{}   "Internal Server Error"
// @Router       /todos/{id} [put]
func (h *TodoHandler) UpdateTodo(c *fiber.Ctx) error {

	var (
		req dto.ReqUpdateTodo
		ctx = c.UserContext()
	)

	if err := c.ParamsParser(&req); err != nil {
		return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
	}

	if err := c.BodyParser(&req); err != nil {
		return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
	}

	if err := req.Validate(); err != nil {
		return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
	}

	todo := req.Transform()

	if err := h.svc.UpdateTodo(ctx, todo); err != nil {
		return err
	}

	return response.SuccessOK(c, nil, "Todo updated successfully")

}

// DeleteTodo godoc
// @Summary      Delete Todo
// @Description  Deletes a todo by ID
// @Tags         Todos
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Todo ID"
// @Success      200  {object}  response.ResponseSuccess{}  "Todo deleted successfully"
// @Failure      400  {object}  response.ResponseFailed{}   "Bad Request"
// @Failure      404  {object}  response.ResponseFailed{}   "Not Found"
// @Failure      500  {object}  response.ResponseFailed{}   "Internal Server Error"
// @Router       /todos/{id} [delete]
func (h *TodoHandler) DeleteTodo(c *fiber.Ctx) error {

	var (
		req dto.ReqDeleteTodo
		ctx = c.UserContext()
	)

	if err := c.ParamsParser(&req); err != nil {
		return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
	}

	if err := req.Validate(); err != nil {
		return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
	}

	if err := h.svc.DeleteTodo(ctx, req.ID); err != nil {
		return err
	}

	return response.SuccessOK(c, nil, "Todo deleted successfully")

}

// ListTodos godoc
// @Summary      List Todos
// @Description  Retrieves a list of todos with optional filtering and pagination
// @Tags         Todos
// @Accept       json
// @Produce      json
// @Param        request  query     dto.ReqListTodo  false  "Request"
// @Success      200      {object}  response.ResponseSuccess{data=dto.ResListTodo}  "Todos retrieved successfully"
// @Failure      400      {object}  response.ResponseFailed{}   "Bad Request"
// @Failure      500      {object}  response.ResponseFailed{}   "Internal Server Error"
// @Router       /todos [get]
func (h *TodoHandler) ListTodos(c *fiber.Ctx) error {

	var (
		req dto.ReqListTodo
		res dto.ResListTodo
		ctx = c.UserContext()
	)

	if err := c.QueryParser(&req); err != nil {
		return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
	}

	if err := req.Validate(); err != nil {
		return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
	}

	filter := req.Transform()
	pagination := req.Pagination

	todos, err := h.svc.ListTodo(ctx, filter, &pagination)
	if err != nil {
		return err
	}

	res.Transform(todos)

	return response.SuccessOKWithPagination(c, res, pagination)

}
