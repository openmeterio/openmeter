package persistedstate

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

type State struct {
	Lines      []billing.LineOrHierarchy
	ByUniqueID map[string]Entity
}

func (s State) Validate() error {
	if s.ByUniqueID == nil {
		return errors.New("by unique id is required")
	}

	return nil
}

type Invoices struct {
	ByID map[string]billing.Invoice
}

func (i Invoices) IsGatheringInvoice(invoiceID string) bool {
	invoice, ok := i.ByID[invoiceID]
	if !ok {
		// If the invoice is not found, we assume that it is gathering, just to be safe.
		return true
	}

	return invoice.Type() == billing.InvoiceTypeGathering
}
