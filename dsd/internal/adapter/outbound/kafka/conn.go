package kafka

import (
	"github.com/IBM/sarama"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"

	"github.com/redhajuanda/komon/logger"
)

type ParamPublisher struct {
	Brokers      []string
	DebugEnabled bool
	TraceEnabled bool
}

type ParamSubscriber struct {
	Brokers       []string
	ConsumerGroup string
	DebugEnabled  bool
	TraceEnabled  bool
}

// NewPublisher creates a new instance of Kafka publisher
func NewPublisher(param ParamPublisher, log logger.Logger) *kafka.Publisher {

	// Configure Sarama (underlying Kafka client)
	saramaCfg := sarama.NewConfig()
	saramaCfg.Producer.Return.Successes = true
	saramaCfg.Producer.Return.Errors = true
	saramaCfg.Version = sarama.DefaultVersion

	// Create publisher config
	publisherConfig := kafka.PublisherConfig{
		Brokers:               param.Brokers,
		Marshaler:             kafka.DefaultMarshaler{},
		OverwriteSaramaConfig: saramaCfg,
	}

	publisher, err := kafka.NewPublisher(
		publisherConfig,
		watermill.NewStdLogger(param.DebugEnabled, param.TraceEnabled),
	)
	if err != nil {
		log.Fatalf("failed to create publisher: %v", err)
	}

	return publisher
}

// NewSubscriber creates a new instance of Kafka subscriber
func NewSubscriber(param ParamSubscriber, log logger.Logger) *kafka.Subscriber {

	saramaCfg := kafka.DefaultSaramaSubscriberConfig()

	saramaCfg.Consumer.Return.Errors = true
	saramaCfg.Version = sarama.DefaultVersion // Use stable Kafka version

	subscriber, err := kafka.NewSubscriber(
		kafka.SubscriberConfig{
			Brokers:               param.Brokers,
			Unmarshaler:           kafka.DefaultMarshaler{},
			ConsumerGroup:         param.ConsumerGroup,
			OverwriteSaramaConfig: saramaCfg,
		},
		watermill.NewStdLogger(param.DebugEnabled, param.TraceEnabled),
	)
	if err != nil {
		log.Fatalf("failed to create subscriber: %v", err)
	}

	return subscriber
}