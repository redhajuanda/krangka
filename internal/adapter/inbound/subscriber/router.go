package subscriber

import "github.com/redhajuanda/krangka/internal/adapter/inbound/subscriber/middleware"

// RegisterRoutes registers the handlers for the Subscriber
func (w *Subscriber) RegisterRoutes() {

	// add middleware
	w.router.AddMiddleware(middleware.RequestID())
	w.router.AddMiddleware(middleware.Idempotence(w.idempotency, "", w.cfg.Event.Idempotency.TTL))

	// register handlers
	for _, handler := range w.handlers {
		handler.RegisterRoutes(w.router)
	}

}
