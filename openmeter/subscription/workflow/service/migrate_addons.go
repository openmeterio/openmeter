package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type validateMigrationAddonsInput struct {
	Namespace string
	Target    subscription.SubscriptionSpec
	At        time.Time
	Addons    []subscriptionaddon.SubscriptionAddon
}

func (i validateMigrationAddonsInput) Validate() error {
	var errs []error
	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}
	if i.Target.Plan == nil {
		errs = append(errs, errors.New("target plan reference is required"))
	}
	if i.At.IsZero() {
		errs = append(errs, errors.New("migration time is required"))
	}
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (s *service) validateMigrationAddons(ctx context.Context, i validateMigrationAddonsInput) error {
	if err := i.Validate(); err != nil {
		return err
	}
	target, at := i.Target, i.At
	namespace := i.Namespace
	for _, addon := range i.Addons {
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

type buildMigrationTargetInput struct {
	Current subscription.SubscriptionView
	Plan    subscription.Plan
	At      time.Time
	Addons  []subscriptionaddon.SubscriptionAddon
}

func (i buildMigrationTargetInput) Validate() error {
	var errs []error
	if i.Plan == nil {
		errs = append(errs, errors.New("target plan is required"))
	}
	if i.At.IsZero() {
		errs = append(errs, errors.New("migration time is required"))
	}
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// buildMigrationTarget builds a new spec from the plan and applies the addons.
// Quantities that ended before migration do not limit changes to the new plan.
func (s *service) buildMigrationTarget(ctx context.Context, i buildMigrationTargetInput) (subscription.SubscriptionSpec, error) {
	if err := i.Validate(); err != nil {
		return subscription.SubscriptionSpec{}, err
	}
	target, err := subscription.NewSpecFromPlan(i.Plan, i.Current.Spec.CreateSubscriptionCustomerInput)
	if err != nil {
		return subscription.SubscriptionSpec{}, subscriptionworkflow.MapSubscriptionErrors(err)
	}
	if err := (buildMigratedSpecInput{Current: i.Current, Target: target, At: i.At}).Validate(); err != nil {
		return subscription.SubscriptionSpec{}, err
	}
	if err := s.validateMigrationAddons(ctx, validateMigrationAddonsInput{Namespace: i.Current.Subscription.Namespace, Target: target, At: i.At, Addons: i.Addons}); err != nil {
		return subscription.SubscriptionSpec{}, err
	}
	at := i.At
	prospectiveAddons := make([]subscriptionaddon.SubscriptionAddon, 0, len(i.Addons))
	for _, addon := range i.Addons {
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
	diffs, err := asDiffs(i.Current, prospectiveAddons)
	if err != nil {
		return subscription.SubscriptionSpec{}, err
	}
	for _, diff := range diffs {
		if err := target.Apply(diff.GetApplies(), subscription.ApplyContext{CurrentTime: at}); err != nil {
			return subscription.SubscriptionSpec{}, models.NewGenericValidationError(fmt.Errorf("applying addons to target plan: %w", err))
		}
	}

	return target, nil
}
