package subscription

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/models"
)

type Repository interface {
	models.CadencedResourceRepo[Subscription]

	// Returns the current customer subscription
	GetCustomerSubscription(ctx context.Context, customerID models.NamespacedID) (Subscription, error)

	// Returns the subscription by ID
	GetSubscription(ctx context.Context, subscriptionID models.NamespacedID) (Subscription, error)

	// Create a new subscription
	CreateSubscription(ctx context.Context, namespace string, subscription CreateSubscriptionInput) (Subscription, error)

	// Patches
	// GetSubscriptionPatches returns the patches of a subscription
	GetSubscriptionPatches(ctx context.Context, subscriptionID models.NamespacedID) ([]SubscriptionPatch, error)

	// CreateSubscriptionPatches creates patches for a subscription
	CreateSubscriptionPatches(ctx context.Context, subscriptionID models.NamespacedID, patches []CreateSubscriptionPatchInput) ([]SubscriptionPatch, error)
}
