package reconciler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	"github.com/openmeterio/openmeter/pkg/clock"
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
func (r *Reconciler) ListSubscriptions(ctx context.Context, in ReconcilerListSubscriptionsInput) ([]subscriptionsync.SubscriptionToReconcile, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	var out []subscriptionsync.SubscriptionToReconcile

	err := r.forEachCandidatePage(ctx, in, clock.Now(), func(page []subscriptionsync.SubscriptionToReconcile) error {
		out = append(out, page...)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

// forEachCandidatePage calls fn with each page of subscriptions to reconcile:
// first the non-deleted ones that are due at asOf, then the deleted ones awaiting
// cleanup. The next page is listed only after fn returns, so a caller can sync a
// large backlog without holding all of it.
//
// asOf stays the same for every page, so a deadline passing while we page cannot
// move a subscription across the cursor.
func (r *Reconciler) forEachCandidatePage(ctx context.Context, in ReconcilerListSubscriptionsInput, asOf time.Time, fn func([]subscriptionsync.SubscriptionToReconcile) error) error {
	err := forEachPage(fn, func(cursor *paginationv2.Cursor) (paginationv2.Result[subscriptionsync.SubscriptionToReconcile], error) {
		page, err := r.subscriptionSync.ListSubscriptionsToReconcile(ctx, subscriptionsync.ListSubscriptionsToReconcileInput{
			Namespaces: in.Namespaces,
			Customers:  in.Customers,
			AsOf:       asOf,
			Lookback:   in.Lookback,
			ForceSync:  in.ForceSync,
			Cursor:     cursor,
			Limit:      defaultWindowSize,
		})
		if err != nil {
			return page, fmt.Errorf("listing subscriptions awaiting reconciliation: %w", err)
		}

		return page, nil
	})
	if err != nil {
		return err
	}

	return forEachPage(fn, func(cursor *paginationv2.Cursor) (paginationv2.Result[subscriptionsync.SubscriptionToReconcile], error) {
		page, err := r.subscriptionSync.ListDeletedSubscriptionsToReconcile(ctx, subscriptionsync.ListDeletedSubscriptionsToReconcileInput{
			Namespaces: in.Namespaces,
			Customers:  in.Customers,
			Cursor:     cursor,
			Limit:      defaultWindowSize,
		})
		if err != nil {
			return page, fmt.Errorf("listing deleted subscriptions awaiting reconciliation: %w", err)
		}

		return page, nil
	})
}

// forEachPage follows the cursor from the first page to the last, passing each
// non-empty page to fn before listing the next one.
func forEachPage(
	fn func([]subscriptionsync.SubscriptionToReconcile) error,
	listPage func(cursor *paginationv2.Cursor) (paginationv2.Result[subscriptionsync.SubscriptionToReconcile], error),
) error {
	var cursor *paginationv2.Cursor

	for {
		page, err := listPage(cursor)
		if err != nil {
			return err
		}

		// A cursor that does not move forward in (created_at, id) order would list
		// pages we have already seen, forever. Checked before the page is handed
		// on, so a repeated page is not synced again.
		if next := page.NextCursor; next != nil && cursor != nil {
			advanced := next.Time.After(cursor.Time) || (next.Time.Equal(cursor.Time) && next.ID > cursor.ID)
			if !advanced {
				return errors.New("listing subscriptions to reconcile: the pagination cursor did not advance")
			}
		}

		if len(page.Items) > 0 {
			if err := fn(page.Items); err != nil {
				return err
			}
		}

		if page.NextCursor == nil {
			return nil
		}

		cursor = page.NextCursor
	}
}

type ReconcilerAllInput struct {
	ReconcilerListSubscriptionsInput
	ForceResyncOldSubscriptions bool
}

// All syncs every subscription that needs periodic reconciliation, one page of
// candidates at a time. A failed sync does not stop the run: its subscription
// stays due and the error is returned once the run ends.
func (r *Reconciler) All(ctx context.Context, in ReconcilerAllInput) error {
	if err := in.Validate(); err != nil {
		return err
	}

	// One selection time for the whole run: every page compares deadlines against
	// it, and a sync that succeeds after it makes ours unnecessary.
	selectedAt := clock.Now()

	var syncOptions []subscriptionsync.SynchronizeSubscriptionOption
	if in.ForceResyncOldSubscriptions {
		syncOptions = append(syncOptions, subscriptionsync.WithForceResyncOldSubscriptions())
	}

	if !in.ForceSync {
		// A forced run syncs no matter what other syncs did in the meantime.
		syncOptions = append(syncOptions, subscriptionsync.WithSkipIfSyncedSince(selectedAt))
	}

	var errs []error

	selected := 0

	err := r.forEachCandidatePage(ctx, in.ReconcilerListSubscriptionsInput, selectedAt, func(page []subscriptionsync.SubscriptionToReconcile) error {
		selected += len(page)

		// Logged per page, so a large backlog (for example the first run after
		// switching to v2 scheduling) can be watched draining.
		r.logger.InfoContext(ctx, "reconciling a page of subscriptions",
			slog.Int("page_size", len(page)),
			slog.Int("selected_so_far", selected),
			slog.Bool("force_sync", in.ForceSync),
		)

		for _, sub := range page {
			// Once the run is canceled every remaining sync would fail the same way.
			if err := ctx.Err(); err != nil {
				return err
			}

			if err := r.subscriptionSync.SyncByID(ctx, sub.NamespacedID, clock.Now(), syncOptions...); err != nil {
				r.logger.ErrorContext(ctx, "failed to reconcile subscription",
					"namespace", sub.Namespace, "subscription_id", sub.ID, "error", err)

				errs = append(errs, fmt.Errorf("reconciling subscription %s/%s: %w", sub.Namespace, sub.ID, err))
			}
		}

		return nil
	})
	if err != nil {
		errs = append(errs, err)
	}

	r.logger.InfoContext(ctx, "periodic subscription reconciliation finished",
		slog.Int("selected", selected),
		slog.Int("failed", len(errs)),
	)

	return errors.Join(errs...)
}
