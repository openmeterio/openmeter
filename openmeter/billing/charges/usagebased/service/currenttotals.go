package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
	billingfeaturemeter "github.com/openmeterio/openmeter/openmeter/billing/featuremeter"
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
			ID:        charge.Intent.GetCustomerID(),
		},
		Expand: billing.CustomerOverrideExpand{
			Customer: true,
		},
	})
	if err != nil {
		return usagebased.GetCurrentTotalsResult{}, fmt.Errorf("get customer override: %w", err)
	}

	featureMeters, err := s.featureMeterResolver.Resolve(ctx, charge.Namespace, []billingfeaturemeter.FeatureMeterRef{{
		IDOrKey:      charge.GetFeatureKeyOrID(),
		RequireMeter: true,
	}})
	if err != nil {
		return usagebased.GetCurrentTotalsResult{}, fmt.Errorf("resolve feature meters: %w", err)
	}

	featureMeter, err := charge.ResolveFeatureMeter(featureMeters)
	if err != nil {
		return usagebased.GetCurrentTotalsResult{}, err
	}

	now := clock.Now()

	dueTotals, err := s.rater.GetTotalsForUsage(ctx, usagebasedrating.GetTotalsForUsageInput{
		Charge:                  charge,
		Customer:                customerOverride,
		FeatureMeter:            featureMeter,
		StoredAtLT:              now,
		IgnoreMinimumCommitment: now.Before(charge.Intent.GetEffectiveServicePeriod().To),
	})
	if err != nil {
		return usagebased.GetCurrentTotalsResult{}, fmt.Errorf("get totals for usage: %w", err)
	}

	return usagebased.GetCurrentTotalsResult{
		Charge:          charge,
		DueTotals:       dueTotals.Totals,
		MeteredQuantity: dueTotals.MeteredQuantity,
	}, nil
}
