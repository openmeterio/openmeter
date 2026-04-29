package rating

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/models"
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
	chargeWithDetailedLines, err := s.ensureDetailedLinesLoadedForRating(ctx, in.Charge, in.ServicePeriodTo)
	if err != nil {
		return GetDetailedRatingForUsageResult{}, err
	}
	in.Charge = chargeWithDetailedLines

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

	// Let's fetch invoice based realizations that are before the current run's service period to
	eligibleRealizations := lo.Filter(charge.Realizations, func(run usagebased.RealizationRun, _ int) bool {
		if run.Type != usagebased.RealizationRunTypeFinalRealization && run.Type != usagebased.RealizationRunTypePartialInvoice {
			return false
		}

		return run.ServicePeriodTo.Before(in.ServicePeriodTo)
	})

	// Let's sort the eligible realizations by service period to
	slices.SortStableFunc(eligibleRealizations, func(a, b usagebased.RealizationRun) int {
		return cmp.Compare(a.ServicePeriodTo.UnixNano(), b.ServicePeriodTo.UnixNano())
	})

	servicePeriodFrom := charge.Intent.ServicePeriod.From
	priorPeriods := make([]ratingPriorPeriod, 0, len(eligibleRealizations))

	for _, realization := range eligibleRealizations {
		servicePeriod := timeutil.ClosedPeriod{
			From: servicePeriodFrom,
			To:   realization.ServicePeriodTo,
		}

		// TODO: Later persist prior-period value snapshots for previous runs to avoid re-querying them. This only
		// helps if we have customers with a lot of interim invoices.
		//
		// The future optimization should snapshot prior runs as follows:
		//
		// - Determine the previous run as the latest non-current run with `ServicePeriodTo < current ServicePeriodTo`.
		// - For monotonic-compatible meters (monotonic + SUM), load that previous run's stored prior-period values.
		// - Re-query prior run quantities using the current run's `StoredAtLT`.
		// - Iterate prior runs from newest to oldest.
		// - Once a freshly queried prior quantity matches the value stored by the previous run, stop querying older periods.
		// - Copy the remaining older prior-period values from the previous run instead.
		// - This avoids expensive meter queries for older periods when the previous snapshot already proves they have not changed.
		//
		// An alternative approach is to do two queries and aggregate from there (for SUM, COUNT, MIN, MAX, etc.).
		//
		// - Query the meter between [intent.ServicePeriodFrom ... servicePeriod.To) capped by [previousStoredAtLT ... currentStoredAtLT)
		// - Query the meter between [servicePeriod.To ... currentServicePeriod.To) capped by currentStoredAtLT
		// - Aggregate the two results

		priorPeriodQty, err := s.snapshotQuantity(ctx, snapshotQuantityInput{
			Customer:      in.Customer.Customer,
			FeatureMeter:  in.FeatureMeter,
			ServicePeriod: servicePeriod,
			StoredAtLT:    in.StoredAtLT,
		})
		if err != nil {
			return GetDetailedRatingForUsageResult{}, fmt.Errorf("get prior period quantity: %w", err)
		}

		priorPeriods = append(priorPeriods, ratingPriorPeriod{
			MeteredQuantity: priorPeriodQty,
			ServicePeriod:   servicePeriod,
			DetailedLines:   realization.DetailedLines.OrEmpty(),
		})

		servicePeriodFrom = servicePeriod.To
	}

	return s.rateWithLateEvents(ctx, rateWithLateEventsInput{
		Intent: charge.Intent,
		CurrentPeriod: ratingCurrentPeriod{
			MeteredQuantity: currentQuantity,
			ServicePeriod:   currentRunServicePeriod,
		},
		PriorPeriod: priorPeriods,
	})
}

type rateWithLateEventsInput struct {
	Intent usagebased.Intent

	CurrentPeriod ratingCurrentPeriod
	PriorPeriod   []ratingPriorPeriod
}

func (i rateWithLateEventsInput) Validate() error {
	var errs []error
	if err := i.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent: %w", err))
	}

	intentServicePeriod := i.Intent.ServicePeriod
	if !intentServicePeriod.ContainsPeriodInclusive(i.CurrentPeriod.ServicePeriod) {
		errs = append(errs, fmt.Errorf("current period service period must be contained in intent service period: [%s..%s] vs [%s..%s]",
			intentServicePeriod.From.Format(time.RFC3339), intentServicePeriod.To.Format(time.RFC3339),
			i.CurrentPeriod.ServicePeriod.From.Format(time.RFC3339), i.CurrentPeriod.ServicePeriod.To.Format(time.RFC3339)))
	}

	for _, priorPeriod := range i.PriorPeriod {
		if !intentServicePeriod.ContainsPeriodInclusive(priorPeriod.ServicePeriod) {
			errs = append(errs, fmt.Errorf("prior period service period must be contained in intent service period: [%s..%s] vs [%s..%s]",
				intentServicePeriod.From.Format(time.RFC3339), intentServicePeriod.To.Format(time.RFC3339),
				priorPeriod.ServicePeriod.From.Format(time.RFC3339), priorPeriod.ServicePeriod.To.Format(time.RFC3339)))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type ratingCurrentPeriod struct {
	// MeteredQuantity is the metered quantity for [intent.ServicePeriodFrom ... servicePeriod.To) capped by StoredAtLT of the current run
	MeteredQuantity alpacadecimal.Decimal

	// ServicePeriod is the service period for the current period (from is intent.ServicePeriod.From if this is the first run or
	// the previous run's servicePeriod.To if this is not the first run)
	ServicePeriod timeutil.ClosedPeriod
}

type ratingPriorPeriod struct {
	// MeteredQuantity is the metered quantity for [intent.ServicePeriodFrom ... servicePeriod.To) capped by StoredAtLT of the current run
	MeteredQuantity alpacadecimal.Decimal

	// ServicePeriod is the service period for the prior period (from is intent.ServicePeriod.From, for the first item or
	// servicePeriod.From of the previous item)
	ServicePeriod timeutil.ClosedPeriod

	// DetailedLines are the detailed lines billed for the prior period
	DetailedLines usagebased.DetailedLines
}

func (s *service) rateWithLateEvents(ctx context.Context, in rateWithLateEventsInput) (GetDetailedRatingForUsageResult, error) {
	if err := in.Validate(); err != nil {
		return GetDetailedRatingForUsageResult{}, err
	}

	var opts []billingrating.GenerateDetailedLinesOption
	// Minimum commitment is charged only on the final run, not on interim snapshots.
	if in.CurrentPeriod.ServicePeriod.To.Before(in.Intent.ServicePeriod.To) {
		opts = append(opts, billingrating.WithMinimumCommitmentIgnored())
	}

	// TODO[later]: Implement the proper rating logic using prior period usage qtys for late event processing
	ratingResult, err := s.ratingService.GenerateDetailedLines(usagebased.RateableIntent{
		Intent:        in.Intent,
		ServicePeriod: in.CurrentPeriod.ServicePeriod,
		MeterValue:    in.CurrentPeriod.MeteredQuantity,
	}, opts...)
	if err != nil {
		return GetDetailedRatingForUsageResult{}, fmt.Errorf("rating: %w", err)
	}

	ratingResult.DetailedLines = withServicePeriodInDetailedLineChildUniqueReferenceIDs(
		ratingResult.DetailedLines,
		in.CurrentPeriod.ServicePeriod,
	)

	return GetDetailedRatingForUsageResult{
		Totals: ratingResult.Totals,
		DetailedLines: mapBillingRatingDetailedLinesToUsageBasedDetailedLines(
			in.Intent,
			in.CurrentPeriod.ServicePeriod,
			ratingResult.DetailedLines,
		),
		Quantity: in.CurrentPeriod.MeteredQuantity,
	}, nil
}
