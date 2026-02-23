package subscriber

import (
	"context"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redhajuanda/komon/logger"
	"github.com/redhajuanda/krangka/configs"
	"github.com/redhajuanda/krangka/internal/core/port/outbound"
)

type Handler interface {
	RegisterRoutes(router *message.Router)
}

// Subscriber is a worker that subscribes to the note.created, note.updated, and note.deleted events
type Subscriber struct {
	cfg         *configs.Config
	log         logger.Logger
	router      *message.Router
	idempotency outbound.Idempotency
	handlers    []Handler
}

// NewSubscriber creates a new Subscriber instance
func NewSubscriber(cfg *configs.Config, log logger.Logger, idempotency outbound.Idempotency, handlers []Handler, closeTimeout time.Duration) *Subscriber {

	logger := watermill.NewStdLogger(false, false)
	router, err := message.NewRouter(message.RouterConfig{CloseTimeout: closeTimeout}, logger)
	if err != nil {
		log.Fatal(err)
	}

	return &Subscriber{
		cfg:         cfg,
		log:         log,
		router:      router,
		idempotency: idempotency,
		handlers:    handlers,
	}

}

// OnStart starts the Subscriber worker
func (w *Subscriber) OnStart(ctx context.Context) error {

	// register routes
	w.RegisterRoutes()

	// start router
	go func() {
		if err := w.router.Run(context.WithoutCancel(ctx)); err != nil {
			w.log.Errorf("failed to run router: %v", err)
		}
	}()

	w.log.Infof("worker subscriber note started")
	return nil

}

// OnStop stops the Subscriber worker
func (w *Subscriber) OnStop(ctx context.Context) error {

	return w.router.Close()

}
