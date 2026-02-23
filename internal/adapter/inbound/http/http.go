package http

import (
	"context"
	"time"

	"github.com/redhajuanda/komon/logger"
	"github.com/redhajuanda/komon/tracer"
	"github.com/redhajuanda/krangka/configs"

	"github.com/gofiber/fiber/v2"
)

type HTTP struct {
	cfg      *configs.Config
	log      logger.Logger
	router   *fiber.App
	handlers []Handler
}

type Handler interface {
	RegisterRoutes(app *fiber.App)
}

// NewHTTP creates a new instance of HTTP server
func NewHTTP(cfg *configs.Config, log logger.Logger, handlers []Handler) *HTTP {
	return &HTTP{
		cfg: cfg,
		log: log,
		router: fiber.New(fiber.Config{
			EnablePrintRoutes:     cfg.Http.EnablePrintRoutes,
			ErrorHandler:          ErrorHandlers(cfg, log),
			DisableStartupMessage: cfg.Http.DisableStartupMessage,
			ReadTimeout:           cfg.Http.ReadTimeout,
			WriteTimeout:          cfg.Http.WriteTimeout,
			IdleTimeout:           cfg.Http.IdleTimeout,
		}),
		handlers: handlers,
	}
}

// OnStart starts the HTTP server and handles graceful shutdown
// It blocks until a shutdown signal is received or an error occurs
func (h *HTTP) OnStart(ctx context.Context) error {

	// Setup trace provider
	err := tracer.SetTraceProvider(h.cfg.Otel.URL, h.cfg.App.Name, h.cfg.Otel.Exporter, h.cfg.Otel.SampleRate)
	if err != nil {
		h.log.Errorf("failed to set trace provider: %v", err)
	}

	h.log.SkipSource().Infof("Starting HTTP server on port %s", h.cfg.Http.Port)
	h.log.SkipSource().Info("Server will gracefully shutdown on SIGINT/SIGTERM")

	// Register all custom routes
	h.RegisterRoutes()
	// Register swagger routes
	h.RegisterSwaggerRoutes()
	// Register health check routes
	h.RegisterHealthCheckRoutes()
	// Register metrics routes
	h.RegisterMetricsRoutes()

	// Start HTTP server in goroutine so we can handle signals and graceful shutdown
	go func() {
		if err := h.router.Listen(":" + h.cfg.Http.Port); err != nil {
			h.log.Errorf("Server stopped: %v", err)
		}
	}()

	// Give the server a moment to start listening
	time.Sleep(200 * time.Millisecond)
	h.log.SkipSource().Infof("HTTP server is ready and accepting connections on port %s", h.cfg.Http.Port)

	return nil

}

// OnStop gracefully shuts down the HTTP server
func (h *HTTP) OnStop(ctx context.Context) error {

	h.log.SkipSource().Info("Shutting down HTTP server...")

	// Shutdown the HTTP server
	if err := h.router.ShutdownWithContext(ctx); err != nil {
		h.log.SkipSource().Errorf("HTTP server shutdown failed: %v", err)
		return err
	}

	h.log.SkipSource().Info("HTTP server shut down successfully")

	return nil

}
