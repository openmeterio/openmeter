package addondiff

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
)

// FIXME: find a better place for this
func GetAffectedItemIDs(view subscription.SubscriptionView, addon subscriptionaddon.SubscriptionAddon) map[string][]string {
	affected := map[string][]string{}

	for _, inst := range addon.GetInstances() {
		instPer := inst.CadencedModel.AsPeriod()

		for _, rc := range inst.RateCards {
			rcKey := rc.AddonRateCard.RateCard.Key()

			for _, p := range view.Phases {
				for _, items := range p.ItemsByKey {
					for _, item := range items {
						if item.Spec.RateCard.Key() != rcKey {
							continue
						}

						itemPer := item.SubscriptionItem.CadencedModel.AsPeriod()

						if instPer.Intersection(itemPer) != nil {
							if _, ok := affected[rcKey]; !ok {
								affected[rcKey] = []string{}
							}

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
