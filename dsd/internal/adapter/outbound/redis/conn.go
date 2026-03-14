package redis

import (
	"fmt"
	"strings"

	"github.com/redhajuanda/komon/cache"
	_ "github.com/redhajuanda/komon/cache/redis"
	"github.com/redhajuanda/komon/logger"
)

// Redis is a wrapper around the Redis Sentinel connection
type Redis struct {
	cache.Cache
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

// New creates a new Redis connection
// Example: redis://<user>:<pass>@localhost:6379/<db>?minIdleConns=<minIdleConns>&poolSize=<poolSize>&prefix="
// Example: redis-sentinel://<user>:<pass>@localhost:26379/<db>?master=mymaster&minIdleConns=<minIdleConns>&poolSize=<poolSize>&prefix="
func New(param Param, log logger.Logger) *Redis {

	protocol := "redis"
	if param.Sentinel {
		protocol = "redis-sentinel"
	}
	url := fmt.Sprintf("%s://:%s@%s?db=%d&prefix=&minIdleConns=%d&poolSize=%d", protocol, param.Password, strings.Join(param.Hosts, ","), param.DB, param.MinIdleConns, param.PoolSize)

	rediscache, err := cache.New(url)
	if err != nil {
		log.Fatalf("failed to create redis cache: %v", err)
	}

	return &Redis{
		Cache: rediscache,
	}

}

// Close closes the Redis connection
func (r *Redis) Close() error {
	return r.Cache.Close()
}