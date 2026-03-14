package outbound

//go:generate mockgen -source=publisher.go -destination=../../../mocks/outbound/mock_publisher.go -package=mocks_outbound

import "github.com/ThreeDotsLabs/watermill/message"

// Publisher is the emitting part of a Pub/Sub.
type Publisher interface {
	// Publish publishes provided messages to the given topic.
	//
	// Publish can be synchronous or asynchronous - it depends on the implementation.
	//
	// Most publisher implementations don't support atomic publishing of messages.
	// This means that if publishing one of the messages fails, the next messages will not be published.
	//
	// Publish does not work with a single Context.
	// Use the Context() method of each message instead.
	//
	// Publish must be thread safe.
	Publish(topic string, messages ...*message.Message) error
	// Close should flush unsent messages if publisher is async.
	Close() error
}

// Publishers is a map of publishers by target
type Publishers map[PublisherTarget]Publisher

// PublisherTarget is the target of the publisher
type PublisherTarget string

// PublisherTarget constants
const (
	// PublisherTargetRedisstream is the target of the redisstream publisher
	PublisherTargetRedisstream PublisherTarget = "redisstream"
	// PublisherTargetKafka is the target of the kafka publisher
	PublisherTargetKafka PublisherTarget = "kafka"
)
