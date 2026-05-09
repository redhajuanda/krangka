package outbound

//go:generate mockgen -source=dlock.go -destination=../../../mocks/outbound/mock_dlock.go -package=mocks_outbound

import (
	"context"
	"time"
)

// DLocker is the outbound contract for a distributed lock (duplicated from
// github.com/redhajuanda/komon/dlock so the core does not import komon).
// Implementations must match that method set; the Redis adapter returns komon's DLocker as this type.
type DLocker interface {
	// TryLock attempts to acquire the lock for id once. It returns
	// ErrNotAcquired immediately if another holder owns the lock.
	TryLock(ctx context.Context, id string, ttl time.Duration) error

	// Lock blocks until the lock for id is acquired, ctx is done, or the
	// configured retry budget is exhausted. Returns ctx.Err() on cancellation
	// and ErrNotAcquired when retries are exhausted.
	Lock(ctx context.Context, id string, ttl time.Duration) error

	// Unlock releases a lock previously acquired by this instance for id.
	Unlock(ctx context.Context, id string) error

	// Close releases any resources owned by the locker. It is safe to call
	// multiple times.
	Close() error
}
