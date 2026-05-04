package rating

import (
	"context"
	"fmt"
	"time"

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

// GetTotalsForUsage returns the rated totals for the charge at the requested stored-at offset.
// It avoids generating detailed lines, so prefer it over GetDetailedRatingForUsage when only totals are needed.
func (s *service) GetTotalsForUsage(ctx context.Context, in GetTotalsForUsageInput) (totals.Totals, error) {
	if err := in.Validate(); err != nil {
		return totals.Totals{}, err
	}

	snapshotQuantity, err := s.snapshotQuantity(ctx, snapshotQuantityInput{
		Customer:      in.Customer.Customer,
		FeatureMeter:  in.FeatureMeter,
		ServicePeriod: in.Charge.Intent.ServicePeriod,
		StoredAtLT:    in.StoredAtLT,
	})
	if err != nil {
		return totals.Totals{}, fmt.Errorf("get snapshot quantity: %w", err)
	}

	opts := []billingrating.GenerateDetailedLinesOption{}
	if in.IgnoreMinimumCommitment {
		opts = append(opts, billingrating.WithMinimumCommitmentIgnored())
	}

	ratingResult, err := s.ratingService.GenerateDetailedLines(usagebased.RateableIntent{
		Intent:        in.Charge.Intent,
		MeterValue:    snapshotQuantity,
		ServicePeriod: in.Charge.Intent.ServicePeriod,
	}, opts...)
	if err != nil {
		return totals.Totals{}, fmt.Errorf("rating totals: %w", err)
	}

	return ratingResult.Totals, nil
}
