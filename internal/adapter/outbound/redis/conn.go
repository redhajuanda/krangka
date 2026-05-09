package redis

import (
	"context"

	scache "github.com/redhajuanda/komon/cache"
	"github.com/redhajuanda/komon/common"
	"github.com/redhajuanda/komon/logger"
	"github.com/redhajuanda/krangka/internal/core/port/outbound"
)

// Redis exposes komon cache operations backed by Redis.
type Redis struct {
	scache.Cache
}

var _ outbound.Cache = (*Redis)(nil)

// Param configures Redis for the cache client.
type Param struct {
	Sentinel                   bool
	MasterName                 string
	Username                   string
	Password                   string
	Hosts                      []string
	DB                         int
	MinIdleConns               int
	PoolSize                   int
	SentinelMasterDialOverride string
	UseMsgPack                 bool
}

// NewRedis creates a komon-backed Redis cache.
func NewRedis(param Param, log logger.Logger) *Redis {
	ctx := context.Background()
	opt := scache.RedisOption{
		RedisOption: common.RedisOption{
			Sentinel:                   param.Sentinel,
			MasterName:                 param.MasterName,
			Username:                   param.Username,
			Password:                   param.Password,
			Hosts:                      param.Hosts,
			DB:                         param.DB,
			PoolSize:                   param.PoolSize,
			MinIdleCon:                 param.MinIdleConns,
			SentinelMasterDialOverride: param.SentinelMasterDialOverride,
		},
		UseMsgPack: param.UseMsgPack,
	}
	rdb, err := scache.NewRedis(ctx, opt)
	if err != nil {
		log.Fatalf("failed to create redis cache: %v", err)
	}
	return &Redis{
		Cache: rdb,
	}
}
