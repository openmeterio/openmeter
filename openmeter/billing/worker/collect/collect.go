package billingworkercollect

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billingservice "github.com/openmeterio/openmeter/openmeter/billing/service"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
)

type InvoiceCollector struct {
	billing billing.Service

	logger *slog.Logger
}

type ListCollectableInvoicesInput struct {
	Namespaces   []string
	InvoiceIDs   []string
	Customers    []string
	CollectionAt time.Time
}

func (i ListCollectableInvoicesInput) Validate() error {
	var errs []error

	if i.CollectionAt.IsZero() {
		errs = append(errs, fmt.Errorf("collectionAt time must not be zero"))
	}

	return errors.Join(errs...)
}

func (a *InvoiceCollector) ListCollectableInvoices(ctx context.Context, params ListCollectableInvoicesInput) ([]billing.Invoice, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	resp, err := a.billing.ListInvoices(ctx, billing.ListInvoicesInput{
		Namespaces:       params.Namespaces,
		IDs:              params.InvoiceIDs,
		Customers:        params.Customers,
		CollectionAt:     lo.ToPtr(params.CollectionAt),
		ExtendedStatuses: []billing.InvoiceStatus{billing.InvoiceStatusGathering},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list collectable invoices: %w", err)
	}

	return resp.Items, nil
}

type CollectCustomerInvoiceInput struct {
	CustomerID string
	AsOf       *time.Time
}

func (i CollectCustomerInvoiceInput) Validate() error {
	var errs []error

	if i.CustomerID == "" {
		errs = append(errs, fmt.Errorf("customer id must not be empty"))
	}

	if i.AsOf != nil && i.AsOf.IsZero() {
		errs = append(errs, fmt.Errorf("asOf time must not be zero"))
	}

	return errors.Join(errs...)
}

func (a *InvoiceCollector) CollectCustomerInvoice(ctx context.Context, params CollectCustomerInvoiceInput) ([]billing.Invoice, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	resp, err := a.billing.ListInvoices(ctx, billing.ListInvoicesInput{
		Customers:        []string{params.CustomerID},
		ExtendedStatuses: []billing.InvoiceStatus{billing.InvoiceStatusGathering},
		Expand: billing.InvoiceExpand{
			Lines: true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get gathering invoice(s) for customer [customer=%s]: %w", params.CustomerID, err)
	}

	if len(resp.Items) == 0 {
		return nil, nil
	}

	invoice := resp.Items[0]
	if params.AsOf == nil || params.AsOf.IsZero() {
		invoice = billingservice.GetInvoiceWithEarliestCollectionAt(resp.Items)
		params.AsOf = lo.ToPtr(billingservice.GetEarliestValidInvoiceAt(invoice.Lines))
	}

	a.logger.DebugContext(ctx, "collecting customer invoices", "customer", params.CustomerID, "asOf", params.AsOf)

	invoices, err := a.billing.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customerentity.CustomerID{
			Namespace: invoice.Namespace,
			ID:        invoice.Customer.CustomerID,
		},
		AsOf: params.AsOf,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice(s) for customer [customer=%s]: %w", params.CustomerID, err)
	}

	return invoices, nil
}

// All runs invoice collection for all customers
func (a *InvoiceCollector) All(ctx context.Context, namespaces []string, customerIDs []string, batchSize int) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	a.logger.InfoContext(ctx, "listing invoices waiting for collection")

	invoices, err := a.ListCollectableInvoices(ctx, ListCollectableInvoicesInput{
		Namespaces:   namespaces,
		Customers:    customerIDs,
		CollectionAt: time.Now(),
	})
	if err != nil {
		return fmt.Errorf("failed to list invoices to collect: %w", err)
	}

	if len(invoices) == 0 {
		return nil
	}

	batches := [][]billing.Invoice{
		invoices,
	}
	if batchSize > 0 {
		batches = lo.Chunk(invoices, batchSize)
	}

	a.logger.DebugContext(ctx, "found invoices to collect", "count", len(invoices), "batchSize", batchSize)

	errChan := make(chan error, len(invoices))
	closeErrChan := sync.OnceFunc(func() {
		close(errChan)
	})
	defer closeErrChan()

	for _, batch := range batches {
		var wg sync.WaitGroup
		for _, invoice := range batch {
			wg.Add(1)

			go func() {
				defer wg.Done()

				_, err = a.CollectCustomerInvoice(ctx, CollectCustomerInvoiceInput{
					CustomerID: invoice.Customer.CustomerID,
				})
				if err != nil {
					err = fmt.Errorf("failed to collect invoice for customer [namespace=%s invoice=%s customer=%s]: %w", invoice.Namespace, invoice.ID, invoice.Customer.CustomerID, err)
				}

				errChan <- err
			}()
		}

		wg.Wait()
	}
	closeErrChan()

	var errs []error
	for err = range errChan {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

type Config struct {
	BillingService billing.Service
	Logger         *slog.Logger
}

func NewInvoiceCollector(config Config) (*InvoiceCollector, error) {
	if config.BillingService == nil {
		return nil, fmt.Errorf("billing service is required")
	}

	if config.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	return &InvoiceCollector{
		billing: config.BillingService,
		logger:  config.Logger,
	}, nil
}
