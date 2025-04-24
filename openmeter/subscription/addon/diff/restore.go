package addondiff

import (
	"fmt"
	"reflect"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (d *diffable) restore() subscription.AppliesToSpec {
	return subscription.NewAppliesToSpec(func(spec *subscription.SubscriptionSpec, _ subscription.ApplyContext) error {
		for _, p := range spec.GetSortedPhases() {
			pCad, err := spec.GetPhaseCadence(p.PhaseKey)
			if err != nil {
				return fmt.Errorf("failed to get phase cadence for phase %s: %w", p.PhaseKey, err)
			}

			for itemsKey := range p.ItemsByKey {
				aPer := d.addon.CadencedModel.AsPeriod()

				affectingAddonRateCard, ok := lo.Find(d.addon.RateCards, func(rc subscriptionaddon.SubscriptionAddonRateCard) bool {
					return rc.AddonRateCard.Key() == itemsKey
				})

				// If there's no matching key in the addon, we can skip
				if !ok {
					continue
				}

				// Let's find the items that should be deleted
				rmIdxs := []int{}
				for idx, item := range p.ItemsByKey[itemsKey] {
					itemPer := item.GetCadence(pCad).AsPeriod()

					if !aPer.IsSupersetOf(itemPer) {
						continue
					}

					// Let's try to undo the effects of the addon RateCard
					target := item.RateCard.Clone()

					if item.Annotations == nil {
						item.Annotations = models.Annotations{}
					}

					if err := affectingAddonRateCard.Restore(target, item.Annotations); err != nil {
						return fmt.Errorf("failed to restore addon rate card %s: %w", affectingAddonRateCard.AddonRateCard.Key(), err)
					}

					item.RateCard = target

					// Let's do a stupid check about whether the item can be deleted
					chk := zeroRateCardCheck{
						itemAnnotations: item.Annotations,
						rc:              target,
					}

					if chk.CanDelete() {
						rmIdxs = append(rmIdxs, idx)
					}
				}

				filteredItems := make([]*subscription.SubscriptionItemSpec, 0, len(p.ItemsByKey[itemsKey])-len(rmIdxs))
				for idx, item := range p.ItemsByKey[itemsKey] {
					if lo.Contains(rmIdxs, idx) {
						continue
					}
					filteredItems = append(filteredItems, item)
				}

				// Two items can be merged if they are
				// - subsequent
				// - identical (except relative cadence)
				mergedItems := make([]*subscription.SubscriptionItemSpec, 0, len(filteredItems))

				var targetItem *subscription.SubscriptionItemSpec
				for idx := range filteredItems {
					if targetItem == nil {
						targetItem = filteredItems[idx]
					}

					if idx+1 >= len(filteredItems) {
						break
					}

					targetCadence := targetItem.GetCadence(pCad)

					testItem := filteredItems[idx+1]
					testCadence := testItem.GetCadence(pCad)

					canMerge := func() bool {
						if !targetItem.RateCard.Equal(testItem.RateCard) {
							return false
						}

						if (targetItem.Annotations == nil) != (testItem.Annotations == nil) {
							return false
						}

						if targetItem.Annotations != nil && !reflect.DeepEqual(targetItem.Annotations, testItem.Annotations) {
							return false
						}

						if !reflect.DeepEqual(targetItem.BillingBehaviorOverride, testItem.BillingBehaviorOverride) {
							return false
						}

						return true
					}()

					if canMerge {
						combinedPer := targetCadence.AsPeriod().Union(testCadence.AsPeriod())

						if combinedPer.From != nil && !combinedPer.From.Equal(pCad.ActiveFrom) {
							targetItem.ActiveFromOverrideRelativeToPhaseStart = lo.ToPtr(isodate.Between(pCad.ActiveFrom, *combinedPer.From))
						}

						if combinedPer.To == nil {
							targetItem.ActiveToOverrideRelativeToPhaseStart = nil
						}

						if combinedPer.To != nil && !combinedPer.To.Equal(pCad.ActiveFrom) {
							targetItem.ActiveToOverrideRelativeToPhaseStart = lo.ToPtr(isodate.Between(pCad.ActiveFrom, *combinedPer.To))
						}
					} else {
						mergedItems = append(mergedItems, targetItem)
						targetItem = testItem
					}
				}

				if targetItem != nil {
					mergedItems = append(mergedItems, targetItem)
				}

				p.ItemsByKey[itemsKey] = mergedItems

				if len(p.ItemsByKey[itemsKey]) == 0 {
					delete(p.ItemsByKey, itemsKey)
				}
			}
		}

		return nil
	})
}
