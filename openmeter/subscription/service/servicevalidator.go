package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) validateCanContinue(ctx context.Context, view subscription.SubscriptionView) error {
	// Check if there are any future subscriptions scheduled for this customer
	// If so, we should not allow continuing this subscription
	futureSubscriptions, err := s.SubscriptionRepo.GetAllForCustomerSince(ctx, models.NamespacedID{
		ID:        view.Customer.ID,
		Namespace: view.Subscription.Namespace,
	}, view.Subscription.ActiveFrom)
	if err != nil {
		return fmt.Errorf("failed to get future subscriptions: %w", err)
	}

	// Filter out the current subscription from the list
	futureSubscriptions = lo.Filter(futureSubscriptions, func(sub subscription.Subscription, _ int) bool {
		return sub.ID != view.Subscription.ID
	})

	if len(futureSubscriptions) > 0 {
		return models.NewGenericConflictError(
			fmt.Errorf("cannot continue subscription as there are future subscriptions scheduled for this customer"),
		)
	}

	return nil
}

// validateTimelineWithSpec validates that a new subscription spec fits into the existing timeline
func (s *service) validateTimelineWithSpec(ctx context.Context, namespace string, customerId string, spec subscription.SubscriptionSpec, currentSubID *string) error {
	// Let's build a timeline of every already scheduled subscription
	scheduled, err := s.SubscriptionRepo.GetAllForCustomerSince(ctx, models.NamespacedID{
		ID:        customerId,
		Namespace: namespace,
	}, spec.ActiveFrom)
	if err != nil {
		return fmt.Errorf("failed to get scheduled subscriptions: %w", err)
	}

	// If we're updating an existing subscription, filter it out from the timeline check
	if currentSubID != nil {
		scheduled = lo.Filter(scheduled, func(sub subscription.Subscription, _ int) bool {
			return sub.ID != *currentSubID
		})
	}

	// Sanity check, validate that the scheduled timeline is consistent
	if err := subscription.ValidateSubscriptionTimeline(scheduled); err != nil {
		return fmt.Errorf("inconsistency error: %w", err)
	}

	// Now check that the new Spec also fits into the timeline
	if err := subscription.ValidateSubscriptionTimelineWithNew(scheduled, spec, namespace); err != nil {
		return err
	}

	return nil
}
