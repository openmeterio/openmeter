package client

import (
	"errors"
	"fmt"

	"github.com/stripe/stripe-go/v80"
	"golang.org/x/net/context"
)

// AddInvoiceLines is the input for adding invoice lines to a Stripe invoice.
func (c *stripeClient) AddInvoiceLines(ctx context.Context, input AddInvoiceLinesInput) (*stripe.Invoice, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("stripe add invoice lines: invalid input: %w", err)
	}

	return c.client.Invoices.AddLines(input.StripeInvoiceID, &stripe.InvoiceAddLinesParams{
		Lines: input.Lines,
	})
}

// UpdateInvoiceLines is the input for updating invoice lines on a Stripe invoice.
func (c *stripeClient) UpdateInvoiceLines(ctx context.Context, input UpdateInvoiceLinesInput) (*stripe.Invoice, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("stripe update invoice lines: invalid input: %w", err)
	}

	return c.client.Invoices.UpdateLines(input.StripeInvoiceID, &stripe.InvoiceUpdateLinesParams{
		Lines: input.Lines,
	})
}

// RemoveInvoiceLines is the input for removing invoice lines from a Stripe invoice.
func (c *stripeClient) RemoveInvoiceLines(ctx context.Context, input RemoveInvoiceLinesInput) (*stripe.Invoice, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("stripe update invoice lines: invalid input: %w", err)
	}

	return c.client.Invoices.RemoveLines(input.StripeInvoiceID, &stripe.InvoiceRemoveLinesParams{
		Lines: input.Lines,
	})
}

// AddInvoiceLinesInput is the input for adding lines to an invoice in Stripe.
type AddInvoiceLinesInput struct {
	StripeInvoiceID string
	Lines           []*stripe.InvoiceAddLinesLineParams
}

func (i AddInvoiceLinesInput) Validate() error {
	if i.StripeInvoiceID == "" {
		return errors.New("stripe invoice id is required")
	}

	if len(i.Lines) == 0 {
		return errors.New("at least one line is required")
	}

	return nil
}

// UpdateInvoiceLinesInput is the input for updating lines on an invoice in Stripe.
type UpdateInvoiceLinesInput struct {
	StripeInvoiceID string
	Lines           []*stripe.InvoiceUpdateLinesLineParams
}

func (i UpdateInvoiceLinesInput) Validate() error {
	if i.StripeInvoiceID == "" {
		return errors.New("stripe invoice id is required")
	}

	if len(i.Lines) == 0 {
		return errors.New("at least one line is required")
	}

	return nil
}

// RemoveInvoiceLinesInput is the input for deleting lines on an invoice in Stripe.
type RemoveInvoiceLinesInput struct {
	StripeInvoiceID string
	Lines           []*stripe.InvoiceRemoveLinesLineParams
}

func (i RemoveInvoiceLinesInput) Validate() error {
	if i.StripeInvoiceID == "" {
		return errors.New("stripe invoice id is required")
	}

	if len(i.Lines) == 0 {
		return errors.New("at least one line is required")
	}

	return nil
}
