package addondiff

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
)

// FIXME: find a better place for this
func GetAffectedItemIDs(view subscription.SubscriptionView, addon subscriptionaddon.SubscriptionAddon) map[string][]string {
	affected := map[string][]string{}

	// We'll assume that a given SubscriptionPhase in general will have more Items than a SubscriptionAddon

	instances := addon.GetInstances()

	if len(instances) == 0 {
		return affected
	}

	for _, p := range view.Phases {
		for _, items := range p.ItemsByKey {
			for _, item := range items {
				for _, inst := range instances {
					if inst.Quantity == 0 {
						continue
					}

					itemPer := item.SubscriptionItem.CadencedModel.AsPeriod()
					instPer := inst.CadencedModel.AsPeriod()

					// If there's no intersection nothing can match it
					if instPer.Intersection(itemPer) == nil {
						continue
					}

					for _, rc := range inst.RateCards {
						rcKey := rc.AddonRateCard.RateCard.Key()

						if _, ok := affected[rcKey]; !ok {
							affected[rcKey] = []string{}
						}

						if productcatalog.AddonRateCardMatcherForAGivenPlanRateCard(item.SubscriptionItem.RateCard)(rc.AddonRateCard.RateCard) {
							affected[rcKey] = append(affected[rcKey], item.SubscriptionItem.ID)
						}
					}
				}
			}
		}
	}

	// Let's dedupe
	return lo.MapEntries(affected, func(key string, value []string) (string, []string) {
		return key, lo.Uniq(value)
	})
}
