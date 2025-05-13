package client

import (
	"context"

	"github.com/samber/lo"
	"github.com/stripe/stripe-go/v80"
)

// GetCustomer returns the stripe customer by stripe customer ID
func (c *stripeAppClient) GetCustomer(ctx context.Context, stripeCustomerID string) (StripeCustomer, error) {
	stripeCustomer, err := c.client.Customers.Get(stripeCustomerID, &stripe.CustomerParams{
		Expand: []*string{
			lo.ToPtr("invoice_settings.default_payment_method"),
			lo.ToPtr("tax"),
		},
	})
	if err != nil {
		// Stripe customer not found error
		if stripeErr, ok := err.(*stripe.Error); ok && stripeErr.Code == stripe.ErrorCodeResourceMissing {
			return StripeCustomer{}, StripeCustomerNotFoundError{
				StripeCustomerID: stripeCustomerID,
			}
		}

		return StripeCustomer{}, c.providerError(err)
	}

	customer := StripeCustomer{
		StripeCustomerID: stripeCustomer.ID,
	}

	if stripeCustomer.Email != "" {
		customer.Email = &stripeCustomer.Email
	}

	if stripeCustomer.Currency != "" {
		customer.Currency = lo.ToPtr(string(stripeCustomer.Currency))
	}

	if stripeCustomer.InvoiceSettings != nil {
		invoiceSettings := *stripeCustomer.InvoiceSettings

		if stripeCustomer.InvoiceSettings.DefaultPaymentMethod != nil {
			customer.DefaultPaymentMethod = lo.ToPtr(toStripePaymentMethod(invoiceSettings.DefaultPaymentMethod))
		}
	}

	if stripeCustomer.Tax != nil {
		customer.Tax = &StripeCustomerTax{
			AutomaticTax: StripeCustomerAutomaticTax(stripeCustomer.Tax.AutomaticTax),
		}
	}

	return customer, nil
}

// CreateCustomer creates a stripe customer
func (c *stripeAppClient) CreateCustomer(ctx context.Context, input CreateStripeCustomerInput) (StripeCustomer, error) {
	if err := input.Validate(); err != nil {
		return StripeCustomer{}, err
	}

	// Create customer
	stripeCustomer, err := c.client.Customers.New(&stripe.CustomerParams{
		Name:  input.Name,
		Email: input.Email,
		Metadata: map[string]string{
			SetupIntentDataMetadataNamespace:  input.AppID.Namespace,
			SetupIntentDataMetadataCustomerID: input.CustomerID.ID,
		},
	})
	if err != nil {
		return StripeCustomer{}, c.providerError(err)
	}

	out := StripeCustomer{
		StripeCustomerID: stripeCustomer.ID,
	}

	return out, nil
}
