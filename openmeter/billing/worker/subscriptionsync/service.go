package subscriptionsync

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/subscription"
)

type Service interface {
	EventHandler
	SyncService
	SyncStateService
}

type SyncService interface {
	SynchronizeSubscriptionAndInvoiceCustomer(ctx context.Context, subs subscription.SubscriptionView, asOf time.Time) error
	SynchronizeSubscription(ctx context.Context, subs subscription.SubscriptionView, asOf time.Time) error
}

type EventHandler interface {
	HandleCancelledEvent(ctx context.Context, event *subscription.CancelledEvent) error
	HandleSubscriptionSyncEvent(ctx context.Context, event *subscription.SubscriptionSyncEvent) error
	HandleInvoiceCreation(ctx context.Context, event *billing.StandardInvoiceCreatedEvent) error
}

type SyncStateService interface {
	GetSyncStates(ctx context.Context, input GetSyncStatesInput) ([]SyncState, error)
}
