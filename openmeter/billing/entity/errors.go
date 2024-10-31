package billingentity

import "fmt"

var (
	// We want to return the invoice regardless of the validation issues, as invoices can have issues at numerous
	// levels (e.g. missing default profile, missing customer override, missing billing profile, provider issues, etc.).
	//
	// We also return the validation issues. By adding codes to the validation issues, it will be easier for the clients (e.g. frontend) to handle the specific errors
	// without having to resort to string matching.
	//
	// Given that invoicing depends on the billing and customer override service, we need to have these error types in place for
	// all.

	ErrDefaultProfileAlreadyExists  = NewValidationError("default_profile_exists", "default profile already exists")
	ErrDefaultProfileNotFound       = NewValidationError("default_profile_not_found", "default profile not found")
	ErrProfileNotFound              = NewValidationError("profile_not_found", "profile not found")
	ErrProfileAlreadyDeleted        = NewValidationError("profile_already_deleted", "profile already deleted")
	ErrProfileConflict              = NewValidationError("profile_update_conflict", "profile has been already updated")
	ErrProfileReferencedByOverrides = NewValidationError("profile_referenced", "profile is referenced by customer overrides")

	ErrCustomerOverrideNotFound       = NewValidationError("customer_override_not_found", "customer override not found")
	ErrCustomerOverrideConflict       = NewValidationError("customer_override_conflict", "customer override has been already updated conflict")
	ErrCustomerOverrideAlreadyDeleted = NewValidationError("customer_override_deleted", "customer override already deleted")
	ErrCustomerNotFound               = NewValidationError("customer_not_found", "customer not found")

	ErrInvoiceCannotAdvance      = NewValidationError("invoice_cannot_advance", "invoice cannot advance")
	ErrInvoiceActionNotAvailable = NewValidationError("invoice_action_not_available", "invoice action not available")
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

type ConflictError struct {
	ID     string
	Entity string
	Err    error
}

func (e ConflictError) Error() string {
	if e.ID == "" {
		return e.Err.Error()
	}

	return fmt.Sprintf("%s [%s/%s]", e.Err, e.Entity, e.ID)
}

func (e ConflictError) Unwrap() error {
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
