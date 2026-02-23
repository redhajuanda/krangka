package redisstream

import (
	"context"
	"log"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/redis/go-redis/v9"
)

type ParamRedis struct {
	Sentinel     bool
	MasterName   string
	Username     string
	Password     string
	Hosts        []string
	DB           int
	MinIdleConns int
	PoolSize     int
}

// initRedisClient initializes the Redis client
func initRedisClient(param ParamRedis) (*redis.Client, error) {

	var (
		rdb *redis.Client
	)

	if param.Sentinel {
		rdb = redis.NewFailoverClient(&redis.FailoverOptions{
			SentinelAddrs: param.Hosts,
			MasterName:    param.MasterName,
			Password:      param.Password,
			DB:            param.DB,
			PoolSize:      param.PoolSize,
			MinIdleConns:  param.MinIdleConns,
		})
	} else {
		rdb = redis.NewClient(&redis.Options{
			Addr:         param.Hosts[0],
			Password:     param.Password,
			DB:           param.DB,
			PoolSize:     param.PoolSize,
			MinIdleConns: param.MinIdleConns,
		})
	}

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return rdb, nil
}

type ParamPublisher struct {
	ParamRedis
	DefaultMaxlen int64
}

// NewPublisher creates a new instance of Redis stream publisher
func NewPublisher(param ParamPublisher) *redisstream.Publisher {

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

type ParamSubscriber struct {
	ParamRedis
	SubscriberID  string
	ConsumerGroup string
}

// NewSubscriber creates a new instance of Redis stream subscriber
func NewSubscriber(param ParamSubscriber) *redisstream.Subscriber {

	rdb, err := initRedisClient(param.ParamRedis)
	if err != nil {
		log.Fatalf("failed to init sentinel: %v", err)
	}

	subscriber, err := redisstream.NewSubscriber(
		redisstream.SubscriberConfig{
			Client:        rdb,
			Unmarshaller:  redisstream.DefaultMarshallerUnmarshaller{},
			ConsumerGroup: param.ConsumerGroup,
		},
		watermill.NewStdLogger(false, false),
	)
	if err != nil {
		log.Fatalf("failed to create subscriber: %v", err)
	}

	return subscriber

}
