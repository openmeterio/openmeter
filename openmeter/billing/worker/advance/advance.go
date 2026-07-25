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
	invoice billing.Service

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

				_, err = a.AdvanceInvoice(ctx, invoice.GetInvoiceID())
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

func (a *AutoAdvancer) ListInvoicesToAdvance(ctx context.Context, namespaces []string, ids []string) ([]billing.StandardInvoice, error) {
	invoices, err := a.invoice.ListStandardInvoicesPendingAdvancement(ctx, billing.ListStandardInvoicesPendingAdvancementInput{
		Namespaces: namespaces,
		IDs:        ids,
		AsOf:       time.Now(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list invoices pending advancement: %w", err)
	}

	return invoices, nil
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
