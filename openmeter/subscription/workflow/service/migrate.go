package service

import (
	"context"
	"errors"
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

func (s *service) MigrateToPlan(ctx context.Context, input subscriptionworkflow.MigrateSubscriptionWorkflowInput) (subscription.SubscriptionView, error) {
	if err := input.Validate(); err != nil {
		return subscription.SubscriptionView{}, err
	}
	return transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (subscription.SubscriptionView, error) {
		var def subscription.SubscriptionView
		// Subscription updates use a customer-scoped lock. Hold that same lock from
		// the fresh read through persistence so concurrent updates cannot stale the diff.
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

		// Resolve edit timing against the existing timeline, then reject target
		// configurations that would require a subscription-level billing reset.
		if err := input.Timing.ValidateForAction(subscription.SubscriptionActionUpdate, &current); err != nil {
			return def, err
		}
		at, err := input.Timing.ResolveForSpec(current.Spec)
		if err != nil {
			return def, err
		}
		target, err := subscription.NewSpecFromPlan(input.Plan, current.Spec.CreateSubscriptionCustomerInput)
		if err != nil {
			return def, subscriptionworkflow.MapSubscriptionErrors(err)
		}
		migration := migrationSpecInput{Current: current, Target: target, At: at}
		if err := migration.Validate(); err != nil {
			return def, err
		}

		// Load purchases and validate their future quantities against the target plan.
		// Apply them to the prospective spec: restoring the live view first could
		// merge historical item versions and change their billing identities.
		addons, err := s.AddonService.List(ctx, input.SubscriptionID.Namespace, subscriptionaddon.ListSubscriptionAddonsInput{SubscriptionID: input.SubscriptionID.ID})
		if err != nil {
			return def, err
		}
		migration.Addons = addons.Items
		if err := migration.validateAddons(ctx, s.PlanAddonService); err != nil {
			return def, err
		}
		if err := migration.applyAddons(); err != nil {
			return def, err
		}

		// Patch only differing schedules. The normal update path materializes items,
		// entitlements, and the plan reference together in this transaction.
		spec, err := migration.updatedSpec()
		if err != nil {
			return def, err
		}
		if _, err := s.Service.Update(ctx, input.SubscriptionID, spec, subscription.WithCostBasisEffectiveAt(at)); err != nil {
			return def, err
		}
		return s.Service.GetView(ctx, input.SubscriptionID)
	})
}

// migrationSpecInput holds the locked current offering and its prospective
// replacement. Addons are composed onto Target before generating item patches.
type migrationSpecInput struct {
	Current subscription.SubscriptionView
	Target  subscription.SubscriptionSpec
	At      time.Time
	Addons  []subscriptionaddon.SubscriptionAddon
}

func (i migrationSpecInput) Validate() error {
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

func (i migrationSpecInput) updatedSpec() (subscription.SubscriptionSpec, error) {
	patches, err := patch.DiffItems(patch.DiffItemsInput{Current: i.Current.Spec, Target: i.Target, At: i.At})
	if err != nil {
		return subscription.SubscriptionSpec{}, err
	}
	spec := i.Current.AsSpec()
	if err := spec.ApplyMany(lo.Map(patches, subscription.ToApplies), subscription.ApplyContext{CurrentTime: i.At}); err != nil {
		return subscription.SubscriptionSpec{}, subscriptionworkflow.MapSubscriptionErrors(err)
	}
	spec.Plan = i.Target.Plan
	if err := spec.ValidateAlignment(); err != nil {
		return subscription.SubscriptionSpec{}, err
	}

	return spec, nil
}
