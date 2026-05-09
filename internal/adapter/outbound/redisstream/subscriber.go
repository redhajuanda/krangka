package redisstream

import (
	"context"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/redhajuanda/komon/logger"
)

type ParamSubscriber struct {
	ParamRedis
	SubscriberID  string
	ConsumerGroup string
}

// NewSubscriber creates a new instance of Redis stream subscriber
func NewSubscriber(param ParamSubscriber, log logger.Logger) *redisstream.Subscriber {

	rdb, err := initRedisClient(param.ParamRedis)
	if err != nil {
		log.Fatalf("failed to init sentinel: %v", err)
	}

	subscriber, err := redisstream.NewSubscriber(
		redisstream.SubscriberConfig{
			Client:        rdb,
			Unmarshaller:  redisstream.DefaultMarshallerUnmarshaller{},
			ConsumerGroup: param.ConsumerGroup,
			// Recreate the stream and consumer group on NOGROUP errors.
			// This can happen after a Redis Sentinel failover where the new
			// master hasn't replicated the consumer group from the old master.
			ShouldStopOnReadErrors: func(err error) bool {
				if strings.Contains(err.Error(), "NOGROUP") {
					log.Info("consumer group not found, recreating after sentinel failover...")

					createErr := rdb.XGroupCreateMkStream(
						context.Background(),
						param.ConsumerGroup,
						param.ConsumerGroup,
						"0",
					).Err()

					if createErr != nil && !strings.Contains(createErr.Error(), "BUSYGROUP") {
						log.Errorf("failed to recreate consumer group: %v", createErr)
					}

					time.Sleep(500 * time.Millisecond)
					return false
				}
				return false
			},
		},
		watermill.NewStdLogger(false, false),
	)
	if err != nil {
		log.Fatalf("failed to create subscriber: %v", err)
	}

	return subscriber

}
