package invoice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"
)

var ErrInvoiceIsImmutable = errors.New("invoice is immutable")

type GetInvoiceParams struct {
	ExpandItems bool
}

type Service interface {
	// CreateInvoiceItems adds line items to the specified invoice. If invoice is null, the line items will be added to the pending line items.
	CreateInvoiceItems(ctx context.Context, invoice *InvoiceID, items []InvoiceItem) error

	// GetInvoice returns the invoice with the specified ID.
	GetInvoice(ctx context.Context, invoice *InvoiceID, params GetInvoiceParams) (*Invoice, error)

	// GetPendingInvoiceItems returns the pending line items for the specified customer.
	GetPendingInvoiceItems(ctx context.Context, customerID CustomerID) (*Invoice, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateInvoiceItems(ctx context.Context, invoiceID *InvoiceID, items []InvoiceItem) error {
	// TODO: let's add TX

	// Let's validate if we are allowed to modify the invoice
	if invoiceID != nil {
		invoice, err := s.repo.GetInvoice(ctx, *invoiceID, RepoGetInvoiceInput{})
		if err != nil {
			return err
		}

		// Make sure that even if an invoice is deleted, but we failed to set the status we are returning the good error
		if invoice.DeletedAt != nil {
			return fmt.Errorf("invoice %s is deleted: %w", invoice.ID, ErrInvoiceIsImmutable)
		}

		if lo.Contains(InvoiceImmutableStatuses, invoice.Status) {
			return fmt.Errorf("invoice %s is immutable (status=%s): %w", invoice.ID, invoice.Status, ErrInvoiceIsImmutable)
		}
	}

	return s.repo.CreateInvoiceItems(ctx, invoiceID, items)
}

func (s *service) GetInvoice(ctx context.Context, invoice *InvoiceID, params GetInvoiceParams) (*Invoice, error) {
	return s.repo.GetInvoice(ctx, *invoice, RepoGetInvoiceInput{
		ExpandItems: params.ExpandItems,
	})
}

func (s *service) GetPendingInvoiceItems(ctx context.Context, customerID CustomerID) (*Invoice, error) {
	items, err := s.repo.GetPendingInvoiceItems(ctx, customerID)
	if err != nil {
		return nil, err
	}

	return &Invoice{
		Key: "pending",
		// TODO:
		// BillingProfileID, WorkflowConfigID, ProviderConfig, ProviderReference -> from customer entity
		Status:    InvoiceStatusPendingCreation,
		Items:     items,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),

		// TODO: totalAmount
	}, nil
}
