package subscriptionhandler

import (
	"fmt"
	"iter"
	"strings"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/datex"
)

type PhaseIterator struct {
	subscriptionID string
	phase          subscription.SubscriptionPhaseView
	iterationEnd   time.Time

	flattenedItems   []subscriptionItemView
	periodIndexByKey map[string]int

	currentItem int
}

type rateCardWithPeriod struct {
	RateCard subscription.RateCard
	Period   billing.Period
	UniqueID string
}

type subscriptionItemView struct {
	subscription.SubscriptionItemView

	lastGenerationTime time.Time
	// index stores the index of the item in the subscriptionPhaseView's ItemsByKey
	index int

	// done is true if the item will not yield more periods
	done bool
}

// NewPhaseIterator creates a new PhaseIterator for the given phase and subscription
//
// It is guaranteed that all items that are starting before phaseEnd are returned. The call
// might return more items if needed, but it always honors the phase's end.
func NewPhaseIterator(phase subscription.SubscriptionPhaseView, subs subscription.SubscriptionView, end time.Time) *PhaseIterator {
	it := &PhaseIterator{
		phase:            phase,
		subscriptionID:   subs.Subscription.ID,
		iterationEnd:     end,
		periodIndexByKey: make(map[string]int, len(phase.ItemsByKey)),
	}

	it.flattenedItems = make([]subscriptionItemView, 0, len(phase.ItemsByKey))
	for _, items := range phase.ItemsByKey {
		for i, item := range items {
			it.flattenedItems = append(it.flattenedItems, subscriptionItemView{
				SubscriptionItemView: item,
				index:                i,
			})
		}
	}

	return it
}

func (it *PhaseIterator) GetMinimumPeriodEndAfter(t time.Time) time.Time {
	panic("TODO")
}

func (it *PhaseIterator) Seq() iter.Seq[rateCardWithPeriod] {
	// Let's find the maximum billing cadence of an item, this will be the limit of a single pass
	// of generation per item

	maxCadence := datex.Period{}
	for _, item := range it.flattenedItems {
		if item.Spec.RateCard.BillingCadence.DurationApprox() > maxCadence.DurationApprox() {
			// TODO: When can this be nil?
			// TODO: What about fee items or recurring fee items?
			maxCadence = *item.Spec.RateCard.BillingCadence
		}
	}

	if maxCadence.IsZero() {
		// We cannot generate anything, as there is no cadence and the algorithm would just
		// loop infinitely
		return func(yield func(rateCardWithPeriod) bool) {
			return
		}
	}

	return func(yield func(rateCardWithPeriod) bool) {
		iterationStartEpoch := it.phase.SubscriptionPhase.ActiveFrom

		for i := range it.flattenedItems {
			it.flattenedItems[i].lastGenerationTime = iterationStartEpoch
		}

		defer it.Reset()

		for {
			for i := range it.flattenedItems {
				item := &it.flattenedItems[i]
				haltAfter, _ := maxCadence.AddTo(item.lastGenerationTime)

				// TODO: active from to overrides
				for {
					itemPeriodStart := item.lastGenerationTime
					itemPeriodEnd, _ := item.Spec.RateCard.BillingCadence.AddTo(itemPeriodStart)

					if !itemPeriodStart.Before(it.iterationEnd) {
						// Phase ended we should stop

						item.done = true
						break
					}

					generatedItem := rateCardWithPeriod{
						RateCard: item.Spec.RateCard,
						Period: billing.Period{
							Start: item.lastGenerationTime,
							End:   itemPeriodEnd,
						},

						// TODO: let's have a stable sorting on the items in case there are more than one in the subscriptionphaseview
						// so that we are not changing the liens for each generation
						UniqueID: strings.Join([]string{
							it.subscriptionID,
							it.phase.SubscriptionPhase.Key,
							item.Spec.ItemKey,
							fmt.Sprintf("period[%d]", it.periodIndexByKey[item.Spec.ItemKey]),
						}, "/"),
					}

					// Let's compensate for any active from/active to overrides
					generatedItem, shouldYield := it.shouldYield(generatedItem, item)
					if shouldYield {
						if !yield(generatedItem) {
							return
						}

						it.periodIndexByKey[item.Spec.ItemKey]++
					}

					item.lastGenerationTime = itemPeriodEnd

					if !itemPeriodEnd.Before(haltAfter) {
						break
					}
				}
			}

			if it.areAllItemsDone() {
				return
			}
		}
	}
}

// shouldYield generates an item with a period to compensate for any active from/active to overrides
// it returns true if the item should be yielded, false otherwise
func (i *PhaseIterator) shouldYield(generatedItem rateCardWithPeriod, item *subscriptionItemView) (rateCardWithPeriod, bool) {
	// Stage 1: Filtering
	if !generatedItem.Period.End.After(item.SubscriptionItem.ActiveFrom) {
		// This item is not really present in the phase, let's just skip it
		return generatedItem, false
	}

	if item.SubscriptionItem.ActiveTo != nil && !generatedItem.Period.Start.Before(*item.SubscriptionItem.ActiveTo) {
		// This item is not active yet, let's skip it
		return generatedItem, false
	}

	// Let's compensate for any active from/active to overrides
	if item.SubscriptionItem.ActiveFrom.After(generatedItem.Period.Start) {
		generatedItem.Period.Start = item.SubscriptionItem.ActiveFrom
	}

	if item.SubscriptionItem.ActiveTo != nil && item.SubscriptionItem.ActiveTo.Before(generatedItem.Period.End) {
		generatedItem.Period.End = *item.SubscriptionItem.ActiveTo
	}

	return generatedItem, true
}

func (i *PhaseIterator) Reset() {
	// TODO
}

func (i *PhaseIterator) areAllItemsDone() bool {
	for _, item := range i.flattenedItems {
		if !item.done {
			return false
		}
	}

	return true
}
