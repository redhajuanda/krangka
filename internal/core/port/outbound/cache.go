package outbound

import (
	"context"
	"time"

	"github.com/redhajuanda/komon/cache"
)

// Cache is a contract for the cache
type Cache interface {
	Set(ctx context.Context, key string, value any, expiration time.Duration) error
	Increment(ctx context.Context, key string, expiration time.Duration) (int64, error)
	Get(ctx context.Context, key string) ([]byte, error)
	GetObject(ctx context.Context, key string, doc any) error
	GetString(ctx context.Context, key string) (string, error)
	GetInt(ctx context.Context, key string) (int64, error)
	GetFloat(ctx context.Context, key string) (float64, error)
	Exist(ctx context.Context, key string) bool
	Delete(ctx context.Context, key string, opts ...cache.DeleteOptions) error
	GetKeys(ctx context.Context, pattern string) []string
	RemainingTime(ctx context.Context, key string) int
	Close() error
}
