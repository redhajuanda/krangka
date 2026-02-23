package inbound

import "context"

type Outbox interface {
	RunOutbox(ctx context.Context) error
}
