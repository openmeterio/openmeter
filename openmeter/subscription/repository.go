package subscription

import "context"

type Repository interface {
	// Returns the current customer subscription
	GetCustomerSubscription(ctx context.Context, customerID string) (Subscription, error)
}
