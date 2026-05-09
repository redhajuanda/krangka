package middleware

import (
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

// CORS returns Fiber middleware that sets Cross-Origin Resource Sharing headers for browser clients.
// allowedOrigins lists allowed request origins
func CORS(allowedOrigins []string) fiber.Handler {
	return cors.New(cors.Config{
		AllowOriginsFunc: func(origin string) bool {
			u, err := url.Parse(origin)
			if err != nil {
				return false
			}

			host := u.Hostname()

			for _, allowed := range allowedOrigins {
				if strings.HasPrefix(allowed, "*.") {
					domain := strings.TrimPrefix(allowed, "*.")
					if host == domain || strings.HasSuffix(host, "."+domain) {
						return true
					}
				} else {
					if origin == allowed {
						return true
					}
				}
			}

			return false
		},
	})
}
