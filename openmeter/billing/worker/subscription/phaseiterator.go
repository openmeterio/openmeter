package billingworkersubscription

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// timeInfinity is a big enough time that we can use to represent infinity (biggest possible date for our system)
var (
	timeInfinity = time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC)
	maxSafeIter  = 1000
)

type PhaseIterator struct {
	// sub is the Subscription
	sub subscription.SubscriptionView
	// phaseCadence is the cadence of the phase that is being iterated
	phaseCadence models.CadencedModel
	// phase is the phase that is being iterated
	phase subscription.SubscriptionPhaseView

	// observability
	logger *slog.Logger
	tracer trace.Tracer
}

type subscriptionItemWithPeriod struct {
	subscription.SubscriptionItemView
	Period             billing.Period
	UniqueID           string
	NonTruncatedPeriod billing.Period
	PhaseID            string
	InvoiceAligned     bool
}

func (r subscriptionItemWithPeriod) IsTruncated() bool {
	return !r.Period.Equal(r.NonTruncatedPeriod)
}

// PeriodPercentage returns the percentage of the period that is actually billed, compared to the non-truncated period
// can be used to calculate prorated prices
func (r subscriptionItemWithPeriod) PeriodPercentage() alpacadecimal.Decimal {
	nonTruncatedPeriodLength := int64(r.NonTruncatedPeriod.Duration())

	// If the period is empty, we can't calculate the percentage, so we return 1 (100%) to prevent
	// any proration
	if nonTruncatedPeriodLength == 0 {
		return alpacadecimal.NewFromInt(1)
	}

	return alpacadecimal.NewFromInt(int64(r.Period.Duration())).Div(alpacadecimal.NewFromInt(nonTruncatedPeriodLength))
}

func NewPhaseIterator(logger *slog.Logger, tracer trace.Tracer, subs subscription.SubscriptionView, phaseKey string) (*PhaseIterator, error) {
	phase, ok := subs.GetPhaseByKey(phaseKey)
	if !ok {
		return nil, fmt.Errorf("phase %s not found in subscription %s", phaseKey, subs.Subscription.ID)
	}

	if phase == nil {
		return nil, fmt.Errorf("unexpected nil: phase %s not found in subscription %s", phaseKey, subs.Subscription.ID)
	}

	phaseCadence, err := subs.Spec.GetPhaseCadence(phaseKey)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate Cadence for phase %s: %w", phaseKey, err)
	}

	it := &PhaseIterator{
		logger:       logger,
		tracer:       tracer,
		sub:          subs,
		phase:        *phase,
		phaseCadence: phaseCadence,
	}

	return it, nil
}

func (it *PhaseIterator) HasInvoicableItems() bool {
	// If the phase is 0 length it never activates so no items should be generated whatsoever
	if it.phaseCadence.ActiveTo != nil && it.phaseCadence.ActiveTo.Equal(it.phaseCadence.ActiveFrom) {
		return false
	}

	return it.phase.Spec.HasBillables()
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
	for _, itemsByKey := range it.phase.ItemsByKey {
		for _, item := range itemsByKey {
			if item.Spec.RateCard.AsMeta().Price == nil {
				continue
			}

			if item.SubscriptionItem.RateCard.AsMeta().Price.Type() == productcatalog.FlatPriceType {
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

func (it *PhaseIterator) Generate(ctx context.Context, iterationEnd time.Time) ([]subscriptionItemWithPeriod, error) {
	ctx, span := it.tracer.Start(ctx, "billing.worker.subscription.phaseiterator.Generate", trace.WithAttributes(
		attribute.String("phase_key", it.phase.Spec.PhaseKey),
	))
	defer span.End()

	if it.sub.Subscription.BillablesMustAlign {
		return it.generateAligned(ctx, iterationEnd)
	}

	return it.generate(iterationEnd)
}

func (it *PhaseIterator) generateAligned(ctx context.Context, iterationEnd time.Time) ([]subscriptionItemWithPeriod, error) {
	ctx, span := it.tracer.Start(ctx, "billing.worker.subscription.phaseiterator.generateAligned")
	defer span.End()

	if !it.sub.Subscription.BillablesMustAlign {
		return nil, fmt.Errorf("aligned generation is not supported for non-aligned subscriptions")
	}

	items := []subscriptionItemWithPeriod{}

	for _, itemsByKey := range it.phase.ItemsByKey {
		for version, item := range itemsByKey {
			err, breaks := func() (error, bool) {
				ctx, span := it.tracer.Start(ctx, "billing.worker.subscription.phaseiterator.generateAligned.loop", trace.WithAttributes(
					attribute.String("itemKey", item.Spec.ItemKey),
					attribute.Int("itemVersion", version),
					attribute.String("phaseKey", it.phase.Spec.PhaseKey),
				))
				defer span.End()

				logger := it.logger.With("itemKey", item.Spec.ItemKey, "itemVersion", version, "phaseKey", it.phase.Spec.PhaseKey)

				// Let's drop non-billable items
				if item.Spec.RateCard.AsMeta().Price == nil {
					return nil, false // continue
				}

				if item.Spec.RateCard.GetBillingCadence() == nil {
					generatedItem, err := it.generateOneTimeAlignedItem(item, version)
					if err != nil {
						logger.ErrorContext(ctx, "failed to generate one-time aligned item", slog.Any("error", err))
						return err, false
					}

					if generatedItem == nil {
						// One time item is not billable yet, let's skip it
						return nil, true // break
					}

					items = append(items, *generatedItem)
					return nil, false // continue
				}

				periodIdx := 0

				at := item.SubscriptionItem.ActiveFrom
				for {
					err, breaks := func() (error, bool) {
						ctx, span := it.tracer.Start(ctx, "billing.worker.subscription.phaseiterator.generateAligned.loop.inner", trace.WithAttributes(
							attribute.Int("periodIdx", periodIdx),
						))
						defer span.End()

						// We start when the item activates, then advance until either
						logger := logger.With("periodIdx", periodIdx)

						// We start when the item activates, then advance until either
						nonTruncatedPeriod, err := it.sub.Spec.GetAlignedBillingPeriodAt(it.phase.Spec.PhaseKey, at)
						if err != nil {
							logger.ErrorContext(ctx, "failed to get aligned billing period", slog.Any("error", err))
							return err, false
						}

						// Period otherwise is truncated by activeFrom and activeTo times
						period := timeutil.ClosedPeriod{
							From: nonTruncatedPeriod.From,
							To:   nonTruncatedPeriod.To,
						}

						if item.SubscriptionItem.ActiveFrom.After(period.From) {
							period.From = item.SubscriptionItem.ActiveFrom
						}

						if item.SubscriptionItem.ActiveTo != nil && item.SubscriptionItem.ActiveTo.Before(period.To) {
							period.To = *item.SubscriptionItem.ActiveTo
						}

						// Let's build the line
						generatedItem := subscriptionItemWithPeriod{
							SubscriptionItemView: item,
							InvoiceAligned:       true,

							Period: billing.Period{
								Start: period.From,
								End:   period.To,
							},

							UniqueID: strings.Join([]string{
								it.sub.Subscription.ID,
								it.phase.Spec.PhaseKey,
								item.Spec.ItemKey,
								fmt.Sprintf("v[%d]", version),
								fmt.Sprintf("period[%d]", periodIdx),
							}, "/"),

							NonTruncatedPeriod: billing.Period{
								Start: nonTruncatedPeriod.From,
								End:   nonTruncatedPeriod.To,
							},
							PhaseID: it.phase.SubscriptionPhase.ID,
						}

						items = append(items, generatedItem)

						// We advance the period counter for ID-ing
						periodIdx++
						// And we try to go to the next period (end times are exclusive)
						at = period.To

						// 1. it deactivates
						if item.SubscriptionItem.ActiveTo != nil && !at.Before(*item.SubscriptionItem.ActiveTo) {
							logger.DebugContext(ctx, "exiting loop due to item deactivation", slog.Time("at", at), slog.Time("activeTo", *item.SubscriptionItem.ActiveTo))
							return nil, true
						}
						// 2. the phase ends
						if it.phaseCadence.ActiveTo != nil && !at.Before(*it.phaseCadence.ActiveTo) {
							logger.DebugContext(ctx, "exiting loop due to phase end", slog.Time("at", at), slog.Time("activeTo", *it.phaseCadence.ActiveTo))
							return nil, true
						}
						// 3. we reach the iteration end
						if !at.Before(iterationEnd) {
							logger.DebugContext(ctx, "exiting loop due to iteration end", slog.Time("at", at), slog.Time("iterationEnd", iterationEnd))
							return nil, true
						}
						// 4. we reach the max iterations
						if periodIdx > maxSafeIter {
							logger.ErrorContext(ctx, "max iterations reached", slog.Any("iterator", it), slog.String("stack", string(debug.Stack())))
							return nil, true
						}

						logger.DebugContext(ctx, "iterating", slog.Time("at", at))

						return nil, false
					}()
					if err != nil {
						return err, false
					}

					if breaks {
						break
					}
				}

				return nil, false // continue
			}()

			if err != nil {
				return nil, err
			}

			if breaks {
				break
			}
		}
	}

	return it.truncateItemsIfNeeded(items), nil
}

func (it *PhaseIterator) generate(iterationEnd time.Time) ([]subscriptionItemWithPeriod, error) {
	out := []subscriptionItemWithPeriod{}
	for _, itemsByKey := range it.phase.ItemsByKey {
		slices.SortFunc(itemsByKey, func(i, j subscription.SubscriptionItemView) int {
			return timeutil.Compare(i.SubscriptionItem.ActiveFrom, j.SubscriptionItem.ActiveFrom)
		})

		for versionID, item := range itemsByKey {
			// Let's drop non-billable items
			if item.Spec.RateCard.AsMeta().Price == nil {
				continue
			}

			if item.Spec.RateCard.GetBillingCadence() == nil {
				generatedItem, err := it.generateOneTimeItem(item, versionID)
				if err != nil {
					return nil, err
				}

				if generatedItem == nil {
					// One time item is not billable yet, let's skip it
					break
				}

				out = append(out, *generatedItem)
				continue
			}

			start := item.SubscriptionItem.ActiveFrom
			periodID := 0

			for {
				// Should not happen here
				bCad := item.Spec.RateCard.GetBillingCadence()
				if bCad == nil {
					return nil, fmt.Errorf("no billing cadence found for item %s", item.Spec.ItemKey)
				}

				end, _ := bCad.AddTo(start)

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
						it.sub.Subscription.ID,
						it.phase.Spec.PhaseKey,
						item.Spec.ItemKey,
						fmt.Sprintf("v[%d]", versionID),
						fmt.Sprintf("period[%d]", periodID),
					}, "/"),

					NonTruncatedPeriod: nonTruncatedPeriod,
					PhaseID:            it.phase.SubscriptionPhase.ID,
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
		if item.Spec.RateCard.AsMeta().Price != nil && item.Spec.RateCard.AsMeta().Price.Type() == productcatalog.FlatPriceType {
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

func (it *PhaseIterator) generateOneTimeAlignedItem(item subscription.SubscriptionItemView, versionID int) (*subscriptionItemWithPeriod, error) {
	if item.Spec.RateCard.AsMeta().Price == nil {
		return nil, nil
	}

	alignedPeriod, err := it.sub.Spec.GetAlignedBillingPeriodAt(it.phase.Spec.PhaseKey, item.SubscriptionItem.ActiveFrom)
	if err != nil {
		// If there isn't a period to align with, we generate a simple oneTime item
		return it.generateOneTimeItem(item, versionID)
	}

	nonTruncatedPeriod := billing.Period{
		Start: alignedPeriod.From,
		End:   alignedPeriod.To,
	}

	period := billing.Period{
		Start: item.SubscriptionItem.ActiveFrom,
	}

	end := lo.CoalesceOrEmpty(item.SubscriptionItem.ActiveTo, it.phaseCadence.ActiveTo)
	if end == nil {
		// One time items are not usage based, so the price object will be a flat price
		price := item.SubscriptionItem.RateCard.AsMeta().Price

		if price == nil {
			// If an item has no price it is not in scope for line generation
			return nil, nil
		}

		if price.Type() != productcatalog.FlatPriceType {
			return nil, fmt.Errorf("cannot determine period end for one-time item %s", item.Spec.ItemKey)
		}

		flatFee, err := price.AsFlat()
		if err != nil {
			return nil, err
		}

		if flatFee.PaymentTerm == productcatalog.InArrearsPaymentTerm {
			// If the item is InArrears but we cannot determine when that time is, let's just skip this item until we
			// can determine the end of period
			return nil, nil
		}

		// For in-advance fees we just specify an empty period, which is fine for non UBP items
		period.End = item.SubscriptionItem.ActiveFrom
	} else {
		period.End = *end
	}

	return &subscriptionItemWithPeriod{
		InvoiceAligned:       true,
		SubscriptionItemView: item,
		Period:               period,
		NonTruncatedPeriod:   nonTruncatedPeriod,
		UniqueID: strings.Join([]string{
			it.sub.Subscription.ID,
			it.phase.Spec.PhaseKey,
			item.Spec.ItemKey,
			fmt.Sprintf("v[%d]", versionID),
		}, "/"),
		PhaseID: it.phase.SubscriptionPhase.ID,
	}, nil
}

func (it *PhaseIterator) generateOneTimeItem(item subscription.SubscriptionItemView, versionID int) (*subscriptionItemWithPeriod, error) {
	period := billing.Period{
		Start: item.SubscriptionItem.ActiveFrom,
	}

	end := lo.CoalesceOrEmpty(item.SubscriptionItem.ActiveTo, it.phaseCadence.ActiveTo)
	if end == nil {
		// One time items are not usage based, so the price object will be a flat price
		price := item.SubscriptionItem.RateCard.AsMeta().Price

		if price == nil {
			// If an item has no price it is not in scope for line generation
			return nil, nil
		}

		if price.Type() != productcatalog.FlatPriceType {
			return nil, fmt.Errorf("cannot determine period end for one-time item %s", item.Spec.ItemKey)
		}

		flatFee, err := price.AsFlat()
		if err != nil {
			return nil, err
		}

		if flatFee.PaymentTerm == productcatalog.InArrearsPaymentTerm {
			// If the item is InArrears but we cannot determine when that time is, let's just skip this item until we
			// can determine the end of period
			return nil, nil
		}

		// For in-advance fees we just specify an empty period, which is fine for non UBP items
		period.End = item.SubscriptionItem.ActiveFrom
	} else {
		period.End = *end
	}

	return &subscriptionItemWithPeriod{
		SubscriptionItemView: item,
		Period:               period,
		NonTruncatedPeriod:   period,
		UniqueID: strings.Join([]string{
			it.sub.Subscription.ID,
			it.phase.Spec.PhaseKey,
			item.Spec.ItemKey,
			fmt.Sprintf("v[%d]", versionID),
		}, "/"),
		PhaseID: it.phase.SubscriptionPhase.ID,
	}, nil
}
