package middleware

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redhajuanda/komon/logger"
)

const LocalsStartTime = "start_time"

func LoggerMiddleware(log logger.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Locals(LocalsStartTime, time.Now())

		err := c.Next()

		if err != nil {
			return err
		}

		status := c.Response().StatusCode()
		latency := time.Since(c.Locals(LocalsStartTime).(time.Time))

		log.SkipSource().
			WithContext(c.Context()).
			WithParam("method", c.Method()).
			WithParam("path", c.Path()).
			WithParam("status", status).
			WithParam("latency", fmt.Sprintf("%dms", latency.Milliseconds())).
			Infof("%s %s %d", c.Method(), c.Path(), status)

		return nil
	}
}
