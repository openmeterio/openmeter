package addondiff

import (
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// getApplyForRateCard returns a function that applies a SubscriptionAddonRateCard to a SubscriptionSpec
func (d *diffable) getApplyForRateCard(rc subscriptionaddon.SubscriptionAddonRateCard) subscription.AppliesToSpec {
	return subscription.NewAppliesToSpec(func(spec *subscription.SubscriptionSpec, actx subscription.ApplyContext) error {
		phaseAtCadenceStart, ok := spec.GetCurrentPhaseAt(d.addon.ActiveFrom)
		if !ok {
			return fmt.Errorf("no phase found at %s", d.addon.ActiveFrom)
		}

		phases := spec.GetSortedPhases()

		lastPhaseKey := phases[len(phases)-1].PhaseKey
		lastPhase, ok := spec.Phases[lastPhaseKey]
		if !ok {
			return fmt.Errorf("no last phase found at %s", lastPhaseKey)
		}

		phaseAtCadenceEnd := lastPhase
		if d.addon.ActiveTo != nil {
			phaseAtCadenceEnd, ok = spec.GetCurrentPhaseAt(*d.addon.ActiveTo)
			if !ok {
				return fmt.Errorf("no phase found at %s", *d.addon.ActiveTo)
			}
		}

		// We're gonna go through all phases, and focus on the period in our cadence.
		// In that period:
		// - there must always be an item for the provided key
		// - any existing item must be updated
		reachedFinal := false
		reachedFirst := false
		for _, phase := range spec.GetSortedPhases() {
			if reachedFinal {
				break
			}

			if phase.PhaseKey == phaseAtCadenceStart.PhaseKey {
				reachedFirst = true
			}

			if phase.PhaseKey == phaseAtCadenceEnd.PhaseKey {
				reachedFinal = true
			}

			if !reachedFirst {
				continue
			}

			// Let's calculate periods
			addPer := d.addon.CadencedModel.AsPeriod()

			pCad, err := spec.GetPhaseCadence(phase.PhaseKey)
			if err != nil {
				return fmt.Errorf("failed to get phase cadence for %s: %w", phase.PhaseKey, err)
			}

			addInPhase := pCad.AsPeriod().Intersection(addPer)
			if addInPhase == nil {
				// If the addon is not effectual in the phase, nothing to do here
				continue
			}

			items := phase.ItemsByKey[rc.AddonRateCard.Key()]

			newItems := make([]*subscription.SubscriptionItemSpec, 0, len(items))

			// We'll use gaps to track any items that need to be created
			gaps := []timeutil.OpenPeriod{
				*addInPhase, // We'll assume there's nothing in the phase, then keep subtracting from it
			}

			// We need to update all items
			for _, item := range items {
				itemPer := item.GetCadence(pCad).AsPeriod()

				{
					// Let's subtract the item from the gaps
					nGaps := make([]timeutil.OpenPeriod, 0, len(gaps))
					for _, g := range gaps {
						nGaps = append(nGaps, g.Difference(itemPer)...)
					}

					slices.SortFunc(nGaps, func(a, b timeutil.OpenPeriod) int {
						if a.From == nil {
							return 1
						}

						if b.From == nil {
							return -1
						}

						return a.From.Compare(*b.From)
					})

					gaps = nGaps
				}

				inter := itemPer.Intersection(addPer)
				if inter == nil {
					newItems = append(newItems, item)

					continue
				}

				// We need to split the item:
				// - the old shape will be kept for the difference
				diff := itemPer.Difference(*inter)

				for _, diffPer := range diff {
					inst := subscription.SubscriptionItemSpec{
						CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
							CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
								PhaseKey: phase.PhaseKey,
								ItemKey:  item.ItemKey,
								RateCard: item.RateCard.Clone(),
							},
							Annotations: item.Annotations,
						},
					}

					d.setItemRelativeCadence(&inst, pCad, diffPer)

					newItems = append(newItems, &inst)
				}

				// - the new shape will be calced for the intersection
				inst := subscription.SubscriptionItemSpec{
					CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
						CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
							PhaseKey: phase.PhaseKey,
							ItemKey:  item.ItemKey,
							RateCard: item.RateCard.Clone(),
						},
						Annotations: item.Annotations,
					},
				}

				for range d.addon.Quantity {
					err := rc.Apply(inst.RateCard)
					if err != nil {
						return fmt.Errorf("failed to extend rate card %s: %w", rc.AddonRateCard.Key(), err)
					}
				}

				d.setItemRelativeCadence(&inst, pCad, *inter)

				newItems = append(newItems, &inst)
			}

			// Let's create new items for the gaps
			for _, gap := range gaps {
				inst := subscription.SubscriptionItemSpec{
					CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
						CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
							PhaseKey: phase.PhaseKey,
							ItemKey:  rc.AddonRateCard.Key(),
							RateCard: rc.AddonRateCard.RateCard.Clone(),
						},
					},
				}

				for range d.addon.Quantity - 1 {
					err := rc.Apply(inst.RateCard)
					if err != nil {
						return fmt.Errorf("failed to extend rate card %s: %w", rc.AddonRateCard.Key(), err)
					}
				}

				d.setItemRelativeCadence(&inst, pCad, gap)

				newItems = append(newItems, &inst)
			}

			slices.SortFunc(newItems, func(a, b *subscription.SubscriptionItemSpec) int {
				return a.GetCadence(pCad).ActiveFrom.Compare(b.GetCadence(pCad).ActiveFrom)
			})

			phase.ItemsByKey[rc.AddonRateCard.Key()] = newItems
		}

		return nil
	})
}

// setItemRelativeCadence sets the cadence of an item to match target
func (d *diffable) setItemRelativeCadence(item *subscription.SubscriptionItemSpec, phaseCadence models.CadencedModel, target timeutil.OpenPeriod) {
	if target.From != nil {
		diff := isodate.Between(phaseCadence.ActiveFrom, *target.From)

		if !diff.IsZero() {
			item.ActiveFromOverrideRelativeToPhaseStart = &diff
		}
	}

	if target.To != nil {
		diff := isodate.Between(phaseCadence.ActiveFrom, *target.To)

		if phaseCadence.ActiveTo == nil || !target.To.Equal(*phaseCadence.ActiveTo) {
			item.ActiveToOverrideRelativeToPhaseStart = &diff
		}
	}
}
