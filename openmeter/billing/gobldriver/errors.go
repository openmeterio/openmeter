package gobldriver

import (
	"github.com/invopop/validation"

	goblvalidation "github.com/openmeterio/openmeter/pkg/gobl/validation"
)

var (
	ErrLoadingTimezoneLocation       = goblvalidation.NewError("invalid_tz", "error loading timezone location")
	ErrMissingPaymentMethod          = goblvalidation.NewError("missing_payment_method", "missing payment method")
	ErrMissingCustomerBillingAddress = goblvalidation.NewError("customer_billing_address_not_found", "missing customer billing address")

	ErrNumberConversion = goblvalidation.NewError("number_conversion", "error converting number")
)

func NewWithMessage(err validation.Error, msg string) validation.Error {
	// TODO: msg is a template, so we can use params if we really want to
	return validation.NewError(err.Code(), msg)
}

func upsertErrors(err validation.Errors) validation.Errors {
	if err == nil {
		return validation.Errors{}
	}

	return err
}
