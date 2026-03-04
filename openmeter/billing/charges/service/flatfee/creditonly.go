package flatfee

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
)

func (s *service) AdvanceCreditOnlyCharge(ctx context.Context, charge charges.FlatFeeCharge) (charges.FlatFeeCharge, error) {
	if charge.Intent.SettlementMode != productcatalog.CreditOnlySettlementMode {
		return charge, nil
	}

	if charge.Status == charges.ChargeStatusFinal {
		return charge, nil
	}

	if clock.Now().Before(charge.Intent.InvoiceAt) {
		return charge, nil
	}

	// Let's realize the charge from credits
	createCreditRealizations, err := s.flatFeeHandler.OnFlatFeeAssignedToInvoice(ctx, charges.OnFlatFeeAssignedToInvoiceInput{
		Charge:            charge,
		ServicePeriod:     charge.Intent.ServicePeriod,
		PreTaxTotalAmount: charge.Intent.AmountAfterProration,
	})
	if err != nil {
		return charge, err
	}

	// Sanity check: we are in credit only mode so let's make sure that the returned realizations are covering for the entire amount
	totalCreditRealizationAmount := alpacadecimal.Zero
	for _, realization := range createCreditRealizations {
		totalCreditRealizationAmount = totalCreditRealizationAmount.Add(realization.Amount)
	}

	if !totalCreditRealizationAmount.Equals(charge.Intent.AmountAfterProration) {
		return charge, fmt.Errorf("credit realizations do not cover the entire amount: %s != %s", totalCreditRealizationAmount, charge.Intent.AmountAfterProration)
	}

	createdCreditRealizations, err := s.adapter.CreateCreditRealizations(ctx, charge.GetChargeID(), createCreditRealizations)
	if err != nil {
		return charge, err
	}

	charge.State.CreditRealizations = createdCreditRealizations
	charge.Status = charges.ChargeStatusFinal

	_, err = s.adapter.UpdateFlatFeeCharge(ctx, charge)
	if err != nil {
		return charge, err
	}

	return charge, nil
}
