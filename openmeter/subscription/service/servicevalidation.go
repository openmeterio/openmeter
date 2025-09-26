package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) validateCreate(ctx context.Context, cust customer.Customer, spec subscription.SubscriptionSpec) error {
	// Let's make sure the method was called properly
	if spec.CustomerId != cust.ID {
		return fmt.Errorf("customer ID mismatch: %s != %s", spec.CustomerId, cust.ID)
	}

	// Now, let's validate that create is possible

	// 1. Valiate the spec
	if err := spec.Validate(); err != nil {
		return fmt.Errorf("spec is invalid: %w", err)
	}

	// 2. Let's make sure Create is possible based on the transition rules
	if err := subscription.NewStateMachine(subscription.SubscriptionStatusInactive).CanTransitionOrErr(ctx, subscription.SubscriptionActionCreate); err != nil {
		return err
	}

	// 3. Let's make sure the customer has a subject if needed
	if _, err := cust.UsageAttribution.GetSubjectKey(); err != nil {
		if spec.HasEntitlements() {
			return models.NewGenericValidationError(errors.New("customer has no subject but subscription has entitlements"))
		}

		if spec.HasMeteredBillables() {
			return models.NewGenericValidationError(errors.New("customer has no subject but subscription has metered billables"))
		}
	}

	// 4. Let's make sure the currency is valid
	if spec.HasBillables() {
		if cust.Currency != nil && (string(*cust.Currency) != string(spec.Currency)) {
			return models.NewGenericValidationError(fmt.Errorf("currency mismatch: customer currency is %s, but subscription currency is %s", *cust.Currency, spec.Currency))
		}
	}

	// 5. Let's make sure there's no scheduling conflict (no overlapping subscriptions)
	multiSubscriptionEnabled, err := s.FeatureFlags.IsFeatureEnabled(ctx, subscription.MultiSubscriptionEnabledFF)
	if err != nil {
		return fmt.Errorf("failed to check if multi-subscription is enabled: %w", err)
	}

	if multiSubscriptionEnabled {
		// TODO[galexi]: Implement feature specific validation
	} else {
		// We're gonna validate uniqueness on the subscription level
		// Let's build a timeline of every already schedueld subscription
		scheduled, err := s.SubscriptionRepo.GetAllForCustomerSince(ctx, models.NamespacedID{
			ID:        spec.CustomerId,
			Namespace: cust.Namespace,
		}, clock.Now())
		if err != nil {
			return fmt.Errorf("failed to get scheduled subscriptions: %w", err)
		}

		scheduledInps := lo.Map(scheduled, func(i subscription.Subscription, _ int) subscription.CreateSubscriptionEntityInput {
			return i.AsEntityInput()
		})

		subscriptionTimeline := models.NewSortedCadenceList(scheduledInps)

		// Sanity check, lets validate that the scheduled timeline is consistent (without the new spec)
		if overlaps := subscriptionTimeline.GetOverlaps(); len(overlaps) > 0 {
			return errors.New("inconsistency error: already scheduled subscriptions are overlapping")
		}

		// Now let's check that the new Spec also fits into the timeline
		subscriptionTimeline = models.NewSortedCadenceList(append(scheduledInps, spec.ToCreateSubscriptionEntityInput(cust.Namespace)))

		if overlaps := subscriptionTimeline.GetOverlaps(); len(overlaps) > 0 {
			return subscription.ErrOnlySingleSubscriptionAllowed
		}
	}

	return nil
}

func (s *service) validateUpdate(ctx context.Context, currentView subscription.SubscriptionView, newSpec subscription.SubscriptionSpec) error {
	// Let's make sure edit is possible based on the transition rules
	if err := subscription.NewStateMachine(
		currentView.Subscription.GetStatusAt(clock.Now()),
	).CanTransitionOrErr(ctx, subscription.SubscriptionActionUpdate); err != nil {
		return err
	}

	// Fetch the customer & validate the customer
	cus, err := s.CustomerService.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerID: &customer.CustomerID{
			Namespace: currentView.Subscription.Namespace,
			ID:        currentView.Subscription.CustomerId,
		},
	})
	if err != nil {
		return err
	}

	if cus != nil && cus.IsDeleted() {
		return models.NewGenericPreConditionFailedError(
			fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID),
		)
	}

	if cus == nil {
		return fmt.Errorf("customer is nil")
	}

	if _, err := cus.UsageAttribution.GetSubjectKey(); err != nil {
		if newSpec.HasEntitlements() {
			return models.NewGenericValidationError(errors.New("customer has no subject but subscription has entitlements"))
		}

		if newSpec.HasMeteredBillables() {
			return models.NewGenericValidationError(errors.New("customer has no subject but subscription has metered billables"))
		}
	}

	if newSpec.HasBillables() {
		if cus.Currency != nil {
			if string(*cus.Currency) != string(newSpec.Currency) {
				return models.NewGenericValidationError(fmt.Errorf("currency mismatch: customer currency is %s, but subscription currency is %s", *cus.Currency, newSpec.Currency))
			}
		}
	}

	return nil
}

func (s *service) validateCancel(ctx context.Context, view subscription.SubscriptionView, timing subscription.Timing) error {
	// Let's make sure Cancel is possible based on the transition rules
	if err := subscription.NewStateMachine(
		view.Subscription.GetStatusAt(clock.Now()),
	).CanTransitionOrErr(ctx, subscription.SubscriptionActionCancel); err != nil {
		return err
	}

	spec := view.AsSpec()

	// Let's try to decode when the subscription should be canceled
	if err := timing.ValidateForAction(subscription.SubscriptionActionCancel, &view); err != nil {
		return fmt.Errorf("invalid cancelation timing: %w", err)
	}

	cancelTime, err := timing.ResolveForSpec(view.Spec)
	if err != nil {
		return fmt.Errorf("failed to get cancelation time: %w", err)
	}

	// Cancellation means that we deactivate everything by that deadline (set ActiveTo)
	// The different Cadences of the Spec are derived from the Subscription Cadence

	spec.ActiveTo = lo.ToPtr(cancelTime)

	if err := spec.Validate(); err != nil {
		return fmt.Errorf("spec is invalid after setting cancelation time: %w", err)
	}

	return nil
}

func (s *service) validateContinue(ctx context.Context, view subscription.SubscriptionView) error {
	// Let's make sure Continue is possible based on the transition rules
	if err := subscription.NewStateMachine(
		view.Subscription.GetStatusAt(clock.Now()),
	).CanTransitionOrErr(ctx, subscription.SubscriptionActionContinue); err != nil {
		return err
	}

	// Continuation means, that we recalculate the deactivation deadlines as if there was no cancellation
	// This is handled by the SubscriptionSpec as all Cadences are derived from the Subscription Cadence
	spec := view.AsSpec()

	spec.ActiveTo = nil

	if err := spec.Validate(); err != nil {
		return fmt.Errorf("spec is invalid after unsetting cancelation time: %w", err)
	}

	// Let's make sure there won't be any scheduling conflicts after continuing (no overlapping subscriptions)

	// Let's build a timeline of every already schedueld subscription
	scheduled, err := s.SubscriptionRepo.GetAllForCustomerSince(ctx, models.NamespacedID{
		ID:        spec.CustomerId,
		Namespace: view.Subscription.Namespace,
	}, clock.Now())
	if err != nil {
		return fmt.Errorf("failed to get scheduled subscriptions: %w", err)
	}

	// Let's filter out the current subscription from the scheduled list
	scheduled = lo.Filter(scheduled, func(i subscription.Subscription, _ int) bool {
		return i.ID != view.Subscription.ID
	})

	scheduledInps := lo.Map(scheduled, func(i subscription.Subscription, _ int) subscription.CreateSubscriptionEntityInput {
		return i.AsEntityInput()
	})

	subscriptionTimeline := models.NewSortedCadenceList(scheduledInps)

	// Sanity check, lets validate that the scheduled timeline is consistent (before continuing)
	if overlaps := subscriptionTimeline.GetOverlaps(); len(overlaps) > 0 {
		return errors.New("inconsistency error: already scheduled subscriptions are overlapping")
	}

	// Now let's check that the new Spec also fits into the timeline
	subscriptionTimeline = models.NewSortedCadenceList(append(scheduledInps, spec.ToCreateSubscriptionEntityInput(view.Subscription.Namespace)))

	if overlaps := subscriptionTimeline.GetOverlaps(); len(overlaps) > 0 {
		return models.NewGenericConflictError(errors.New("continued subscription would overlap with existing ones"))
	}
	return nil
}
