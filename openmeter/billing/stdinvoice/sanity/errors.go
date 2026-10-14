package stdinvoicesanity

import "github.com/openmeterio/openmeter/openmeter/billing"

const ValidationComponent billing.ComponentName = "billing.invoice.sanity"

var (
	ErrLinesNotExpanded      = billing.NewValidationError("invoice_sanity_lines_not_expanded", "invoice lines must be expanded for sanity validation")
	ErrNilLine               = billing.NewValidationError("invoice_sanity_nil_line", "invoice line must not be nil")
	ErrInvalidCurrency       = billing.NewValidationError("invoice_sanity_invalid_currency", "invoice currency is invalid for discount reconciliation")
	ErrDiscountTotalMismatch = billing.NewValidationError("invoice_sanity_discount_total_mismatch", "discount entries total does not match discounts total")
)
