package plansubscription

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
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
	Next    subscription.SubscriptionView
}

type MigrateSubscriptionRequest struct {
	ID            models.NamespacedID
	TargetVersion *int
	StartingPhase *string
}

type ChangeSubscriptionRequest struct {
	ID            models.NamespacedID
	WorkflowInput subscriptionworkflow.ChangeSubscriptionWorkflowInput
	PlanInput     PlanInput

	// Only used if existing plan is provided
	StartingPhase *string
}

type CreateSubscriptionRequest struct {
	WorkflowInput subscriptionworkflow.CreateSubscriptionWorkflowInput
	PlanInput     PlanInput
}
