package http

import (
	"fmt"
	"time"

	"github.com/redhajuanda/komon/fail"
	"github.com/redhajuanda/komon/logger"
	"github.com/redhajuanda/komon/tracer"
	"github.com/redhajuanda/krangka/configs"
	"github.com/redhajuanda/krangka/internal/adapter/inbound/http/middleware"
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
			errPublished   string
			errInternal    string
		)

		if internalErr, ok := err.(*fail.Fail); ok {

			if internalErr.OriginalError() == nil {
				log.WithContext(ctx).WithStack(internalErr).Error(internalErr)
			} else {
				log.WithContext(ctx).WithStack(internalErr.OriginalError()).Error(internalErr.OriginalError())
				errInternal = internalErr.OriginalError().Error()
			}

			utils.LocalDebug(cfg, internalErr.OriginalError())

			errPublished = internalErr.GetFailure().Message
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

		} else if fiberErr, ok := err.(*fiber.Error); ok {

			errPublished = fiberErr.Message
			responseFailed = response.ResponseFailed{
				Success:    false,
				Message:    fiberErr.Message,
				HTTPStatus: fiberErr.Code,
				Metadata: response.Metadata{
					RequestID:     c.GetRespHeader(tracer.RequestIDHeader),
					CorrelationID: c.GetRespHeader(tracer.CorrelationIDHeader),
				},
			}

		} else {

			is := fail.Wrap(err).WithFailure(fail.ErrInternalServer)

			log.WithContext(ctx).WithStack(err).Error(err)
			utils.LocalDebug(cfg, err)

			errPublished = is.GetFailure().Message
			errInternal = err.Error()
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

		}

		logRequest(c, log, responseFailed.HTTPStatus, errPublished, errInternal)

		return c.Status(responseFailed.HTTPStatus).JSON(responseFailed)
	}
}

func logRequest(c fiber.Ctx, log logger.Logger, status int, errPublished, errInternal string) {
	var latency time.Duration
	if t, ok := c.Locals(middleware.LocalsStartTime).(time.Time); ok {
		latency = time.Since(t)
	}

	entry := log.SkipSource().
		WithContext(c.Context()).
		WithParam("method", c.Method()).
		WithParam("path", c.Path()).
		WithParam("status", status).
		WithParam("latency", fmt.Sprintf("%dms", latency.Milliseconds())).
		WithParam("error_published", errPublished)

	if errInternal != "" {
		entry = entry.WithParam("error_internal", errInternal)
	}

	if status >= 500 {
		entry.Errorf("%s %s %d", c.Method(), c.Path(), status)
	} else {
		entry.Warnf("%s %s %d", c.Method(), c.Path(), status)
	}
}
