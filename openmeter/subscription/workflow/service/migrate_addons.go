package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (i migrationSpecInput) validateAddons(ctx context.Context, planAddons planaddon.Service) error {
	target, at := i.Target, i.At
	namespace := i.Current.Subscription.Namespace
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
			compatibility, err := planAddons.GetPlanAddon(ctx, planaddon.GetPlanAddonInput{
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

// applyAddons ignores expired purchase segments: they must not constrain the
// offering being amended. Future quantity boundaries, including zero, survive.
func (i *migrationSpecInput) applyAddons() error {
	at := i.At
	target := &i.Target
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
		return err
	}
	for _, diff := range diffs {
		if err := target.Apply(diff.GetApplies(), subscription.ApplyContext{CurrentTime: at}); err != nil {
			return models.NewGenericValidationError(fmt.Errorf("applying addons to target plan: %w", err))
		}
	}

	return nil
}
