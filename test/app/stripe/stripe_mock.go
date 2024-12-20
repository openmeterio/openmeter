package appstripe

import (
	"context"

	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stripe/stripe-go/v80"

	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	"github.com/openmeterio/openmeter/pkg/models"
)

type StripeClientMock struct {
	mock.Mock

	StripeAccountID string
}

func (c *StripeClientMock) SetupWebhook(ctx context.Context, input stripeclient.SetupWebhookInput) (stripeclient.StripeWebhookEndpoint, error) {
	return stripeclient.StripeWebhookEndpoint{
		EndpointID: "we_123",
		Secret:     "whsec_123",
	}, input.Validate()
}

func (c *StripeClientMock) DeleteWebhook(ctx context.Context, input stripeclient.DeleteWebhookInput) error {
	return input.Validate()
}

func (c *StripeClientMock) GetAccount(ctx context.Context) (stripeclient.StripeAccount, error) {
	return stripeclient.StripeAccount{
		StripeAccountID: c.StripeAccountID,
	}, nil
}

func (c *StripeClientMock) GetCustomer(ctx context.Context, stripeCustomerID string) (stripeclient.StripeCustomer, error) {
	return stripeclient.StripeCustomer{
		StripeCustomerID: stripeCustomerID,
		DefaultPaymentMethod: &stripeclient.StripePaymentMethod{
			ID:    "pm_123",
			Name:  "ACME Inc.",
			Email: "acme@test.com",
			BillingAddress: &models.Address{
				City:       lo.ToPtr("San Francisco"),
				PostalCode: lo.ToPtr("94103"),
				State:      lo.ToPtr("CA"),
				Country:    lo.ToPtr(models.CountryCode("US")),
				Line1:      lo.ToPtr("123 Market St"),
			},
		},
	}, nil
}

func (c *StripeClientMock) CreateCustomer(ctx context.Context, input stripeclient.CreateStripeCustomerInput) (stripeclient.StripeCustomer, error) {
	if err := input.Validate(); err != nil {
		return stripeclient.StripeCustomer{}, err
	}

	return stripeclient.StripeCustomer{
		StripeCustomerID: "cus_123",
	}, input.Validate()
}

func (c *StripeClientMock) CreateCheckoutSession(ctx context.Context, input stripeclient.CreateCheckoutSessionInput) (stripeclient.StripeCheckoutSession, error) {
	if err := input.Validate(); err != nil {
		return stripeclient.StripeCheckoutSession{}, err
	}

	return stripeclient.StripeCheckoutSession{
		SessionID:     "cs_123",
		SetupIntentID: "seti_123",
		Mode:          stripe.CheckoutSessionModeSetup,
		URL:           "https://checkout.stripe.com/cs_123/test",
	}, input.Validate()
}

func (c *StripeClientMock) GetPaymentMethod(ctx context.Context, paymentMethodID string) (stripeclient.StripePaymentMethod, error) {
	return stripeclient.StripePaymentMethod{
		ID:    "pm_123",
		Name:  "ACME Inc.",
		Email: "acme@test.com",
		BillingAddress: &models.Address{
			City:       lo.ToPtr("San Francisco"),
			PostalCode: lo.ToPtr("94103"),
			State:      lo.ToPtr("CA"),
			Country:    lo.ToPtr(models.CountryCode("US")),
			Line1:      lo.ToPtr("123 Market St"),
		},
	}, nil
}

// Invoice

func (c *StripeClientMock) CreateInvoice(ctx context.Context, input stripeclient.CreateInvoiceInput) (*stripe.Invoice, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	args := c.Called(input)
	return args.Get(0).(*stripe.Invoice), args.Error(1)
}

func (c *StripeClientMock) UpdateInvoice(ctx context.Context, input stripeclient.UpdateInvoiceInput) (*stripe.Invoice, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	args := c.Called(input)
	return args.Get(0).(*stripe.Invoice), args.Error(1)
}

func (c *StripeClientMock) DeleteInvoice(ctx context.Context, input stripeclient.DeleteInvoiceInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	args := c.Called(input)
	return args.Error(1)
}

func (c *StripeClientMock) FinalizeInvoice(ctx context.Context, input stripeclient.FinalizeInvoiceInput) (*stripe.Invoice, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	args := c.Called(input)
	return args.Get(0).(*stripe.Invoice), args.Error(1)
}

// Invoice Lines

func (c *StripeClientMock) AddInvoiceLines(ctx context.Context, input stripeclient.AddInvoiceLinesInput) (*stripe.Invoice, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	args := c.Called(input)
	return args.Get(0).(*stripe.Invoice), args.Error(1)
}

func (c *StripeClientMock) UpdateInvoiceLines(ctx context.Context, input stripeclient.UpdateInvoiceLinesInput) (*stripe.Invoice, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	args := c.Called(input)
	return args.Get(0).(*stripe.Invoice), args.Error(1)
}

func (c *StripeClientMock) RemoveInvoiceLines(ctx context.Context, input stripeclient.RemoveInvoiceLinesInput) (*stripe.Invoice, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	args := c.Called(input)
	return args.Get(0).(*stripe.Invoice), args.Error(1)
}
