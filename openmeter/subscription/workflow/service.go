package subscriptionworkflow

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Service interface {
	CreateFromPlan(ctx context.Context, inp CreateSubscriptionWorkflowInput, plan subscription.Plan) (subscription.SubscriptionView, error)
	EditRunning(ctx context.Context, subscriptionID models.NamespacedID, customizations []subscription.Patch, timing subscription.Timing) (subscription.SubscriptionView, error)
	ChangeToPlan(ctx context.Context, subscriptionID models.NamespacedID, inp ChangeSubscriptionWorkflowInput, plan subscription.Plan) (current subscription.Subscription, new subscription.SubscriptionView, err error)
	Restore(ctx context.Context, subscriptionID models.NamespacedID) (subscription.Subscription, error)
}

type CreateSubscriptionWorkflowInput struct {
	ChangeSubscriptionWorkflowInput
	Namespace  string
	CustomerID string
}

type ChangeSubscriptionWorkflowInput struct {
	subscription.Timing
	models.MetadataModel
	Name        string
	Description *string
}
