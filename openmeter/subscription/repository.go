package subscription

import "context"

type Repository interface {
	// Returns the current customer subscription
	GetCustomerSubscription(ctx context.Context, customerID string) (Subscription, error)

	// Returns the subscription by ID
	GetSubscription(ctx context.Context, subscriptionID string) (Subscription, error)

	// Create a new subscription
	CreateSubscription(ctx context.Context, subscription CreateSubscriptionInput) (Subscription, error)

	// Patches
	// GetSubscriptionPatches returns the patches of a subscription
	GetSubscriptionPatches(ctx context.Context, subscriptionID string) ([]SubscriptionPatch, error)
}
