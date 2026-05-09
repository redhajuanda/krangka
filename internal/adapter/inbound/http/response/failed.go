package response

import (
	"github.com/gofiber/fiber/v3"
	"github.com/redhajuanda/komon/tracer"
)

type ResponseFailed struct {
	Success    bool     `json:"success" example:"false"`
	Message    string   `json:"message"`
	Data       any      `json:"data"`
	ErrorCode  string   `json:"error_code"`
	HTTPStatus int      `json:"-"`
	Metadata   Metadata `json:"metadata"`
}

func Failed(c fiber.Ctx, httpcode int, errorCode string, msg string) error {
	return c.Status(httpcode).JSON(ResponseFailed{
		Success:    false,
		Message:    msg,
		HTTPStatus: httpcode,
		ErrorCode:  errorCode,
		Metadata: Metadata{
			RequestID:     c.GetRespHeader(tracer.RequestIDHeader),
			CorrelationID: c.GetRespHeader(tracer.CorrelationIDHeader),
		},
	})
}
