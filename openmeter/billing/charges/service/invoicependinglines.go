package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/samber/lo"
)

func (s *service) InvoicePendingLines(ctx context.Context, input billing.InvoicePendingLinesInput) ([]billing.StandardInvoice, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// TODO: we need to handle the namespace lockdowns as an argument to this service too

	return withBillingTransactionForInvoiceManipulation(ctx, s, input.Customer, func(ctx context.Context) ([]billing.StandardInvoice, error) {
		// Step 1: Let's have all the lines that are billable prepared on the gathering invoice (including line splitting)
		billableLines, err := s.billingService.PrepareBillableLines(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("preparing billable lines: %w", err)
		}

		if billableLines == nil {
			// Should not happen, but we want to be defensive, but we are not surfacing this error to the caller.
			return nil, fmt.Errorf("billable lines are nil")
		}

		createdInvoices := make([]billing.StandardInvoice, 0, len(billableLines.LinesByCurrency))

		for currency, inScopeLines := range billableLines.LinesByCurrency {
			// Step 2: We need to allocate credit amounts to the billable lines
			linesWithCreditAllocations, err := s.allocateCreditAmountsToBillableLines(ctx, input.Customer.Namespace, inScopeLines)
			if err != nil {
				return nil, fmt.Errorf("allocating credit amounts: %w", err)
			}

			// Step 3: We need to create the standard invoices from the billable lines
			createdInvoice, err := s.billingService.CreateStandardInvoiceFromGatheringLines(ctx, billing.CreateStandardInvoiceFromGatheringLinesInput{
				Customer: input.Customer,
				Currency: currency,
				Lines:    inScopeLines,
				PostCreationCalculationHook: func(invoice billing.StandardInvoice, line billing.StandardLine) (billing.LineMutators, error) {
					if chargesManaged, ok := linesWithCreditAllocations.chargesManagedLines[line.GetLineID()]; ok {
						return []billing.LineMutator{
							billing.NewSetCreditsAppliedOperation(convertCreditRealizationToCreditsApplied(chargesManaged.Realizations)),
						}, nil
					}

					return nil, nil
				},
			})
			if err != nil {
				return nil, fmt.Errorf("creating standard invoice from gathering lines: %w", err)
			}

			createdInvoices = append(createdInvoices, *createdInvoice)
		}

		return createdInvoices, nil
	})
}

type gatheringLineWithCreditAllocations struct {
	GatheringLine billing.GatheringLine
	Realizations  charges.CreditRealizations
}

type gatheringLinesWithCreditAllocationsResult struct {
	chargesManagedLines          map[billing.LineID]gatheringLineWithCreditAllocations
	gatheringLinesWithoutCharges billing.GatheringLines
}

func (s *service) allocateCreditAmountsToBillableLines(ctx context.Context, namespace string, billableLines billing.GatheringLines) (*gatheringLinesWithCreditAllocationsResult, error) {
	if billableLines == nil {
		return nil, fmt.Errorf("billable lines are nil")
	}

	// Let's collect all the lines that are managed by charges so that we can start allocating credit amount to them
	chargesManagedGatheringLines := lo.FilterSliceToMap(billableLines, func(line billing.GatheringLine) (billing.LineID, gatheringLineWithCreditAllocations, bool) {
		if line.ChargeID == nil {
			return billing.LineID{}, gatheringLineWithCreditAllocations{}, false
		}

		return line.GetLineID(), gatheringLineWithCreditAllocations{
			GatheringLine: line,
			Realizations:  nil,
		}, true
	})

	unmanagedLines := lo.Filter(billableLines, func(line billing.GatheringLine, _ int) bool {
		return line.ChargeID == nil
	})

	// Let's get all the charges that are managed by the lines
	chargeIDs := lo.MapToSlice(chargesManagedGatheringLines, func(lineID billing.LineID, line gatheringLineWithCreditAllocations) string {
		return *line.GatheringLine.ChargeID
	})

	affectedCharges, err := s.adapter.GetChargesByIDs(ctx, namespace, chargeIDs)
	if err != nil {
		return nil, fmt.Errorf("getting charges by IDs: %w", err)
	}

	chargesById := make(map[charges.ChargeID]charges.Charge)
	for _, charge := range affectedCharges {
		id, err := charge.GetChargeID()
		if err != nil {
			return nil, fmt.Errorf("getting charge ID: %w", err)
		}

		chargesById[id] = charge
	}

	// Let's iterate over the charges and allocate credit amounts to the lines that are managed by them
	for lineID, gatheringLine := range chargesManagedGatheringLines {
		if gatheringLine.GatheringLine.ChargeID == nil {
			return nil, fmt.Errorf("charge ID is nil for line [lineID=%s]", lineID)
		}

		chargeID := charges.ChargeID{
			Namespace: namespace,
			ID:        *gatheringLine.GatheringLine.ChargeID,
		}
		charge, ok := chargesById[chargeID]
		if !ok {
			return nil, fmt.Errorf("charge not found for line [lineID=%s]: %w", lineID, charges.NewChargeNotFoundError(namespace, chargeID.ID))
		}

		switch charge.Type() {
		case charges.ChargeTypeFlatFee:
			flatFee, err := charge.AsFlatFeeCharge()
			if err != nil {
				return nil, err
			}

			creditAllocations, err := s.flatFeeService.PostLineAssignedToInvoice(ctx, flatFee, gatheringLine.GatheringLine)
			if err != nil {
				return nil, fmt.Errorf("post line assigned to invoice: %w", err)
			}

			gatheringLine.Realizations = creditAllocations
		case charges.ChargeTypeUsageBased:
			usageBased, err := charge.AsUsageBasedCharge()
			if err != nil {
				return nil, err
			}

			return nil, fmt.Errorf("usage based charges are not supported [chargeID=%s]: %w", usageBased.ID, charges.ErrUnsupported)
		default:
			return nil, fmt.Errorf("charge type is not supported: %s", charge.Type())
		}

		chargesManagedGatheringLines[lineID] = gatheringLine
	}

	return &gatheringLinesWithCreditAllocationsResult{
		chargesManagedLines:          chargesManagedGatheringLines,
		gatheringLinesWithoutCharges: unmanagedLines,
	}, nil
}

func convertCreditRealizationToCreditsApplied(creditRealizations charges.CreditRealizations) billing.CreditsApplied {
	return lo.Map(creditRealizations, func(creditRealization charges.CreditRealization, _ int) billing.CreditApplied {
		return billing.CreditApplied{
			Amount:              creditRealization.Amount,
			CreditRealizationID: creditRealization.ID,
		}
	})
}
