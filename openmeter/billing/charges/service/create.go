package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type chargesWithInvoiceNowActions struct {
	charges         charges.Charges
	invoiceNowLines []invoicePendingLinesInput
}

func (s *service) Create(ctx context.Context, input charges.CreateInput) (charges.Charges, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	if err := s.validateNamespaceLockdown(input.Namespace); err != nil {
		return nil, err
	}

	result, err := transaction.Run(ctx, s.adapter, func(ctx context.Context) (*chargesWithInvoiceNowActions, error) {
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

		// Let's create all the credit purchase charges
		for _, intent := range intentsByType.CreditPurchase {
			result, err := s.creditPurchaseService.Create(ctx, creditpurchase.CreateInput{
				Namespace: input.Namespace,
				Intent:    intent.Value,
			})
			if err != nil {
				return nil, err
			}

			// For invoice settlement, prepare the gathering line (actual invoicing happens after TX commits)
			if result.GatheringLineToCreate != nil {
				shouldInvoiceNow := false
				if !result.Charge.Intent.ServicePeriod.From.After(clock.Now()) {
					shouldInvoiceNow = true
				}

				gatheringLinesToCreate = append(gatheringLinesToCreate, gatheringLineWithCustomerID{
					gatheringLine: *result.GatheringLineToCreate,
					customerID: customer.CustomerID{
						Namespace: input.Namespace,
						ID:        result.Charge.Intent.CustomerID,
					},
					ShouldInvoiceNow: shouldInvoiceNow,
				})
			}

			createdCharges = append(createdCharges, charges.WithIndex[charges.Charge]{
				Index: intent.Index,
				Value: charges.NewCharge(result.Charge),
			})
		}

		// Let's generate the gathering lines for the flat fees
		invoiceNowLines, err := s.createGatheringLines(ctx, gatheringLinesToCreate)
		if err != nil {
			return nil, err
		}

		// Let's map the created charges to the original intents
		result := make([]charges.Charge, len(input.Intents))
		for _, createdCharge := range createdCharges {
			result[createdCharge.Index] = createdCharge.Value
		}

		return &chargesWithInvoiceNowActions{
			charges:         result,
			invoiceNowLines: invoiceNowLines,
		}, nil
	})
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, fmt.Errorf("result is nil")
	}

	// TODO: once we have proper state machine for credit purchases, we can remove this and mek the
	// autoAdvanceCreatedCharges handle the invoice now actions.
	if len(result.invoiceNowLines) > 0 {
		if err := s.invokeInvoiceNowOnCreate(ctx, result.invoiceNowLines); err != nil {
			return nil, fmt.Errorf("invoking invoice now on create: %w", err)
		}
	}

	return s.autoAdvanceCreatedCharges(ctx, result.charges)
}

// autoAdvanceCreatedCharges post-processes newly created charges
// it handles credit-only usage-based and flat fee charges
// a separate transaction is used to make sure that we persist the creation state even if the advancing fails (as
// a worker will try to advance the charges again).
func (s *service) autoAdvanceCreatedCharges(ctx context.Context, created charges.Charges) (charges.Charges, error) {
	// Collect unique customer IDs that have newly created credit-only charges.
	customerIDs := make(map[customer.CustomerID]struct{})
	for _, c := range created {
		switch c.Type() {
		case meta.ChargeTypeUsageBased:
			ub, err := c.AsUsageBasedCharge()
			if err != nil {
				return nil, err
			}

			if ub.Intent.SettlementMode != productcatalog.CreditOnlySettlementMode {
				continue
			}

			customerIDs[customer.CustomerID{Namespace: ub.Namespace, ID: ub.Intent.CustomerID}] = struct{}{}

		case meta.ChargeTypeFlatFee:
			ff, err := c.AsFlatFeeCharge()
			if err != nil {
				return nil, err
			}

			if ff.Intent.SettlementMode != productcatalog.CreditOnlySettlementMode {
				continue
			}

			customerIDs[customer.CustomerID{Namespace: ff.Namespace, ID: ff.Intent.CustomerID}] = struct{}{}
		}
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
	gatheringLine    billing.GatheringLine
	customerID       customer.CustomerID
	ShouldInvoiceNow bool
}

func (s *service) invokeInvoiceNowOnCreate(ctx context.Context, invoiceNowLines []invoicePendingLinesInput) error {
	if len(invoiceNowLines) == 0 {
		return nil
	}

	invoiceNowArgs := lo.GroupByMap(invoiceNowLines, func(item invoicePendingLinesInput) (customer.CustomerID, string) {
		return item.CustomerID, item.LineID
	})

	for customerID, lines := range invoiceNowArgs {
		if _, err := s.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer:            customerID,
			IncludePendingLines: mo.Some(lines),
			AsOf:                lo.ToPtr(clock.Now()),
		}); err != nil {
			return fmt.Errorf("invoking invoice now on create: %w", err)
		}
	}

	return nil
}

// creditPurchaseWithPendingGatheringLine holds a credit purchase charge and its associated gathering line
// for deferred invoicing after the Create transaction commits.
type creditPurchaseWithPendingGatheringLine struct {
	charge        creditpurchase.Charge
	gatheringLine billing.GatheringLine
	customerID    customer.CustomerID
	currency      currencyx.Code
}

type invoicePendingLinesInput struct {
	CustomerID customer.CustomerID
	LineID     string
}

func (s *service) createGatheringLines(ctx context.Context, gatheringLinesToCreate []gatheringLineWithCustomerID) ([]invoicePendingLinesInput, error) {
	if len(gatheringLinesToCreate) == 0 {
		return nil, nil
	}

	gatheringLinesByCurrencyAndCustomer := lo.GroupBy(gatheringLinesToCreate, func(item gatheringLineWithCustomerID) currencyAndCustomerID {
		return currencyAndCustomerID{
			currency:   item.gatheringLine.Currency,
			customerID: item.customerID,
		}
	})

	invoiceNowLines := make([]invoicePendingLinesInput, 0, len(gatheringLinesToCreate))

	for custAndCurrency, lines := range gatheringLinesByCurrencyAndCustomer {
		// Let's create the gathering invoice on invoicing side
		result, err := s.billingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
			Customer: custAndCurrency.customerID,
			Currency: custAndCurrency.currency,
			Lines: lo.Map(lines, func(item gatheringLineWithCustomerID, _ int) billing.GatheringLine {
				return item.gatheringLine
			}),
		})
		if err != nil {
			return nil, fmt.Errorf("creating pending invoice lines for charges: %w", err)
		}

		for idx, line := range result.Lines {
			if lines[idx].ShouldInvoiceNow {
				invoiceNowLines = append(invoiceNowLines, invoicePendingLinesInput{
					CustomerID: custAndCurrency.customerID,
					LineID:     line.ID,
				})
			}
		}
	}

	return invoiceNowLines, nil
}
