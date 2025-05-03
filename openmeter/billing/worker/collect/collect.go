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
	"github.com/openmeterio/openmeter/openmeter/customer"
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
		a.logger.DebugContext(ctx, "no invoices found for customer to be collected", "customer", params.CustomerID)

		return nil, nil
	}

	// Calculate asOf parameter
	asOf := lo.FromPtr(params.AsOf)
	if asOf.IsZero() {
		customerProfile, err := a.billing.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
			Customer: customer.CustomerID{
				Namespace: resp.Items[0].Namespace,
				ID:        resp.Items[0].Customer.CustomerID,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get customer profile [customer=%s]: %w", params.CustomerID, err)
		}

		asOf, _ = customerProfile.MergedProfile.WorkflowConfig.Collection.Interval.Negate().AddTo(time.Now())
	}

	// Calculate alignedAsOf time which is set to the latest invoiceAt time which is still before the time defined by asOf.
	var alignedAsOf time.Time
	for _, invoice := range resp.Items {
		if invoice.Lines.IsAbsent() {
			a.logger.WarnContext(ctx, "skipping invoice as lines not fetched", "customer", params.CustomerID, "invoice", invoice.ID)

			continue
		}

		latestInvoiceAt := billingservice.GetLatestValidInvoiceAtAsOf(invoice.Lines, asOf)
		if latestInvoiceAt.IsZero() {
			a.logger.DebugContext(ctx, "empty invoice found", "customer", invoice.Customer.CustomerID, "invoice", invoice.ID)

			continue
		}

		if latestInvoiceAt.Before(asOf) && latestInvoiceAt.After(alignedAsOf) {
			alignedAsOf = latestInvoiceAt
		}

		// Stop iteration as the asOf is already aligned in this case
		if latestInvoiceAt.Equal(asOf) {
			alignedAsOf = latestInvoiceAt

			break
		}
	}

	if alignedAsOf.IsZero() {
		a.logger.DebugContext(ctx, "customer has no lines to be collected", "customer", params.CustomerID, "asOf", asOf.UTC().Format(time.RFC3339))

		return nil, nil
	}

	a.logger.DebugContext(ctx, "collecting customer invoices", "customer", params.CustomerID, "asOf", alignedAsOf)

	invoices, err := a.billing.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customer.CustomerID{
			Namespace: resp.Items[0].Namespace,
			ID:        resp.Items[0].Customer.CustomerID,
		},
		AsOf: lo.ToPtr(alignedAsOf),
	})
	if err != nil {
		if errors.Is(err, billing.ErrNamespaceLocked) {
			a.logger.WarnContext(ctx, "namespace is locked, skipping collection", "customer", params.CustomerID)

			return nil, nil
		}

		return nil, fmt.Errorf("failed to create invoice(s) for customer [customer=%s]: %w", params.CustomerID, err)
	}

	return invoices, nil
}

func (a *InvoiceCollector) GetCollectionConfig(ctx context.Context, customer customer.CustomerID) (billing.CollectionConfig, error) {
	customerProfile, err := a.billing.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
		Customer: customer,
	})
	if err != nil {
		return billing.CollectionConfig{}, fmt.Errorf("failed to get collection configfor customer [namespace=%s customer=%s]: %w",
			customer.Namespace, customer.ID, err,
		)
	}

	return customerProfile.MergedProfile.WorkflowConfig.Collection, nil
}

func (a *InvoiceCollector) GetAsOfForCustomer(ctx context.Context, customer customer.CustomerID) (time.Time, error) {
	return a.GetAsOfForCustomerAt(ctx, customer, time.Now())
}

func (a *InvoiceCollector) GetAsOfForCustomerAt(ctx context.Context, customer customer.CustomerID, at time.Time) (time.Time, error) {
	collectionConfig, err := a.GetCollectionConfig(ctx, customer)
	if err != nil {
		return time.Time{}, err
	}

	collectionInterval, ok := collectionConfig.Interval.Duration()
	if !ok {
		return time.Time{}, fmt.Errorf("failed to cast collection interval for customer [namespace=%s customer=%s]: %w",
			customer.Namespace, customer.ID, err)
	}

	return at.Add(-1 * collectionInterval), nil
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
