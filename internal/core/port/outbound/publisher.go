package outbound

import "github.com/ThreeDotsLabs/watermill/message"

// Publisher is a contract for the publisher
type Publisher message.Publisher

// Publishers is a map of publishers by target
type Publishers map[PublisherTarget]message.Publisher

// PublisherTarget is the target of the publisher
type PublisherTarget string

// PublisherTarget constants
const (
	// PublisherTargetRedisstream is the target of the redisstream publisher
	PublisherTargetRedisstream PublisherTarget = "redisstream"
	// PublisherTargetKafka is the target of the kafka publisher
	PublisherTargetKafka PublisherTarget = "kafka"
)
