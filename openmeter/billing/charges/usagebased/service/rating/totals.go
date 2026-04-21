package rating

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
)

// GetTotalsForUsage returns the rated totals for the charge at the requested stored-at offset.
// It avoids generating detailed lines, so prefer it over GetDetailedLinesForUsage when only totals are needed.
func (s *Service) GetTotalsForUsage(ctx context.Context, in GetRatingForUsageInput) (totals.Totals, error) {
	if err := in.Validate(); err != nil {
		return totals.Totals{}, err
	}

	snapshotQuantity, err := s.snapshotQuantity(ctx, snapshotQuantityInput{
		Customer:       in.Customer.Customer,
		FeatureMeter:   in.FeatureMeter,
		ServicePeriod:  in.Charge.Intent.ServicePeriod,
		StoredAtOffset: in.StoredAtOffset,
	})
	if err != nil {
		return totals.Totals{}, fmt.Errorf("get snapshot quantity: %w", err)
	}

	ratingResult, err := s.ratingService.GenerateDetailedLines(usagebased.RateableIntent{
		Intent:     in.Charge.Intent,
		MeterValue: snapshotQuantity,
	})
	if err != nil {
		return totals.Totals{}, fmt.Errorf("rating totals: %w", err)
	}

	return ratingResult.Totals, nil
}
