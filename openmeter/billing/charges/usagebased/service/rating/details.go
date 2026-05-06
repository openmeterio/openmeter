package rating

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating/delta"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type GetDetailedRatingForUsageInput struct {
	// Charge general data

	// Charge contains the charge intent and prior runs.
	Charge usagebased.Charge

	// Current run's data

	// ServicePeriodTo defines the rated event-time upper bound for the current run.
	ServicePeriodTo time.Time
	// StoredAtLT defines the stored-at cutoff for current and prior snapshots.
	StoredAtLT time.Time

	// Metering values

	Customer     billing.CustomerOverrideWithDetails
	FeatureMeter feature.FeatureMeter
}

func (i GetDetailedRatingForUsageInput) Validate() error {
	if err := i.Charge.Validate(); err != nil {
		return fmt.Errorf("charge: %w", err)
	}

	if i.Customer.Customer == nil {
		return fmt.Errorf("customer is required")
	}

	if i.FeatureMeter.Meter == nil {
		return fmt.Errorf("feature meter is required")
	}

	period := i.Charge.Intent.ServicePeriod
	if i.ServicePeriodTo.IsZero() {
		return fmt.Errorf("service period to is required")
	}

	if !i.ServicePeriodTo.After(period.From) {
		return fmt.Errorf("service period to must be after charge service period from")
	}

	if i.ServicePeriodTo.After(period.To) {
		return fmt.Errorf("service period to must not be after charge service period to")
	}

	if i.StoredAtLT.IsZero() {
		return fmt.Errorf("stored at lt is required")
	}

	return nil
}

type GetDetailedRatingForUsageResult struct {
	Totals        totals.Totals
	DetailedLines usagebased.DetailedLines
	// Quantity is the current run's meter value between [Charge.Intent.ServicePeriod.From, ServicePeriodTo)
	// capped at StoredAtLT.
	Quantity alpacadecimal.Decimal
}

func (s *service) GetDetailedRatingForUsage(ctx context.Context, in GetDetailedRatingForUsageInput) (GetDetailedRatingForUsageResult, error) {
	if err := in.Validate(); err != nil {
		return GetDetailedRatingForUsageResult{}, err
	}

	charge, err := s.ensureDetailedLinesLoadedForRating(ctx, in.Charge, in.ServicePeriodTo)
	if err != nil {
		return GetDetailedRatingForUsageResult{}, err
	}

	currentRunServicePeriod := timeutil.ClosedPeriod{
		From: charge.Intent.ServicePeriod.From,
		To:   in.ServicePeriodTo,
	}

	currentQuantity, err := s.snapshotQuantity(ctx, snapshotQuantityInput{
		Customer:      in.Customer.Customer,
		FeatureMeter:  in.FeatureMeter,
		ServicePeriod: currentRunServicePeriod,
		StoredAtLT:    in.StoredAtLT,
	})
	if err != nil {
		return GetDetailedRatingForUsageResult{}, fmt.Errorf("get current quantity: %w", err)
	}

	// Let's fetch invoice based realizations that are before the current run's service period to.
	eligibleRealizations := lo.Filter(charge.Realizations, func(run usagebased.RealizationRun, _ int) bool {
		if run.Type != usagebased.RealizationRunTypeFinalRealization && run.Type != usagebased.RealizationRunTypePartialInvoice {
			return false
		}

		return run.ServicePeriodTo.Before(in.ServicePeriodTo)
	})

	switch charge.State.RatingEngine {
	case usagebased.RatingEngineDelta:
		alreadyBilledDetailedLines := make(usagebased.DetailedLines, 0, len(eligibleRealizations))
		for _, realization := range eligibleRealizations {
			alreadyBilledDetailedLines = append(alreadyBilledDetailedLines, realization.DetailedLines.OrEmpty()...)
		}

		out, err := s.deltaRater.Rate(ctx, delta.Input{
			Intent: charge.Intent,
			CurrentPeriod: delta.CurrentPeriod{
				MeteredQuantity: currentQuantity,
				ServicePeriod:   currentBillingPeriod(currentRunServicePeriod, eligibleRealizations),
			},
			AlreadyBilledDetailedLines: alreadyBilledDetailedLines,
		})
		if err != nil {
			return GetDetailedRatingForUsageResult{}, err
		}

		return GetDetailedRatingForUsageResult{
			Totals:        out.DetailedLines.SumTotals(),
			DetailedLines: out.DetailedLines,
			Quantity:      currentQuantity,
		}, nil
	default:
		return GetDetailedRatingForUsageResult{}, fmt.Errorf("unsupported rating engine: %s", charge.State.RatingEngine)
	}
}

func (s *service) ensureDetailedLinesLoadedForRating(ctx context.Context, charge usagebased.Charge, servicePeriodTo time.Time) (usagebased.Charge, error) {
	if len(charge.Realizations) == 0 {
		return charge, nil
	}

	if !lo.EveryBy(charge.Realizations, func(run usagebased.RealizationRun) bool {
		return !run.ServicePeriodTo.Before(servicePeriodTo) || run.DetailedLines.IsPresent()
	}) {
		expandedCharge, err := s.detailedLinesFetcher.FetchDetailedLines(ctx, charge)
		if err != nil {
			return usagebased.Charge{}, fmt.Errorf("fetch detailed lines: %w", err)
		}

		charge = expandedCharge
	}

	for idx, run := range charge.Realizations {
		// Extra safety: the fetcher contract should return all prior-run detailed
		// lines, but rating must not proceed with incomplete prior runs as we will overcharge
		// customers.
		if run.ServicePeriodTo.Before(servicePeriodTo) && !run.DetailedLines.IsPresent() {
			return usagebased.Charge{}, fmt.Errorf("prior runs[%d]: detailed lines must be expanded", idx)
		}
	}

	return charge, nil
}

func currentBillingPeriod(currentRunServicePeriod timeutil.ClosedPeriod, eligibleRealizations usagebased.RealizationRuns) timeutil.ClosedPeriod {
	currentBillingPeriod := currentRunServicePeriod
	for _, realization := range eligibleRealizations {
		if realization.ServicePeriodTo.After(currentBillingPeriod.From) {
			currentBillingPeriod.From = realization.ServicePeriodTo
		}
	}

	return currentBillingPeriod
}
