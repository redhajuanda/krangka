package dlock

import (
	"context"

	"github.com/redhajuanda/komon/common"
	komondlock "github.com/redhajuanda/komon/dlock"
	"github.com/redhajuanda/komon/logger"
	"github.com/redhajuanda/krangka/internal/core/port/outbound"
)

// Dlock wraps the komon Redis-backed distributed locker.
type Dlock struct {
	komondlock.DLocker
}

var _ outbound.DLocker = (*Dlock)(nil)

// Param configures the Redis backing store for the lock.
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
}

// NewDlock builds a Redis-backed Dlock from application config.
func NewDlock(param Param, log logger.Logger) *Dlock {
	ctx := context.Background()
	opt := komondlock.RedisOption{
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
	}
	inner, err := komondlock.NewRedis(ctx, opt)
	if err != nil {
		log.Fatalf("failed to create dlock: %v", err)
	}

	log.Info("dlock initialized successfully")

	return &Dlock{DLocker: inner}
}
