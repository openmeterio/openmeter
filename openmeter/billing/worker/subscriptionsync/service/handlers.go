package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

// HandleCancelledEvent is a handler for the subscription cancel event, it will make sure that
// we synchronize the
func (s *Service) HandleCancelledEvent(ctx context.Context, event *subscription.CancelledEvent) error {
	now := clock.Now()

	// For canceled events, we skip the pre-sync invoice creation, as we don't want to create an invoice that we
	// might need to change immediately after the sync.

	if event.Spec.ActiveTo == nil {
		// Let's do one sync, just to make sure we have at least the new items lined up
		err := s.synchronizeSubscriptionAndInvoiceCustomer(ctx, newSubscriptionReferenceOrView(event.SubscriptionView), now)
		if err != nil {
			return err
		}

		return errors.New("active_to is required for canceled events")
	}

	// Let's sync up to the end of the subscription
	err := s.synchronizeSubscriptionAndInvoiceCustomer(ctx, newSubscriptionReferenceOrView(event.SubscriptionView), *event.Spec.ActiveTo)
	if err != nil {
		return err
	}

	return nil
}

// HandleInvoiceCreation is a handler for the invoice creation event, it will make sure that
// we are backfilling the items consumed by invoice creation into the gathering invoice.
func (s *Service) HandleInvoiceCreation(ctx context.Context, event *billing.StandardInvoiceCreatedEvent) error {
	if event == nil {
		return nil
	}

	if event.Invoice.Status == billing.StandardInvoiceStatusGathering {
		return nil
	}

	affectedSubscriptions := lo.Uniq(
		lo.Map(
			lo.Filter(event.Invoice.Lines.OrEmpty(), func(line *billing.StandardLine, _ int) bool {
				return line.Subscription != nil
			}),
			func(line *billing.StandardLine, _ int) string {
				return line.Subscription.SubscriptionID
			}),
	)

	for _, subscriptionID := range affectedSubscriptions {
		// We use the current time as reference point instead of the invoice, as if we are delayed
		// we might want to provision more lines
		if err := s.synchronizeSubscriptionAndInvoiceCustomer(ctx, newSubscriptionReferenceOrView(models.NamespacedID{
			Namespace: event.Invoice.Namespace,
			ID:        subscriptionID,
		}), clock.Now()); err != nil {
			return fmt.Errorf("syncing subscription[%s]: %w", subscriptionID, err)
		}
	}

	return nil
}

// HandleDeletedEvent is a handler for the subscription deleted event, it will make sure that
// we synchronize the subscription and invoice customer.
func (s *Service) HandleDeletedEvent(ctx context.Context, event *subscription.DeletedEvent) error {
	_, err := s.synchronizeSubscription(ctx, newSubscriptionReferenceOrView(event.Subscription.NamespacedID), clock.Now())
	return err
}
