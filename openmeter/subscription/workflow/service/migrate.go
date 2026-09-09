package service

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (s *service) MigrateToPlan(ctx context.Context, input subscriptionworkflow.MigrateSubscriptionWorkflowInput) (subscription.SubscriptionView, error) {
	if err := input.Validate(); err != nil {
		return subscription.SubscriptionView{}, err
	}
	return transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (subscription.SubscriptionView, error) {
		var def subscription.SubscriptionView
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
		if current.Spec.Plan == nil {
			return def, models.NewGenericValidationError(errors.New("custom subscriptions cannot be migrated"))
		}
		if err := (subscription.MigratePlanInput{
			SubscriptionID: input.SubscriptionID,
			CurrentPlan:    *current.Spec.Plan,
			TargetPlan:     *input.Plan.ToCreateSubscriptionPlanInput().Plan,
		}).Validate(); err != nil {
			return def, err
		}
		if err := input.Timing.ValidateForAction(subscription.SubscriptionActionUpdate, &current); err != nil {
			return def, err
		}
		at, err := input.Timing.ResolveForSpec(current.Spec)
		if err != nil {
			return def, err
		}
		if !current.Subscription.IsActiveAt(at) {
			return def, models.NewGenericValidationError(errors.New("subscription must be active at migration time"))
		}
		target, err := subscription.NewSpecFromPlan(input.Plan, current.Spec.CreateSubscriptionCustomerInput)
		if err != nil {
			return def, subscriptionworkflow.MapSubscriptionErrors(err)
		}
		if !current.Spec.BillingCadence.Equal(&target.BillingCadence) ||
			current.Spec.SettlementMode != target.SettlementMode ||
			!reflect.DeepEqual(current.Spec.ProRatingConfig, target.ProRatingConfig) {
			return def, models.NewGenericValidationError(errors.New("migration cannot change billing cadence, settlement mode, or proration configuration; use subscription change"))
		}
		addons, err := s.AddonService.List(ctx, input.SubscriptionID.Namespace, subscriptionaddon.ListSubscriptionAddonsInput{
			SubscriptionID: input.SubscriptionID.ID,
		})
		if err != nil {
			return def, err
		}
		if err := s.validateMigrationAddons(ctx, input.SubscriptionID.Namespace, target, addons.Items, at); err != nil {
			return def, err
		}
		// Build the prospective offering with the existing purchases and quantity
		// schedule. Diff the composed result so untouched addon items keep their
		// identities; restoring and rematerializing the live view would renumber them.
		// Historical purchases must not constrain the new offering. Keep only
		// quantity segments overlapping the part of the subscription being amended.
		prospectiveAddons := make([]subscriptionaddon.SubscriptionAddon, 0, len(addons.Items))
		for _, addon := range addons.Items {
			var quantities []timeutil.Timed[subscriptionaddon.SubscriptionAddonQuantity]
			for _, instance := range addon.GetInstances() {
				if instance.ActiveTo != nil && !instance.ActiveTo.After(at) {
					continue
				}
				startsAt := instance.ActiveFrom
				if startsAt.Before(at) {
					startsAt = at
				}
				if target.ActiveTo != nil && !startsAt.Before(*target.ActiveTo) {
					continue
				}
				quantities = append(quantities, (subscriptionaddon.SubscriptionAddonQuantity{
					ActiveFrom: startsAt, Quantity: instance.Quantity,
				}).AsTimed())
			}
			addon.Quantities = timeutil.NewTimeline(quantities)
			prospectiveAddons = append(prospectiveAddons, addon)
		}
		diffs, err := asDiffs(current, prospectiveAddons)
		if err != nil {
			return def, err
		}
		for _, diff := range diffs {
			if err := target.Apply(diff.GetApplies(), subscription.ApplyContext{CurrentTime: at}); err != nil {
				return def, models.NewGenericValidationError(fmt.Errorf("applying addons to target plan: %w", err))
			}
		}
		patches, err := patch.DiffItems(patch.DiffItemsInput{Current: current.Spec, Target: target, At: at})
		if err != nil {
			return def, err
		}
		spec := current.AsSpec()
		if err := spec.ApplyMany(lo.Map(patches, subscription.ToApplies), subscription.ApplyContext{CurrentTime: at}); err != nil {
			return def, subscriptionworkflow.MapSubscriptionErrors(err)
		}
		spec.Plan = target.Plan
		if err := spec.ValidateAlignment(); err != nil {
			return def, err
		}
		if _, err := s.Service.Update(ctx, input.SubscriptionID, spec, subscription.WithCostBasisEffectiveAt(at)); err != nil {
			return def, err
		}
		return s.Service.GetView(ctx, input.SubscriptionID)
	})
}

func (s *service) validateMigrationAddons(ctx context.Context, namespace string, target subscription.SubscriptionSpec, addons []subscriptionaddon.SubscriptionAddon, at time.Time) error {
	for _, addon := range addons {
		for _, instance := range addon.GetInstances() {
			if instance.Quantity == 0 || (instance.ActiveTo != nil && !instance.ActiveTo.After(at)) {
				continue
			}
			startsAt := instance.ActiveFrom
			if startsAt.Before(at) {
				startsAt = at
			}
			if target.ActiveTo != nil && !startsAt.Before(*target.ActiveTo) {
				continue
			}
			compatibility, err := s.PlanAddonService.GetPlanAddon(ctx, planaddon.GetPlanAddonInput{
				NamespacedModel: models.NamespacedModel{Namespace: namespace},
				PlanIDOrKey:     target.Plan.Id,
				AddonIDOrKey:    addon.Addon.ID,
			})
			if err != nil {
				if models.IsGenericNotFoundError(err) {
					return models.NewGenericValidationError(fmt.Errorf("addon %s is not compatible with target plan version", addon.Addon.Key))
				}
				return err
			}
			phase, ok := target.Phases[compatibility.FromPlanPhase]
			if !ok {
				return models.NewGenericValidationError(fmt.Errorf("addon %s compatibility phase is missing", addon.Addon.Key))
			}
			compatibleFrom, _ := phase.StartAfter.AddTo(target.ActiveFrom)
			if startsAt.Before(compatibleFrom) || (compatibility.MaxQuantity != nil && instance.Quantity > *compatibility.MaxQuantity) {
				return models.NewGenericValidationError(fmt.Errorf("addon %s quantity schedule is incompatible with target plan version", addon.Addon.Key))
			}
		}
	}
	return nil
}
