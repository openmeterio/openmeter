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
	// Cancel a running subscription at the provided time
	Cancel(ctx context.Context, subscriptionID models.NamespacedID, at time.Time) (Subscription, error)
	// Continue a canceled subscription (effectively undoing the cancellation)
	Continue(ctx context.Context, subscriptionID models.NamespacedID) (Subscription, error)
	// Get the subscription with the given ID
	Get(ctx context.Context, subscriptionID models.NamespacedID) (Subscription, error)
	// GetView returns a full view of the subscription with the given ID
	GetView(ctx context.Context, subscriptionID models.NamespacedID) (SubscriptionView, error)
}

type WorkflowService interface {
	CreateFromPlan(ctx context.Context, inp CreateFromPlanInput) (SubscriptionView, error)
	EditRunning(ctx context.Context, subscriptionID models.NamespacedID, customizations []Patch) (SubscriptionView, error)
}

type CreateFromPlanInput struct {
	Namespace   string
	ActiveFrom  time.Time
	CustomerID  string
	Name        string
	Description *string
	models.AnnotatedModel

	Plan PlanRef
}
