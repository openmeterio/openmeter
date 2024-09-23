package invoice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/samber/lo"
)

var ErrInvoiceIsImmutable = errors.New("invoice is immutable")

type GetInvoiceParams struct {
	ExpandItems bool
}

type Service interface {
	// CreateInvoiceItems adds line items to the specified invoice. If invoice is null, the line items will be added to the pending line items.
	CreateInvoiceItems(ctx context.Context, invoice *InvoiceID, items []InvoiceItem) ([]InvoiceItem, error)

	// GetInvoice returns the invoice with the specified ID.
	GetInvoice(ctx context.Context, invoice *InvoiceID, params GetInvoiceParams) (*Invoice, error)

	// GetPendingInvoiceItems returns the pending line items for the specified customer.
	GetPendingInvoiceItems(ctx context.Context, customerID customer.CustomerID) (*Invoice, error)
}

type Config struct {
	Repository      Repository
	CustomerService customer.Service
}

func (c Config) Validate() error {
	if c.Repository == nil {
		return errors.New("repository is required")
	}

	if c.CustomerService == nil {
		return errors.New("customer service is required")
	}
	return nil
}

type service struct {
	repo     Repository
	customer customer.Service
}

func NewService(config Config) (Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &service{
		repo:     config.Repository,
		customer: config.CustomerService,
	}, nil
}

func (s *service) CreateInvoiceItems(ctx context.Context, invoiceID *InvoiceID, items []InvoiceItem) ([]InvoiceItem, error) {
	// TODO: let's add TX

	// Let's validate if we are allowed to modify the invoice
	if invoiceID != nil {
		invoice, err := s.repo.GetInvoice(ctx, *invoiceID, RepoGetInvoiceInput{})
		if err != nil {
			return nil, err
		}

		// Make sure that even if an invoice is deleted, but we failed to set the status we are returning the good error
		if invoice.DeletedAt != nil {
			return nil, fmt.Errorf("invoice %s is deleted: %w", invoice.ID, ErrInvoiceIsImmutable)
		}

		if lo.Contains(InvoiceImmutableStatuses, invoice.Status) {
			return nil, fmt.Errorf("invoice %s is immutable (status=%s): %w", invoice.ID, invoice.Status, ErrInvoiceIsImmutable)
		}

		for _, item := range items {
			if item.Invoice == nil || *item.Invoice != invoice.ID {
				return nil, errors.New("invoice ID should be set to the invoice ID of the invoice")
			}
		}
	} else {
		for _, item := range items {
			if item.Invoice != nil {
				return nil, errors.New("invoice should be nil for pending invoice items")
			}
		}
	}

	return s.repo.CreateInvoiceItems(ctx, invoiceID, items)
}

func (s *service) GetInvoice(ctx context.Context, invoice *InvoiceID, params GetInvoiceParams) (*Invoice, error) {
	return s.repo.GetInvoice(ctx, *invoice, RepoGetInvoiceInput{
		ExpandItems: params.ExpandItems,
	})
}

func (s *service) GetPendingInvoiceItems(ctx context.Context, customerID customer.CustomerID) (*Invoice, error) {
	items, err := s.repo.GetPendingInvoiceItems(ctx, customerID)
	if err != nil {
		return nil, err
	}

	// TODO: if we want this in the same txn we either use multi stage commits or we need to move this to the repo
	customer, err := s.customer.GetCustomer(ctx, customer.CustomerID{
		Namespace: customerID.Namespace,
		ID:        customerID.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	return &Invoice{
		Key: "pending",
		// TODO:
		// BillingProfileID, WorkflowConfigID, ProviderConfig, ProviderReference -> from customer entity
		Status:    InvoiceStatusPendingCreation,
		Items:     items,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),

		Customer: InvoiceCustomer{
			CustomerID:     customerID.ID,
			Name:           customer.Name,
			BillingAddress: customer.BillingAddress,
		},

		// TODO: totalAmount
	}, nil
}
