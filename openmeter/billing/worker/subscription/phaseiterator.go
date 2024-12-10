package billingworkersubscription

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timex"
)

// timeInfinity is a big enough time that we can use to represent infinity (biggest possible date for our system)
var timeInfinity = time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC)

type PhaseIterator struct {
	// subscriptionID is the ID of the subscription that is being iterated (used for unique ID generation)
	subscriptionID string
	// phaseKey is the key of the phase that is being iterated (used for unique ID generation)
	phaseKey string
	// phaseID is the database ID of the phase that is being iterated (used for DB references)
	phaseID string
	// phaseCadence is the cadence of the phase that is being iterated
	phaseCadence models.CadencedModel

	items [][]subscription.SubscriptionItemView
}

type subscriptionItemWithPeriod struct {
	subscription.SubscriptionItemView
	Period             billing.Period
	UniqueID           string
	NonTruncatedPeriod billing.Period
	PhaseID            string
}

func (r subscriptionItemWithPeriod) IsTruncated() bool {
	return !r.Period.Equal(r.NonTruncatedPeriod)
}

// PeriodPercentage returns the percentage of the period that is actually billed, compared to the non-truncated period
// can be used to calculate prorated prices
func (r subscriptionItemWithPeriod) PeriodPercentage() alpacadecimal.Decimal {
	return alpacadecimal.NewFromInt(int64(r.Period.Duration())).Div(alpacadecimal.NewFromInt(int64(r.NonTruncatedPeriod.Duration())))
}

func NewPhaseIterator(subs subscription.SubscriptionView, phaseKey string) (*PhaseIterator, error) {
	it := &PhaseIterator{
		subscriptionID: subs.Subscription.ID,
		phaseKey:       phaseKey,
	}

	return it, it.ResolvePhaseData(subs, phaseKey)
}

func (it *PhaseIterator) ResolvePhaseData(subs subscription.SubscriptionView, phaseKey string) error {
	phaseCadence := models.CadencedModel{}
	var currentPhase *subscription.SubscriptionPhaseView

	slices.SortFunc(subs.Phases, func(i, j subscription.SubscriptionPhaseView) int {
		return timex.Compare(i.SubscriptionPhase.ActiveFrom, j.SubscriptionPhase.ActiveFrom)
	})

	for i, phase := range subs.Phases {
		if phase.SubscriptionPhase.Key == phaseKey {
			phaseCadence.ActiveFrom = phase.SubscriptionPhase.ActiveFrom

			if i < len(subs.Phases)-1 {
				phaseCadence.ActiveTo = lo.ToPtr(subs.Phases[i+1].SubscriptionPhase.ActiveFrom)
			}

			currentPhase = &phase

			break
		}
	}

	if currentPhase == nil {
		return fmt.Errorf("phase %s not found in subscription %s", phaseKey, subs.Subscription.ID)
	}

	it.phaseCadence = phaseCadence
	it.phaseID = currentPhase.SubscriptionPhase.ID

	it.items = make([][]subscription.SubscriptionItemView, 0, len(currentPhase.ItemsByKey))
	for _, items := range currentPhase.ItemsByKey {
		slices.SortFunc(items, func(i, j subscription.SubscriptionItemView) int {
			return timex.Compare(i.SubscriptionItem.ActiveFrom, j.SubscriptionItem.ActiveFrom)
		})

		it.items = append(it.items, items)
	}

	return nil
}

func (it *PhaseIterator) HasInvoicableItems() bool {
	for _, itemsByKey := range it.items {
		for _, item := range itemsByKey {
			if item.Spec.RateCard.Price != nil {
				return true
			}
		}
	}

	return false
}

func (it *PhaseIterator) PhaseEnd() *time.Time {
	return it.phaseCadence.ActiveTo
}

func (it *PhaseIterator) PhaseStart() time.Time {
	return it.phaseCadence.ActiveFrom
}

// GetMinimumBillableTime returns the minimum time that we can bill for the phase (e.g. the first time we would be
// yielding a line item)
func (it *PhaseIterator) GetMinimumBillableTime() time.Time {
	minTime := timeInfinity
	for _, itemsByKey := range it.items {
		for _, item := range itemsByKey {
			if item.Spec.RateCard.Price == nil {
				continue
			}

			if item.SubscriptionItem.RateCard.Price.Type() == productcatalog.FlatPriceType {
				if item.SubscriptionItem.ActiveFrom.Before(minTime) {
					minTime = item.SubscriptionItem.ActiveFrom
				}
			} else {
				// Let's make sure that truncation won't filter out the item
				period := billing.Period{
					Start: item.SubscriptionItem.ActiveFrom,
					End:   timeInfinity,
				}

				if item.SubscriptionItem.ActiveTo != nil {
					period.End = *item.SubscriptionItem.ActiveTo
				}

				if it.phaseCadence.ActiveTo != nil && period.End.After(*it.phaseCadence.ActiveTo) {
					period.End = *it.phaseCadence.ActiveTo
				}

				period = period.Truncate(billing.DefaultMeterResolution)
				if period.IsEmpty() {
					continue
				}

				if period.Start.Before(minTime) {
					minTime = period.Start
				}
			}
		}
	}

	return minTime
}

func (it *PhaseIterator) Generate(iterationEnd time.Time) ([]subscriptionItemWithPeriod, error) {
	out := []subscriptionItemWithPeriod{}
	for _, itemsByKey := range it.items {
		slices.SortFunc(itemsByKey, func(i, j subscription.SubscriptionItemView) int {
			return timex.Compare(i.SubscriptionItem.ActiveFrom, j.SubscriptionItem.ActiveFrom)
		})

		for versionID, item := range itemsByKey {
			// Let's drop non-billable items
			if item.Spec.RateCard.Price == nil {
				continue
			}

			if item.Spec.RateCard.BillingCadence == nil {
				generatedItem, err := it.generateOneTimeItem(item, versionID)
				if err != nil {
					return nil, err
				}
				out = append(out, generatedItem)
				continue
			}

			start := item.SubscriptionItem.ActiveFrom
			periodID := 0

			for {
				end, _ := item.Spec.RateCard.BillingCadence.AddTo(start)

				nonTruncatedPeriod := billing.Period{
					Start: start,
					End:   end,
				}

				if item.SubscriptionItem.ActiveTo != nil && item.SubscriptionItem.ActiveTo.Before(end) {
					end = *item.SubscriptionItem.ActiveTo
				}

				if it.phaseCadence.ActiveTo != nil && end.After(*it.phaseCadence.ActiveTo) {
					end = *it.phaseCadence.ActiveTo
				}

				generatedItem := subscriptionItemWithPeriod{
					SubscriptionItemView: item,
					Period: billing.Period{
						Start: start,
						End:   end,
					},

					UniqueID: strings.Join([]string{
						it.subscriptionID,
						it.phaseKey,
						item.Spec.ItemKey,
						fmt.Sprintf("v[%d]", versionID),
						fmt.Sprintf("period[%d]", periodID),
					}, "/"),

					NonTruncatedPeriod: nonTruncatedPeriod,
					PhaseID:            it.phaseID,
				}

				out = append(out, generatedItem)

				periodID++
				start = end

				// Either we have reached the end of the phase
				if it.phaseCadence.ActiveTo != nil && !start.Before(*it.phaseCadence.ActiveTo) {
					break
				}

				// We have reached the end of the active range
				if item.SubscriptionItem.ActiveTo != nil && !start.Before(*item.SubscriptionItem.ActiveTo) {
					break
				}

				// Or we have reached the iteration end
				if !start.Before(iterationEnd) {
					break
				}
			}
		}
	}

	return it.truncateItemsIfNeeded(out), nil
}

func (it *PhaseIterator) truncateItemsIfNeeded(in []subscriptionItemWithPeriod) []subscriptionItemWithPeriod {
	out := make([]subscriptionItemWithPeriod, 0, len(in))
	// We need to sanitize the output to compensate for the 1min resolution of meters
	for _, item := range in {
		// We only need to sanitize the items that are not flat priced, flat prices can be handled in any resolution
		if item.Spec.RateCard.Price != nil && item.Spec.RateCard.Price.Type() == productcatalog.FlatPriceType {
			out = append(out, item)
			continue
		}

		item.Period = item.Period.Truncate(billing.DefaultMeterResolution)
		if item.Period.IsEmpty() {
			continue
		}

		item.NonTruncatedPeriod = item.NonTruncatedPeriod.Truncate(billing.DefaultMeterResolution)

		out = append(out, item)
	}

	return out
}

func (it *PhaseIterator) generateOneTimeItem(item subscription.SubscriptionItemView, versionID int) (subscriptionItemWithPeriod, error) {
	end := lo.CoalesceOrEmpty(item.SubscriptionItem.ActiveTo, it.phaseCadence.ActiveTo)
	if end == nil {
		// TODO[later]: implement open ended gathering line items, as that's a valid use case to for example:
		// Have a plan, that has an open ended billing item for flat fee, then the end user uses progressive billing
		// to bill the end user if the usage gets above $1000. Non-gathering lines must have a period end.
		return subscriptionItemWithPeriod{}, fmt.Errorf("cannot determine phase end for item %s", item.Spec.ItemKey)
	}

	period := billing.Period{
		Start: item.SubscriptionItem.ActiveFrom,
		End:   *end,
	}

	return subscriptionItemWithPeriod{
		SubscriptionItemView: item,
		Period:               period,
		NonTruncatedPeriod:   period,
		UniqueID: strings.Join([]string{
			it.subscriptionID,
			it.phaseKey,
			item.Spec.ItemKey,
			fmt.Sprintf("v[%d]", versionID),
		}, "/"),
		PhaseID: it.phaseID,
	}, nil
}
