package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/clock"
)

func (s *service) GetCurrentTotals(ctx context.Context, input usagebased.GetCurrentTotalsInput) (usagebased.GetCurrentTotalsResult, error) {
	if err := input.Validate(); err != nil {
		return usagebased.GetCurrentTotalsResult{}, err
	}

	charge, err := s.adapter.GetByID(ctx, usagebased.GetByIDInput{
		ChargeID: input.ChargeID,
		Expands:  meta.Expands{meta.ExpandRealizations},
	})
	if err != nil {
		return usagebased.GetCurrentTotalsResult{}, fmt.Errorf("get charge: %w", err)
	}

	customerOverride, err := s.customerOverrideService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
		Customer: customer.CustomerID{
			Namespace: charge.Namespace,
			ID:        charge.Intent.CustomerID,
		},
		Expand: billing.CustomerOverrideExpand{
			Customer: true,
		},
	})
	if err != nil {
		return usagebased.GetCurrentTotalsResult{}, fmt.Errorf("get customer override: %w", err)
	}

	featureMeters, err := s.featureService.ResolveFeatureMeters(ctx, charge.Namespace, []string{charge.Intent.FeatureKey})
	if err != nil {
		return usagebased.GetCurrentTotalsResult{}, fmt.Errorf("resolve feature meters: %w", err)
	}

	featureMeter, err := featureMeters.Get(charge.Intent.FeatureKey, true)
	if err != nil {
		return usagebased.GetCurrentTotalsResult{}, fmt.Errorf("get feature meter: %w", err)
	}

	ratingResult, err := s.getRatingForUsage(ctx, getRatingForUsageInput{
		Charge:         charge,
		Customer:       customerOverride,
		FeatureMeter:   featureMeter,
		StoredAtOffset: clock.Now().Add(-usagebased.InternalCollectionPeriod),
	})
	if err != nil {
		return usagebased.GetCurrentTotalsResult{}, fmt.Errorf("get rating for usage: %w", err)
	}

	return usagebased.GetCurrentTotalsResult{
		Charge:    charge,
		Quantity:  ratingResult.Quantity,
		DueTotals: ratingResult.Totals,
	}, nil
}
