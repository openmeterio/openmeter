package appstripe

import (
	"context"
	"slices"
	"strings"

	"github.com/stretchr/testify/mock"
	"github.com/stripe/stripe-go/v80"

	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
)

type StripeClientMock struct {
	mock.Mock
}

func (c *StripeClientMock) Restore() {
	c.ExpectedCalls = c.ExpectedCalls[:0]
}

func (c *StripeClientMock) GetAccount(ctx context.Context) (stripeclient.StripeAccount, error) {
	args := c.Called()
	return args.Get(0).(stripeclient.StripeAccount), args.Error(1)
}

func (c *StripeClientMock) SetupWebhook(ctx context.Context, input stripeclient.SetupWebhookInput) (stripeclient.StripeWebhookEndpoint, error) {
	if err := input.Validate(); err != nil {
		return stripeclient.StripeWebhookEndpoint{}, err
	}

	args := c.Called(input)
	return args.Get(0).(stripeclient.StripeWebhookEndpoint), args.Error(1)
}

type StripeAppClientMock struct {
	mock.Mock
}

func (c *StripeAppClientMock) Restore() {
	c.ExpectedCalls = c.ExpectedCalls[:0]
}

func (c *StripeAppClientMock) DeleteWebhook(ctx context.Context, input stripeclient.DeleteWebhookInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	args := c.Called(input)
	return args.Error(0)
}

func (c *StripeAppClientMock) GetAccount(ctx context.Context) (stripeclient.StripeAccount, error) {
	args := c.Called()
	return args.Get(0).(stripeclient.StripeAccount), args.Error(1)
}

func (c *StripeAppClientMock) GetCustomer(ctx context.Context, stripeCustomerID string) (stripeclient.StripeCustomer, error) {
	args := c.Called(stripeCustomerID)
	return args.Get(0).(stripeclient.StripeCustomer), args.Error(1)
}

func (c *StripeAppClientMock) CreateCustomer(ctx context.Context, input stripeclient.CreateStripeCustomerInput) (stripeclient.StripeCustomer, error) {
	if err := input.Validate(); err != nil {
		return stripeclient.StripeCustomer{}, err
	}

	args := c.Called(input)
	return args.Get(0).(stripeclient.StripeCustomer), args.Error(1)
}

func (c *StripeAppClientMock) CreateCheckoutSession(ctx context.Context, input stripeclient.CreateCheckoutSessionInput) (stripeclient.StripeCheckoutSession, error) {
	if err := input.Validate(); err != nil {
		return stripeclient.StripeCheckoutSession{}, err
	}

	args := c.Called(input)
	return args.Get(0).(stripeclient.StripeCheckoutSession), args.Error(1)
}

func (c *StripeAppClientMock) GetPaymentMethod(ctx context.Context, paymentMethodID string) (stripeclient.StripePaymentMethod, error) {
	args := c.Called(paymentMethodID)
	return args.Get(0).(stripeclient.StripePaymentMethod), args.Error(1)
}

// Invoice

func (c *StripeAppClientMock) CreateInvoice(ctx context.Context, input stripeclient.CreateInvoiceInput) (*stripe.Invoice, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	args := c.Called(input)
	return args.Get(0).(*stripe.Invoice), args.Error(1)
}

func (c *StripeAppClientMock) GetInvoice(ctx context.Context, input stripeclient.GetInvoiceInput) (*stripe.Invoice, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	args := c.Called(input)
	return args.Get(0).(*stripe.Invoice), args.Error(1)
}

func (c *StripeAppClientMock) UpdateInvoice(ctx context.Context, input stripeclient.UpdateInvoiceInput) (*stripe.Invoice, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	args := c.Called(input)
	return args.Get(0).(*stripe.Invoice), args.Error(1)
}

func (c *StripeAppClientMock) DeleteInvoice(ctx context.Context, input stripeclient.DeleteInvoiceInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	args := c.Called(input)
	return args.Error(1)
}

func (c *StripeAppClientMock) FinalizeInvoice(ctx context.Context, input stripeclient.FinalizeInvoiceInput) (*stripe.Invoice, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	args := c.Called(input)
	return args.Get(0).(*stripe.Invoice), args.Error(1)
}

// Invoice Lines
func (c *StripeAppClientMock) ListInvoiceLineItems(ctx context.Context, stripeInvoiceID string) ([]*stripe.InvoiceLineItem, error) {
	args := c.Called(stripeInvoiceID)
	return args.Get(0).([]*stripe.InvoiceLineItem), args.Error(1)
}

func (c *StripeAppClientMock) AddInvoiceLines(ctx context.Context, input stripeclient.AddInvoiceLinesInput) ([]stripeclient.StripeInvoiceItemWithLineID, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	c.StableSortInvoiceItemParams(input.Lines)

	args := c.Called(input)
	return args.Get(0).([]stripeclient.StripeInvoiceItemWithLineID), args.Error(1)
}

func (c *StripeAppClientMock) StableSortInvoiceItemParams(input []*stripe.InvoiceItemParams) {
	slices.SortFunc(input, func(a, b *stripe.InvoiceItemParams) int {
		return strings.Compare(*a.Description, *b.Description)
	})
}

func (c *StripeAppClientMock) UpdateInvoiceLines(ctx context.Context, input stripeclient.UpdateInvoiceLinesInput) ([]*stripe.InvoiceItem, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	c.StableSortStripeInvoiceItemWithID(input.Lines)

	args := c.Called(input)
	return args.Get(0).([]*stripe.InvoiceItem), args.Error(1)
}

func (c *StripeAppClientMock) StableSortStripeInvoiceItemWithID(lines []*stripeclient.StripeInvoiceItemWithID) {
	slices.SortFunc(lines, func(a, b *stripeclient.StripeInvoiceItemWithID) int {
		return strings.Compare(a.ID, b.ID)
	})
}

func (c *StripeAppClientMock) RemoveInvoiceLines(ctx context.Context, input stripeclient.RemoveInvoiceLinesInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	args := c.Called(input)
	return args.Error(0)
}

// Portal
func (c *StripeAppClientMock) CreatePortalSession(ctx context.Context, input stripeclient.CreatePortalSessionInput) (stripeclient.PortalSession, error) {
	if err := input.Validate(); err != nil {
		return stripeclient.PortalSession{}, err
	}

	args := c.Called(input)
	return args.Get(0).(stripeclient.PortalSession), args.Error(1)
}
