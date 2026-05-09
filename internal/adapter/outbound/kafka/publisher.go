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
