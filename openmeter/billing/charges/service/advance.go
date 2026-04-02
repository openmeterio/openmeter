package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) AdvanceCharges(ctx context.Context, input charges.AdvanceChargesInput) (charges.Charges, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	if err := s.validateNamespaceLockdown(input.Customer.Namespace); err != nil {
		return nil, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (charges.Charges, error) {
		inScopeCharges, err := s.ListCharges(ctx, charges.ListChargesInput{
			Namespace:   input.Customer.Namespace,
			StatusNotIn: []meta.ChargeStatus{meta.ChargeStatusFinal},
			CustomerIDs: []string{input.Customer.ID},
			Expands:     meta.Expands{meta.ExpandRealizations},
		})
		if err != nil {
			return nil, fmt.Errorf("list charges: %w", err)
		}

		chargesByType, err := chargesByType(inScopeCharges.Items)
		if err != nil {
			return nil, fmt.Errorf("get charges by type: %w", err)
		}

		if len(chargesByType.usageBased) == 0 && len(chargesByType.flatFees) == 0 {
			return charges.Charges{}, nil
		}

		advancedCharges := make(charges.Charges, 0, len(chargesByType.usageBased)+len(chargesByType.flatFees))

		// Advance credit-only flat fee charges
		for _, charge := range chargesByType.flatFees {
			if charge.Intent.SettlementMode != productcatalog.CreditOnlySettlementMode {
				continue
			}

			advancedCharge, err := s.flatFeeService.AdvanceCharge(ctx, flatfee.AdvanceChargeInput{
				ChargeID: charge.GetChargeID(),
			})
			if err != nil {
				return nil, fmt.Errorf("advance flat fee charge %s: %w", charge.ID, err)
			}

			if advancedCharge == nil {
				continue
			}

			advancedCharges = append(advancedCharges, charges.NewCharge(*advancedCharge))
		}

		// Advance usage-based charges
		if len(chargesByType.usageBased) > 0 {
			customerOverride, err := s.billingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
				Customer: input.Customer,
				Expand: billing.CustomerOverrideExpand{
					Customer: true,
				},
			})
			if err != nil {
				return nil, fmt.Errorf("get customer override: %w", err)
			}

			featureMeters, err := s.resolveFeatureMeters(ctx, input.Customer.Namespace, chargesByType.usageBased)
			if err != nil {
				return nil, err
			}

			for _, charge := range chargesByType.usageBased {
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
