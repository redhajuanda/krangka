package outbound

//go:generate mockgen -source=dlock.go -destination=../../../mocks/outbound/mock_dlock.go -package=mocks_outbound

import "context"

// DLocker distributed locker interface
type DLocker interface {
	TryLock(ctx context.Context, id string, ttl int) error
	Lock(ctx context.Context, id string, ttl int) error
	Unlock(ctx context.Context, id string) error
	Close() error
}
