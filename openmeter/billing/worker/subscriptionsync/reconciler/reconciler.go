package reconciler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	paginationv2 "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

const (
	defaultWindowSize = 10_000

	// DefaultLookback matches the age after which sync skips an ended
	// subscription, so a missed final sync stays listed for as long as a sync
	// would still do billing work.
	DefaultLookback = 30 * 24 * time.Hour
)

// Reconciler is a component that periodically reconciles the subscription state with the billing state
// this is essential, as the invoice creation is purley event driven. In case a processing error happens,
// we might fail to create an invoice, and stop processing the subscription.
type Reconciler struct {
	subscriptionSync subscriptionsync.Service
	logger           *slog.Logger
}

type ReconcilerConfig struct {
	SubscriptionSync subscriptionsync.Service
	Logger           *slog.Logger
}

func (c ReconcilerConfig) Validate() error {
	if c.SubscriptionSync == nil {
		return errors.New("subscriptionSync is required")
	}

	if c.Logger == nil {
		return errors.New("logger is required")
	}

	return nil
}

func NewReconciler(config ReconcilerConfig) (*Reconciler, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Reconciler{
		subscriptionSync: config.SubscriptionSync,
		logger:           config.Logger,
	}, nil
}

type ReconcilerListSubscriptionsInput struct {
	Namespaces []string
	Customers  []string
	Lookback   time.Duration
	// ForceSync selects every subscription in the lookback regardless of its sync
	// state. Deleted subscriptions with completed cleanup stay excluded.
	ForceSync bool
}

func (i ReconcilerListSubscriptionsInput) Validate() error {
	if i.Lookback <= 0 {
		return errors.New("lookback must be greater than 0")
	}

	return nil
}

// ListSubscriptions returns the subscriptions periodic reconciliation has to
// sync: non-deleted ones active within the lookback whose sync state is due, and
// deleted ones whose cleanup has not been recorded. The two candidate queries
// decide what is due; every subscription returned here gets synced.
func (r *Reconciler) ListSubscriptions(ctx context.Context, in ReconcilerListSubscriptionsInput) ([]models.NamespacedID, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	// Use one timestamp for all pages, so a deadline passing while we page
	// cannot move a subscription across the cursor.
	now := clock.Now()

	due, err := paginationv2.CollectAll(ctx, paginationv2.NewPaginator(func(ctx context.Context, cursor *paginationv2.Cursor) (paginationv2.Result[models.NamespacedID], error) {
		return r.subscriptionSync.ListSubscriptionsToReconcile(ctx, subscriptionsync.ListSubscriptionsToReconcileInput{
			Namespaces: in.Namespaces,
			Customers:  in.Customers,
			AsOf:       now,
			Lookback:   in.Lookback,
			ForceSync:  in.ForceSync,
			Cursor:     cursor,
			Limit:      defaultWindowSize,
		})
	}), nil)
	if err != nil {
		return nil, fmt.Errorf("listing subscriptions awaiting reconciliation: %w", err)
	}

	deleted, err := paginationv2.CollectAll(ctx, paginationv2.NewPaginator(func(ctx context.Context, cursor *paginationv2.Cursor) (paginationv2.Result[models.NamespacedID], error) {
		return r.subscriptionSync.ListDeletedSubscriptionsToReconcile(ctx, subscriptionsync.ListDeletedSubscriptionsToReconcileInput{
			Namespaces: in.Namespaces,
			Customers:  in.Customers,
			Cursor:     cursor,
			Limit:      defaultWindowSize,
		})
	}), nil)
	if err != nil {
		return nil, fmt.Errorf("listing deleted subscriptions awaiting reconciliation: %w", err)
	}

	return append(due, deleted...), nil
}

type ReconcilerAllInput struct {
	ReconcilerListSubscriptionsInput
	ForceResyncOldSubscriptions bool
}

func (r *Reconciler) All(ctx context.Context, in ReconcilerAllInput) error {
	subscriptionIDs, err := r.ListSubscriptions(ctx, in.ReconcilerListSubscriptionsInput)
	if err != nil {
		return fmt.Errorf("failed to list subscriptions: %w", err)
	}

	var syncOptions []subscriptionsync.SynchronizeSubscriptionOption
	if in.ForceResyncOldSubscriptions {
		syncOptions = append(syncOptions, subscriptionsync.WithForceResyncOldSubscriptions())
	}

	// Logged before syncing, so a large backlog (for example the first run after
	// switching to v2 scheduling) can be watched draining from run to run.
	r.logger.InfoContext(ctx, "selected subscriptions for periodic reconciliation",
		slog.Int("total", len(subscriptionIDs)),
		slog.Bool("force_sync", in.ForceSync),
	)

	var outErr error
	for _, subscriptionID := range subscriptionIDs {
		if err := r.subscriptionSync.SyncByID(ctx, subscriptionID, clock.Now(), syncOptions...); err != nil {
			r.logger.ErrorContext(ctx, "failed to reconcile subscription", "error", err)

			outErr = errors.Join(outErr, fmt.Errorf("failed to reconcile subscription: %w", err))
		}
	}

	return outErr
}
