package plansubscription

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
)

type PlanSubscriptionService interface {
	Create(ctx context.Context, request CreateSubscriptionRequest) (subscription.Subscription, error)
	Migrate(ctx context.Context, request MigrateSubscriptionRequest) (SubscriptionChangeResponse, error)
	Change(ctx context.Context, request ChangeSubscriptionRequest) (SubscriptionChangeResponse, error)
}

// Generic response where a customer's subscription is changed to a different one.
type SubscriptionChangeResponse struct {
	Current subscription.Subscription
	New     subscription.SubscriptionView
}

type MigrateSubscriptionRequest struct {
	ID            models.NamespacedID
	TargetVersion int
}

type ChangeSubscriptionRequest struct {
	ID            models.NamespacedID
	WorkflowInput subscription.ChangeSubscriptionWorkflowInput
	PlanInput     PlanInput
}

type CreateSubscriptionRequest struct {
	WorkflowInput subscription.CreateSubscriptionWorkflowInput
	PlanInput     PlanInput
}
