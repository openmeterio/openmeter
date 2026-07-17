package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
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
		if cust.Currency != nil && *cust.Currency != spec.InvoiceCurrency {
			return models.NewGenericValidationError(fmt.Errorf("currency mismatch: customer currency is %s, but subscription invoice currency is %s", *cust.Currency, spec.InvoiceCurrency))
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

	if err := newSpec.Validate(); err != nil {
		return fmt.Errorf("spec is invalid: %w", err)
	}

	if currentView.Subscription.InvoiceCurrency != newSpec.InvoiceCurrency {
		return models.NewGenericValidationError(fmt.Errorf(
			"cannot change subscription invoice currency from %s to %s",
			currentView.Subscription.InvoiceCurrency,
			newSpec.InvoiceCurrency,
		))
	}

	if currentView.Subscription.CostBasisMode.OrDefault() != newSpec.CostBasisMode.OrDefault() {
		return models.NewGenericValidationError(fmt.Errorf(
			"cannot change subscription cost basis mode from %s to %s",
			currentView.Subscription.CostBasisMode.OrDefault(),
			newSpec.CostBasisMode.OrDefault(),
		))
	}

	if err := validateMaterializedItemCurrenciesUnchanged(currentView.Spec, newSpec); err != nil {
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
			if *cus.Currency != newSpec.InvoiceCurrency {
				return models.NewGenericValidationError(fmt.Errorf("currency mismatch: customer currency is %s, but subscription invoice currency is %s", *cus.Currency, newSpec.InvoiceCurrency))
			}
		}
	}

	return nil
}

func validateMaterializedItemCurrenciesUnchanged(currentSpec, newSpec subscription.SubscriptionSpec) error {
	for phaseKey, currentPhase := range currentSpec.Phases {
		newPhase, ok := newSpec.Phases[phaseKey]
		if !ok || currentPhase == nil || newPhase == nil {
			continue
		}

		for itemKey, currentItems := range currentPhase.ItemsByKey {
			newItems, ok := newPhase.ItemsByKey[itemKey]
			if !ok {
				continue
			}

			var establishedCurrency currencyx.CurrencyIdentity
			for _, currentItem := range currentItems {
				if currentItem == nil || currentItem.RateCard == nil {
					continue
				}

				meta := currentItem.RateCard.AsMeta()
				if meta.Price != nil && meta.Currency != nil {
					establishedCurrency = meta.Currency
					break
				}
			}

			for idx, currentItem := range currentItems {
				if idx >= len(newItems) || currentItem == nil || newItems[idx] == nil || currentItem.RateCard == nil || newItems[idx].RateCard == nil {
					continue
				}

				currentCurrency := currentItem.RateCard.AsMeta().Currency
				if currentCurrency == nil {
					continue
				}

				newCurrency := newItems[idx].RateCard.AsMeta().Currency
				if newCurrency == nil || !currentCurrency.Equal(newCurrency) {
					return models.NewGenericValidationError(fmt.Errorf(
						"cannot change currency of subscription item %q[%d] in phase %q",
						itemKey,
						idx,
						phaseKey,
					))
				}
			}

			for idx, newItem := range newItems {
				if newItem == nil || newItem.RateCard == nil {
					continue
				}

				meta := newItem.RateCard.AsMeta()
				if meta.Price == nil || meta.Currency == nil {
					continue
				}

				if establishedCurrency == nil {
					establishedCurrency = meta.Currency
					continue
				}

				if !establishedCurrency.Equal(meta.Currency) {
					return models.NewGenericValidationError(fmt.Errorf(
						"cannot change currency of subscription item %q[%d] in phase %q",
						itemKey,
						idx,
						phaseKey,
					))
				}
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
