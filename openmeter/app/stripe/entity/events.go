package appstripeentity

import "errors"

type CustomerPaymentSetupSucceededEventAppData struct {
	StripeCustomerID string `json:"stripeCustomerId"`
	PaymentMethodID  string `json:"paymentMethodId"`
}

func (i CustomerPaymentSetupSucceededEventAppData) Validate() error {
	if i.StripeCustomerID == "" {
		return errors.New("stripe customer id is required")
	}

	if i.PaymentMethodID == "" {
		return errors.New("payment method id is required")
	}

	return nil
}
