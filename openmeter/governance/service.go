package governance

import "context"

// Service evaluates feature access for customers by composing the customer, entitlement,
// and feature services. It owns no persistence of its own.
type Service interface {
	QueryAccess(ctx context.Context, input QueryAccessInput) (QueryResult, error)
}
