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

type createResult struct {
	charges                       charges.Charges
	pendingInvoiceCreditPurchases []charges.WithIndex[*creditPurchaseWithPendingGatheringLine]
}

func (s *service) Create(ctx context.Context, input charges.CreateInput) (charges.Charges, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	result, err := transaction.Run(ctx, s.adapter, func(ctx context.Context) (createResult, error) {
		intentsByType, err := input.Intents.ByType()
		if err != nil {
			return createResult{}, err
		}

		createdCharges := make([]charges.WithIndex[charges.Charge], 0, len(input.Intents))
		gatheringLinesToCreate := make([]gatheringLineWithCustomerID, 0, len(input.Intents))
		var pendingInvoiceCreditPurchases []charges.WithIndex[*creditPurchaseWithPendingGatheringLine]

		// Let's create all the flat fee charges in bulk and record any gathering lines to create
		flatFees, err := s.flatFeeService.Create(ctx, flatfee.CreateInput{
			Namespace: input.Namespace,
			Intents: lo.Map(intentsByType.FlatFee, func(intent charges.WithIndex[flatfee.Intent], _ int) flatfee.Intent {
				return intent.Value
			}),
		})
		if err != nil {
			return createResult{}, err
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
			return createResult{}, err
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
			return createResult{}, err
		}

		// Let's create all the credit purchase charges
		for _, intent := range intentsByType.CreditPurchase {
			result, err := s.creditPurchaseService.Create(ctx, creditpurchase.CreateInput{
				Namespace: input.Namespace,
				Intent:    intent.Value,
			})
			if err != nil {
				return createResult{}, err
			}

			// For invoice settlement, prepare the gathering line (actual invoicing happens after TX commits)
			if result.GatheringLineToCreate != nil {
				pendingInvoiceCreditPurchases = append(pendingInvoiceCreditPurchases, charges.WithIndex[*creditPurchaseWithPendingGatheringLine]{
					Index: intent.Index,
					Value: &creditPurchaseWithPendingGatheringLine{
						charge:        result.Charge,
						gatheringLine: *result.GatheringLineToCreate,
						customerID: customer.CustomerID{
							Namespace: input.Namespace,
							ID:        result.Charge.Intent.CustomerID,
						},
						currency: result.GatheringLineToCreate.Currency,
					},
				})
			}

			createdCharges = append(createdCharges, charges.WithIndex[charges.Charge]{
				Index: intent.Index,
				Value: charges.NewCharge(result.Charge),
			})
		}

		// Let's map the created charges to the original intents
		result := make([]charges.Charge, len(input.Intents))
		for _, createdCharge := range createdCharges {
			result[createdCharge.Index] = createdCharge.Value
		}

		return createResult{
			charges:                       result,
			pendingInvoiceCreditPurchases: pendingInvoiceCreditPurchases,
		}, nil
	})
	if err != nil {
		return nil, err
	}

	// TODO: mark
	out := result.charges

	// Handle invoice credit purchase post-creation steps outside the main transaction
	// to avoid lock contention with transactionForInvoiceManipulation.
	for _, pending := range result.pendingInvoiceCreditPurchases {
		updatedCharge, err := s.handleInvoiceCreditPurchasePostCreate(ctx, pending.Value)
		if err != nil {
			return nil, fmt.Errorf("post-create invoice credit purchase: %w", err)
		}

		out[pending.Index] = charges.NewCharge(updatedCharge)
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

// creditPurchaseWithPendingGatheringLine holds a credit purchase charge and its associated gathering line
// for deferred invoicing after the Create transaction commits.
type creditPurchaseWithPendingGatheringLine struct {
	charge        creditpurchase.Charge
	gatheringLine billing.GatheringLine
	customerID    customer.CustomerID
	currency      currencyx.Code
}

// handleInvoiceCreditPurchasePostCreate handles the post-creation steps for invoice credit purchases.
// This MUST be called outside the Create transaction to avoid lock contention.
// This function is idempotent - it can be safely called multiple times.
func (s *service) handleInvoiceCreditPurchasePostCreate(ctx context.Context, pending *creditPurchaseWithPendingGatheringLine) (creditpurchase.Charge, error) {
	charge := pending.charge

	// Idempotency check: if InvoiceSettlement is already set, processing was already done.
	if charge.State.InvoiceSettlement != nil {
		return charge, nil
	}

	// Create the gathering line on the billing side
	result, err := s.billingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: pending.customerID,
		Currency: pending.currency,
		Lines:    []billing.GatheringLine{pending.gatheringLine},
	})
	if err != nil {
		return creditpurchase.Charge{}, fmt.Errorf("creating pending invoice lines for credit purchase: %w", err)
	}

	if result == nil || len(result.Lines) == 0 {
		return creditpurchase.Charge{}, fmt.Errorf("no gathering lines created for credit purchase")
	}

	createdLine := result.Lines[0]

	// Invoke InvoicePendingLines to convert the gathering line to a standard invoice
	// if the invoice_at is in the past or now. InvoicePendingLines will also create
	// the InvoiceSettlement via PostLineAssignedToInvoice.
	if !createdLine.InvoiceAt.After(clock.Now()) {
		invoices, err := s.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer:            pending.customerID,
			IncludePendingLines: mo.Some([]string{createdLine.ID}),
			AsOf:                lo.ToPtr(clock.Now()),
		})
		if err != nil {
			return creditpurchase.Charge{}, fmt.Errorf("invoicing pending lines for credit purchase: %w", err)
		}

		if len(invoices) == 0 || len(invoices[0].Lines.OrEmpty()) == 0 {
			return creditpurchase.Charge{}, fmt.Errorf("no standard invoice created for credit purchase")
		}

		// Re-fetch the charge to get the updated state with InvoiceSettlement
		updatedCharges, err := s.GetByIDs(ctx, charges.GetByIDsInput{
			Namespace: charge.Namespace,
			ChargeIDs: []string{charge.ID},
			Expands: meta.Expands{
				meta.ExpandRealizations,
			},
		})
		if err != nil {
			return creditpurchase.Charge{}, fmt.Errorf("re-fetching credit purchase charge: %w", err)
		}

		if len(updatedCharges) == 0 {
			return creditpurchase.Charge{}, fmt.Errorf("credit purchase charge not found after invoicing [id=%s]", charge.ID)
		}

		cpCharge, err := updatedCharges[0].AsCreditPurchaseCharge()
		if err != nil {
			return creditpurchase.Charge{}, fmt.Errorf("converting to credit purchase charge: %w", err)
		}

		return cpCharge, nil
	}
	// For future-dated lines (InvoiceAt > now), the gathering line is created but
	// InvoiceSettlement is not set yet. It will be created later by InvoicePendingLines
	// via PostLineAssignedToInvoice when InvoiceAt arrives.

	return charge, nil
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
