package billingentity

import "context"

type InvoicingApp interface {
	// ValidateInvoice validates if the app can run for the given invoice
	ValidateInvoice(ctx context.Context, invoice Invoice) error
}
