package handler

import (
	"github.com/redhajuanda/komon/fail"
	"github.com/redhajuanda/komon/logger"
	"github.com/redhajuanda/krangka/configs"
	"github.com/redhajuanda/krangka/internal/adapter/inbound/http/handler/dto"
	"github.com/redhajuanda/krangka/internal/adapter/inbound/http/response"
	"github.com/redhajuanda/krangka/internal/core/port/inbound"

	"github.com/gofiber/fiber/v3"
)

// NoteHandler is a struct that encapsulates the configuration, logger, and service
// for handling HTTP requests related to notes in the application. It provides methods to interact with the
// application's note services, such as creating, updating, deleting, and retrieving notes.
type NoteHandler struct {
	cfg *configs.Config
	log logger.Logger
	svc inbound.Note
}

// NewNoteHandler creates a new instance of NoteHandler with the provided configuration, logger, and service.
// It initializes the handler with the necessary dependencies for handling HTTP requests.
func NewNoteHandler(cfg *configs.Config, log logger.Logger, svc inbound.Note) *NoteHandler {
	return &NoteHandler{
		cfg: cfg,
		log: log,
		svc: svc,
	}
}

// RegisterRoutes registers the HTTP routes for the NoteHandler.
func (h *NoteHandler) RegisterRoutes(app fiber.Router) {

	app.Get("/notes/:id", h.GetNoteByID)
	app.Post("/notes", h.CreateNote)
	app.Put("/notes/:id", h.UpdateNote)
	app.Delete("/notes/:id", h.DeleteNote)
	app.Get("/notes", h.ListNotes)

}

// GetNoteByID godoc
// @Summary      Get Note by ID
// @Description  Retrieves a note by its id
// @Tags         Notes
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Note ID"
// @Success      200  {object}  response.ResponseSuccess{data=dto.ResGetNoteByID}  "Note retrieved successfully"
// @Failure      400  {object}  response.ResponseFailed{}   "Bad Request"
// @Failure      404  {object}  response.ResponseFailed{}   "Not Found"
// @Failure      500  {object}  response.ResponseFailed{}   "Internal Server Error"
// @Router       /notes/{id} [get]
func (h *NoteHandler) GetNoteByID(c fiber.Ctx) error {

	var (
		req dto.ReqGetNoteByID
		res dto.ResGetNoteByID
		ctx = c.Context()
	)

	if err := c.Bind().URI(&req); err != nil {
		return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
	}

	if err := req.Validate(); err != nil {
		return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
	}

	note, err := h.svc.GetNoteByID(ctx, req.ID)
	if err != nil {
		return err
	}

	res.Transform(note)

	return response.SuccessOK(c, res, "Note retrieved successfully")

}

// CreateNote godoc
// @Summary      Create Note
// @Description  Creates a new note
// @Tags         Notes
// @Accept       json
// @Produce      json
// @Param        note  body      dto.ReqCreateNote  true  "Note data"
// @Success      201   {object}  response.ResponseSuccess{data=dto.ResCreateNote}  "Note created successfully"
// @Failure      400   {object}  response.ResponseFailed{}   "Bad Request"
// @Failure      500   {object}  response.ResponseFailed{}   "Internal Server Error"
// @Router       /notes [post]
func (h *NoteHandler) CreateNote(c fiber.Ctx) error {

	var (
		req = dto.ReqCreateNote{}
		res = dto.ResCreateNote{}
		ctx = c.Context()
	)

	if err := c.Bind().Body(&req); err != nil {
		return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
	}

	if err := req.Validate(); err != nil {
		return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
	}

	note := req.Transform()

	err := h.svc.CreateNote(ctx, note)
	if err != nil {
		return err
	}

	res.Transform(note)

	return response.SuccessCreated(c, res, "Note created successfully")

}

// UpdateNote godoc
// @Summary      Update Note
// @Description  Updates an existing note by ID
// @Tags         Notes
// @Accept       json
// @Produce      json
// @Param        id    path      string            true  "Note ID"
// @Param        note  body      dto.ReqUpdateNote true  "Note data"
// @Success      200   {object}  response.ResponseSuccess{}  "Note updated successfully"
// @Failure      400   {object}  response.ResponseFailed{}   "Bad Request"
// @Failure      404   {object}  response.ResponseFailed{}   "Not Found"
// @Failure      500   {object}  response.ResponseFailed{}   "Internal Server Error"
// @Router       /notes/{id} [put]
func (h *NoteHandler) UpdateNote(c fiber.Ctx) error {

	var (
		req dto.ReqUpdateNote
		ctx = c.Context()
	)

	if err := c.Bind().URI(&req); err != nil {
		return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
	}

	if err := c.Bind().Body(&req); err != nil {
		return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
	}

	if err := req.Validate(); err != nil {
		return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
	}

	note := req.Transform()

	if err := h.svc.UpdateNote(ctx, note); err != nil {
		return err
	}

	return response.SuccessOK(c, nil, "Note updated successfully")

}

// DeleteNote godoc
// @Summary      Delete Note
// @Description  Deletes a note by ID
// @Tags         Notes
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Note ID"
// @Success      200  {object}  response.ResponseSuccess{}  "Note deleted successfully"
// @Failure      400  {object}  response.ResponseFailed{}   "Bad Request"
// @Failure      404  {object}  response.ResponseFailed{}   "Not Found"
// @Failure      500  {object}  response.ResponseFailed{}   "Internal Server Error"
// @Router       /notes/{id} [delete]
func (h *NoteHandler) DeleteNote(c fiber.Ctx) error {

	var (
		req dto.ReqDeleteNote
		ctx = c.Context()
	)

	if err := c.Bind().URI(&req); err != nil {
		return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
	}

	if err := req.Validate(); err != nil {
		return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
	}

	if err := h.svc.DeleteNote(ctx, req.ID); err != nil {
		return err
	}

	return response.SuccessOK(c, nil, "Note deleted successfully")

}

// ListNotes godoc
// @Summary      List Notes
// @Description  Retrieves a list of notes with optional filtering and pagination
// @Tags         Notes
// @Accept       json
// @Produce      json
// @Param        request  query     dto.ReqListNote  false  "Request"
// @Success      200      {object}  response.ResponseSuccess{data=dto.ResListNote}  "Notes retrieved successfully"
// @Failure      400      {object}  response.ResponseFailed{}   "Bad Request"
// @Failure      500      {object}  response.ResponseFailed{}   "Internal Server Error"
// @Router       /notes [get]
func (h *NoteHandler) ListNotes(c fiber.Ctx) error {

	var (
		req dto.ReqListNote
		res dto.ResListNote
		ctx = c.Context()
	)

	if err := c.Bind().Query(&req); err != nil {
		return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
	}

	if err := req.Validate(); err != nil {
		return fail.Wrap(err).WithFailure(fail.ErrBadRequest)
	}

	filter := req.Transform()
	pagination := req.Pagination

	notes, err := h.svc.ListNote(ctx, filter, &pagination)
	if err != nil {
		return err
	}

	res.Transform(notes)

	return response.SuccessOKWithPagination(c, res, pagination)

}
