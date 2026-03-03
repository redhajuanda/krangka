package middleware

import (
	"fmt"

	"github.com/cockroachdb/errors"
	"github.com/gofiber/fiber/v3"
)

func RecoverMiddleware() fiber.Handler {
	return func(c fiber.Ctx) (err error) {
		defer func() {

			if r := recover(); r != nil {

				fmt.Printf("%+v\n", r)

				switch v := r.(type) {
				case string:
					err = fmt.Errorf("panic: %s", v)
				case error:
					err = fmt.Errorf("panic: %w", v)
				default:
					err = fmt.Errorf("panic: %#v", v)
				}

				err = errors.Wrap(err, "panic caught in middleware")

			}

		}()
		return c.Next()
	}
}
