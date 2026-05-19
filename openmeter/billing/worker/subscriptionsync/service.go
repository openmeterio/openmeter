package subscriptionsync

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Service interface {
	EventHandler
	SyncService
	SyncStateService
}

type SyncService interface {
	SyncByViewAndInvoiceCustomer(ctx context.Context, view subscription.SubscriptionView, asOf time.Time) error
	SyncByIDAndInvoiceCustomer(ctx context.Context, subscriptionID models.NamespacedID, asOf time.Time) error
	SyncByView(ctx context.Context, view subscription.SubscriptionView, asOf time.Time, opts ...SynchronizeSubscriptionOption) error
	SyncByID(ctx context.Context, subscriptionID models.NamespacedID, asOf time.Time, opts ...SynchronizeSubscriptionOption) error
}

type EventHandler interface {
	HandleCancelledEvent(ctx context.Context, event *subscription.CancelledEvent) error
	HandleDeletedEvent(ctx context.Context, event *subscription.DeletedEvent) error
	HandleSubscriptionSyncEvent(ctx context.Context, event *subscription.SubscriptionSyncEvent) error
	HandleInvoiceCreation(ctx context.Context, event *billing.StandardInvoiceCreatedEvent) error
}

type SyncStateService interface {
	GetSyncStates(ctx context.Context, input GetSyncStatesInput) ([]SyncState, error)
}

type SynchronizeSubscriptionOptions struct {
	DryRun bool
}

type SynchronizeSubscriptionOption func(*SynchronizeSubscriptionOptions)

func EnableDryRun() SynchronizeSubscriptionOption {
	return func(o *SynchronizeSubscriptionOptions) {
		o.DryRun = true
	}
}
