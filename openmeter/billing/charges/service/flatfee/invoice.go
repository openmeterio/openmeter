package flatfee

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/samber/lo"
)

// TODO: Once we have proper UBP handling this should happen on the already converted StandardLine but for now we should be fine with this approach.
func (s *service) PostLineAssignedToInvoice(ctx context.Context, charge charges.FlatFeeCharge, line billing.GatheringLine) (charges.CreditRealizations, error) {
	creditAllocations, err := s.flatFeeHandler.OnFlatFeeAssignedToInvoice(ctx, charges.OnFlatFeeAssignedToInvoiceInput{
		Charge:            charge,
		ServicePeriod:     line.ServicePeriod,
		PreTaxTotalAmount: charge.Intent.AmountAfterProration,
	})
	if err != nil {
		return nil, fmt.Errorf("on flat fee assigned to invoice: %w", err)
	}

	// TODO: If we want we can bulk insert these into the database for better performance (for now it's fine)
	realizations, err := s.adapter.CreateCreditRealizations(ctx, charge.GetChargeID(), creditAllocations)
	if err != nil {
		return nil, fmt.Errorf("creating credit realizations: %w", err)
	}

	return realizations, nil
}

func (s *service) PostInvoiceIssued(ctx context.Context, charge charges.FlatFeeCharge, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	if charge.State.AccruedUsage != nil {
		// Lifecycle violation: this should not happen as we should not be able to issue an invoice if the charge already has an accrued usage.
		return fmt.Errorf("accrued invoice usage already exists for charge %s", charge.GetChargeID())
	}

	ledgerTransactionRef, err := s.flatFeeHandler.OnFlatFeeStandardInvoiceUsageAccrued(ctx, charges.OnFlatFeeStandardInvoiceUsageAccruedInput{
		Charge:        charge,
		ServicePeriod: lineWithHeader.Line.Period.ToClosedPeriod(),
		Totals:        lineWithHeader.Line.Totals,
	})
	if err != nil {
		return fmt.Errorf("on flat fee standard invoice usage accrued: %w", err)
	}

	accruedUsage := charges.StandardInvoiceAccruedUsage{
		LineID:            lo.ToPtr(lineWithHeader.Line.ID),
		ServicePeriod:     lineWithHeader.Line.Period.ToClosedPeriod(),
		Mutable:           false,
		Totals:            lineWithHeader.Line.Totals,
		LedgerTransaction: &ledgerTransactionRef,
	}

	_, err = s.adapter.CreateStandardInvoiceAccruedUsage(ctx, charge.GetChargeID(), accruedUsage)
	if err != nil {
		return fmt.Errorf("creating standard invoice accrued usage: %w", err)
	}

	return nil
}
