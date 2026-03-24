package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) AdvanceCharges(ctx context.Context, input charges.AdvanceChargesInput) (charges.Charges, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (charges.Charges, error) {
		chargeMetas, err := s.metaAdapter.ListByCustomer(ctx, meta.ListByCustomerInput{
			Customer: input.Customer,
		})
		if err != nil {
			return nil, fmt.Errorf("list charges by customer: %w", err)
		}

		usageBasedChargeMetas := lo.Filter(chargeMetas, func(chargeMeta meta.Charge, _ int) bool {
			return chargeMeta.Type == meta.ChargeTypeUsageBased
		})
		if len(usageBasedChargeMetas) == 0 {
			return charges.Charges{}, nil
		}

		customerOverride, err := s.billingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
			Customer: input.Customer,
			Expand: billing.CustomerOverrideExpand{
				Customer: true,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("get customer override: %w", err)
		}

		usageBasedCharges, err := s.usageBasedService.GetByMetas(ctx, usagebased.GetByMetasInput{
			Namespace: input.Customer.Namespace,
			Charges:   usageBasedChargeMetas,
			Expands:   meta.Expands{meta.ExpandRealizations},
		})
		if err != nil {
			return nil, fmt.Errorf("get usage based charges: %w", err)
		}

		featureMeters, err := s.resolveFeatureMeters(ctx, input.Customer.Namespace, usageBasedCharges)
		if err != nil {
			return nil, err
		}

		advancedCharges := make(charges.Charges, 0, len(usageBasedCharges))
		for _, charge := range usageBasedCharges {
			featureMeter, err := featureMeters.Get(charge.Intent.FeatureKey, true)
			if err != nil {
				return nil, fmt.Errorf("get feature meter for charge %s: %w", charge.ID, err)
			}

			advancedCharge, err := s.usageBasedService.AdvanceCharge(ctx, usagebased.AdvanceChargeInput{
				ChargeID:         charge.GetChargeID(),
				CustomerOverride: customerOverride,
				FeatureMeter:     featureMeter,
			})
			if err != nil {
				return nil, fmt.Errorf("advance usage based charge %s: %w", charge.ID, err)
			}

			if advancedCharge == nil {
				continue
			}

			advancedCharges = append(advancedCharges, charges.NewCharge(*advancedCharge))
		}

		return advancedCharges, nil
	})
}

func (s *service) resolveFeatureMeters(ctx context.Context, namespace string, charges []usagebased.Charge) (feature.FeatureMeters, error) {
	featureKeys := lo.Uniq(lo.Map(charges, func(charge usagebased.Charge, _ int) string {
		return charge.Intent.FeatureKey
	}))

	featureMeters, err := s.featureService.ResolveFeatureMeters(ctx, namespace, featureKeys)
	if err != nil {
		return nil, fmt.Errorf("resolve feature meters: %w", err)
	}

	return featureMeters, nil
}
