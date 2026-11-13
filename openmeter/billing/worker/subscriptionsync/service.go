package subscriptionsync

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type Service interface {
	SyncService
	SyncStateService
}

type SyncService interface {
	SyncByID(ctx context.Context, subscriptionID models.NamespacedID, asOf time.Time, opts ...SynchronizeSubscriptionOption) error
}

type SyncStateService interface {
	GetSyncStates(ctx context.Context, input GetSyncStatesInput) ([]SyncState, error)
}

type SynchronizeSubscriptionOptions struct {
	DryRun              bool
	InvoicePendingLines bool
}

// WithInvoicePendingLines collects the customer's pending lines after a successful
// subscription sync, so fees billed in advance can be invoiced immediately.
// Dry runs never collect pending lines.
func WithInvoicePendingLines() SynchronizeSubscriptionOption {
	return func(o *SynchronizeSubscriptionOptions) {
		o.InvoicePendingLines = true
	}
}

type SynchronizeSubscriptionOption func(*SynchronizeSubscriptionOptions)

func EnableDryRun() SynchronizeSubscriptionOption {
	return func(o *SynchronizeSubscriptionOptions) {
		o.DryRun = true
	}
}
