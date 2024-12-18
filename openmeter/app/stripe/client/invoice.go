package client

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"
	"github.com/stripe/stripe-go/v80"

	"github.com/openmeterio/openmeter/pkg/currencyx"
)

// GetInvoice retrieves a Stripe invoice.
func (c *stripeClient) GetInvoice(ctx context.Context, input GetInvoiceInput) (*stripe.Invoice, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("stripe get invoice: invalid input: %w", err)
	}

	return c.client.Invoices.Get(input.StripeInvoiceID, nil)
}

// CreateInvoice creates a new invoice for a customer in Stripe.
func (c *stripeClient) CreateInvoice(ctx context.Context, input CreateInvoiceInput) (*stripe.Invoice, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("stripe create invoice: invalid input: %w", err)
	}

	params := &stripe.InvoiceParams{
		Currency: lo.ToPtr(string(input.Currency)),
		Customer: lo.ToPtr(input.StripeCustomerID),
		// FinalizeInvoice will advance the invoice
		AutoAdvance: lo.ToPtr(false),
		// When charging automatically, Stripe will attempt to pay this invoice using the default source attached to the customer.
		// When sending an invoice, Stripe will email this invoice to the customer with payment instructions. Defaults to charge_automatically.
		CollectionMethod: lo.ToPtr(string(stripe.InvoiceCollectionMethodChargeAutomatically)),
		// If not set, defaults to the default payment method in the customer’s invoice settings.
		DefaultPaymentMethod: input.StripeDefaultPaymentMethodID,
	}

	if input.DueDate != nil {
		params.DueDate = lo.ToPtr(input.DueDate.Unix())
	}

	return c.client.Invoices.New(params)
}

// UpdateInvoice updates a Stripe invoice Stripe.
func (c *stripeClient) UpdateInvoice(ctx context.Context, input UpdateInvoiceInput) (*stripe.Invoice, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("stripe update invoice: invalid input: %w", err)
	}

	params := &stripe.InvoiceParams{}

	if input.DueDate != nil {
		params.DueDate = lo.ToPtr(input.DueDate.Unix())
	}

	return c.client.Invoices.Update(input.StripeInvoiceID, params)
}

// DeleteInvoice deletes a Stripe invoice.
// Stripe only allows deleting invoices in draft state.
func (c *stripeClient) DeleteInvoice(ctx context.Context, input DeleteInvoiceInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("stripe delete invoice: invalid input: %w", err)
	}

	_, err := c.client.Invoices.Del(input.StripeInvoiceID, nil)
	return err
}

// GetInvoiceInput is the input to get a Stripe invoice.
type GetInvoiceInput struct {
	StripeInvoiceID string
}

func (i GetInvoiceInput) Validate() error {
	if i.StripeInvoiceID == "" {
		return errors.New("stripe invoice id is required")
	}

	return nil
}

// CreateInvoiceInput is the input for creating a new invoice in Stripe.
type CreateInvoiceInput struct {
	StripeCustomerID             string
	StripeDefaultPaymentMethodID *string
	Currency                     currencyx.Code
	DueDate                      *time.Time
}

func (i CreateInvoiceInput) Validate() error {
	if i.StripeCustomerID == "" {
		return errors.New("stripe customer id is required")
	}

	if i.Currency == "" {
		return errors.New("currency is required")
	}

	if i.DueDate != nil && i.DueDate.IsZero() {
		return errors.New("due date cannot be zero")
	}

	return nil
}

// UpdateInvoiceInput is the input for updating an invoice in Stripe.
type UpdateInvoiceInput struct {
	StripeInvoiceID string
	DueDate         *time.Time
}

func (i UpdateInvoiceInput) Validate() error {
	if i.StripeInvoiceID == "" {
		return errors.New("stripe invoice id is required")
	}

	if i.DueDate != nil && i.DueDate.IsZero() {
		return errors.New("due date cannot be zero")
	}

	return nil
}

// DeleteInvoiceInput is the input for deleting an invoice in Stripe.
type DeleteInvoiceInput struct {
	StripeInvoiceID string
}

func (i DeleteInvoiceInput) Validate() error {
	if i.StripeInvoiceID == "" {
		return errors.New("stripe invoice id is required")
	}

	return nil
}
