package rating

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
)

type GetDetailedLinesForUsageInput struct {
	Charge          usagebased.Charge
	PriorRuns       usagebased.RealizationRuns
	Customer        billing.CustomerOverrideWithDetails
	FeatureMeter    feature.FeatureMeter
	ServicePeriodTo time.Time
	StoredAtLT      time.Time
	// IgnoreMinimumCommitment suppresses minimum commitment while still applying the rest of the billing mutators.
	IgnoreMinimumCommitment bool
}

func (i GetDetailedLinesForUsageInput) Validate() error {
	if err := i.Charge.Validate(); err != nil {
		return fmt.Errorf("charge: %w", err)
	}

	if i.Customer.Customer == nil {
		return fmt.Errorf("customer is required")
	}

	if i.FeatureMeter.Meter == nil {
		return fmt.Errorf("feature meter is required")
	}

	if i.ServicePeriodTo.IsZero() {
		return fmt.Errorf("service period to is required")
	}

	period := i.Charge.Intent.ServicePeriod
	if !i.ServicePeriodTo.After(period.From) {
		return fmt.Errorf("service period to must be after charge service period from")
	}

	if i.ServicePeriodTo.After(period.To) {
		return fmt.Errorf("service period to must not be after charge service period to")
	}

	if i.StoredAtLT.IsZero() {
		return fmt.Errorf("stored at lt is required")
	}

	if err := i.PriorRuns.Validate(); err != nil {
		return fmt.Errorf("prior runs: %w", err)
	}

	for idx, run := range i.PriorRuns {
		if !run.DetailedLines.IsPresent() {
			return fmt.Errorf("prior runs[%d]: detailed lines must be expanded", idx)
		}
	}

	return nil
}

type GetRatingForUsageResult struct {
	billingrating.GenerateDetailedLinesResult
	Quantity alpacadecimal.Decimal
}

// GetDetailedLinesForUsage returns the rated detailed lines together with the metered quantity snapshot
// used to compute them. Prefer GetTotalsForUsage when only totals are needed because it is faster.
func (s *service) GetDetailedLinesForUsage(ctx context.Context, in GetDetailedLinesForUsageInput) (GetRatingForUsageResult, error) {
	if err := in.Validate(); err != nil {
		return GetRatingForUsageResult{}, err
	}

	servicePeriod := in.Charge.Intent.ServicePeriod
	servicePeriod.To = in.ServicePeriodTo

	snapshotQuantity, err := s.snapshotQuantity(ctx, snapshotQuantityInput{
		Customer:      in.Customer.Customer,
		FeatureMeter:  in.FeatureMeter,
		ServicePeriod: servicePeriod,
		StoredAtLT:    in.StoredAtLT,
	})
	if err != nil {
		return GetRatingForUsageResult{}, fmt.Errorf("get snapshot quantity: %w", err)
	}

	var opts []billingrating.GenerateDetailedLinesOption
	if in.IgnoreMinimumCommitment {
		opts = append(opts, billingrating.WithMinimumCommitmentIgnored())
	}

	intent := in.Charge.Intent
	intent.ServicePeriod = servicePeriod

	ratingResult, err := s.ratingService.GenerateDetailedLines(usagebased.RateableIntent{
		Intent:     intent,
		MeterValue: snapshotQuantity,
	}, opts...)
	if err != nil {
		return GetRatingForUsageResult{}, fmt.Errorf("rating: %w", err)
	}

	ratingResult.DetailedLines = withServicePeriodInDetailedLineChildUniqueReferenceIDs(
		ratingResult.DetailedLines,
		servicePeriod,
	)

	return GetRatingForUsageResult{
		GenerateDetailedLinesResult: ratingResult,
		Quantity:                    snapshotQuantity,
	}, nil
}
