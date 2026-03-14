package http

import (
	"encoding/json"

	"gitlab.sicepat.tech/pka/sds/internal/adapter/inbound/http/docs"
	"gitlab.sicepat.tech/pka/sds/internal/adapter/inbound/http/middleware"

	"github.com/gofiber/contrib/v3/swaggo"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// RegisterRoutes registers the routes for the HTTP server
func (h *HTTP) RegisterRoutes() {

	h.router.Use(middleware.SecurityHeader())
	h.router.Use(middleware.RequestIDMiddleware())
	h.router.Use(middleware.RecoverMiddleware())

	for _, handler := range h.handlers {
		handler.RegisterRoutes(h.router)
	}
}

// RegisterSwaggerRoutes registers the swagger routes for the HTTP server
func (h *HTTP) RegisterSwaggerRoutes() {

	// skip swagger in production
	if h.cfg.GetEnv().IsProd() {
		return
	}

	// Serve custom swagger spec with dynamic base path
	h.router.Get("/swagger/doc.json", h.customSwaggerSpec)

	// Serve swagger UI
	h.router.Get("/docs/*", swaggo.New(swaggo.Config{
		InstanceName:           h.cfg.App.Name,
		Title:                  h.cfg.App.Name,
		URL:                    "/swagger/doc.json",
		DeepLinking:            true,
		DocExpansion:           "none",
		DisplayRequestDuration: true,
		TryItOutEnabled:        true,
	}))

}

// customSwaggerSpec is a custom swagger spec for the HTTP server
func (h *HTTP) customSwaggerSpec(c fiber.Ctx) error {

	// Get the original swagger spec
	spec := docs.SwaggerInfo.ReadDoc()

	// Parse and modify the spec
	var swaggerSpec map[string]any
	if err := json.Unmarshal([]byte(spec), &swaggerSpec); err != nil {
		return err
	}

	return c.JSON(swaggerSpec)
}

// RegisterHealthCheckRoutes registers the health check routes for the HTTP server
func (h *HTTP) RegisterHealthCheckRoutes() {

	h.router.Get("/health", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

}

// RegisterMetricsRoutes registers the metrics routes for the HTTP server
func (h *HTTP) RegisterMetricsRoutes() {

	registry := prometheus.NewRegistry()
	// Register default Go collectors (memory, goroutines, etc.)
	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	h.router.Get("/metrics", func(c fiber.Ctx) error {

		handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
			ErrorLog:      nil,
			ErrorHandling: promhttp.ContinueOnError,
		})

		return adaptor.HTTPHandler(handler)(c)
	})
}