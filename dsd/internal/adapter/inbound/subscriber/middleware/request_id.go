package middleware

import (
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redhajuanda/komon/tracer"
)

// RequestID is a middleware that sets the request ID and correlation ID to the message context
func RequestID() message.HandlerMiddleware {
	return func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			ctx := msg.Context()
			ctx = tracer.SetRequestID(ctx, msg.Metadata.Get(tracer.RequestIDHeader))
			msg.Metadata.Set(tracer.RequestIDHeader, tracer.GetRequestID(ctx))
			ctx = tracer.SetCorrelationID(ctx, msg.Metadata.Get(tracer.CorrelationIDHeader))
			msg.Metadata.Set(tracer.CorrelationIDHeader, tracer.GetCorrelationID(ctx))
			msg.SetContext(ctx)
			return h(msg)
		}
	}
}