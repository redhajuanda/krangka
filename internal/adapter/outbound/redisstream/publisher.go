package redisstream

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/redhajuanda/komon/logger"
)

type ParamPublisher struct {
	ParamRedis
	DefaultMaxlen int64
}

// NewPublisher creates a new instance of Redis stream publisher
func NewPublisher(param ParamPublisher, log logger.Logger) *redisstream.Publisher {

	rdb, err := initRedisClient(param.ParamRedis)
	if err != nil {
		log.Fatalf("failed to init sentinel: %v", err)
	}

	publisher, err := redisstream.NewPublisher(
		redisstream.PublisherConfig{
			Client:        rdb,
			Marshaller:    redisstream.DefaultMarshallerUnmarshaller{},
			DefaultMaxlen: param.DefaultMaxlen,
		},
		watermill.NewStdLogger(false, false),
	)
	if err != nil {
		log.Fatalf("failed to create publisher: %v", err)
	}

	return publisher

}
