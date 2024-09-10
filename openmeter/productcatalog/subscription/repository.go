package subscription

import (
	"context"
	"time"
)

type SubscriptionRepoCreateInput struct{}

type SubscriptionRepo interface {
	// Create a new subscription.
	Create(ctx context.Context, subscription SubscriptionRepoCreateInput) (Subscription, error)
}

type CustomerSubscriptionRepo interface {
	// GetActiveSubscriptionsAt returns the active subscriptions for a customer at the given time.
	//
	// Each customer can have multiple active subscriptions at a time, given that:
	// - At most one of them is trialing
	// - At most of them is non-trialing
	GetActiveSubscriptionsAt(ctx context.Context, customerID string, at time.Time) ([]Subscription, error)

	// GetEffectiveAt returns the subscription that is effective at the given time.
	//
	// An effective subscription is an active subscription. If there are multiple active subscriptions
	// the trialing subscription is effective.
	GetEffectiveAt(ctx context.Context, customerID string, at time.Time) (Subscription, error)

	// GetCurrentAt returns the subscription that is current at the given time.
	// The current subscription is a subscription that cannot be trialing.
	GetCurrentAt(ctx context.Context, customerID string, at time.Time) (Subscription, error)
}
