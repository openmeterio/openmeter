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
	invoice billing.StandardInvoiceService

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

	batches := [][]billing.StandardInvoice{
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
func (a *AutoAdvancer) ListInvoicesPendingAutoAdvance(ctx context.Context, namespaces []string, ids []string) ([]billing.StandardInvoice, error) {
	resp, err := a.invoice.ListStandardInvoices(ctx, billing.ListStandardInvoicesInput{
		ExtendedStatuses: []billing.StandardInvoiceStatus{billing.StandardInvoiceStatusDraftWaitingAutoApproval},
		DraftUntil:       lo.ToPtr(time.Now()),
		Namespaces:       namespaces,
		IDs:              ids,
	})
	if err != nil {
		return nil, err
	}

	return resp.Items, nil
}

// ListInvoicesPendingCollection lists invoices that are due to be collected
func (a *AutoAdvancer) ListInvoicesPendingCollection(ctx context.Context, namespaces []string, ids []string) ([]billing.StandardInvoice, error) {
	resp, err := a.invoice.ListStandardInvoices(ctx, billing.ListStandardInvoicesInput{
		ExtendedStatuses: []billing.StandardInvoiceStatus{billing.StandardInvoiceStatusDraftWaitingForCollection},
		CollectionAt:     lo.ToPtr(time.Now()),
		Namespaces:       namespaces,
		IDs:              ids,
	})
	if err != nil {
		return nil, err
	}

	return resp.Items, nil
}

// ListStuckInvoicesNeedingAdvance lists invoices that are stuck in some advanceable state (this is a fail-safe mechanism)
func (a *AutoAdvancer) ListStuckInvoicesNeedingAdvance(ctx context.Context, namespaces []string, ids []string) ([]billing.StandardInvoice, error) {
	resp, err := a.invoice.ListStandardInvoices(ctx, billing.ListStandardInvoicesInput{
		HasAvailableAction: []billing.InvoiceAvailableActionsFilter{billing.InvoiceAvailableActionsFilterAdvance},
		Namespaces:         namespaces,
		IDs:                ids,
	})
	if err != nil {
		return nil, err
	}

	return resp.Items, nil
}

func (a *AutoAdvancer) ListInvoicesToAdvance(ctx context.Context, namespace []string, ids []string) ([]billing.StandardInvoice, error) {
	autoAdvanceInvoices, err := a.ListInvoicesPendingAutoAdvance(ctx, namespace, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to list invoices to auto-advance: %w", err)
	}

	collectingInvoices, err := a.ListInvoicesPendingCollection(ctx, namespace, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to list invoices to collect: %w", err)
	}

	stuckInvoices, err := a.ListStuckInvoicesNeedingAdvance(ctx, namespace, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to list invoices that can be advanced: %w", err)
	}

	allInvoices := append(autoAdvanceInvoices, stuckInvoices...)
	allInvoices = append(allInvoices, collectingInvoices...)
	return lo.UniqBy(allInvoices, func(i billing.StandardInvoice) string {
		return i.ID
	}), nil
}

func (a *AutoAdvancer) AdvanceInvoice(ctx context.Context, id billing.InvoiceID) (billing.StandardInvoice, error) {
	invoice, err := a.invoice.AdvanceInvoice(ctx, id)
	if err != nil {
		// ErrInvoiceCannotAdvance is returned when the invoice cannot be advanced due to state machine settings
		// thus we can safely ignore this error, we will retry
		if errors.Is(err, billing.ErrInvoiceCannotAdvance) {
			invoice, err := a.invoice.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
				Invoice: id,
			})
			if err != nil {
				return billing.StandardInvoice{}, fmt.Errorf("failed to get invoice by id: %w", err)
			}

			logArgs := []any{
				slog.String("invoice_id", invoice.ID),
				slog.String("namespace", invoice.Namespace),
				slog.Time("updated_at", invoice.UpdatedAt),
				slog.Any("status", invoice.Status),
				slog.Any("status_details", invoice.StatusDetails),
			}

			if invoice.DraftUntil != nil {
				logArgs = append(logArgs, slog.String("draft_until", invoice.DraftUntil.Format(time.RFC3339)))
			}

			a.logger.WarnContext(ctx, "invoice cannot be advanced by billing-worker's advancer", logArgs...)

			return invoice, nil
		}
	}

	return invoice, err
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
