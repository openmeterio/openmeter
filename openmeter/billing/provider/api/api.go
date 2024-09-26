package providerapi

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

type InvoiceValidator interface {
	// ValidateInvoice validates the given invoice, called by the gobldriver and the returned
	// error(s) are added to the global error list. Notes:
	// - please use errors.join to combine multiple errors, they will be unwrapped by the gobldriver
	// - please use github.com/invopop/validation.NewError to create new errors, so that error codes are present
	//   in the validation results
	ValidateInvoice(context.Context, billing.Invoice) error
}

type TaxProvider interface {
	InvoiceValidator
}

type InvoiceProvider interface {
	InvoiceValidator
}

type PaymentProvider interface {
	InvoiceValidator
}
