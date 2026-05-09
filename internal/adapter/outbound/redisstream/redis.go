package redisstream

import (
	"context"

	"github.com/redis/go-redis/v9"

	"github.com/redhajuanda/komon/common"
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
	// SentinelMasterDialOverride routes master TCP dials to this host:port (local Docker + app on host).
	SentinelMasterDialOverride string
}

// initRedisClient initializes the Redis client
func initRedisClient(param ParamRedis) (*redis.Client, error) {
	return common.NewRedisClient(context.Background(), common.RedisOption{
		Sentinel:                   param.Sentinel,
		MasterName:                 param.MasterName,
		Username:                   param.Username,
		Password:                   param.Password,
		Hosts:                      param.Hosts,
		DB:                         param.DB,
		PoolSize:                   param.PoolSize,
		MinIdleCon:                 param.MinIdleConns,
		SentinelMasterDialOverride: param.SentinelMasterDialOverride,
	})
}
