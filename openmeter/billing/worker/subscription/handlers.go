package billingworkersubscription

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// HandleCancelledEvent is a handler for the subscription cancel event, it will make sure that
// we synchronize the
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

			if errors.Is(err, billing.ErrNamespaceLocked) {
				h.logger.WarnContext(ctx, "namespace is locked, skipping invoice creation", "customer_id", event.Customer.ID)
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
func (h *Handler) HandleInvoiceCreation(ctx context.Context, event billing.EventInvoice) error {
	if event.Invoice.Status == billing.InvoiceStatusGathering {
		return nil
	}

	affectedSubscriptions := lo.Uniq(
		lo.Map(
			lo.Filter(event.Invoice.Lines.OrEmpty(), func(line *billing.Line, _ int) bool {
				return line.Status == billing.InvoiceLineStatusValid &&
					line.Subscription != nil
			}),
			func(line *billing.Line, _ int) string {
				return line.Subscription.SubscriptionID
			}),
	)

	for _, subscriptionID := range affectedSubscriptions {
		subsView, err := h.subscriptionService.GetView(ctx, models.NamespacedID{
			Namespace: event.Invoice.Namespace,
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

func (h *Handler) HandleSubscriptionCreated(ctx context.Context, subs subscription.SubscriptionView, asOf time.Time) error {
	if err := h.SyncronizeSubscription(ctx, subs, asOf); err != nil {
		return err
	}

	if subs.Spec.HasBillables() {
		// Let's check if there are any pending lines that we can invoice now (those will be in-advance fees, so we don't have to wait
		// for the collection period)

		gatheringInvoices, err := h.billingService.ListInvoices(ctx, billing.ListInvoicesInput{
			Namespaces:       []string{subs.Subscription.Namespace},
			Customers:        []string{subs.Customer.ID},
			ExtendedStatuses: []billing.InvoiceStatus{billing.InvoiceStatusGathering},
			Currencies:       []currencyx.Code{subs.Spec.Currency},
			Expand: billing.InvoiceExpand{
				Lines: true,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to list gathering invoices: %w", err)
		}

		now := clock.Now()

		invoicableLines := []string{}

		for _, invoice := range gatheringInvoices.Items {
			inScopeLines := lo.Filter(invoice.Lines.OrEmpty(), func(line *billing.Line, _ int) bool {
				if line.Subscription.SubscriptionID != subs.Subscription.ID {
					return false
				}

				if line.Status != billing.InvoiceLineStatusValid {
					return false
				}

				return !line.InvoiceAt.After(now)
			})

			if len(inScopeLines) == 0 {
				continue
			}

			invoicableLines = append(invoicableLines, lo.Map(inScopeLines, func(line *billing.Line, _ int) string {
				return line.ID
			})...)
		}

		if len(invoicableLines) > 0 {
			invoices, err := h.billingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
				Customer: customer.CustomerID{
					Namespace: subs.Subscription.Namespace,
					ID:        subs.Customer.ID,
				},

				IncludePendingLines: mo.Some(invoicableLines),
			})
			if err != nil {
				if errors.Is(err, billing.ErrNamespaceLocked) {
					h.logger.WarnContext(ctx, "namespace is locked, skipping invoice creation", "customer_id", subs.Customer.ID)
					return nil
				}

				return fmt.Errorf("failed to create invoice: %w", err)
			}

			h.logger.Info("created invoice on subscription creation trigger",
				"customer_id", subs.Customer.ID,
				"invoice_ids", lo.Map(invoices, func(inv billing.Invoice, _ int) string {
					return fmt.Sprintf("%s/%s", inv.Number, inv.ID)
				}),
			)
		}
	}

	return nil
}
