package ledger

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/charges"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func NewLedgerChargesHandler() (charges.Handler, error) {
	return charges.NewHandlerRouter(getHandlerForCharge)
}

func getHandlerForCharge(charge charges.Charge) (charges.Handler, error) {
	if charge.Intent.SettlementMode == productcatalog.InvoiceOnlySettlementMode {
		return &invoiceOnlyHandler{}, nil
	}

	return nil, fmt.Errorf("cannot handle charge %s with settlement mode %s", charge.ID, charge.Intent.SettlementMode)
}

type invoiceOnlyHandler struct {
	charges.NoOpHandler
}

func (h *invoiceOnlyHandler) OnStandardInvoiceRealizationAuthorized(ctx context.Context, charge charges.Charge, realization charges.StandardInvoiceRealizationWithLine) (charges.Charge, error) {
	// Note on manual edits:
	// - In invoice only mode we can book based on the line, and not the realization to allow for manual edits to the line if needed.
	// - In any credit based settlement mode, we don't allow for manual edits, so we can book based on the realization is we want.

	// Book the realization to the ledger to the customer's ledger account, and add an outstanding balance to the customer's outstanding account.

	return charge, nil
}

func (h *invoiceOnlyHandler) OnStandardInvoiceRealizationSettled(ctx context.Context, charge charges.Charge, realization charges.StandardInvoiceRealizationWithLine) (charges.Charge, error) {
	// Book the realization to the customer's outstanding account from the wash (bank) account.

	return charge, nil
}
