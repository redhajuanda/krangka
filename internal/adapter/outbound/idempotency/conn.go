package idempotency

import (
	"context"

	"github.com/redhajuanda/komon/common"
	komonidem "github.com/redhajuanda/komon/idempotency"
	"github.com/redhajuanda/komon/logger"
	"github.com/redhajuanda/krangka/internal/core/port/outbound"
)

// Idempotency wraps the komon Redis-backed idempotency store.
type Idempotency struct {
	*komonidem.Store
}

// Param configures Redis for idempotency (aligned with cache / dlock Redis settings).
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

// NewIdempotency creates an Idempotency backed by komon.
func NewIdempotency(param Param, log logger.Logger) *Idempotency {
	ctx := context.Background()
	keyOpt := komonidem.WithKeyPrefix("idempotency:")
	s, err := komonidem.NewRedis(ctx, komonidem.RedisOption{
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
	}, keyOpt)
	if err != nil {
		log.Fatalf("failed to create idempotency: %v", err)
	}
	return &Idempotency{Store: s}
}

// Close releases Redis resources owned by this adapter.
func (i *Idempotency) Close() error {
	if i.Store == nil {
		return nil
	}
	return i.Store.Close()
}

var _ outbound.Idempotency = (*Idempotency)(nil)
