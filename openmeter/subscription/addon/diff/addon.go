package addondiff

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
)

func GetDiffableFromAddon(
	view subscription.SubscriptionView,
	addon subscriptionaddon.SubscriptionAddon,
) (Diffable, error) {
	instances := addon.GetInstances()

	if len(instances) == 0 {
		// no-op
		return &someDiffable{
			ApplyFn:   func(spec *subscription.SubscriptionSpec, actx subscription.ApplyContext) error { return nil },
			RestoreFn: func(spec *subscription.SubscriptionSpec, actx subscription.ApplyContext) error { return nil },
		}, nil
	}

	// As there's no overlap between the instances, we can just apply them in (any) sequence

	diffs := lo.Map(instances, func(instance subscriptionaddon.SubscriptionAddonInstance, _ int) Diffable {
		return &diffable{
			view:  view,
			addon: instance,
		}
	})

	return &someDiffable{
		ApplyFn: func(spec *subscription.SubscriptionSpec, actx subscription.ApplyContext) error {
			applieses := lo.Map(diffs, func(diff Diffable, _ int) subscription.AppliesToSpec {
				return diff.GetApplies()
			})

			agg := subscription.NewAggregateAppliesToSpec(applieses)

			return agg.ApplyTo(spec, actx)
		},
		RestoreFn: func(spec *subscription.SubscriptionSpec, actx subscription.ApplyContext) error {
			applieses := lo.Map(diffs, func(diff Diffable, _ int) subscription.AppliesToSpec {
				return diff.GetRestores()
			})

			agg := subscription.NewAggregateAppliesToSpec(applieses)

			return agg.ApplyTo(spec, actx)
		},
	}, nil
}

var _ Diffable = &diffable{}

type diffable struct {
	view  subscription.SubscriptionView
	addon subscriptionaddon.SubscriptionAddonInstance
}

func (d *diffable) GetApplies() subscription.AppliesToSpec {
	if d.addon.Quantity == 0 {
		return subscription.NewAppliesToSpec(func(spec *subscription.SubscriptionSpec, actx subscription.ApplyContext) error {
			return nil
		})
	}

	applieses := lo.Map(d.addon.RateCards, func(rc subscriptionaddon.SubscriptionAddonRateCard, _ int) subscription.AppliesToSpec {
		return d.getApplyForRC(rc)
	})

	return subscription.NewAggregateAppliesToSpec(applieses)
}

func (d *diffable) GetRestores() subscription.AppliesToSpec {
	panic("not implemented")
}
