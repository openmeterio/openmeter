package subscriptions

import (
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
	models "github.com/openmeterio/openmeter/pkg/models"
)

func ConvertSubscriptionToAPISubscription(subscription subscription.Subscription) api.BillingSubscription {
	subscriptionAPI := api.BillingSubscription{
		Id:            subscription.ID,
		CustomerId:    subscription.CustomerId,
		BillingAnchor: subscription.BillingAnchor,
		Status:        api.BillingSubscriptionStatus(subscription.GetStatusAt(clock.Now())),
		CreatedAt:     &subscription.CreatedAt,
		UpdatedAt:     &subscription.UpdatedAt,
		DeletedAt:     subscription.DeletedAt,
	}

	// Only set if the subscription is created from a plan
	if subscription.PlanRef != nil {
		subscriptionAPI.PlanId = &subscription.PlanRef.Id
		subscriptionAPI.PlanKey = &subscription.PlanRef.Key
		subscriptionAPI.PlanVersion = &subscription.PlanRef.Version
	}

	return subscriptionAPI
}

// ConvertFromCreateSubscriptionRequestToCreateSubscriptionWorkflowInput converts a create subscription request to a create subscription workflow input
func ConvertFromCreateSubscriptionRequestToCreateSubscriptionWorkflowInput(namespace string, name string, createSubscriptionRequest api.BillingSubscriptionCreate) (subscriptionworkflow.CreateSubscriptionWorkflowInput, error) {
	workflowInput := subscriptionworkflow.CreateSubscriptionWorkflowInput{
		Namespace:     namespace,
		CustomerID:    createSubscriptionRequest.CustomerId,
		BillingAnchor: createSubscriptionRequest.BillingAnchor,
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Name: name,
			Timing: subscription.Timing{
				// TODO: accept from request
				Enum: lo.ToPtr(subscription.TimingImmediate),
			},
			BillingAnchor: createSubscriptionRequest.BillingAnchor,
			MetadataModel: models.MetadataModel{
				Metadata: models.Metadata(lo.FromPtrOr(createSubscriptionRequest.Labels, api.Labels{})),
			},
		},
	}

	return workflowInput, nil
}
