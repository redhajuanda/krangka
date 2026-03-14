package middleware

import "github.com/gofiber/fiber/v3"

func SecurityHeader() fiber.Handler {

	return func(c fiber.Ctx) error {
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("Strict-Transport-Security", "max-age=16070400; includeSubDomains")
		c.Set("Cache-Control", "no-store")
		c.Set("Referrer-Policy", "no-referrer")
		c.Set("Content-Security-Policy", "default-src 'self' http: https: data: blob: 'unsafe-inline'")
		c.Set("Permissions-Policy", "fullscreen=(self), geolocation=(), microphone=()")
		c.Set("X-Permitted-Cross-Domain-Policies", "none")

		return c.Next()
	}
}