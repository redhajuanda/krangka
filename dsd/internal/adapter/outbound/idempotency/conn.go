package idempotency

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redhajuanda/komon/fail"
	"github.com/redhajuanda/komon/logger"
	"gitlab.sicepat.tech/pka/sds/internal/core/port/outbound"
	"github.com/redis/go-redis/v9"
)

const idempotencyKeyPrefix = "idempotency"

// Idempotency implements outbound.Idempotency using Redis SET NX with TTL.
type Idempotency struct {
	rdb *redis.Client
}

type Param struct {
	Sentinel     bool
	MasterName   string
	Username     string
	Password     string
	Hosts        []string
	DB           int
	MinIdleConns int
	PoolSize     int
}

// NewIdempotency creates an Idempotency that uses a dedicated Redis client.
// Uses the same Param as the cache Redis for config consistency.
func NewIdempotency(param Param, log logger.Logger) *Idempotency {

	rdb, err := initRedisClient(param)
	if err != nil {
		log.Fatalf("failed to create idempotency: %v", err)
	}
	return &Idempotency{rdb: rdb}

}

// initRedisClient initializes the Redis client
func initRedisClient(param Param) (*redis.Client, error) {
	if param.Sentinel {
		rdb := redis.NewFailoverClient(&redis.FailoverOptions{
			SentinelAddrs: param.Hosts,
			MasterName:    param.MasterName,
			Username:      param.Username,
			Password:      param.Password,
			DB:            param.DB,
			PoolSize:      param.PoolSize,
			MinIdleConns:  param.MinIdleConns,
		})
		if err := rdb.Ping(context.Background()).Err(); err != nil {
			return nil, fail.Wrap(err)
		}
		return rdb, nil
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:         param.Hosts[0],
		Username:     param.Username,
		Password:     param.Password,
		DB:           param.DB,
		PoolSize:     param.PoolSize,
		MinIdleConns: param.MinIdleConns,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fail.Wrap(err)
	}
	return rdb, nil
}

// TryClaim implements outbound.Idempotency.
// Uses Redis SET key NX EX ttl — atomic "set if not exists" with expiry.
func (s *Idempotency) TryClaim(ctx context.Context, topic, messageID string, ttl time.Duration) (bool, error) {

	key := fmt.Sprintf("%s:%s:%s", idempotencyKeyPrefix, topic, messageID)
	_, err := s.rdb.SetArgs(ctx, key, "1", redis.SetArgs{Mode: "NX", TTL: ttl}).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil // key already exists
		}
		return false, fail.Wrapf(err, "idempotency TryClaim failed, key: %s", key)
	}
	return true, nil
}

// Close releases the Redis connection. Call when shutting down the worker.
func (s *Idempotency) Close() error {
	return s.rdb.Close()
}

// Ensure Idempotency implements the port
var _ outbound.Idempotency = (*Idempotency)(nil)