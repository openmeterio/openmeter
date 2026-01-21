package billing

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
)

var (
	// We want to return the invoice regardless of the validation issues, as invoices can have issues at numerous
	// levels (e.g. missing default profile, missing customer override, missing billing profile, provider issues, etc.).
	//
	// We also return the validation issues. By adding codes to the validation issues, it will be easier for the clients (e.g. frontend) to handle the specific errors
	// without having to resort to string matching.
	//
	// Given that invoicing depends on the billing and customer override service, we need to have these error types in place for
	// all.

	ErrDefaultProfileNotFound        = NewValidationError("default_profile_not_found", "default profile not found")
	ErrProfileNotFound               = NewValidationError("profile_not_found", "profile not found")
	ErrProfileAlreadyDeleted         = NewValidationError("profile_already_deleted", "profile already deleted")
	ErrProfileReferencedByOverrides  = NewValidationError("profile_referenced", "profile is referenced by customer overrides")
	ErrDefaultProfileCannotBeDeleted = NewValidationError("default_profile_cannot_be_deleted", "default profile cannot be deleted")
	ErrDefaultProfileCannotBeUnset   = NewValidationError("default_profile_cannot_be_unset", "default profile cannot be unset")

	ErrCustomerOverrideNotFound       = NewValidationError("customer_override_not_found", "customer override not found")
	ErrCustomerOverrideAlreadyDeleted = NewValidationError("customer_override_deleted", "customer override already deleted")
	ErrCustomerNotFound               = NewValidationError("customer_not_found", "customer not found")
	ErrCustomerDeleted                = NewValidationError("customer_deleted", "customer has been deleted")

	ErrFieldRequired             = NewValidationError("field_required", "field is required")
	ErrFieldMustBePositive       = NewValidationError("field_must_be_positive", "field must be positive")
	ErrFieldMustBePositiveOrZero = NewValidationError("field_must_be_positive_or_zero", "field must be positive or zero")

	ErrInvoiceCannotAdvance                  = NewValidationError("invoice_cannot_advance", "invoice cannot advance")
	ErrInvoiceCannotBeEdited                 = NewValidationError("invoice_cannot_be_edited", "invoice cannot be edited in the current state")
	ErrInvoiceActionNotAvailable             = NewValidationError("invoice_action_not_available", "invoice action not available")
	ErrInvoiceProgressiveBillingNotSupported = NewValidationError("invoice_progressive_billing_not_supported", "progressive billing is not supported")
	ErrInvoiceLinesNotBillable               = NewValidationError("invoice_lines_not_billable", "invoice lines are not billable")
	ErrInvoiceEmpty                          = NewValidationError("invoice_empty", "invoice is empty")
	ErrInvoiceNotFound                       = NewValidationError("invoice_not_found", "invoice not found")
	ErrInvoiceDeleteFailed                   = NewValidationError("invoice_delete_failed", "invoice delete failed")
	ErrInvoiceCannotDeleteGathering          = NewValidationError("invoice_cannot_delete_gathering", "gathering invoices cannot be deleted, please delete the upcoming lines instead")

	ErrInvoiceLineFeatureHasNoMeters                             = NewValidationError("invoice_line_feature_has_no_meters", "usage based invoice line: feature has no meters")
	ErrInvoiceLineVolumeSplitNotSupported                        = NewValidationError("invoice_line_graduated_split_not_supported", "graduated tiered pricing is not supported for split periods")
	ErrInvoiceLineNoTiers                                        = NewValidationError("invoice_line_no_tiers", "usage based invoice line: no tiers found")
	ErrInvoiceLineMissingOpenEndedTier                           = NewValidationError("invoice_line_missing_open_ended_tier", "usage based invoice line: missing open ended tier")
	ErrInvoiceLineDeleteInvalidStatus                            = NewValidationError("invoice_line_delete_invalid_status", "invoice line cannot be deleted in the current state (only valid lines can be deleted)")
	ErrInvoiceLineNoPeriodChangeForSplitLine                     = NewValidationError("invoice_line_no_period_change_for_split_line", "invoice line period cannot be changed for split lines")
	ErrInvoiceLineProgressiveBillingUsageDiscountUpdateForbidden = NewValidationError("invoice_line_progressive_billing_usage_discount_update_forbidden", "usage discount cannot be updated on a partially invoiced line")
	ErrInvoiceCreateNoLines                                      = NewValidationError("invoice_create_no_lines", "the new invoice would have no lines")
	ErrInvoiceCreateUBPLineCustomerUsageAttributionInvalid       = NewValidationError("invoice_create_ubp_line_customer_has_no_subjects", "creating an usage based line: customer usage attribution is invalid")
	ErrInvoiceCreateUBPLinePeriodIsEmpty                         = NewValidationError("invoice_create_ubp_line_period_is_empty", "creating an usage based line: truncated period is empty")
	ErrInvoiceLineCurrencyMismatch                               = NewValidationError("invoice_line_currency_mismatch", "invoice line currency mismatch")

	ErrInvoiceDiscountInvalidLineReference                  = NewValidationError("invoice_discount_invalid_line_reference", "invoice discount references non-existing line")
	ErrInvoiceDiscountNoWildcardDiscountOnGatheringInvoices = NewValidationError("invoice_discount_no_wildcard_discount_on_gathering_invoices", "wildcard discount on gathering invoices is not allowed")

	ErrNamespaceLocked = NewValidationError("namespace_locked", "namespace is locked")
)

const (
	ImmutableInvoiceHandlingNotSupportedErrorCode = "immutable_invoice_handling_not_supported"
)

var _ error = (*NotFoundError)(nil)

const (
	EntityCustomerOverride = "BillingCustomerOverride"
	EntityCustomer         = "Customer"
	EntityDefaultProfile   = "DefaultBillingProfile"
	EntityInvoice          = "Invoice"
	EntityInvoiceLine      = "InvoiceLine"
)

type NotFoundError struct {
	ID     string
	Entity string
	Err    error
}

func (e NotFoundError) Error() string {
	// ID can be empty for default profiles
	if e.ID == "" {
		return e.Err.Error()
	}

	return fmt.Sprintf("%s [%s/%s]", e.Err, e.Entity, e.ID)
}

func (e NotFoundError) Unwrap() error {
	return e.Err
}

type genericError struct {
	Err error
}

func EncodeValidationIssues[T error](err T) map[string]interface{} {
	validationIssues, _ := ToValidationIssues(err)

	if len(validationIssues) == 0 {
		return map[string]interface{}{}
	}

	// For HTTP calls we are usually interested in the first issue
	// so we return it as the main error
	out := validationIssues[0].EncodeAsErrorExtension()

	if len(validationIssues) > 1 {
		out["additionalIssues"] = validationIssues[1:]
	}

	return out
}

var _ error = (*ValidationError)(nil)

type ValidationError genericError

func (e ValidationError) Error() string {
	return e.Err.Error()
}

func (e ValidationError) Unwrap() error {
	return e.Err
}

var _ error = (*UpdateAfterDeleteError)(nil)

type UpdateAfterDeleteError genericError

func (e UpdateAfterDeleteError) Error() string {
	return e.Err.Error()
}

func (e UpdateAfterDeleteError) Unwrap() error {
	return e.Err
}

// AppError represents an error when interacting with an app.
var _ error = (*AppError)(nil)

type AppError struct {
	AppID   app.AppID
	AppType app.AppType
	Err     error
}

func (e AppError) Error() string {
	return fmt.Sprintf("app %s type with id %s in namespace %s: %s", e.AppType, e.AppID.ID, e.AppID.Namespace, e.Err.Error())
}
