package dlock

import (
	"fmt"
	"strings"

	"github.com/redhajuanda/komon/lock"
	_ "github.com/redhajuanda/komon/lock/redis"
	"github.com/redhajuanda/komon/logger"
)

// DLock is a wrapper around the DLock connection
type DLock struct {
	lock.DLocker
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

// New creates a new distributed lock connection
// Example: redis://<user>:<pass>@localhost:6379/<db>?minIdleConns=<minIdleConns>&poolSize=<poolSize>
// Example: redis-sentinel://<user>:<pass>@localhost:26379/<db>?master=mymaster&minIdleConns=<minIdleConns>&poolSize=<poolSize>
func New(param Param, log logger.Logger) *DLock {

	protocol := "redis"
	if param.Sentinel {
		protocol = "redis-sentinel"
	}
	url := fmt.Sprintf("%s://:%s@%s?db=%d&prefix=&minIdleConns=%d&poolSize=%d", protocol, param.Password, strings.Join(param.Hosts, ","), param.DB, param.MinIdleConns, param.PoolSize)

	dLock, err := lock.New(url)
	if err != nil {
		log.Fatalf("failed to create dlock: %v", err)
	}
	return &DLock{
		DLocker: dLock,
	}

}

// Close closes the DLock connection
func (d *DLock) Close() error {
	return d.DLocker.Close()
}