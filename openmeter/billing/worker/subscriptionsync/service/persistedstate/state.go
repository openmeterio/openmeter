package persistedstate

import "github.com/openmeterio/openmeter/openmeter/billing"

type State struct {
	Lines      []billing.LineOrHierarchy
	ByUniqueID map[string]billing.LineOrHierarchy
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
