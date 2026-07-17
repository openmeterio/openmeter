package subscriptions

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/labels"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func FromAPISubscriptionSortField(ctx context.Context, field string) (subscription.OrderBy, error) {
	switch field {
	case "id":
		return subscription.OrderByID, nil
	case "active_from":
		return subscription.OrderByActiveFrom, nil
	case "active_to":
		return subscription.OrderByActiveTo, nil
	default:
		return "", apierrors.NewUnsupportedSortFieldError(
			ctx, field, "id", "active_from", "active_to",
		)
	}
}

func ToAPIBillingSubscription(subscription subscription.Subscription) api.BillingSubscription {
	costBasisPins := make([]api.BillingSubscriptionCostBasisPin, 0, len(subscription.CostBasisPins))
	for _, pin := range subscription.CostBasisPins {
		costBasisPins = append(costBasisPins, api.BillingSubscriptionCostBasisPin{
			CustomCurrencyId: pin.CustomCurrencyID,
			InvoiceCurrency:  pin.InvoiceCurrency.String(),
			CostBasisId:      pin.CostBasis.ID,
		})
	}

	subscriptionAPI := api.BillingSubscription{
		Id:              subscription.ID,
		CustomerId:      subscription.CustomerId,
		InvoiceCurrency: subscription.InvoiceCurrency.String(),
		CostBasisMode:   api.BillingSubscriptionCostBasisMode(subscription.CostBasisMode.OrDefault()),
		CostBasisPins:   costBasisPins,
		BillingAnchor:   subscription.BillingAnchor,
		SettlementMode:  lo.ToPtr(api.BillingSettlementMode(subscription.SettlementMode)),
		Status:          api.BillingSubscriptionStatus(subscription.GetStatusAt(clock.Now())),
		Labels:          labels.FromMetadataAnnotations(subscription.Metadata, subscription.Annotations),
		CreatedAt:       subscription.CreatedAt,
		UpdatedAt:       subscription.UpdatedAt,
		DeletedAt:       subscription.DeletedAt,
	}

	// Only set if the subscription is created from a plan
	if subscription.PlanRef != nil {
		subscriptionAPI.PlanId = &subscription.PlanRef.Id
	}

	return subscriptionAPI
}

func FromAPIBillingSubscriptionEditTimingEnum(t api.BillingSubscriptionEditTimingEnum) (subscription.Timing, error) {
	switch string(t) {
	case "immediate":
		return subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)}, nil
	case "next_billing_cycle":
		return subscription.Timing{Enum: lo.ToPtr(subscription.TimingNextBillingCycle)}, nil
	default:
		return subscription.Timing{}, models.NewGenericValidationError(fmt.Errorf("invalid timing: %s", t))
	}
}

func FromAPIBillingSubscriptionEditTiming(t api.BillingSubscriptionEditTiming) (subscription.Timing, error) {
	// Try decoding as a custom RFC3339 datetime first, otherwise it would also decode as a "string enum"
	// and we'd never be able to distinguish enum vs datetime.
	if custom, err := t.AsDateTime(); err == nil {
		return subscription.Timing{Custom: &custom}, nil
	}

	enum, err := t.AsBillingSubscriptionEditTimingEnum()
	if err != nil {
		return subscription.Timing{}, models.NewGenericValidationError(fmt.Errorf("invalid timing"))
	}

	return FromAPIBillingSubscriptionEditTimingEnum(enum)
}

// FromAPIBillingSubscriptionCreate converts a create subscription request to a create subscription workflow input.
func FromAPIBillingSubscriptionCreate(
	namespace string,
	customerID customer.CustomerID,
	subscriptionName string,
	createSubscriptionRequest api.BillingSubscriptionCreate,
) (subscriptionworkflow.CreateSubscriptionWorkflowInput, error) {
	metadata, err := labels.ToMetadata(createSubscriptionRequest.Labels)
	if err != nil {
		return subscriptionworkflow.CreateSubscriptionWorkflowInput{}, err
	}

	workflowInput := subscriptionworkflow.CreateSubscriptionWorkflowInput{
		Namespace:     namespace,
		CustomerID:    customerID.ID,
		BillingAnchor: createSubscriptionRequest.BillingAnchor,
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Name:          subscriptionName,
			CostBasisMode: subscription.CostBasisMode(lo.FromPtr(createSubscriptionRequest.CostBasisMode)),
			Timing: subscription.Timing{
				// TODO: accept from request
				Enum: lo.ToPtr(subscription.TimingImmediate),
			},
			BillingAnchor: createSubscriptionRequest.BillingAnchor,
			MetadataModel: models.MetadataModel{
				Metadata: metadata,
			},
		},
	}

	return workflowInput, nil
}
