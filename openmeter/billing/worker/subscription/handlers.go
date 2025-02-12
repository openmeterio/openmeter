package billingworkersubscription

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

// HandleCancelledEvent is a handler for the subscription cancel event, it will make sure that
// we syncronize the
func (h *Handler) HandleCancelledEvent(ctx context.Context, event *subscription.CancelledEvent) error {
	now := clock.Now()

	if event.Spec.ActiveTo == nil {
		// Let's do one sync, just to make sure we have at least the new items lined up
		err := h.SyncronizeSubscription(ctx, event.SubscriptionView, now)
		if err != nil {
			return err
		}

		return errors.New("active_to is required for canceled events")
	}

	// Let's sync up to the end of the subscription
	err := h.SyncronizeSubscription(ctx, event.SubscriptionView, *event.Spec.ActiveTo)
	if err != nil {
		return err
	}

	if event.Spec.ActiveTo.Before(now) {
		invoices, err := h.billingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customer.CustomerID{
				Namespace: event.Subscription.Namespace,
				ID:        event.Customer.ID,
			},
			AsOf: event.Spec.ActiveTo,
		})
		if err != nil {
			// Let's wait for the collector to create the invoice in case we run into any errors
			if errors.Is(err, billing.ErrInvoiceCreateNoLines) {
				// Safe to ignore, maybe there are no billable items at all
				return nil
			}

			h.logger.WarnContext(ctx, "failed to create invoice", "error", err, "customer_id", event.Customer.ID)
			return nil
		}

		h.logger.Info("created invoice(s) on subscription cancel trigger",
			"customer_id", event.Customer.ID,
			"invoice_ids", lo.Map(invoices, func(invoice billing.Invoice, _ int) string {
				return invoice.ID
			}),
		)

		return nil
	}

	return nil
}

// HandleInvoiceCreation is a handler for the invoice creation event, it will make sure that
// we are backfilling the items consumed by invoice creation into the gathering invoice.
func (h *Handler) HandleInvoiceCreation(ctx context.Context, invoice billing.EventInvoice) error {
	if invoice.Status == billing.InvoiceStatusGathering {
		return nil
	}

	affectedSubscriptions := lo.Uniq(
		lo.Map(
			lo.Filter(invoice.Lines.OrEmpty(), func(line *billing.Line, _ int) bool {
				return line.Status == billing.InvoiceLineStatusValid &&
					line.Subscription != nil
			}),
			func(line *billing.Line, _ int) string {
				return line.Subscription.SubscriptionID
			}),
	)

	for _, subscriptionID := range affectedSubscriptions {
		subsView, err := h.subscriptionService.GetView(ctx, models.NamespacedID{
			Namespace: invoice.Namespace,
			ID:        subscriptionID,
		})
		if err != nil {
			return fmt.Errorf("getting subscription view[%s]: %w", subscriptionID, err)
		}

		// We use the current time as reference point instead of the invoice, as if we are delayed
		// we might want to provision more lines
		if err := h.SyncronizeSubscription(ctx, subsView, clock.Now()); err != nil {
			return fmt.Errorf("syncing subscription[%s]: %w", subscriptionID, err)
		}
	}

	return nil
}
