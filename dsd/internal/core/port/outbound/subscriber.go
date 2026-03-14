package outbound

//go:generate mockgen -source=subscriber.go -destination=../../../mocks/outbound/mock_subscriber.go -package=mocks

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
)

// Subscriber is the consuming part of the Pub/Sub.
type Subscriber interface {
	// Subscribe returns an output channel with messages from the provided topic.
	// The channel is closed after Close() is called on the subscriber.
	//
	// To receive the next message, `Ack()` must be called on the received message.
	// If message processing fails and the message should be redelivered `Nack()` should be called instead.
	//
	// When the provided ctx is canceled, the subscriber closes the subscription and the output channel.
	// The provided ctx is passed to all produced messages.
	// When Nack or Ack is called on the message, the context of the message is canceled.
	Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error)
	// Close closes all subscriptions with their output channels and flushes offsets etc. when needed.
	Close() error
}