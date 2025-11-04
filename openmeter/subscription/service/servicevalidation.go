package service

import (
	"context"
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

	// 3. Let's make sure the currency is valid
	if spec.HasBillables() {
		if cust.Currency != nil && (string(*cust.Currency) != string(spec.Currency)) {
			return models.NewGenericValidationError(fmt.Errorf("currency mismatch: customer currency is %s, but subscription currency is %s", *cust.Currency, spec.Currency))
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

	return nil
}

func numNotGrouped[T any, K comparable](source []T, grouped map[K][]T) int {
	count := len(source)
	for _, group := range grouped {
		count -= len(group)
	}

	return count
}
