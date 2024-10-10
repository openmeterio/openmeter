package client

import "fmt"

var _ error = (*StripeCustomerNotFoundError)(nil)

type StripeCustomerNotFoundError struct {
	StripeCustomerID string
}

func (e StripeCustomerNotFoundError) Error() string {
	return fmt.Sprintf("stripe customer %s not found", e.StripeCustomerID)
}

var _ error = (*StripePaymentMethodNotFoundError)(nil)

type StripePaymentMethodNotFoundError struct {
	StripePaymentMethodID string
}

func (e StripePaymentMethodNotFoundError) Error() string {
	return fmt.Sprintf("stripe customer %s not found", e.StripePaymentMethodID)
}
