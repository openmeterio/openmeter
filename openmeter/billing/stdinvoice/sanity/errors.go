package stdinvoicesanity

import "github.com/openmeterio/openmeter/openmeter/billing"

const ValidationComponent billing.ComponentName = "billing.invoice.sanity"

var (
	ErrLinesNotExpanded            = billing.NewValidationError("invoice_sanity_lines_not_expanded", "invoice lines must be expanded for sanity validation")
	ErrNilLine                     = billing.NewValidationError("invoice_sanity_nil_line", "invoice line must not be nil")
	ErrInvalidCurrency             = billing.NewValidationError("invoice_sanity_invalid_currency", "invoice currency is invalid for sanity validation")
	ErrDiscountTotalMismatch       = billing.NewValidationError("invoice_sanity_discount_total_mismatch", "discount entries total does not match discounts total")
	ErrDiscountSignMismatch        = billing.NewValidationError("invoice_sanity_discount_sign_mismatch", "discount amount sign does not match the detailed line amount")
	ErrCreditTotalMismatch         = billing.NewValidationError("invoice_sanity_credit_total_mismatch", "credit entries total does not match credits total")
	ErrInvalidDetailCategory       = billing.NewValidationError("invoice_sanity_invalid_detail_category", "detailed line category is invalid")
	ErrCommitmentPlacementMismatch = billing.NewValidationError("invoice_sanity_commitment_placement_mismatch", "gross value is stored in the wrong category total")
	ErrTotalArithmeticMismatch     = billing.NewValidationError("invoice_sanity_total_arithmetic_mismatch", "total does not match its components")
	ErrLineTotalsMismatch          = billing.NewValidationError("invoice_sanity_line_totals_mismatch", "line totals do not match active detailed lines")
	ErrInvoiceTotalsMismatch       = billing.NewValidationError("invoice_sanity_invoice_totals_mismatch", "invoice totals do not match active lines")
	ErrCreditConservationMismatch  = billing.NewValidationError("invoice_sanity_credit_conservation_mismatch", "parent and detailed credit allocations do not reconcile")
	ErrInvalidCreditAmount         = billing.NewValidationError("invoice_sanity_invalid_credit_amount", "applied credit amount must be positive")
	ErrMissingCreditRealizationID  = billing.NewValidationError("invoice_sanity_missing_credit_realization_id", "applied credit must identify its realization")
	ErrCreditAllocationLimit       = billing.NewValidationError("invoice_sanity_credit_allocation_limit", "applied credits exceed the positive pretax balance")
	ErrCurrencyMismatch            = billing.NewValidationError("invoice_sanity_currency_mismatch", "line currency must match invoice currency")
	ErrTotalsPrecision             = billing.NewValidationError("invoice_sanity_totals_precision", "calculated totals must use invoice currency precision")
	ErrMissingIdentity             = billing.NewValidationError("invoice_sanity_missing_identity", "persisted invoice identities must not be empty")
	ErrDuplicateIdentity           = billing.NewValidationError("invoice_sanity_duplicate_identity", "persisted identities must be unique within their entity kind")
	ErrOwnershipMismatch           = billing.NewValidationError("invoice_sanity_ownership_mismatch", "line ownership must match the invoice")
	ErrInvalidServicePeriod        = billing.NewValidationError("invoice_sanity_invalid_service_period", "service period must not end before it starts")
	ErrInvoicePeriodMismatch       = billing.NewValidationError("invoice_sanity_invoice_period_mismatch", "invoice period must equal the active parent service period envelope")
	ErrNegativeParentTotal         = billing.NewValidationError("invoice_sanity_negative_parent_total", "parent payable total must not be negative")
	ErrMissingUsageBased           = billing.NewValidationError("invoice_sanity_missing_usage_based", "standard line must contain usage-based pricing data")
	ErrMissingPrice                = billing.NewValidationError("invoice_sanity_missing_price", "standard line must contain a price")
)
