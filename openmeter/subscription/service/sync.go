package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) sync(ctx context.Context, view subscription.SubscriptionView, newSpec subscription.SubscriptionSpec) (subscription.Subscription, error) {
	if err := s.prepareSpecCurrencies(ctx, view.Subscription.Namespace, &newSpec); err != nil {
		return subscription.Subscription{}, err
	}

	return s.syncPrepared(ctx, view, newSpec)
}

func (s *service) syncPrepared(ctx context.Context, view subscription.SubscriptionView, newSpec subscription.SubscriptionSpec) (subscription.Subscription, error) {
	return transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (subscription.Subscription, error) {
		var def subscription.Subscription

		if err := validateSyncTarget(view, newSpec); err != nil {
			return def, err
		}

		plan, err := newSyncPlan(view, newSpec)
		if err != nil {
			return def, fmt.Errorf("failed to plan subscription sync: %w", err)
		}

		if !view.Subscription.CadencedModel.Equal(models.CadencedModel{
			ActiveFrom: newSpec.ActiveFrom,
			ActiveTo:   newSpec.ActiveTo,
		}) {
			if _, err := s.SubscriptionRepo.SetEndOfCadence(
				ctx,
				view.Subscription.NamespacedID,
				newSpec.ActiveTo,
			); err != nil {
				return def, fmt.Errorf("failed to set end of cadence: %w", err)
			}
		}

		if err := plan.Execute(ctx, s, view.Customer); err != nil {
			return def, err
		}
		if !view.Subscription.PlanRef.NilEqual(newSpec.Plan) {
			// validateSyncTarget has already checked a later version of the same
			// plan. Persist its reference atomically with the materialized items.
			if err := s.SubscriptionRepo.AdvancePlanReference(ctx, subscription.AdvancePlanReferenceInput{
				SubscriptionID: view.Subscription.NamespacedID,
				CurrentPlan:    *view.Subscription.PlanRef,
				TargetPlan:     *newSpec.Plan,
			}); err != nil {
				return def, err
			}
		}

		return s.Get(ctx, view.Subscription.NamespacedID)
	})
}

func validateSyncTarget(view subscription.SubscriptionView, newSpec subscription.SubscriptionSpec) error {
	if view.Subscription.CustomerId != newSpec.CustomerId {
		return fmt.Errorf("cannot change customer id")
	}
	if !view.Subscription.PlanRef.NilEqual(newSpec.Plan) {
		if view.Subscription.PlanRef == nil || newSpec.Plan == nil {
			return fmt.Errorf("cannot change plan")
		}
		if err := (subscription.AdvancePlanReferenceInput{
			SubscriptionID: view.Subscription.NamespacedID,
			CurrentPlan:    *view.Subscription.PlanRef,
			TargetPlan:     *newSpec.Plan,
		}).Validate(); err != nil {
			return fmt.Errorf("cannot change plan: %w", err)
		}
	}
	if !view.Subscription.ActiveFrom.Equal(newSpec.ActiveFrom) {
		return fmt.Errorf("cannot change subscription start")
	}
	if view.Subscription.SettlementMode != newSpec.SettlementMode {
		return fmt.Errorf("cannot change settlement mode")
	}

	return nil
}
