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
	"github.com/openmeterio/openmeter/openmeter/customer"
)

type InvoiceCollector struct {
	gatheringInvoices  billing.GatheringInvoiceService
	billingService     billing.Service
	lockedNamespaces   []string
	maxLinesPerInvoice int

	logger *slog.Logger
}

type ListCustomersToCollectInput struct {
	Namespaces  []string
	InvoiceIDs  []string
	CustomerIDs []string
	AsOf        time.Time
}

func (i ListCustomersToCollectInput) Validate() error {
	var errs []error

	if i.AsOf.IsZero() {
		errs = append(errs, fmt.Errorf("asOf time must not be zero"))
	}

	return errors.Join(errs...)
}

func (a *InvoiceCollector) ListCustomersToCollect(ctx context.Context, params ListCustomersToCollectInput) ([]customer.CustomerID, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	customers, err := a.gatheringInvoices.ListCustomerIDsPendingCollection(ctx, billing.ListCustomerIDsPendingCollectionInput{
		Namespaces:         params.Namespaces,
		ExcludedNamespaces: a.lockedNamespaces,
		InvoiceIDs:         params.InvoiceIDs,
		CustomerIDs:        params.CustomerIDs,
		AsOf:               params.AsOf,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list customers to collect: %w", err)
	}

	return customers, nil
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

	invoices, err := a.billingService.InvoicePendingLines(
		ctx,
		billing.InvoicePendingLinesInput{
			Customer:          params.CustomerID,
			AsOf:              lo.ToPtr(params.AsOf),
			ForceAsyncAdvance: true,
		},
		// We want to make sure that system collection does not use progressive billing.
		billing.WithPartialInvoiceLinesDisabled(),
		billing.WithMaxLinesPerInvoice(a.maxLinesPerInvoice),
	)
	if err != nil {
		if errors.Is(err, billing.ErrNamespaceLocked) {
			a.logger.WarnContext(ctx, "namespace is locked, skipping collection", "customer", params.CustomerID)

			return nil, nil
		}

		if errors.Is(err, billing.ErrInvoiceCreateNoLines) {
			a.logger.WarnContext(ctx, "no invoices generated for customer during collection (possible data inconsistency), recalculating gathering invoices", "customer", params.CustomerID)

			if err := a.gatheringInvoices.RecalculateGatheringInvoices(ctx, params.CustomerID); err != nil {
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

	customerIDs, err := a.ListCustomersToCollect(ctx, ListCustomersToCollectInput{
		Namespaces:  namespaces,
		CustomerIDs: customerIDFilter,
		AsOf:        time.Now(),
	})
	if err != nil {
		return fmt.Errorf("failed to list customers to collect: %w", err)
	}

	if len(customerIDs) == 0 {
		return nil
	}

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
	GatheringInvoiceService billing.GatheringInvoiceService
	BillingService          billing.Service
	Logger                  *slog.Logger
	LockedNamespaces        []string
	MaxLinesPerInvoice      int
}

func NewInvoiceCollector(config Config) (*InvoiceCollector, error) {
	if config.GatheringInvoiceService == nil {
		return nil, fmt.Errorf("gathering invoice service is required")
	}

	if config.BillingService == nil {
		return nil, fmt.Errorf("billing service is required")
	}

	if config.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	return &InvoiceCollector{
		gatheringInvoices:  config.GatheringInvoiceService,
		billingService:     config.BillingService,
		logger:             config.Logger,
		lockedNamespaces:   config.LockedNamespaces,
		maxLinesPerInvoice: config.MaxLinesPerInvoice,
	}, nil
}
