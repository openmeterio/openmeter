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

// HandleSubscriptionChange treats a lifecycle event as a prompt to reconcile
// current state. The current cancellation end is included in the sync horizon.
func (s *Service) HandleSubscriptionChange(ctx context.Context, subscriptionID models.NamespacedID) error {
	return s.synchronizeSubscriptionAndInvoiceCustomer(
		ctx,
		newSubscriptionReferenceOrView(subscriptionID),
		syncHorizon{asOf: clock.Now(), includeCurrentEnd: true},
	)
}

func (s *Service) HandleCancelledEvent(ctx context.Context, event *subscription.CancelledEvent) error {
	if event == nil {
		return nil
	}
	// Reconcile the current subscription even if a later continuation already
	// committed. The event's timestamp and view are not a reliable version.
	if err := s.HandleSubscriptionChange(ctx, event.Subscription.NamespacedID); err != nil {
		return err
	}
	if event.Spec.ActiveTo == nil {
		return errors.New("active_to is required for canceled events")
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
		if err := s.synchronizeSubscriptionAndInvoiceCustomer(
			ctx,
			newSubscriptionReferenceOrView(models.NamespacedID{
				Namespace: event.Invoice.Namespace,
				ID:        subscriptionID,
			}),
			syncHorizon{asOf: clock.Now()},
		); err != nil {
			return fmt.Errorf("syncing subscription[%s]: %w", subscriptionID, err)
		}
	}

	return nil
}

// HandleDeletedEvent is a handler for the subscription deleted event, it will make sure that
// we synchronize the subscription and invoice customer.
func (s *Service) HandleDeletedEvent(ctx context.Context, event *subscription.DeletedEvent) error {
	_, err := s.synchronizeSubscription(
		ctx,
		newSubscriptionReferenceOrView(event.Subscription.NamespacedID),
		syncHorizon{asOf: clock.Now()},
	)
	return err
}
