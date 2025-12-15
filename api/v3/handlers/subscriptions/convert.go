package subscriptions

import (
	"fmt"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/customer"
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
		Labels:        lo.ToPtr(api.Labels(subscription.Metadata)),
		CreatedAt:     &subscription.CreatedAt,
		UpdatedAt:     &subscription.UpdatedAt,
		DeletedAt:     subscription.DeletedAt,
	}

	// Only set if the subscription is created from a plan
	if subscription.PlanRef != nil {
		subscriptionAPI.PlanId = &subscription.PlanRef.Id
	}

	return subscriptionAPI
}

func ConvertBillingSubscriptionEditTimingEnumToSubscriptionTiming(t api.BillingSubscriptionEditTimingEnum) (subscription.Timing, error) {
	switch string(t) {
	case "immediate":
		return subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)}, nil
	case "next_billing_cycle":
		return subscription.Timing{Enum: lo.ToPtr(subscription.TimingNextBillingCycle)}, nil
	default:
		return subscription.Timing{}, models.NewGenericValidationError(fmt.Errorf("invalid timing: %s", t))
	}
}

func ConvertBillingSubscriptionEditTimingToSubscriptionTiming(t api.BillingSubscriptionEditTiming) (subscription.Timing, error) {
	// Try decoding as a custom RFC3339 datetime first, otherwise it would also decode as a "string enum"
	// and we'd never be able to distinguish enum vs datetime.
	if custom, err := t.AsBillingSubscriptionEditTiming1(); err == nil {
		return subscription.Timing{Custom: &custom}, nil
	}

	enum, err := t.AsBillingSubscriptionEditTimingEnum()
	if err != nil {
		return subscription.Timing{}, models.NewGenericValidationError(fmt.Errorf("invalid timing"))
	}

	return ConvertBillingSubscriptionEditTimingEnumToSubscriptionTiming(enum)
}

// ConvertFromCreateSubscriptionRequestToCreateSubscriptionWorkflowInput converts a create subscription request to a create subscription workflow input
func ConvertFromCreateSubscriptionRequestToCreateSubscriptionWorkflowInput(
	namespace string,
	customerID customer.CustomerID,
	subscriptionName string,
	createSubscriptionRequest api.BillingSubscriptionCreate,
) (subscriptionworkflow.CreateSubscriptionWorkflowInput, error) {
	workflowInput := subscriptionworkflow.CreateSubscriptionWorkflowInput{
		Namespace:     namespace,
		CustomerID:    customerID.ID,
		BillingAnchor: createSubscriptionRequest.BillingAnchor,
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Name: subscriptionName,
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
