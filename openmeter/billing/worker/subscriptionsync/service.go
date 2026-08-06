package subscriptionsync

import (
	"context"
	"errors"
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
	SyncByViewAndInvoiceCustomer(ctx context.Context, view subscription.SubscriptionView, asOf time.Time, opts ...SynchronizeSubscriptionOption) error
	SyncByIDAndInvoiceCustomer(ctx context.Context, subscriptionID models.NamespacedID, asOf time.Time, opts ...SynchronizeSubscriptionOption) error
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
	DryRun                          bool
	SkipCustomCurrencySubscriptions bool
}

type SynchronizeSubscriptionOption func(*SynchronizeSubscriptionOptions)

func EnableDryRun() SynchronizeSubscriptionOption {
	return func(o *SynchronizeSubscriptionOptions) {
		o.DryRun = true
	}
}

// SkipCustomCurrencySubscriptions makes automatic reconciliation ignore subscriptions
// that billing cannot represent yet. Explicit sync callers should omit this option so
// the unsupported operation remains visible to them.
func SkipCustomCurrencySubscriptions() SynchronizeSubscriptionOption {
	return func(o *SynchronizeSubscriptionOptions) {
		o.SkipCustomCurrencySubscriptions = true
	}
}

// ErrCustomCurrencyBillingNotSupported is returned when explicit sync reaches
// a subscription that billing cannot represent without currency conversion.
var ErrCustomCurrencyBillingNotSupported = errors.New("billing sync does not support subscriptions with custom-currency priced items")
