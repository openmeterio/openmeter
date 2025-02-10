package billingworkeradvance

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

type AutoAdvancer struct {
	invoice billing.InvoiceService

	logger *slog.Logger
}

// All runs auto-advance for all eligible invoices
func (a *AutoAdvancer) All(ctx context.Context, namespaces []string, batchSize int) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	a.logger.InfoContext(ctx, "listing invoices waiting for auto approval")

	invoices, err := a.ListInvoicesToAdvance(ctx, namespaces, nil)
	if err != nil {
		return fmt.Errorf("failed to list invoices to advance: %w", err)
	}

	batches := [][]billing.Invoice{
		invoices,
	}
	if batchSize > 0 {
		batches = lo.Chunk(invoices, batchSize)
	}

	a.logger.DebugContext(ctx, "found invoices to approve", "count", len(invoices), "batchSize", batchSize)

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

				_, err = a.AdvanceInvoice(ctx, invoice.InvoiceID())
				if err != nil {
					err = fmt.Errorf("failed to auto-advance invoice [namespace=%s id=%s]: %w", invoice.Namespace, invoice.ID, err)
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

// ListInvoicesPendingAutoAdvance lists invoices that are due to be auto-advanced
func (a *AutoAdvancer) ListInvoicesPendingAutoAdvance(ctx context.Context, namespaces []string, ids []string) ([]billing.Invoice, error) {
	resp, err := a.invoice.ListInvoices(ctx, billing.ListInvoicesInput{
		ExtendedStatuses: []billing.InvoiceStatus{billing.InvoiceStatusDraftWaitingAutoApproval},
		DraftUntil:       lo.ToPtr(time.Now()),
		Namespaces:       namespaces,
		IDs:              ids,
	})
	if err != nil {
		return nil, err
	}

	return resp.Items, nil
}

// ListStuckInvoicesNeedingAdvance lists invoices that are stuck in some advancable state (this is a fail-safe mechanism)
func (a *AutoAdvancer) ListStuckInvoicesNeedingAdvance(ctx context.Context, namespaces []string, ids []string) ([]billing.Invoice, error) {
	resp, err := a.invoice.ListInvoices(ctx, billing.ListInvoicesInput{
		HasAvailableAction: []billing.InvoiceAvailableActionsFilter{billing.InvoiceAvailableActionsFilterAdvance},
		Namespaces:         namespaces,
		IDs:                ids,
	})
	if err != nil {
		return nil, err
	}

	return resp.Items, nil
}

func (a *AutoAdvancer) ListInvoicesToAdvance(ctx context.Context, namespace []string, ids []string) ([]billing.Invoice, error) {
	autoAdvanceInvoices, err := a.ListInvoicesPendingAutoAdvance(ctx, namespace, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to list invoices to auto-advance: %w", err)
	}

	stuckInvoices, err := a.ListStuckInvoicesNeedingAdvance(ctx, namespace, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to list invoices that can be advanced: %w", err)
	}

	allInvoices := append(autoAdvanceInvoices, stuckInvoices...)

	return lo.UniqBy(allInvoices, func(i billing.Invoice) string {
		return i.ID
	}), nil
}

func (a *AutoAdvancer) AdvanceInvoice(ctx context.Context, id billing.InvoiceID) (billing.Invoice, error) {
	return a.invoice.AdvanceInvoice(ctx, id)
}

type Config struct {
	BillingService billing.Service
	Logger         *slog.Logger
}

func NewAdvancer(config Config) (*AutoAdvancer, error) {
	if config.BillingService == nil {
		return nil, fmt.Errorf("billing service is required")
	}

	if config.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	return &AutoAdvancer{
		invoice: config.BillingService,
		logger:  config.Logger,
	}, nil
}
