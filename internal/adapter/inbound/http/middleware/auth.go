package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/redhajuanda/komon/fail"
	"github.com/redhajuanda/krangka/shared/libctx"
)

func Auth() fiber.Handler {

	return func(c fiber.Ctx) error {

		authorization := c.Get(fiber.HeaderAuthorization)
		if authorization == "" {
			return fail.New("Unauthorized").WithFailure(fail.ErrUnauthorized)
		}

		if !strings.Contains(authorization, "Bearer") {
			return fail.New("Unauthorized").WithFailure(fail.ErrUnauthorized)
		}

		authorizations := strings.Split(authorization, " ")
		if len(authorizations) != 2 {
			return fail.New("Unauthorized").WithFailure(fail.ErrUnauthorized)
		}

		ctx, err := libctx.SetClaims(c.Context(), authorizations[1])
		if err != nil {
			return fail.Wrap(err).WithFailure(fail.ErrUnauthorized)
		}
		c.SetContext(ctx)
		return c.Next()

	}

}
