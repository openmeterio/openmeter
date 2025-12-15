package service

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/streaming"
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
	PhaseKey string

	PeriodIndex int
	ItemVersion int

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
//
// The response always truncated to capture that billing has 1s resolution.
func (it *PhaseIterator) GetMinimumBillableTime() time.Time {
	minTime := timeInfinity
	for _, itemsByKey := range it.phase.ItemsByKey {
		for _, item := range itemsByKey {
			if item.Spec.RateCard.AsMeta().Price == nil {
				continue
			}

			if item.SubscriptionItem.RateCard.AsMeta().Price.Type() == productcatalog.FlatPriceType {
				if item.SubscriptionItem.ActiveFrom.Before(minTime) {
					minTime = item.SubscriptionItem.ActiveFrom.Truncate(streaming.MinimumWindowSizeDuration)
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

				period = period.Truncate(streaming.MinimumWindowSizeDuration)
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
	span := tracex.Start[[]subscriptionItemWithPeriods](ctx, it.tracer, "billing.worker.subscription.phaseiterator.Generate", trace.WithAttributes(
		attribute.String("phase_key", it.phase.Spec.PhaseKey),
	))

	// Given we are truncating to 1s resolution, we need to make sure that iterationEnd contains the last second as a whole.
	iterationEnd = iterationEnd.Truncate(streaming.MinimumWindowSizeDuration).Add(streaming.MinimumWindowSizeDuration - time.Nanosecond)

	return span.Wrap(func(ctx context.Context) ([]subscriptionItemWithPeriods, error) {
		return it.generateAligned(ctx, iterationEnd)
	})
}

func (it *PhaseIterator) generateAligned(ctx context.Context, iterationEnd time.Time) ([]subscriptionItemWithPeriods, error) {
	span := tracex.Start[[]subscriptionItemWithPeriods](ctx, it.tracer, "billing.worker.subscription.phaseiterator.generateAligned")

	return span.Wrap(func(ctx context.Context) ([]subscriptionItemWithPeriods, error) {
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
	span := tracex.Start[bool](ctx, it.tracer, "billing.worker.subscription.phaseiterator.generateForAlignedItemVersion", trace.WithAttributes(
		attribute.String("itemKey", item.Spec.ItemKey),
		attribute.Int("itemVersion", version),
		attribute.String("phaseKey", it.phase.Spec.PhaseKey),
		attribute.String("subscriptionId", it.sub.Subscription.ID),
		attribute.String("phaseId", it.phase.SubscriptionPhase.ID),
	))

	return span.Wrap(func(ctx context.Context) (bool, error) {
		logger := it.logger.With(
			"itemKey", item.Spec.ItemKey,
			"itemVersion", version,
			"phaseKey", it.phase.Spec.PhaseKey,
			"subscriptionId", it.sub.Subscription.ID,
			"phaseId", it.phase.SubscriptionPhase.ID,
		)

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
	span := tracex.Start[subscriptionItemWithPeriods](ctx, it.tracer, "billing.worker.subscription.phaseiterator.generateForAlignedItemVersionPeriod", trace.WithAttributes(
		attribute.Int("periodIdx", periodIdx),
		attribute.String("periodAt", at.Format(time.RFC3339)),
	))

	return span.Wrap(func(ctx context.Context) (subscriptionItemWithPeriods, error) {
		var empty subscriptionItemWithPeriods

		billingPeriod, err := it.sub.Spec.GetAlignedBillingPeriodAt(at)
		if err != nil {
			logger.ErrorContext(ctx, "failed to get aligned billing period", slog.Any("error", err))
			return empty, err
		}

		if it.sub.Spec.BillingAnchor.IsZero() {
			return empty, fmt.Errorf("billing anchor is zero for aligned generation, this should not happen")
		}

		fullServicePeriod, err := item.Spec.GetFullServicePeriodAt(
			subscription.GetFullServicePeriodAtInput{
				SubscriptionCadence:  it.sub.Subscription.CadencedModel,
				PhaseCadence:         it.phaseCadence,
				ItemCadence:          item.SubscriptionItem.CadencedModel,
				At:                   at,
				AlignedBillingAnchor: it.sub.Spec.BillingAnchor,
			},
		)
		if err != nil {
			logger.ErrorContext(ctx, "failed to get full service period", slog.Any("error", err))
			return empty, err
		}

		inter := fullServicePeriod.Open().Intersection(item.SubscriptionItem.CadencedModel.AsPeriod())

		// .Intersection() treats zero length periods as non-intersecting (to be consistent with .Contains() calls)
		// We need to handle this case separately
		if cl, err := item.SubscriptionItem.CadencedModel.AsPeriod().Closed(); err == nil && cl.From.Equal(cl.To) {
			inter = lo.ToPtr(cl.Open())
		}

		servicePeriod, err := inter.Closed()
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
			PhaseID:     it.phase.SubscriptionPhase.ID,
			PhaseKey:    it.phase.Spec.PhaseKey,
			PeriodIndex: periodIdx,
			ItemVersion: version,

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

func (it *PhaseIterator) truncateItemsIfNeeded(in []subscriptionItemWithPeriods) []subscriptionItemWithPeriods {
	out := make([]subscriptionItemWithPeriods, 0, len(in))
	// We need to sanitize the output to compensate for the 1second resolution of meters
	for _, item := range in {
		isFlatPrice := item.Spec.RateCard.AsMeta().Price != nil && item.Spec.RateCard.AsMeta().Price.Type() == productcatalog.FlatPriceType

		// We truncate the service period to the meter resolution
		item.ServicePeriod = item.ServicePeriod.Truncate(streaming.MinimumWindowSizeDuration)

		// We only allow empty service periods for flat prices.
		if item.ServicePeriod.IsEmpty() && !isFlatPrice {
			continue
		}

		// Let's truncate the billing period and full service period so that when
		// doing any calculations we don't have small rounding errors due to the iterator
		// returning ns precision.
		item.BillingPeriod = item.BillingPeriod.Truncate(streaming.MinimumWindowSizeDuration)
		item.FullServicePeriod = item.FullServicePeriod.Truncate(streaming.MinimumWindowSizeDuration)

		out = append(out, item)
	}

	return out
}

func (it *PhaseIterator) generateOneTimeAlignedItem(item subscription.SubscriptionItemView, versionID int) (*subscriptionItemWithPeriods, error) {
	if item.Spec.RateCard.AsMeta().Price == nil {
		return nil, nil
	}

	itemCadence := item.SubscriptionItem.CadencedModel

	billingPeriod, err := it.sub.Spec.GetAlignedBillingPeriodAt(itemCadence.ActiveFrom)
	if err != nil {
		return nil, fmt.Errorf("failed to get aligned billing period at %s: %w", itemCadence.ActiveFrom, err)
	}

	fullServicePeriod, err := item.Spec.GetFullServicePeriodAt(
		subscription.GetFullServicePeriodAtInput{
			SubscriptionCadence:  it.sub.Subscription.CadencedModel,
			PhaseCadence:         it.phaseCadence,
			ItemCadence:          itemCadence,
			At:                   itemCadence.ActiveFrom,
			AlignedBillingAnchor: billingPeriod.From,
		},
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
		PhaseID:     it.phase.SubscriptionPhase.ID,
		PhaseKey:    it.phase.Spec.PhaseKey,
		PeriodIndex: 0,
		ItemVersion: versionID,

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
