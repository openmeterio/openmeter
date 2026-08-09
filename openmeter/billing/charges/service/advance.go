package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/invoiceupdater"
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

		advancedChargesByID := make(map[string]charges.Charge, len(chargesByType.usageBased)+len(chargesByType.flatFees))
		advancedChargeIDs := make([]string, 0, len(chargesByType.usageBased)+len(chargesByType.flatFees))
		var invoicePatches invoiceupdater.Patches
		pendingAdvancement := make(map[string]InvocableCharge, len(chargesByType.usageBased)+len(chargesByType.flatFees))

		for _, charge := range chargesByType.flatFees {
			result, err := s.flatFeeService.AdvanceCharge(ctx, flatfee.AdvanceChargeInput{
				ChargeID: charge.GetChargeID(),
			})
			if err != nil {
				return nil, fmt.Errorf("advance flat fee charge %s: %w", charge.ID, err)
			}

			mappedResult := mapTriggerPatchResult(result)
			if mappedResult.Charge != nil {
				advancedChargesByID[charge.ID] = *mappedResult.Charge
				advancedChargeIDs = append(advancedChargeIDs, charge.ID)
			}
			invoicePatches = append(invoicePatches, mappedResult.InvoicePatches...)

			if mappedResult.CanAdvance {
				if len(mappedResult.InvoicePatches) == 0 {
					return nil, fmt.Errorf("flat fee charge %s can advance without an invoice-effect boundary", charge.ID)
				}
				pendingAdvancement[charge.ID] = &flatFeeInvocableCharge{
					chargeID:       charge.GetChargeID(),
					flatFeeService: s.flatFeeService,
				}
			}
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
				result, err := s.usageBasedService.AdvanceCharge(ctx, usagebased.AdvanceChargeInput{
					ChargeID:         charge.GetChargeID(),
					CustomerOverride: mo.Some(customerOverride),
					FeatureMeters:    mo.Some(featureMeters),
				})
				if err != nil {
					return nil, fmt.Errorf("advance usage based charge %s: %w", charge.ID, err)
				}

				mappedResult := mapTriggerPatchResult(result)
				if mappedResult.Charge != nil {
					advancedChargesByID[charge.ID] = *mappedResult.Charge
					advancedChargeIDs = append(advancedChargeIDs, charge.ID)
				}
				invoicePatches = append(invoicePatches, mappedResult.InvoicePatches...)

				if mappedResult.CanAdvance {
					if len(mappedResult.InvoicePatches) == 0 {
						return nil, fmt.Errorf("usage based charge %s can advance without an invoice-effect boundary", charge.ID)
					}
					pendingAdvancement[charge.ID] = &usageBasedInvocableCharge{
						chargeID:          charge.GetChargeID(),
						usageBasedService: s.usageBasedService,
					}
				}
			}
		}

		continuedResults, err := s.advanceChargesAndApplyInvoicePatches(ctx, input.Customer, pendingAdvancement, invoicePatches)
		if err != nil {
			return nil, err
		}
		for chargeID, result := range continuedResults {
			if result.Charge != nil {
				advancedChargesByID[chargeID] = *result.Charge
			}
		}

		advancedCharges := make(charges.Charges, 0, len(advancedChargeIDs))
		for _, chargeID := range advancedChargeIDs {
			advancedCharges = append(advancedCharges, advancedChargesByID[chargeID])
		}

		currencies, err := collectCurrencies(advancedCharges)
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

func collectCurrencies(chargeList charges.Charges) ([]currencies.Currency, error) {
	out, err := lo.MapErr(chargeList, func(c charges.Charge, _ int) (currencies.Currency, error) {
		return c.GetCurrency()
	})
	if err != nil {
		return nil, fmt.Errorf("get currencies: %w", err)
	}

	return lo.UniqByErr(out, func(c currencies.Currency) (string, error) {
		return c.Identity()
	})
}
