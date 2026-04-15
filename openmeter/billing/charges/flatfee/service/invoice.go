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

func (s *service) PostLineAssignedToInvoice(ctx context.Context, charge flatfee.Charge, line billing.StandardLine) (creditrealization.Realizations, error) {
	if charge.State.AmountAfterProration.IsZero() {
		return nil, nil
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (creditrealization.Realizations, error) {
		currencyCalculator, err := charge.Intent.Currency.Calculator()
		if err != nil {
			return nil, fmt.Errorf("get currency calculator: %w", err)
		}

		input := flatfee.OnAssignedToInvoiceInput{
			Charge:            charge,
			ServicePeriod:     line.Period.ToClosedPeriod(),
			PreTaxTotalAmount: currencyCalculator.RoundToPrecision(charge.State.AmountAfterProration),
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
		realizations, err := s.realizations.CreateCreditAllocations(ctx, charge, creditAllocations.AsCreateInputs())
		if err != nil {
			return nil, fmt.Errorf("creating credit realizations: %w", err)
		}

		return realizations, nil
	})
}

func (s *service) PostInvoiceIssued(ctx context.Context, charge flatfee.Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		if charge.Realizations.AccruedUsage != nil {
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
