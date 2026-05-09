package outbound

//go:generate mockgen -source=cache.go -destination=../../../mocks/outbound/mock_cache.go -package=mocks_outbound

import (
	"context"

	komoncache "github.com/redhajuanda/komon/cache"
)

// CacheOption configures a single cache operation (TTL, serializer, etc.).
// Type alias to komon cache.Option so implementations remain wire-compatible.
type CacheOption = komoncache.Option

// CacheDataSet is one key/value or hash field for batch/hash APIs.
// Type alias to komon cache.DataSet.
type CacheDataSet = komoncache.DataSet

// Cache is the outbound cache contract (aligned with github.com/redhajuanda/komon/cache.Cache).
type Cache interface {
	// Close closes the cache connection.
	Close() error

	Get(ctx context.Context, key string, dest any, opts ...CacheOption) error
	GetString(ctx context.Context, key string) (string, error)
	GetInt(ctx context.Context, key string) (int64, error)
	Set(ctx context.Context, key string, value any, opts ...CacheOption) error
	Del(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, keys ...string) (bool, error)

	MGet(ctx context.Context, keys []string, dest any, opts ...CacheOption) error
	SetMultiple(ctx context.Context, data []*CacheDataSet, opts ...CacheOption) error

	HSet(ctx context.Context, key string, data []*CacheDataSet, opts ...CacheOption) error
	HGet(ctx context.Context, key string, field string, dest any, opts ...CacheOption) error
	HGetAll(ctx context.Context, key string, dest any, opts ...CacheOption) error
	HMGetPipelined(ctx context.Context, requests map[string]string, opts ...CacheOption) (map[string][]byte, error)
	HMSetPipelined(ctx context.Context, writes map[string][]*CacheDataSet, opts ...CacheOption) error
	UnmarshalValue(raw []byte, dest any, opts ...CacheOption) error

	SetMember(ctx context.Context, key string, data []*CacheDataSet, opts ...CacheOption) error
	GetMember(ctx context.Context, key string, dest any, opts ...CacheOption) (int, error)

	DeleteWithPattern(ctx context.Context, pattern string) error

	// Increment uses a sliding TTL (reset on every call); IncrementFixed sets TTL only on first creation.
	Increment(ctx context.Context, key string, opts ...CacheOption) (int64, error)
	IncrementFixed(ctx context.Context, key string, opts ...CacheOption) (int64, error)
}
