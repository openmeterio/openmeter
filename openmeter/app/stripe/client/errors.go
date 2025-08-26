package client

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

// StripeCustomerNotFoundError
var _ models.GenericError = StripeCustomerNotFoundError{}

func NewStripeCustomerNotFoundError(stripeCustomerID string) *StripeCustomerNotFoundError {
	return &StripeCustomerNotFoundError{
		err: fmt.Errorf("stripe customer %s not found", stripeCustomerID),
	}
}

func IsStripeCustomerNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var e *StripeCustomerNotFoundError

	return errors.As(err, &e)
}

type StripeCustomerNotFoundError struct {
	err error
}

func (e StripeCustomerNotFoundError) Error() string {
	return e.err.Error()
}

func (e StripeCustomerNotFoundError) Unwrap() error {
	return e.err
}

// StripePaymentMethodNotFoundError
var _ models.GenericError = StripePaymentMethodNotFoundError{}

func NewStripePaymentMethodNotFoundError(stripePaymentMethodID string) *StripePaymentMethodNotFoundError {
	return &StripePaymentMethodNotFoundError{
		err: fmt.Errorf("stripe payment method %s not found", stripePaymentMethodID),
	}
}

func IsStripePaymentMethodNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var e *StripePaymentMethodNotFoundError

	return errors.As(err, &e)
}

type StripePaymentMethodNotFoundError struct {
	err error
}

func (e StripePaymentMethodNotFoundError) Error() string {
	return e.err.Error()
}

func (e StripePaymentMethodNotFoundError) Unwrap() error {
	return e.err
}

// StripeInvoiceCustomerTaxLocationInvalid
var _ models.GenericError = StripeInvoiceCustomerTaxLocationInvalidError{}

func NewStripeInvoiceCustomerTaxLocationInvalidError(stripeInvoiceID string, stripeMessage string) *StripeInvoiceCustomerTaxLocationInvalidError {
	return &StripeInvoiceCustomerTaxLocationInvalidError{
		err: fmt.Errorf("stripe invoice %s customer tax location invalid: %s", stripeInvoiceID, stripeMessage),
	}
}

func IsStripeInvoiceCustomerTaxLocationInvalidError(err error) bool {
	if err == nil {
		return false
	}

	var e *StripeInvoiceCustomerTaxLocationInvalidError

	return errors.As(err, &e)
}

type StripeInvoiceCustomerTaxLocationInvalidError struct {
	err error
}

func (e StripeInvoiceCustomerTaxLocationInvalidError) Error() string {
	return e.err.Error()
}

func (e StripeInvoiceCustomerTaxLocationInvalidError) Unwrap() error {
	return e.err
}
