package subscription

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/models"
)

type QueryService interface {
	// Get the subscription with the given ID
	Get(ctx context.Context, subscriptionID models.NamespacedID) (Subscription, error)
	// GetView returns a full view of the subscription with the given ID
	GetView(ctx context.Context, subscriptionID models.NamespacedID) (SubscriptionView, error)
	// List lists the subscriptions matching the set criteria
	List(ctx context.Context, params ListSubscriptionsInput) (SubscriptionList, error)
	// ExpandViews expands the subscriptions to views
	ExpandViews(ctx context.Context, subs []Subscription) ([]SubscriptionView, error)
}

type CommandService interface {
	HookService

	// Create a new subscription accotding to the given spec
	Create(ctx context.Context, namespace string, spec SubscriptionSpec) (Subscription, error)
	// Update the subscription with the given ID to the target spec
	Update(ctx context.Context, subscriptionID models.NamespacedID, target SubscriptionSpec) (Subscription, error)
	// Delete a scheduled subscription with the given ID
	Delete(ctx context.Context, subscriptionID models.NamespacedID) error
	// Cancel a running subscription at the provided time
	Cancel(ctx context.Context, subscriptionID models.NamespacedID, timing Timing) (Subscription, error)
	// Continue a canceled subscription (effectively undoing the cancellation)
	Continue(ctx context.Context, subscriptionID models.NamespacedID) (Subscription, error)
	// UpdateAnnotations updates the annotations of a subscription
	UpdateAnnotations(ctx context.Context, subscriptionID models.NamespacedID, annotations models.Annotations) (*Subscription, error)
}

type Service interface {
	QueryService
	CommandService
}

type HookService interface {
	RegisterHook(SubscriptionCommandHook) error
}
