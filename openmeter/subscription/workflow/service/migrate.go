package service

import (
	"context"
	"errors"
	"maps"
	"reflect"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) MigrateToPlan(ctx context.Context, input subscriptionworkflow.MigrateSubscriptionWorkflowInput) (subscription.Subscription, subscription.SubscriptionView, error) {
	if err := input.Validate(); err != nil {
		return subscription.Subscription{}, subscription.SubscriptionView{}, err
	}

	type result struct {
		current subscription.Subscription
		updated subscription.SubscriptionView
	}

	migrated, err := transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (result, error) {
		var def result

		// Lock the customer before reading the subscription so another update
		// cannot change it while we calculate and save the migration.
		sub, err := s.Service.Get(ctx, input.SubscriptionID)
		if err != nil {
			return def, err
		}

		if err := s.lockCustomer(ctx, sub.CustomerId); err != nil {
			return def, err
		}

		current, err := s.Service.GetView(ctx, input.SubscriptionID)
		if err != nil {
			return def, err
		}

		// Use the same timing rules as a running subscription edit.
		if err := input.Timing.ValidateForAction(subscription.SubscriptionActionUpdate, &current); err != nil {
			return def, err
		}

		at, err := input.Timing.ResolveForSpec(current.Spec)
		if err != nil {
			return def, err
		}

		// Check the addons against the new plan, using their current and future quantities.
		addons, err := s.AddonService.List(ctx, input.SubscriptionID.Namespace, subscriptionaddon.ListSubscriptionAddonsInput{
			SubscriptionID: input.SubscriptionID.ID,
		})
		if err != nil {
			return def, err
		}

		target, err := s.buildMigrationTarget(ctx, buildMigrationTargetInput{
			Current: current,
			Plan:    input.Plan,
			At:      at,
			Addons:  addons.Items,
		})
		if err != nil {
			return def, err
		}

		// Keep unchanged items and save changed items, entitlements, and the plan
		// reference together through the existing update path.
		spec, err := buildMigratedSpec(buildMigratedSpecInput{Current: current, Target: target, At: at})
		if err != nil {
			return def, err
		}

		if _, err := s.Service.Update(ctx, input.SubscriptionID, spec, subscription.WithCostBasisEffectiveAt(at)); err != nil {
			return def, err
		}

		updated, err := s.Service.GetView(ctx, input.SubscriptionID)
		if err != nil {
			return def, err
		}

		// Both response snapshots must describe the change made under this lock.
		return result{current: current.Subscription, updated: updated}, nil
	})

	return migrated.current, migrated.updated, err
}

type buildMigratedSpecInput struct {
	Current subscription.SubscriptionView
	Target  subscription.SubscriptionSpec
	At      time.Time
}

func (i buildMigratedSpecInput) Validate() error {
	var errs []error
	if i.Current.Spec.Plan == nil || i.Target.Plan == nil {
		errs = append(errs, errors.New("custom subscriptions cannot be migrated"))
	} else if err := (subscription.AdvancePlanReferenceInput{
		SubscriptionID: i.Current.Subscription.NamespacedID,
		CurrentPlan:    *i.Current.Spec.Plan, TargetPlan: *i.Target.Plan,
	}).Validate(); err != nil {
		errs = append(errs, err)
	}
	if i.At.IsZero() || !i.Current.Subscription.IsActiveAt(i.At) {
		errs = append(errs, errors.New("subscription must be active at migration time"))
	}
	if !i.Current.Spec.BillingCadence.Equal(&i.Target.BillingCadence) ||
		i.Current.Spec.SettlementMode != i.Target.SettlementMode ||
		!reflect.DeepEqual(i.Current.Spec.ProRatingConfig, i.Target.ProRatingConfig) {
		errs = append(errs, errors.New("migration cannot change billing cadence, settlement mode, or proration configuration; use subscription change"))
	}
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func buildMigratedSpec(i buildMigratedSpecInput) (subscription.SubscriptionSpec, error) {
	if err := i.Validate(); err != nil {
		return subscription.SubscriptionSpec{}, err
	}
	patches, err := patch.DiffItems(patch.DiffItemsInput{Current: i.Current.Spec, Target: i.Target, At: i.At})
	if err != nil {
		return subscription.SubscriptionSpec{}, err
	}
	// Patches replace entries in phase item maps. Copy those maps so the
	// caller's view still describes the subscription before migration.
	spec := i.Current.AsSpec()
	spec.Phases = maps.Clone(spec.Phases)
	for key, phase := range spec.Phases {
		copied := *phase
		copied.ItemsByKey = maps.Clone(phase.ItemsByKey)
		spec.Phases[key] = &copied
	}
	if err := spec.ApplyMany(lo.Map(patches, subscription.ToApplies), subscription.ApplyContext{CurrentTime: i.At}); err != nil {
		return subscription.SubscriptionSpec{}, subscriptionworkflow.MapSubscriptionErrors(err)
	}
	spec.Plan = i.Target.Plan
	if err := spec.ValidateAlignment(); err != nil {
		return subscription.SubscriptionSpec{}, err
	}

	return spec, nil
}
