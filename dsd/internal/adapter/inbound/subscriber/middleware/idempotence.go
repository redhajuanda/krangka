package middleware

import (
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"gitlab.sicepat.tech/pka/sds/internal/core/port/outbound"
)

// Idempotence returns a Watermill handler middleware that ensures each message is processed at most once.
// Uses the message UUID (outbox entry ID) as the idempotency key. Duplicate deliveries are ACKed without reprocessing.
func Idempotence(idempotency outbound.Idempotency, topic string, ttl time.Duration) message.HandlerMiddleware {
	return func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			ctx := msg.Context()
			messageID := msg.UUID
			if messageID == "" {
				messageID = msg.Metadata.Get("id")
			}
			topicKey := topic
			if topicKey == "" {
				topicKey = msg.Metadata.Get("topic")
			}

			claimed, err := idempotency.TryClaim(ctx, topicKey, messageID, ttl)
			if err != nil {
				return nil, err
			}
			if !claimed {
				// Already processed — ACK and skip
				return nil, nil
			}

			return h(msg)
		}
	}
}