package rating

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
)

// GetDetailedLinesForUsage returns the rated detailed lines together with the metered quantity snapshot
// used to compute them. Prefer GetTotalsForUsage when only totals are needed because it is faster.
func (s *Service) GetDetailedLinesForUsage(ctx context.Context, in GetRatingForUsageInput) (GetRatingForUsageResult, error) {
	if err := in.Validate(); err != nil {
		return GetRatingForUsageResult{}, err
	}

	snapshotQuantity, err := s.snapshotQuantity(ctx, snapshotQuantityInput{
		Customer:       in.Customer.Customer,
		FeatureMeter:   in.FeatureMeter,
		ServicePeriod:  in.Charge.Intent.ServicePeriod,
		StoredAtOffset: in.StoredAtOffset,
	})
	if err != nil {
		return GetRatingForUsageResult{}, fmt.Errorf("get snapshot quantity: %w", err)
	}

	ratingResult, err := s.ratingService.GenerateDetailedLines(usagebased.RateableIntent{
		Intent:     in.Charge.Intent,
		MeterValue: snapshotQuantity,
	})
	if err != nil {
		return GetRatingForUsageResult{}, fmt.Errorf("rating: %w", err)
	}

	return GetRatingForUsageResult{
		GenerateDetailedLinesResult: ratingResult,
		Quantity:                    snapshotQuantity,
	}, nil
}
