package billing

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

var (
	ErrDefaultProfileAlreadyExists  = errors.New("default profile already exists")
	ErrDefaultProfileNotFound       = errors.New("default profile not found")
	ErrProfileNotFound              = errors.New("profile not found")
	ErrProfileAlreadyDeleted        = errors.New("profile already deleted")
	ErrProfileConflict              = errors.New("profile has been already updated")
	ErrProfileReferencedByOverrides = errors.New("profile is referenced by customer overrides")
	ErrProfileTaxTypeChange         = errors.New("profile tax type change is not allowed")
	ErrProfileInvoicingTypeChange   = errors.New("profile invoicing type change is not allowed")
	ErrProfilePaymentTypeChange     = errors.New("profile payment type change is not allowed")

	ErrCustomerOverrideNotFound       = errors.New("customer override not found")
	ErrCustomerOverrideConflict       = errors.New("customer override has been already updated conflict")
	ErrCustomerOverrideAlreadyDeleted = errors.New("customer override already deleted")
	ErrCustomerNotFound               = errors.New("customer not found")
)

var _ error = (*NotFoundError)(nil)

const (
	EntityCustomerOverride = "billingCustomerOverride"
	EntityCustomer         = "customer"
	EntityDefaultProfile   = "defaultBillingProfile"
)

type NotFoundError struct {
	models.NamespacedID
	Entity string
	Err    error
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf("%s with id %s not found: %s", e.Entity, e.ID, e.Err)
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
