package billing

import (
	"fmt"

	goblvalidation "github.com/openmeterio/openmeter/pkg/gobl/validation"
)

var (
	// Billing uses invopop's validation package to provide error codes and messages. This is required
	// as invoices can have:
	// - validation errors (coming from gobl)
	// - errors that are not validation errors (coming from the billing service, e.g. missing default profile)
	// - errors that are provider specific (coming from the provider)
	//
	// We want to return the invoice regardless of the validation errors, as invoices can have issues at numerous
	// levels (e.g. missing default profile, missing customer override, missing billing profile, provider issues, etc.).
	//
	// We also return the validation errors. By adding codes to the validation errors, it will be easier for the clients (e.g. frontend) to handle the specific errors
	// without having to resort to string matching.
	//
	// Given that invoicing depends on the billing and customer override service, we need to have these error types in place for
	// all.

	ErrDefaultProfileAlreadyExists  = goblvalidation.NewError("default_profile_exists", "default profile already exists")
	ErrDefaultProfileNotFound       = goblvalidation.NewError("default_profile_not_found", "default profile not found")
	ErrProfileNotFound              = goblvalidation.NewError("profile_not_found", "profile not found")
	ErrProfileAlreadyDeleted        = goblvalidation.NewError("profile_already_deleted", "profile already deleted")
	ErrProfileConflict              = goblvalidation.NewError("profile_update_conflict", "profile has been already updated")
	ErrProfileReferencedByOverrides = goblvalidation.NewError("profile_referenced", "profile is referenced by customer overrides")
	ErrProfileTaxTypeChange         = goblvalidation.NewError("profile_tax_provider_change_forbidden", "profile tax type change is not allowed")
	ErrProfileInvoicingTypeChange   = goblvalidation.NewError("profile_invoicing_provider_change_forbidden", "profile invoicing type change is not allowed")
	ErrProfilePaymentTypeChange     = goblvalidation.NewError("profile_payment_provider_change_forbidden", "profile payment type change is not allowed")

	ErrCustomerOverrideNotFound       = goblvalidation.NewError("customer_override_not_found", "customer override not found")
	ErrCustomerOverrideConflict       = goblvalidation.NewError("customer_override_conflict", "customer override has been already updated conflict")
	ErrCustomerOverrideAlreadyDeleted = goblvalidation.NewError("customer_override_deleted", "customer override already deleted")
	ErrCustomerNotFound               = goblvalidation.NewError("customer_not_found", "customer not found")
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
