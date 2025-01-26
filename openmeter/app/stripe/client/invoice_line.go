package client

import (
	"errors"
	"fmt"

	"github.com/samber/lo"
	"github.com/stripe/stripe-go/v80"
	"golang.org/x/net/context"

	"github.com/openmeterio/openmeter/pkg/slicesx"
)

// AddInvoiceLines is the input for adding invoice lines to a Stripe invoice.
func (c *stripeAppClient) AddInvoiceLines(ctx context.Context, input AddInvoiceLinesInput) ([]*stripe.InvoiceItem, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("stripe add invoice lines: invalid input: %w", err)
	}

	return slicesx.MapWithErr(input.Lines, func(i *stripe.InvoiceItemParams) (*stripe.InvoiceItem, error) {
		i.Invoice = stripe.String(input.StripeInvoiceID)
		return c.client.InvoiceItems.New(i)
	})
}

// UpdateInvoiceLines is the input for updating invoice lines on a Stripe invoice.
func (c *stripeAppClient) UpdateInvoiceLines(ctx context.Context, input UpdateInvoiceLinesInput) ([]*stripe.InvoiceItem, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("stripe update invoice lines: invalid input: %w", err)
	}

	return slicesx.MapWithErr(input.Lines, func(i *StripeInvoiceItemWithID) (*stripe.InvoiceItem, error) {
		return c.client.InvoiceItems.Update(i.ID, i.InvoiceItemParams)
	})
}

// RemoveInvoiceLines is the input for removing invoice lines from a Stripe invoice.
func (c *stripeAppClient) RemoveInvoiceLines(ctx context.Context, input RemoveInvoiceLinesInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("stripe update invoice lines: invalid input: %w", err)
	}

	return errors.Join(lo.Map(input.Lines, func(id string, _ int) error {
		_, err := c.client.InvoiceItems.Del(id, nil)
		return err
	})...)
}

// AddInvoiceLinesInput is the input for adding lines to an invoice in Stripe.
type AddInvoiceLinesInput struct {
	StripeInvoiceID string
	Lines           []*stripe.InvoiceItemParams
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

type StripeInvoiceItemWithID struct {
	*stripe.InvoiceItemParams

	ID string
}

// UpdateInvoiceLinesInput is the input for updating lines on an invoice in Stripe.
type UpdateInvoiceLinesInput struct {
	StripeInvoiceID string
	Lines           []*StripeInvoiceItemWithID
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
	Lines           []string
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
