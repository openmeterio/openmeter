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
	"github.com/openmeterio/openmeter/pkg/framework/tracex"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
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

type subscriptionItemWithPeriods struct {
	subscription.SubscriptionItemView
	// References
	UniqueID string
	PhaseID  string

	// Period Information

	// ServicePeriod is the de-facto service period that the item is billed for
	ServicePeriod billing.Period
	// FullServicePeriod is the full service period that the item is billed for (previously nonTruncatedPeriod)
	FullServicePeriod billing.Period

	// BillingPeriod as determined by alignment and service period
	BillingPeriod billing.Period
}

// PeriodPercentage returns the percentage of the period that is actually billed, compared to the non-truncated period
// can be used to calculate prorated prices
func (r subscriptionItemWithPeriods) PeriodPercentage() alpacadecimal.Decimal {
	fullServicePeriodLength := int64(r.FullServicePeriod.Duration())

	// If the period is empty, we can't calculate the percentage, so we return 1 (100%) to prevent
	// any proration
	if fullServicePeriodLength == 0 {
		return alpacadecimal.NewFromInt(1)
	}

	return alpacadecimal.NewFromInt(int64(r.ServicePeriod.Duration())).Div(alpacadecimal.NewFromInt(fullServicePeriodLength))
}

func (r subscriptionItemWithPeriods) GetInvoiceAt() time.Time {
	// Flat-fee in advance is the only case we bill in advance
	if r.Spec.RateCard.AsMeta().Price.Type() == productcatalog.FlatPriceType {
		flatFee, _ := r.Spec.RateCard.AsMeta().Price.AsFlat()
		if flatFee.PaymentTerm == productcatalog.InAdvancePaymentTerm {
			// In advance invoicing
			// For in advance invoicing we attempt to incoice at the start of the billing period
			return r.BillingPeriod.Start
		}
	}

	// All other items are invoiced after the fact, meaning
	// - not before its billing period is over
	// - not before its service period is over
	return lo.Latest(r.ServicePeriod.End, r.BillingPeriod.End)
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

// Generate generates the lines for the phase so that all active subscription item's are generated up to the point
// where either the item gets deactivated or the last item's invoice_at >= iterationEnd and it's period's end is equal to
// or after iterationEnd.
//
// This ensures that we always have the upcoming lines stored on the gathering invoice.
func (it *PhaseIterator) Generate(ctx context.Context, iterationEnd time.Time) ([]subscriptionItemWithPeriods, error) {
	ctx, span := tracex.Start[[]subscriptionItemWithPeriods](ctx, it.tracer, "billing.worker.subscription.phaseiterator.Generate", trace.WithAttributes(
		attribute.String("phase_key", it.phase.Spec.PhaseKey),
	))

	return span.Wrap(ctx, func(ctx context.Context) ([]subscriptionItemWithPeriods, error) {
		if it.sub.Subscription.BillablesMustAlign {
			return it.generateAligned(ctx, iterationEnd)
		}

		return it.generate(iterationEnd)
	})
}

func (it *PhaseIterator) generateAligned(ctx context.Context, iterationEnd time.Time) ([]subscriptionItemWithPeriods, error) {
	ctx, span := tracex.Start[[]subscriptionItemWithPeriods](ctx, it.tracer, "billing.worker.subscription.phaseiterator.generateAligned")

	return span.Wrap(ctx, func(ctx context.Context) ([]subscriptionItemWithPeriods, error) {
		if !it.sub.Subscription.BillablesMustAlign {
			return nil, fmt.Errorf("aligned generation is not supported for non-aligned subscriptions")
		}

		items := []subscriptionItemWithPeriods{}

		for _, itemsByKey := range it.phase.ItemsByKey {
			err := slicesx.ForEachUntilWithErr(
				itemsByKey,
				func(item subscription.SubscriptionItemView, version int) (breaks bool, err error) {
					return it.generateForAlignedItemVersion(ctx, item, version, iterationEnd, &items)
				},
			)
			if err != nil {
				return nil, err
			}
		}

		return it.truncateItemsIfNeeded(items), nil
	})
}

func (it *PhaseIterator) generateForAlignedItemVersion(ctx context.Context, item subscription.SubscriptionItemView, version int, iterationEnd time.Time, items *[]subscriptionItemWithPeriods) (bool, error) {
	ctx, span := tracex.Start[bool](ctx, it.tracer, "billing.worker.subscription.phaseiterator.generateForAlignedItemVersion", trace.WithAttributes(
		attribute.String("itemKey", item.Spec.ItemKey),
		attribute.Int("itemVersion", version),
		attribute.String("phaseKey", it.phase.Spec.PhaseKey),
	))

	return span.Wrap(ctx, func(ctx context.Context) (bool, error) {
		logger := it.logger.With("itemKey", item.Spec.ItemKey, "itemVersion", version, "phaseKey", it.phase.Spec.PhaseKey)

		// Let's drop non-billable items
		if item.Spec.RateCard.AsMeta().Price == nil {
			return false, nil
		}

		if item.Spec.RateCard.GetBillingCadence() == nil {
			generatedItem, err := it.generateOneTimeAlignedItem(item, version)
			if err != nil {
				logger.ErrorContext(ctx, "failed to generate one-time aligned item", slog.Any("error", err))
				return false, err
			}

			if generatedItem == nil {
				// One time item is not billable yet, let's skip it
				return true, nil
			}

			*items = append(*items, *generatedItem)

			return false, nil
		}

		periodIdx := 0
		at := item.SubscriptionItem.ActiveFrom

		// If the item is already past the subscription end, we can ignore it
		if it.sub.Spec.ActiveTo != nil && !at.Before(*it.sub.Spec.ActiveTo) {
			return true, nil
		}

		// Should not happen, being a bit defensive here
		if it.phaseCadence.ActiveTo != nil && !at.Before(*it.phaseCadence.ActiveTo) {
			return true, nil
		}

		for {
			logger := logger.With("periodIdx", periodIdx, "periodAt", at)

			newItem, err := it.generateForAlignedItemVersionPeriod(ctx, logger, item, version, periodIdx, at)
			if err != nil {
				return false, err
			}

			// Let's increment
			periodIdx = periodIdx + 1
			at = newItem.ServicePeriod.End

			// Check if we have reached the iteration end based on invoiceAt
			if newItem.GetInvoiceAt().After(iterationEnd) {
				logger.DebugContext(ctx, "exiting loop due to iteration end", slog.Time("at", at), slog.Time("iterationEnd", iterationEnd), slog.Time("invoiceAt", newItem.GetInvoiceAt()))
				break
			}

			*items = append(*items, newItem)

			// We start when the item activates, then advance until either
			// 1. it deactivates
			if item.SubscriptionItem.ActiveTo != nil && !at.Before(*item.SubscriptionItem.ActiveTo) {
				logger.DebugContext(ctx, "exiting loop due to item deactivation", slog.Time("at", at), slog.Time("activeTo", *item.SubscriptionItem.ActiveTo))
				break
			}

			// 2. the phase ends
			if it.phaseCadence.ActiveTo != nil && !at.Before(*it.phaseCadence.ActiveTo) {
				logger.DebugContext(ctx, "exiting loop due to phase end", slog.Time("at", at), slog.Time("activeTo", *it.phaseCadence.ActiveTo))
				break
			}

			// 4. we reach the max iterations
			if periodIdx > maxSafeIter {
				logger.ErrorContext(ctx, "max iterations reached", slog.Any("iterator", it), slog.String("stack", string(debug.Stack())))
				break
			}

			logger.DebugContext(ctx, "iterating", slog.Time("at", at))
		}

		return false, nil
	})
}

type generatedVersionPeriodItem struct {
	period    timeutil.ClosedPeriod
	invoiceAt time.Time
	index     int
	item      subscriptionItemWithPeriods
}

func (it *PhaseIterator) generateForAlignedItemVersionPeriod(ctx context.Context, logger *slog.Logger, item subscription.SubscriptionItemView, version int, periodIdx int, at time.Time) (subscriptionItemWithPeriods, error) {
	ctx, span := tracex.Start[subscriptionItemWithPeriods](ctx, it.tracer, "billing.worker.subscription.phaseiterator.generateForAlignedItemVersionPeriod", trace.WithAttributes(
		attribute.Int("periodIdx", periodIdx),
		attribute.String("periodAt", at.Format(time.RFC3339)),
	))

	return span.Wrap(ctx, func(ctx context.Context) (subscriptionItemWithPeriods, error) {
		var empty subscriptionItemWithPeriods

		if !it.sub.Subscription.BillablesMustAlign {
			return empty, fmt.Errorf("aligned generation is not supported for non-aligned subscriptions")
		}

		billingPeriod, err := it.sub.Spec.GetAlignedBillingPeriodAt(it.phase.Spec.PhaseKey, at)
		if err != nil {
			logger.ErrorContext(ctx, "failed to get aligned billing period", slog.Any("error", err))
			return empty, err
		}

		fullServicePeriod, err := item.Spec.GetFullServicePeriodAt(
			it.phaseCadence,
			item.SubscriptionItem.CadencedModel,
			at,
			&billingPeriod.From, // We can use the billing period start as that's already aligned
		)
		if err != nil {
			logger.ErrorContext(ctx, "failed to get full service period", slog.Any("error", err))
			return empty, err
		}

		servicePeriod, err := fullServicePeriod.Open().Intersection(item.SubscriptionItem.CadencedModel.AsPeriod()).Closed()
		if err != nil {
			logger.ErrorContext(ctx, "failed to get service period", slog.Any("error", err))
			return empty, err
		}

		// Let's build the line
		generatedItem := subscriptionItemWithPeriods{
			SubscriptionItemView: item,

			UniqueID: strings.Join([]string{
				it.sub.Subscription.ID,
				it.phase.Spec.PhaseKey,
				item.Spec.ItemKey,
				fmt.Sprintf("v[%d]", version),
				fmt.Sprintf("period[%d]", periodIdx),
			}, "/"),
			PhaseID: it.phase.SubscriptionPhase.ID,

			ServicePeriod: billing.Period{
				Start: servicePeriod.From,
				End:   servicePeriod.To,
			},
			FullServicePeriod: billing.Period{
				Start: fullServicePeriod.From,
				End:   fullServicePeriod.To,
			},
			BillingPeriod: billing.Period{
				Start: billingPeriod.From,
				End:   billingPeriod.To,
			},
		}

		return generatedItem, nil
	})
}

func (it *PhaseIterator) generate(iterationEnd time.Time) ([]subscriptionItemWithPeriods, error) {
	out := []subscriptionItemWithPeriods{}
	for _, itemsByKey := range it.phase.ItemsByKey {
		slices.SortFunc(itemsByKey, func(i, j subscription.SubscriptionItemView) int {
			return timeutil.Compare(i.SubscriptionItem.ActiveFrom, j.SubscriptionItem.ActiveFrom)
		})

		for versionID, item := range itemsByKey {
			// Let's drop non-billable items
			if !item.Spec.RateCard.IsBillable() {
				continue
			}

			price := item.Spec.RateCard.AsMeta().Price
			if price == nil {
				return nil, fmt.Errorf("item %s should have price", item.Spec.ItemKey)
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
				itemCadence := item.SubscriptionItem.CadencedModel
				fullServicePeriod, err := item.Spec.GetFullServicePeriodAt(
					it.phaseCadence,
					itemCadence,
					start,
					nil,
				)
				if err != nil {
					return nil, err
				}

				servicePeriod, err := fullServicePeriod.Open().Intersection(itemCadence.AsPeriod()).Closed()
				if err != nil {
					return nil, fmt.Errorf("failed to get service period: %w", err)
				}

				// As billing is not aligned, we'll simply bill by the full service period, cut short by phase cadence
				billingPeriod := fullServicePeriod
				if it.phaseCadence.ActiveTo != nil && billingPeriod.To.After(*it.phaseCadence.ActiveTo) {
					billingPeriod.To = *it.phaseCadence.ActiveTo
				}

				generatedItem := subscriptionItemWithPeriods{
					SubscriptionItemView: item,

					UniqueID: strings.Join([]string{
						it.sub.Subscription.ID,
						it.phase.Spec.PhaseKey,
						item.Spec.ItemKey,
						fmt.Sprintf("v[%d]", versionID),
						fmt.Sprintf("period[%d]", periodID),
					}, "/"),
					PhaseID: it.phase.SubscriptionPhase.ID,

					FullServicePeriod: billing.Period{
						Start: fullServicePeriod.From,
						End:   fullServicePeriod.To,
					},
					ServicePeriod: billing.Period{
						Start: servicePeriod.From,
						End:   servicePeriod.To,
					},

					BillingPeriod: billing.Period{
						Start: billingPeriod.From,
						End:   billingPeriod.To,
					},
				}

				out = append(out, generatedItem)

				periodID++
				start = servicePeriod.To

				// Either we have reached the end of the phase
				if it.phaseCadence.ActiveTo != nil && !start.Before(*it.phaseCadence.ActiveTo) {
					break
				}

				// We have reached the end of the active range
				if item.SubscriptionItem.ActiveTo != nil && !start.Before(*item.SubscriptionItem.ActiveTo) {
					break
				}

				// Or we have reached the iteration end
				if !start.Before(iterationEnd) && !generatedItem.GetInvoiceAt().Before(iterationEnd) {
					break
				}
			}
		}
	}

	return it.truncateItemsIfNeeded(out), nil
}

func (it *PhaseIterator) truncateItemsIfNeeded(in []subscriptionItemWithPeriods) []subscriptionItemWithPeriods {
	out := make([]subscriptionItemWithPeriods, 0, len(in))
	// We need to sanitize the output to compensate for the 1min resolution of meters
	for _, item := range in {
		// We only need to sanitize the items that are not flat priced, flat prices can be handled in any resolution
		if item.Spec.RateCard.AsMeta().Price != nil && item.Spec.RateCard.AsMeta().Price.Type() == productcatalog.FlatPriceType {
			out = append(out, item)
			continue
		}

		// We truncate the service period to the meter resolution
		item.ServicePeriod = item.ServicePeriod.Truncate(billing.DefaultMeterResolution)
		if item.ServicePeriod.IsEmpty() {
			continue
		}

		out = append(out, item)
	}

	return out
}

func (it *PhaseIterator) generateOneTimeAlignedItem(item subscription.SubscriptionItemView, versionID int) (*subscriptionItemWithPeriods, error) {
	if !it.sub.Subscription.BillablesMustAlign {
		return nil, fmt.Errorf("aligned generation is not supported for non-aligned subscriptions")
	}

	if item.Spec.RateCard.AsMeta().Price == nil {
		return nil, nil
	}

	itemCadence := item.SubscriptionItem.CadencedModel

	billingPeriod, err := it.sub.Spec.GetAlignedBillingPeriodAt(it.phase.Spec.PhaseKey, itemCadence.ActiveFrom)
	if err != nil {
		return nil, fmt.Errorf("failed to get aligned billing period at %s: %w", itemCadence.ActiveFrom, err)
	}

	fullServicePeriod, err := item.Spec.GetFullServicePeriodAt(
		it.phaseCadence,
		itemCadence,
		itemCadence.ActiveFrom,
		&billingPeriod.From, // we can just use the billing period start as that's already aligned
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get full service period at %s: %w", item.SubscriptionItem.ActiveFrom, err)
	}

	// The service period is the intersection of the full service period and the item cadence
	// As fullServicePeriod is a closed period, this intersection will always have both start and end (be closed)
	servicePeriodOpen := fullServicePeriod.Open().Intersection(itemCadence.AsPeriod())

	if servicePeriodOpen == nil && fullServicePeriod.Duration() == time.Duration(0) {
		// If the service period is an instant, we'll bill at the same time as the service period
		servicePeriodOpen = lo.ToPtr(fullServicePeriod.Open())
	}

	if servicePeriodOpen == nil {
		return nil, fmt.Errorf("service period is empty, cadence is [from %s to %s], full service period is [from %s to %s]", itemCadence.ActiveFrom, itemCadence.ActiveTo, fullServicePeriod.From, fullServicePeriod.To)
	}

	servicePeriod, err := servicePeriodOpen.Closed()
	if err != nil {
		return nil, fmt.Errorf("failed to get service period: %w", err)
	}

	return &subscriptionItemWithPeriods{
		SubscriptionItemView: item,

		UniqueID: strings.Join([]string{
			it.sub.Subscription.ID,
			it.phase.Spec.PhaseKey,
			item.Spec.ItemKey,
			fmt.Sprintf("v[%d]", versionID),
		}, "/"),
		PhaseID: it.phase.SubscriptionPhase.ID,

		ServicePeriod: billing.Period{
			Start: servicePeriod.From,
			End:   servicePeriod.To,
		},
		FullServicePeriod: billing.Period{
			Start: fullServicePeriod.From,
			End:   fullServicePeriod.To,
		},
		BillingPeriod: billing.Period{
			Start: billingPeriod.From,
			End:   billingPeriod.To,
		},
	}, nil
}

func (it *PhaseIterator) generateOneTimeItem(item subscription.SubscriptionItemView, versionID int) (*subscriptionItemWithPeriods, error) {
	itemCadence := item.SubscriptionItem.CadencedModel

	fullServicePeriod, err := item.Spec.GetFullServicePeriodAt(
		it.phaseCadence,
		itemCadence,
		item.SubscriptionItem.ActiveFrom,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get full service period: %w", err)
	}

	servicePeriodOpen := fullServicePeriod.Open().Intersection(itemCadence.AsPeriod())

	if servicePeriodOpen == nil && fullServicePeriod.Duration() == time.Duration(0) {
		// If the service period is an instant, we'll bill at the same time as the service period
		servicePeriodOpen = lo.ToPtr(fullServicePeriod.Open())
	}

	if servicePeriodOpen == nil {
		return nil, fmt.Errorf("service period is empty, cadence is [from %s to %s], full service period is [from %s to %s]", itemCadence.ActiveFrom, itemCadence.ActiveTo, fullServicePeriod.From, fullServicePeriod.To)
	}

	servicePeriod, err := servicePeriodOpen.Closed()
	if err != nil {
		return nil, fmt.Errorf("failed to get service period: %w", err)
	}

	// As this is a one-time item, the billing period is the same as the full service period
	billingPeriod := fullServicePeriod

	return &subscriptionItemWithPeriods{
		SubscriptionItemView: item,

		UniqueID: strings.Join([]string{
			it.sub.Subscription.ID,
			it.phase.Spec.PhaseKey,
			item.Spec.ItemKey,
			fmt.Sprintf("v[%d]", versionID),
		}, "/"),
		PhaseID: it.phase.SubscriptionPhase.ID,

		FullServicePeriod: billing.Period{
			Start: fullServicePeriod.From,
			End:   fullServicePeriod.To,
		},
		ServicePeriod: billing.Period{
			Start: servicePeriod.From,
			End:   servicePeriod.To,
		},
		BillingPeriod: billing.Period{
			Start: billingPeriod.From,
			End:   billingPeriod.To,
		},
	}, nil
}
