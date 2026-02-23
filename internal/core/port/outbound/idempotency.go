package outbound

import (
	"context"
	"time"
)

// Idempotency provides atomic "claim" semantics to ensure a message is processed at most once.
// TryClaim atomically records that a message will be processed. If it returns true, the caller must process;
// if false, the message was already claimed (by this or another consumer) — skip and ACK.
type Idempotency interface {
	// TryClaim atomically claims the idempotency key for the given topic and message ID.
	// Returns true if claimed (caller should process), false if already processed (caller should skip/ACK).
	// Keys expire after ttl to limit storage growth.
	TryClaim(ctx context.Context, topic, messageID string, ttl time.Duration) (claimed bool, err error)
}
