package appstripeentity

import (
	"errors"

	"github.com/openmeterio/openmeter/api"
)

// CustomerAppData represents the Stripe associated data for an app used by a customer
type CustomerAppData struct {
	StripeCustomerID             string
	StripeDefaultPaymentMethodID *string
}

func (d CustomerAppData) Validate() error {
	if d.StripeCustomerID == "" {
		return errors.New("stripe customer id is required")
	}

	if d.StripeDefaultPaymentMethodID != nil && *d.StripeDefaultPaymentMethodID == "" {
		return errors.New("stripe default payment method id cannot be empty if provided")
	}

	return nil
}

func (d CustomerAppData) ToAPI() api.StripeCustomerAppData {
	return api.StripeCustomerAppData{
		StripeCustomerId:             d.StripeCustomerID,
		StripeDefaultPaymentMethodId: d.StripeDefaultPaymentMethodID,
	}
}
