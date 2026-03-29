package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

// TODO: Once we have proper UBP handling this should happen on the already converted StandardLine but for now we should be fine with this approach.
func (s *service) PostLineAssignedToInvoice(ctx context.Context, charge flatfee.Charge, line billing.GatheringLine) (creditrealization.Realizations, error) {
	if charge.State.AmountAfterProration.IsZero() {
		return nil, nil
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (creditrealization.Realizations, error) {
		input := flatfee.OnAssignedToInvoiceInput{
			Charge:            charge,
			ServicePeriod:     line.ServicePeriod,
			PreTaxTotalAmount: charge.State.AmountAfterProration,
		}
		if err := input.Validate(); err != nil {
			return nil, fmt.Errorf("validating on assigned to invoice input: %w", err)
		}

		creditAllocations, err := s.handler.OnAssignedToInvoice(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("on flat fee assigned to invoice: %w", err)
		}

		if len(creditAllocations) == 0 {
			return nil, nil
		}

		// TODO: If we want we can bulk insert these into the database for better performance (for now it's fine)
		realizations, err := s.adapter.CreateCreditAllocations(ctx, charge.GetChargeID(), creditAllocations)
		if err != nil {
			return nil, fmt.Errorf("creating credit realizations: %w", err)
		}

		return realizations, nil
	})
}

func (s *service) PostInvoiceIssued(ctx context.Context, charge flatfee.Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		if charge.State.AccruedUsage != nil {
			// Lifecycle violation: this should not happen as we should not be able to issue an invoice if the charge already has an accrued usage.
			return fmt.Errorf("accrued invoice usage already exists for charge %s", charge.GetChargeID())
		}

		if lineWithHeader.Line == nil {
			return fmt.Errorf("postInvoiceIssued: line is nil")
		}

		ledgerTransactionRef, err := s.handler.OnInvoiceUsageAccrued(ctx, flatfee.OnInvoiceUsageAccruedInput{
			Charge:        charge,
			ServicePeriod: lineWithHeader.Line.Period.ToClosedPeriod(),
			Totals:        lineWithHeader.Line.Totals,
		})
		if err != nil {
			return fmt.Errorf("on flat fee standard invoice usage accrued: %w", err)
		}

		accruedUsage := invoicedusage.AccruedUsage{
			LineID:            lo.ToPtr(lineWithHeader.Line.ID),
			ServicePeriod:     lineWithHeader.Line.Period.ToClosedPeriod(),
			Mutable:           false,
			Totals:            lineWithHeader.Line.Totals,
			LedgerTransaction: &ledgerTransactionRef,
		}

		_, err = s.adapter.CreateInvoicedUsage(ctx, charge.GetChargeID(), accruedUsage)
		if err != nil {
			return fmt.Errorf("creating standard invoice accrued usage: %w", err)
		}

		return nil
	})
}
