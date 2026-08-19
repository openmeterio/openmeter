package rating

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
)

type GetTotalsForUsageInput struct {
	Charge                  usagebased.Charge
	Customer                billing.CustomerOverrideWithDetails
	FeatureMeter            feature.FeatureMeter
	StoredAtLT              time.Time
	IgnoreMinimumCommitment bool
}

func (i GetTotalsForUsageInput) Validate() error {
	if err := i.Charge.Validate(); err != nil {
		return fmt.Errorf("charge: %w", err)
	}

	if i.Customer.Customer == nil {
		return fmt.Errorf("customer is required")
	}

	if i.FeatureMeter.Meter == nil {
		return fmt.Errorf("feature meter is required")
	}

	if i.StoredAtLT.IsZero() {
		return fmt.Errorf("stored at lt is required")
	}

	return nil
}

type GetTotalsForUsageResult struct {
	Totals totals.Totals
	// MeteredQuantity is the cumulative snapshot quantity the totals were
	// rated from, so callers can report the usage behind the amounts.
	MeteredQuantity alpacadecimal.Decimal
}

// GetTotalsForUsage returns the rated totals for the charge at the requested stored-at offset.
// It avoids generating detailed lines, so prefer it over GetDetailedRatingForUsage when only totals are needed.
func (s *service) GetTotalsForUsage(ctx context.Context, in GetTotalsForUsageInput) (GetTotalsForUsageResult, error) {
	if err := in.Validate(); err != nil {
		return GetTotalsForUsageResult{}, err
	}

	snapshotQuantity, err := s.snapshotQuantity(ctx, snapshotQuantityInput{
		Customer:      in.Customer.Customer,
		FeatureMeter:  in.FeatureMeter,
		ServicePeriod: in.Charge.Intent.GetEffectiveServicePeriod(),
		StoredAtLT:    in.StoredAtLT,
	})
	if err != nil {
		return GetTotalsForUsageResult{}, fmt.Errorf("get snapshot quantity: %w", err)
	}

	// Totals must stay gross before charge credit allocation; run creation applies credits later and expects gross rating totals here.
	opts := []billingrating.GenerateDetailedLinesOption{
		billingrating.WithCreditsMutatorDisabled(),
	}
	if in.IgnoreMinimumCommitment {
		opts = append(opts, billingrating.WithMinimumCommitmentIgnored())
	}

	ratingResult, err := s.ratingService.GenerateDetailedLines(usagebased.RateableIntent{
		Intent:        in.Charge.Intent.GetEffectiveIntent(),
		MeterValue:    snapshotQuantity,
		ServicePeriod: in.Charge.Intent.GetEffectiveServicePeriod(),
	}, opts...)
	if err != nil {
		return GetTotalsForUsageResult{}, fmt.Errorf("rating totals: %w", err)
	}

	return GetTotalsForUsageResult{
		Totals:          ratingResult.Totals,
		MeteredQuantity: snapshotQuantity,
	}, nil
}
