package response

import (
	"net/http"
	"strings"

	"github.com/redhajuanda/komon/pagination"
	"github.com/redhajuanda/komon/tracer"

	"github.com/gofiber/fiber/v3"
)

// ResponseSuccess struct
type ResponseSuccess struct {
	Success  bool     `json:"success" example:"true"`
	Message  string   `json:"message" example:"success"`
	Data     any      `json:"data"`
	Metadata Metadata `json:"metadata"`
}

// Success responses with a success message and data
func Success(c fiber.Ctx, code int, data any, msg ...string) error {

	responseMsg := buildResponseMsg("Success", msg...)

	if data == nil {
		data = map[string]any{}
	}

	res := ResponseSuccess{
		Success: true,
		Message: responseMsg,
		Metadata: Metadata{
			RequestID:     c.GetRespHeader(tracer.RequestIDHeader),
			CorrelationID: c.GetRespHeader(tracer.CorrelationIDHeader),
		},
		Data: data,
	}
	return c.Status(code).JSON(res)

}

// SuccessOK returns code 200
func SuccessOK(c fiber.Ctx, data any, msg ...string) error {
	return Success(c, http.StatusOK, data, msg...)
}

// SuccessCreated returns code 201
func SuccessCreated(c fiber.Ctx, data any, msg ...string) error {
	return Success(c, http.StatusCreated, data, msg...)
}

func SuccessOKWithPagination(c fiber.Ctx, data any, pagination pagination.Pagination) error {

	if data == nil {
		data = map[string]any{}
	}

	res := ResponseSuccess{
		Success: true,
		Message: "Success",
		Metadata: Metadata{
			RequestID:     c.GetRespHeader(tracer.RequestIDHeader),
			CorrelationID: c.GetRespHeader(tracer.CorrelationIDHeader),
			Pagination:    pagination.Result,
		},
		Data: data,
	}
	return c.Status(http.StatusOK).JSON(res)

}

// Data is an alias for map
type Data map[string]any

func buildResponseMsg(defaultMsg string, msg ...string) string {

	if len(msg) == 0 {
		return defaultMsg
	}
	return strings.Join(msg, ", ")

}
