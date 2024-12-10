package subscriptionhandler

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/subscription"
)

type GetUpcomingLineItemsInput struct {
	SubscriptionView subscription.SubscriptionView
	StartFrom        *time.Time

	Customer billing.ProfileWithCustomerDetails
}

func GetUpcomingLineItems(ctx context.Context, in GetUpcomingLineItemsInput) ([]billing.Line, error) {
	// Given we are event-driven we should at least yield one line item. If that's assigned to
	// a new invoice, this function can be triggered again.

	slices.SortFunc(in.SubscriptionView.Phases, func(i, j subscription.SubscriptionPhaseView) int {
		switch {
		case i.SubscriptionPhase.ActiveFrom.Before(j.SubscriptionPhase.ActiveFrom):
			return -1
		case i.SubscriptionPhase.ActiveFrom.After(j.SubscriptionPhase.ActiveFrom):
			return 1
		default:
			return 0
		}
	})

	// Let's identify the first phase that is invoicable
	firstInvoicablePhaseIdx, found := findFirstInvoicablePhase(in.SubscriptionView.Phases, in.StartFrom)
	if !found {
		// There are no invoicable items in the subscription, so we can return an empty list
		// If the subscription has changed we will just recalculate the line items with the updated
		// contents.
		return nil, nil
	}

	// Let's find out the limit of the generation. As a rule of thumb, we should have at least one line per
	// invoicable item.

	switch in.Customer.Profile.WorkflowConfig.Collection.Alignment {
	case billing.AlignmentKindSubscription:
		// In this case, the end of the generation end will be
	default:
		return nil, fmt.Errorf("unsupported alignment type: %s", in.Customer.Profile.WorkflowConfig.Collection.Alignment)
	}

	for i := firstInvoicablePhaseIdx; i < len(in.SubscriptionView.Phases); i++ {
	}

	return nil, nil
}

func findFirstInvoicablePhase(phases []subscription.SubscriptionPhaseView, startFrom *time.Time) (int, bool) {
	// A phase is invoicable, if it has any items that has price objects set, and if it's activeFrom is before or equal startFrom
	// and the next phase's activeFrom is after startFrom.

	for i, phase := range phases {
		isBillable := false
		// TODO: maybe forEachRateCard or similar
		for _, items := range phase.ItemsByKey {
			for _, item := range items {
				if item.Spec.RateCard.Price != nil {
					isBillable = true
					break
				}
			}
		}

		if !isBillable {
			continue
		}

		if !phase.SubscriptionPhase.ActiveFrom.After(*startFrom) {
			if i == len(phases)-1 {
				return i, true
			}

			if phases[i+1].SubscriptionPhase.ActiveFrom.After(*startFrom) {
				return i, true
			}
		}
	}

	return -1, false
}
