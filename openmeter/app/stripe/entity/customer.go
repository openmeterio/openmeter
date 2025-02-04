package appstripeentity

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/app"
)

var _ app.CustomerData = (*CustomerData)(nil)

type CustomerData struct {
	StripeCustomerID             string
	StripeDefaultPaymentMethodID *string
}

func (d CustomerData) Validate() error {
	if d.StripeCustomerID == "" {
		return errors.New("stripe customer id is required")
	}

	if d.StripeDefaultPaymentMethodID != nil && *d.StripeDefaultPaymentMethodID == "" {
		return errors.New("stripe default payment method id cannot be empty if provided")
	}

	return nil
}
