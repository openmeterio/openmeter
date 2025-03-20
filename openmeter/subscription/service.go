package subscription

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type Service interface {
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
	// Get the subscription with the given ID
	Get(ctx context.Context, subscriptionID models.NamespacedID) (Subscription, error)
	// GetView returns a full view of the subscription with the given ID
	GetView(ctx context.Context, subscriptionID models.NamespacedID) (SubscriptionView, error)
	// List lists the subscriptions matching the set criteria
	List(ctx context.Context, params ListSubscriptionsInput) (SubscriptionList, error)
	// GetAllForCustomerSince returns all subscriptions for the given customer that are active or scheduled to start after the given timestamp
	GetAllForCustomerSince(ctx context.Context, customerID models.NamespacedID, at time.Time) ([]Subscription, error)
}

type WorkflowService interface {
	CreateFromPlan(ctx context.Context, inp CreateSubscriptionWorkflowInput, plan Plan) (SubscriptionView, error)
	EditRunning(ctx context.Context, subscriptionID models.NamespacedID, customizations []Patch, timing Timing) (SubscriptionView, error)
	ChangeToPlan(ctx context.Context, subscriptionID models.NamespacedID, inp ChangeSubscriptionWorkflowInput, plan Plan) (current Subscription, new SubscriptionView, err error)
	Restore(ctx context.Context, subscriptionID models.NamespacedID) (Subscription, error)
}

type CreateSubscriptionWorkflowInput struct {
	ChangeSubscriptionWorkflowInput
	Namespace  string
	CustomerID string
}

type ChangeSubscriptionWorkflowInput struct {
	Timing
	models.MetadataModel
	Name        string
	Description *string
}
