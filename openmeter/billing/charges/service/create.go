package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) Create(ctx context.Context, input charges.CreateInput) (charges.Charges, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	out, err := transaction.Run(ctx, s.adapter, func(ctx context.Context) (charges.Charges, error) {
		intentsByType, err := input.Intents.ByType()
		if err != nil {
			return nil, err
		}

		createdCharges := make([]charges.WithIndex[charges.Charge], 0, len(input.Intents))
		gatheringLinesToCreate := make([]gatheringLineWithCustomerID, 0, len(input.Intents))

		// Let's create all the flat fee charges in bulk and record any gathering lines to create
		flatFees, err := s.flatFeeService.Create(ctx, flatfee.CreateInput{
			Namespace: input.Namespace,
			Intents: lo.Map(intentsByType.FlatFee, func(intent charges.WithIndex[flatfee.Intent], _ int) flatfee.Intent {
				return intent.Value
			}),
		})
		if err != nil {
			return nil, err
		}

		createdCharges = append(
			createdCharges,
			lo.Map(flatFees, func(fee flatfee.ChargeWithGatheringLine, idx int) charges.WithIndex[charges.Charge] {
				return charges.WithIndex[charges.Charge]{
					Index: intentsByType.FlatFee[idx].Index,
					Value: charges.NewCharge(fee.Charge),
				}
			})...,
		)

		for _, fee := range flatFees {
			if fee.GatheringLineToCreate != nil {
				gatheringLinesToCreate = append(gatheringLinesToCreate, gatheringLineWithCustomerID{
					gatheringLine: *fee.GatheringLineToCreate,
					customerID: customer.CustomerID{
						Namespace: input.Namespace,
						ID:        fee.Charge.Intent.CustomerID,
					},
				})
			}
		}

		// Let's create all the usage based charges in bulk
		usageBasedCharges, err := s.usageBasedService.Create(ctx, usagebased.CreateInput{
			Namespace: input.Namespace,
			Intents: lo.Map(intentsByType.UsageBased, func(intent charges.WithIndex[usagebased.Intent], _ int) usagebased.Intent {
				return intent.Value
			}),
		})
		if err != nil {
			return nil, err
		}

		createdCharges = append(
			createdCharges,
			lo.Map(usageBasedCharges, func(charge usagebased.ChargeWithGatheringLine, idx int) charges.WithIndex[charges.Charge] {
				return charges.WithIndex[charges.Charge]{
					Index: intentsByType.UsageBased[idx].Index,
					Value: charges.NewCharge(charge.Charge),
				}
			})...,
		)

		for _, charge := range usageBasedCharges {
			if charge.GatheringLineToCreate != nil {
				gatheringLinesToCreate = append(gatheringLinesToCreate, gatheringLineWithCustomerID{
					gatheringLine: *charge.GatheringLineToCreate,
					customerID: customer.CustomerID{
						Namespace: input.Namespace,
						ID:        charge.Charge.Intent.CustomerID,
					},
				})
			}
		}

		// Let's generate the gathering lines for the flat fees
		if err := s.createGatheringLines(ctx, gatheringLinesToCreate); err != nil {
			return nil, err
		}

		// Let's create all the credit purchase charges in bulk
		for _, intent := range intentsByType.CreditPurchase {
			charge, err := s.creditPurchaseService.Create(ctx, creditpurchase.CreateInput{
				Namespace: input.Namespace,
				Intent:    intent.Value,
			})
			if err != nil {
				return nil, err
			}

			createdCharges = append(createdCharges, charges.WithIndex[charges.Charge]{
				Index: intent.Index,
				Value: charges.NewCharge(charge),
			})
		}

		// Let's map the created charges to the original intents
		result := make([]charges.Charge, len(input.Intents))
		for _, createdCharge := range createdCharges {
			result[createdCharge.Index] = createdCharge.Value
		}

		return result, nil
	})
	if err != nil {
		return nil, err
	}

	return s.autoAdvanceCreatedCharges(ctx, out)
}

// autoAdvanceCreatedCharges post-processes newly created charges
// right now it only handles credit-only usage-based charges
// a separate transaction is used to make sure that we persist the creation state even if the advancing fails (as
// a worker will try to advance the charges again).
func (s *service) autoAdvanceCreatedCharges(ctx context.Context, created charges.Charges) (charges.Charges, error) {
	// Collect unique customer IDs that have newly created credit-only usage-based charges.
	customerIDs := make(map[customer.CustomerID]struct{})
	for _, c := range created {
		if c.Type() != meta.ChargeTypeUsageBased {
			continue
		}

		ub, err := c.AsUsageBasedCharge()
		if err != nil {
			return nil, err
		}

		if ub.Intent.SettlementMode != productcatalog.CreditOnlySettlementMode {
			continue
		}

		customerIDs[customer.CustomerID{Namespace: ub.Namespace, ID: ub.Intent.CustomerID}] = struct{}{}
	}

	if len(customerIDs) == 0 {
		return created, nil
	}

	advancedByID := make(map[string]charges.Charge)
	for custID := range customerIDs {
		advancedCharges, err := s.AdvanceCharges(ctx, charges.AdvanceChargesInput{
			Customer: custID,
		})
		if err != nil {
			return nil, fmt.Errorf("auto-advance charges for customer %s: %w", custID.ID, err)
		}

		for _, advanced := range advancedCharges {
			chargeID, err := advanced.GetChargeID()
			if err != nil {
				return nil, err
			}
			advancedByID[chargeID.ID] = advanced
		}
	}

	if len(advancedByID) == 0 {
		return created, nil
	}

	out := make(charges.Charges, len(created))
	for i, c := range created {
		chargeID, err := c.GetChargeID()
		if err != nil {
			return nil, err
		}

		if advanced, ok := advancedByID[chargeID.ID]; ok {
			out[i] = advanced
		} else {
			out[i] = c
		}
	}

	return out, nil
}

type currencyAndCustomerID struct {
	currency   currencyx.Code
	customerID customer.CustomerID
}

type gatheringLineWithCustomerID struct {
	gatheringLine billing.GatheringLine
	customerID    customer.CustomerID
}

func (s *service) createGatheringLines(ctx context.Context, gatheringLinesToCreate []gatheringLineWithCustomerID) error {
	if len(gatheringLinesToCreate) == 0 {
		return nil
	}

	gatheringLinesByCurrencyAndCustomer := lo.GroupBy(gatheringLinesToCreate, func(item gatheringLineWithCustomerID) currencyAndCustomerID {
		return currencyAndCustomerID{
			currency:   item.gatheringLine.Currency,
			customerID: item.customerID,
		}
	})

	for custAndCurrency, lines := range gatheringLinesByCurrencyAndCustomer {
		// Let's create the gathering invoice on invoicing side
		_, err := s.billingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
			Customer: custAndCurrency.customerID,
			Currency: custAndCurrency.currency,
			Lines: lo.Map(lines, func(item gatheringLineWithCustomerID, _ int) billing.GatheringLine {
				return item.gatheringLine
			}),
		})
		if err != nil {
			return fmt.Errorf("creating pending invoice lines for charges: %w", err)
		}
	}

	return nil
}
