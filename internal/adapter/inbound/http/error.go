package http

import (
	"github.com/redhajuanda/komon/fail"
	"github.com/redhajuanda/komon/logger"
	"github.com/redhajuanda/komon/tracer"
	"github.com/redhajuanda/krangka/configs"
	"github.com/redhajuanda/krangka/internal/adapter/inbound/http/response"
	"github.com/redhajuanda/krangka/shared/utils"

	"github.com/gofiber/fiber/v3"
)

// ErrorHandlers centralizes the error handling for the HTTP server
var ErrorHandlers = func(cfg *configs.Config, log logger.Logger) fiber.ErrorHandler {

	return func(c fiber.Ctx, err error) error {

		var (
			responseFailed = response.ResponseFailed{}
			ctx            = c.Context()
		)

		if internalErr, ok := err.(*fail.Fail); ok {

			if internalErr.OriginalError() == nil {
				log.WithContext(ctx).WithStack(internalErr).Error(internalErr)
			} else {
				log.WithContext(ctx).WithStack(internalErr.OriginalError()).Error(internalErr.OriginalError())
			}

			utils.LocalDebug(cfg, internalErr.OriginalError())

			responseFailed = response.ResponseFailed{
				Success:    false,
				Message:    internalErr.GetFailure().Message,
				Data:       internalErr.Data(),
				ErrorCode:  internalErr.GetFailure().Code,
				HTTPStatus: internalErr.GetFailure().HTTPStatus,
				Metadata: response.Metadata{
					RequestID:     c.GetRespHeader(tracer.RequestIDHeader),
					CorrelationID: c.GetRespHeader(tracer.CorrelationIDHeader),
				},
			}

		} else {

			is := fail.Wrap(err).WithFailure(fail.ErrInternalServer)
			responseFailed = response.ResponseFailed{
				Success:    false,
				Message:    is.GetFailure().Message,
				ErrorCode:  is.GetFailure().Code,
				HTTPStatus: is.GetFailure().HTTPStatus,
				Metadata: response.Metadata{
					RequestID:     c.GetRespHeader(tracer.RequestIDHeader),
					CorrelationID: c.GetRespHeader(tracer.CorrelationIDHeader),
				},
			}

			log.WithContext(ctx).WithStack(err).Error(err)

			utils.LocalDebug(cfg, err)

		}

		return c.Status(responseFailed.HTTPStatus).JSON(responseFailed)
	}
}
