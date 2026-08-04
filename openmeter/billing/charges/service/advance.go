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
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) AdvanceCharges(ctx context.Context, input charges.AdvanceChargesInput) (charges.Charges, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	if err := s.validateNamespaceLockdown(input.Customer.Namespace); err != nil {
		return nil, err
	}

	advancedCharges, err := transaction.Run(ctx, s.adapter, func(ctx context.Context) (charges.Charges, error) {
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

		for _, charge := range chargesByType.flatFees {
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

			featureMeters, err := s.featureService.ResolveFeatureMeters(ctx, input.Customer.Namespace, chargesByType.usageBased.GetFeatureKeysOrIDs()...)
			if err != nil {
				return nil, fmt.Errorf("resolve feature meters: %w", err)
			}

			for _, charge := range chargesByType.usageBased {
				advancedCharge, err := s.usageBasedService.AdvanceCharge(ctx, usagebased.AdvanceChargeInput{
					ChargeID:         charge.GetChargeID(),
					CustomerOverride: customerOverride,
					FeatureMeters:    featureMeters,
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

		currencies, err := collectEarningsRecognitionCurrencies(advancedCharges)
		if err != nil {
			return nil, err
		}

		if err := s.recognizeCustomerEarnings(ctx, input.Customer, currencies...); err != nil {
			return nil, err
		}

		return advancedCharges, nil
	})
	if err != nil {
		return nil, err
	}

	return advancedCharges, nil
}

// collectEarningsRecognitionCurrencies resolves the currency each advanced
// charge should recognize earnings in. Fiat charges recognize in their own
// currency.
//
// Custom-currency charges are omitted regardless of settlement mode. Their
// only recognizable (credit-backed) lineage is created by the shared
// allocation path in the native custom currency - identically for
// credit_only and credit_then_invoice, since both allocate covered usage the
// same way. Recognizing in the invoice fiat currency instead would be a
// silent no-op: credit_then_invoice never creates fiat-denominated lineage
// for the overage (it books straight to receivable/accrued, bypassing
// lineage entirely), so searching in fiat would find nothing while the real,
// recognizable native-currency lineage goes unrecognized. Recognizing
// native custom-currency consumption as revenue directly has no accounting
// design yet, so it stays omitted rather than erroring or being substituted.
func collectEarningsRecognitionCurrencies(chargeList charges.Charges) ([]currencies.Currency, error) {
	out := make([]currencies.Currency, 0, len(chargeList))

	for _, c := range chargeList {
		chargeCurrency, err := c.GetCurrency()
		if err != nil {
			return nil, fmt.Errorf("get currency: %w", err)
		}

		if chargeCurrency.IsCustom() {
			continue
		}

		out = append(out, chargeCurrency)
	}

	return lo.UniqByErr(out, func(c currencies.Currency) (string, error) {
		return c.Identity()
	})
}
