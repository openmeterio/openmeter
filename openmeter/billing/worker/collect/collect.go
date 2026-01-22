package billingworkercollect

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

type InvoiceCollector struct {
	billing          billing.Service
	lockedNamespaces []string

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

func (a *InvoiceCollector) ListCollectableInvoices(ctx context.Context, params ListCollectableInvoicesInput) ([]billing.StandardInvoice, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	resp, err := a.billing.ListInvoices(ctx, billing.ListInvoicesInput{
		Namespaces:       params.Namespaces,
		IDs:              params.InvoiceIDs,
		Customers:        params.Customers,
		CollectionAt:     lo.ToPtr(params.CollectionAt),
		ExtendedStatuses: []billing.StandardInvoiceStatus{billing.StandardInvoiceStatusGathering},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list collectable invoices: %w", err)
	}

	return resp.Items, nil
}

type CollectCustomerInvoiceInput struct {
	CustomerID customer.CustomerID
	AsOf       time.Time
}

func (i CollectCustomerInvoiceInput) Validate() error {
	var errs []error

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid customer id: %w", err))
	}

	if i.AsOf.IsZero() {
		errs = append(errs, fmt.Errorf("asOf time must not be zero"))
	}

	return errors.Join(errs...)
}

func (a *InvoiceCollector) CollectCustomerInvoice(ctx context.Context, params CollectCustomerInvoiceInput) ([]billing.StandardInvoice, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	a.logger.DebugContext(ctx, "collecting customer invoices", "customer", params.CustomerID)

	invoices, err := a.billing.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: params.CustomerID,
		// We want to make sure that system collection does not use progressive billing.
		ProgressiveBillingOverride: lo.ToPtr(false),
	})
	if err != nil {
		if errors.Is(err, billing.ErrNamespaceLocked) {
			a.logger.WarnContext(ctx, "namespace is locked, skipping collection", "customer", params.CustomerID)

			return nil, nil
		}

		if errors.Is(err, billing.ErrInvoiceCreateNoLines) {
			a.logger.WarnContext(ctx, "no invoices generated for customer during collection (possible data inconsistency), recalculating gathering invoices", "customer", params.CustomerID)

			if err := a.billing.RecalculateGatheringInvoices(ctx, params.CustomerID); err != nil {
				return nil, err
			}

			return nil, nil
		}

		return nil, fmt.Errorf("failed to create invoice(s) for customer [customer=%s]: %w", params.CustomerID, err)
	}

	return invoices, nil
}

// All runs invoice collection for all customers
func (a *InvoiceCollector) All(ctx context.Context, namespaces []string, customerIDFilter []string, batchSize int) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	a.logger.InfoContext(ctx, "listing invoices waiting for collection")

	invoices, err := a.ListCollectableInvoices(ctx, ListCollectableInvoicesInput{
		Namespaces:   namespaces,
		Customers:    customerIDFilter,
		CollectionAt: time.Now(),
	})
	if err != nil {
		return fmt.Errorf("failed to list invoices to collect: %w", err)
	}

	if len(invoices) == 0 {
		return nil
	}

	customerIDs := lo.Map(invoices, func(i billing.StandardInvoice, _ int) customer.CustomerID {
		return customer.CustomerID{
			Namespace: i.Namespace,
			ID:        i.Customer.CustomerID,
		}
	})

	customerIDs = lo.Filter(lo.Uniq(customerIDs), func(id customer.CustomerID, _ int) bool {
		return !slices.Contains(a.lockedNamespaces, id.Namespace)
	})

	batches := [][]customer.CustomerID{
		customerIDs,
	}
	if batchSize > 0 {
		batches = lo.Chunk(customerIDs, batchSize)
	}

	a.logger.DebugContext(ctx, "found customers to collect", "count", len(customerIDs), "batchSize", batchSize)

	errChan := make(chan error, len(customerIDs))
	closeErrChan := sync.OnceFunc(func() {
		close(errChan)
	})
	defer closeErrChan()

	for _, batch := range batches {
		var wg sync.WaitGroup
		for _, customerID := range batch {
			wg.Add(1)

			go func() {
				defer wg.Done()

				_, err = a.CollectCustomerInvoice(ctx, CollectCustomerInvoiceInput{
					CustomerID: customerID,
					AsOf:       time.Now(),
				})
				if err != nil {
					err = fmt.Errorf("failed to collect invoice for customer [namespace=%s customer=%s]: %w", customerID.Namespace, customerID.ID, err)
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
	BillingService   billing.Service
	Logger           *slog.Logger
	LockedNamespaces []string
}

func NewInvoiceCollector(config Config) (*InvoiceCollector, error) {
	if config.BillingService == nil {
		return nil, fmt.Errorf("billing service is required")
	}

	if config.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	return &InvoiceCollector{
		billing:          config.BillingService,
		logger:           config.Logger,
		lockedNamespaces: config.LockedNamespaces,
	}, nil
}
