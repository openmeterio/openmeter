package client

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"
	"github.com/stripe/stripe-go/v80"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

// CreateInvoice creates a new invoice for a customer in Stripe.
func (c *stripeAppClient) CreateInvoice(ctx context.Context, input CreateInvoiceInput) (*stripe.Invoice, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("stripe create invoice: invalid input: %w", err)
	}

	params := &stripe.InvoiceParams{
		Currency: lo.ToPtr(string(input.Currency)),
		Customer: lo.ToPtr(input.StripeCustomerID),
		// FinalizeInvoice will advance the invoice
		AutoAdvance: lo.ToPtr(false),
		// If not set, defaults to the default payment method in the customer’s invoice settings.
		DefaultPaymentMethod: input.StripeDefaultPaymentMethodID,
		StatementDescriptor:  input.StatementDescriptor,
	}

	// When charging automatically, Stripe will attempt to pay this invoice using the default source attached to the customer.
	// When sending an invoice, Stripe will email this invoice to the customer with payment instructions.
	switch input.CollectionMethod {
	case billing.CollectionMethodChargeAutomatically:
		params.CollectionMethod = lo.ToPtr(string(stripe.InvoiceCollectionMethodChargeAutomatically))
	case billing.CollectionMethodSendInvoice:
		params.CollectionMethod = lo.ToPtr(string(stripe.InvoiceCollectionMethodSendInvoice))

		// The due date can only be set if we are not charging automatically.
		if input.DueDate != nil {
			params.DueDate = lo.ToPtr(input.DueDate.Unix())
		}

	default:
		return nil, fmt.Errorf("stripe create invoice: invalid collection method: %s", input.CollectionMethod)
	}

	if input.AutomaticTaxEnabled {
		params.AutomaticTax = &stripe.InvoiceAutomaticTaxParams{
			Enabled: lo.ToPtr(true),
		}
	}

	return c.client.Invoices.New(params)
}

// UpdateInvoice updates a Stripe invoice Stripe.
func (c *stripeAppClient) UpdateInvoice(ctx context.Context, input UpdateInvoiceInput) (*stripe.Invoice, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("stripe update invoice: invalid input: %w", err)
	}

	params := &stripe.InvoiceParams{
		StatementDescriptor: input.StatementDescriptor,
	}

	if input.DueDate != nil {
		params.DueDate = lo.ToPtr(input.DueDate.Unix())
	}
	return c.client.Invoices.Update(input.StripeInvoiceID, params)
}

// DeleteInvoice deletes a Stripe invoice.
// Stripe only allows deleting invoices in draft state.
func (c *stripeAppClient) DeleteInvoice(ctx context.Context, input DeleteInvoiceInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("stripe delete invoice: invalid input: %w", err)
	}

	_, err := c.client.Invoices.Del(input.StripeInvoiceID, nil)
	return err
}

// FinalizeInvoice finalizes a Stripe invoice.
func (c *stripeAppClient) FinalizeInvoice(ctx context.Context, input FinalizeInvoiceInput) (*stripe.Invoice, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("stripe finalize invoice: invalid input: %w", err)
	}

	return c.client.Invoices.FinalizeInvoice(input.StripeInvoiceID, &stripe.InvoiceFinalizeInvoiceParams{
		AutoAdvance: lo.ToPtr(input.AutoAdvance),
	})
}

// GetInvoice gets an invoice from Stripe.
func (c *stripeAppClient) GetInvoice(ctx context.Context, input GetInvoiceInput) (*stripe.Invoice, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("stripe get invoice: invalid input: %w", err)
	}

	return c.client.Invoices.Get(input.StripeInvoiceID, nil)
}

// CreateInvoiceInput is the input for creating a new invoice in Stripe.
type CreateInvoiceInput struct {
	AutomaticTaxEnabled          bool
	CollectionMethod             billing.CollectionMethod
	Currency                     currencyx.Code
	DueDate                      *time.Time
	StatementDescriptor          *string
	StripeCustomerID             string
	StripeDefaultPaymentMethodID *string
}

func (i CreateInvoiceInput) Validate() error {
	if i.CollectionMethod == "" {
		return errors.New("collection method is required")
	}

	if i.Currency == "" {
		return errors.New("currency is required")
	}

	if i.DueDate != nil && i.DueDate.IsZero() {
		return errors.New("due date cannot be zero")
	}

	if i.StripeCustomerID == "" {
		return errors.New("stripe customer id is required")
	}

	if i.StatementDescriptor != nil && *i.StatementDescriptor == "" {
		return errors.New("statement descriptor cannot be empty")
	}

	return nil
}

// UpdateInvoiceInput is the input for updating an invoice in Stripe.
type UpdateInvoiceInput struct {
	StripeInvoiceID     string
	DueDate             *time.Time
	StatementDescriptor *string
}

func (i UpdateInvoiceInput) Validate() error {
	if i.StripeInvoiceID == "" {
		return errors.New("stripe invoice id is required")
	}

	if i.DueDate != nil && i.DueDate.IsZero() {
		return errors.New("due date cannot be zero")
	}

	if i.StatementDescriptor != nil && *i.StatementDescriptor == "" {
		return errors.New("statement descriptor cannot be empty")
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

// FinalizeInvoiceInput is the input for finalizing an invoice in Stripe.
type FinalizeInvoiceInput struct {
	StripeInvoiceID string
	AutoAdvance     bool
}

func (i FinalizeInvoiceInput) Validate() error {
	if i.StripeInvoiceID == "" {
		return errors.New("stripe invoice id is required")
	}

	return nil
}

// GetInvoice gets an invoice from Stripe.
type GetInvoiceInput struct {
	StripeInvoiceID string
}

func (i GetInvoiceInput) Validate() error {
	if i.StripeInvoiceID == "" {
		return errors.New("stripe invoice id is required")
	}

	return nil
}
