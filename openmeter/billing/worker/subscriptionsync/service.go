package subscriptionsync

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
	paginationv2 "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

type Service interface {
	ListSubscriptionsToReconcile(context.Context, ListSubscriptionsToReconcileInput) (paginationv2.Result[SubscriptionToReconcile], error)
	ListDeletedSubscriptionsToReconcile(context.Context, ListDeletedSubscriptionsToReconcileInput) (paginationv2.Result[SubscriptionToReconcile], error)

	SyncService
	SyncStateService
}

type SyncService interface {
	SyncByID(ctx context.Context, subscriptionID models.NamespacedID, asOf time.Time, opts ...SynchronizeSubscriptionOption) error
}

type SyncStateService interface {
	GetSyncStatesV2(ctx context.Context, input GetSyncStatesV2Input) ([]SyncStateV2Managed, error)
}

// SubscriptionToReconcile identifies a subscription that needs a sync. The
// customer comes along because subscription sync events are keyed by customer.
type SubscriptionToReconcile struct {
	models.NamespacedID
	CustomerID string
}

// ListSubscriptionsToReconcileInput selects non-deleted subscriptions that were
// active within the lookback and need periodic reconciliation. Deleted
// subscriptions are listed by ListDeletedSubscriptionsToReconcile instead.
//
// A subscription needs reconciliation when it has no sync-state row, no sync has
// recorded an outcome yet, it changed after the last recorded sync, or it is
// active and reached its next reconciliation deadline.
type ListSubscriptionsToReconcileInput struct {
	Namespaces []string
	Customers  []string
	// AsOf is the worker's current time: deadlines are compared against it and
	// the lookback ends at it. Use the same value for every page of one listing.
	AsOf time.Time
	// Lookback limits the list to subscriptions active at some point in the
	// Lookback before AsOf.
	Lookback time.Duration
	// ForceSync lists every subscription in the lookback regardless of its
	// sync state.
	ForceSync bool
	// Cursor pages by creation time and subscription ID. Offset pagination could
	// skip subscriptions when concurrent syncs settle earlier candidates.
	Cursor *paginationv2.Cursor
	Limit  int
}

func (i ListSubscriptionsToReconcileInput) Validate() error {
	var errs []error
	if i.AsOf.IsZero() {
		errs = append(errs, errors.New("as of is required"))
	}

	if i.Lookback <= 0 {
		errs = append(errs, errors.New("lookback must be greater than zero"))
	}

	if i.Limit <= 0 {
		errs = append(errs, errors.New("limit must be greater than zero"))
	}

	if i.Cursor != nil {
		if err := i.Cursor.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("cursor: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type SynchronizeSubscriptionOptions struct {
	DryRun                      bool
	InvoicePendingLines         bool
	ForceResyncOldSubscriptions bool
	// SkipIfSyncedSince is set by WithSkipIfSyncedSince.
	SkipIfSyncedSince *time.Time
}

// WithSkipIfSyncedSince makes the sync a no-op when another sync has succeeded
// after selectedAt and the subscription has not changed since.
//
// Periodic reconciliation selects its candidates first and syncs them later. An
// event handler can sync one of them in between, with a horizon the periodic
// run does not know about: a cancellation syncs up to the subscription end. A
// periodic sync with the current time as its horizon would then remove the
// lines provisioned for the later billing periods.
func WithSkipIfSyncedSince(selectedAt time.Time) SynchronizeSubscriptionOption {
	return func(o *SynchronizeSubscriptionOptions) {
		o.SkipIfSyncedSince = &selectedAt
	}
}

// WithForceResyncOldSubscriptions reconciles billing artifacts even when the
// subscription ended more than 30 days ago.
func WithForceResyncOldSubscriptions() SynchronizeSubscriptionOption {
	return func(o *SynchronizeSubscriptionOptions) {
		o.ForceResyncOldSubscriptions = true
	}
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
