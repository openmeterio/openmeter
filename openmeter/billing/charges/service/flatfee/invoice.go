package flatfee

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
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
