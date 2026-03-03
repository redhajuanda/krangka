package middleware

import (
	"context"

	"github.com/redhajuanda/komon/tracer"

	"github.com/gofiber/fiber/v3"
	"github.com/oklog/ulid/v2"
)

func RequestIDMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {

		rid := ulid.Make().String()
		cid := ulid.Make().String()

		// Set to Fiber context (c.Locals)
		c.Locals(tracer.RequestIDKey, rid)
		c.Locals(tracer.CorrelationIDKey, cid)

		// Set to Go std context (c.Context)
		ctx := context.WithValue(c.Context(), tracer.RequestIDKey, rid)
		ctx = context.WithValue(ctx, tracer.CorrelationIDKey, cid)

		c.SetContext(ctx)

		// Optionally set header (biar bisa di-trace via curl/postman)
		c.Set(tracer.RequestIDHeader, rid)
		c.Set(tracer.CorrelationIDHeader, cid)

		return c.Next()
	}
}
