package usagebasedrating

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
)

type GetRatingForUsageResult struct {
	rating.GenerateDetailedLinesResult
	Quantity alpacadecimal.Decimal
}

type GetRatingForUsageInput struct {
	Charge         usagebased.Charge
	Customer       billing.CustomerOverrideWithDetails
	FeatureMeter   feature.FeatureMeter
	StoredAtOffset time.Time
}

func (i GetRatingForUsageInput) Validate() error {
	if err := i.Charge.Validate(); err != nil {
		return fmt.Errorf("charge: %w", err)
	}

	if i.Customer.Customer == nil {
		return fmt.Errorf("customer is required")
	}

	if i.FeatureMeter.Meter == nil {
		return fmt.Errorf("feature meter is required")
	}

	if i.StoredAtOffset.IsZero() {
		return fmt.Errorf("stored at offset is required")
	}

	return nil
}

func (s *service) GetRatingForUsage(ctx context.Context, in GetRatingForUsageInput) (GetRatingForUsageResult, error) {
	if err := in.Validate(); err != nil {
		return GetRatingForUsageResult{}, err
	}

	snapshotQuantity, err := s.snapshotQuantity(ctx, SnapshotQuantityInput{
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
