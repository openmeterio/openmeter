package reconciler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

const (
	defaultWindowSize = 10_000
)

// Reconciler is a component that periodically reconciles the subscription state with the billing state
// this is essential, as the invoice creation is purley event driven. In case a processing error happens,
// we might fail to create an invoice, and stop processing the subscription.
type Reconciler struct {
	subscriptionSync    subscriptionsync.Service
	subscriptionService subscription.Service
	customerService     customer.Service
	logger              *slog.Logger
}

type ReconcilerConfig struct {
	SubscriptionSync    subscriptionsync.Service
	SubscriptionService subscription.Service
	CustomerService     customer.Service
	Logger              *slog.Logger
}

func (c ReconcilerConfig) Validate() error {
	if c.SubscriptionSync == nil {
		return errors.New("subscriptionSync is required")
	}

	if c.SubscriptionService == nil {
		return errors.New("subscriptionService is required")
	}

	if c.CustomerService == nil {
		return errors.New("customerService is required")
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
		subscriptionSync:    config.SubscriptionSync,
		subscriptionService: config.SubscriptionService,
		customerService:     config.CustomerService,
		logger:              config.Logger,
	}, nil
}

type ReconcilerListSubscriptionsInput struct {
	Namespaces []string
	Customers  []string
	Lookback   time.Duration
}

func (i ReconcilerListSubscriptionsInput) Validate() error {
	if i.Lookback <= 0 {
		return errors.New("lookback must be greater than 0")
	}

	return nil
}

type SubscriptionWithSyncState struct {
	subscription.Subscription
	// TODO: Once Cloud's SubscriptionSyncSendSyncEvents activity uses ShouldReconcile,
	// restore legacy state for deleted subscriptions and remove the legacyIDs filter
	// below. Until then, nil prevents Cloud's legacy checks from skipping cleanup.
	*subscriptionsync.SyncState
	SyncStateV2 *subscriptionsync.SyncStateV2Managed
}

// ShouldReconcile decides whether periodic subscription sync has work to do.
// Deleted subscriptions need a v2 cleanup result; legacy no-billables state
// cannot establish that their billing artifacts were reconciled after deletion.
func (s SubscriptionWithSyncState) ShouldReconcile(now time.Time) bool {
	if s.Subscription.DeletedAt != nil {
		if s.SyncStateV2 == nil {
			return true
		}

		return s.SyncStateV2.LastSuccessState != subscriptionsync.SyncStateV2StateSubscriptionDeleted
	}

	if s.SyncState == nil {
		return true
	}

	if !s.SyncState.HasBillables {
		return false
	}

	// A nil deadline means no further periodic subscription sync is needed.
	if s.SyncState.NextSyncAfter == nil {
		return false
	}

	if s.SyncState.NextSyncAfter.After(now) {
		return false
	}

	return true
}

func (r *Reconciler) ListSubscriptions(ctx context.Context, in ReconcilerListSubscriptionsInput) ([]SubscriptionWithSyncState, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	pageIndex := 1

	var out []SubscriptionWithSyncState

	var customerID *filter.FilterULID
	if len(in.Customers) > 0 {
		customerID = &filter.FilterULID{FilterString: filter.FilterString{In: &in.Customers}}
	}

	for {
		subscriptions, err := r.subscriptionService.List(ctx, subscription.ListSubscriptionsInput{
			Namespaces: in.Namespaces,
			CustomerID: customerID,
			// Keep deleted subscriptions exclusively in the cleanup query.
			DeletedAt: &filter.FilterTime{Exists: lo.ToPtr(false)},
			ActiveInPeriod: &timeutil.StartBoundedPeriod{
				From: clock.Now().Add(-in.Lookback),
				To:   lo.ToPtr(clock.Now()),
			},
			Page: pagination.Page{
				PageNumber: pageIndex,
				PageSize:   defaultWindowSize,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list subscriptions: %w", err)
		}

		if len(subscriptions.Items) == 0 {
			break
		}

		mapped, err := r.mapToSubscriptionWithSyncState(ctx, subscriptions.Items)
		if err != nil {
			return nil, fmt.Errorf("failed to map subscriptions to subscription with sync state: %w", err)
		}

		out = append(out, mapped...)

		pageIndex++
	}

	deletedInput := subscriptionsync.ListDeletedSubscriptionsToReconcileInput{
		Namespaces: in.Namespaces,
		Customers:  in.Customers,
		Limit:      defaultWindowSize,
	}
	for {
		page, err := r.subscriptionSync.ListDeletedSubscriptionsToReconcile(ctx, deletedInput)
		if err != nil {
			return nil, fmt.Errorf("listing deleted subscriptions awaiting reconciliation: %w", err)
		}

		if len(page.Items) == 0 {
			break
		}

		// Keep subscription hydration in the subscription service. The sync
		// adapter selects only IDs so its public contract stays independent of
		// the subscription package, which imports generated Ent code.
		subscriptionIDs := lo.Map(page.Items, func(id models.NamespacedID, _ int) string { return id.ID })
		subscriptions, err := r.subscriptionService.List(ctx, subscription.ListSubscriptionsInput{
			Namespaces:     in.Namespaces,
			ID:             &filter.FilterULID{FilterString: filter.FilterString{In: &subscriptionIDs}},
			IncludeDeleted: true,
			Page: pagination.Page{
				PageNumber: 1,
				PageSize:   len(page.Items),
			},
		})
		if err != nil {
			return nil, fmt.Errorf("loading deleted subscriptions awaiting reconciliation: %w", err)
		}

		mapped, err := r.mapToSubscriptionWithSyncState(ctx, subscriptions.Items)
		if err != nil {
			return nil, fmt.Errorf("loading deleted subscription sync states: %w", err)
		}

		out = append(out, mapped...)

		if page.NextCursor == nil {
			break
		}

		deletedInput.Cursor = page.NextCursor
	}

	return out, nil
}

func (r *Reconciler) mapToSubscriptionWithSyncState(ctx context.Context, subs []subscription.Subscription) ([]SubscriptionWithSyncState, error) {
	deletedIDs := lo.FilterMap(subs, func(item subscription.Subscription, _ int) (models.NamespacedID, bool) {
		return item.NamespacedID, item.DeletedAt != nil
	})
	legacyIDs := lo.FilterMap(subs, func(item subscription.Subscription, _ int) (models.NamespacedID, bool) {
		return item.NamespacedID, item.DeletedAt == nil
	})

	var syncStates []subscriptionsync.SyncState
	if len(legacyIDs) > 0 {
		var err error
		syncStates, err = r.subscriptionSync.GetSyncStates(ctx, legacyIDs)
		if err != nil {
			return nil, fmt.Errorf("getting legacy sync states: %w", err)
		}
	}

	var syncStatesV2 []subscriptionsync.SyncStateV2Managed
	if len(deletedIDs) > 0 {
		var err error
		syncStatesV2, err = r.subscriptionSync.GetSyncStatesV2(ctx, deletedIDs)
		if err != nil {
			return nil, fmt.Errorf("getting v2 sync states: %w", err)
		}
	}

	syncStatesBySubscriptionID := lo.SliceToMap(syncStates, func(state subscriptionsync.SyncState) (models.NamespacedID, subscriptionsync.SyncState) {
		return state.SubscriptionID, state
	})
	syncStatesV2BySubscriptionID := lo.SliceToMap(syncStatesV2, func(state subscriptionsync.SyncStateV2Managed) (models.NamespacedID, subscriptionsync.SyncStateV2Managed) {
		return state.SubscriptionID, state
	})

	return lo.Map(subs, func(item subscription.Subscription, _ int) SubscriptionWithSyncState {
		existingSyncState, ok := syncStatesBySubscriptionID[item.NamespacedID]

		var syncState *subscriptionsync.SyncState
		if ok {
			syncState = lo.ToPtr(existingSyncState)
		}

		var syncStateV2 *subscriptionsync.SyncStateV2Managed
		if state, ok := syncStatesV2BySubscriptionID[item.NamespacedID]; ok {
			syncStateV2 = lo.ToPtr(state)
		}

		return SubscriptionWithSyncState{
			Subscription: item,
			SyncState:    syncState,
			SyncStateV2:  syncStateV2,
		}
	}), nil
}

func (r *Reconciler) ReconcileSubscription(ctx context.Context, subsID models.NamespacedID) error {
	return r.subscriptionSync.SyncByID(ctx, subsID, time.Now())
}

type ReconcilerAllInput struct {
	ReconcilerListSubscriptionsInput
	Force bool
}

func (r *Reconciler) All(ctx context.Context, in ReconcilerAllInput) error {
	subscriptions, err := r.ListSubscriptions(ctx, in.ReconcilerListSubscriptionsInput)
	if err != nil {
		return fmt.Errorf("failed to list subscriptions: %w", err)
	}

	var outErr error
	for _, subscription := range subscriptions {
		if !in.Force && !subscription.ShouldReconcile(clock.Now()) {
			r.logger.InfoContext(ctx, "subscription does not need periodic reconciliation", "subscription_id", subscription.NamespacedID)
			continue
		}

		if err := r.ReconcileSubscription(ctx, subscription.NamespacedID); err != nil {
			r.logger.ErrorContext(ctx, "failed to reconcile subscription", "error", err)

			outErr = errors.Join(outErr, fmt.Errorf("failed to reconcile subscription: %w", err))
		}
	}

	return outErr
}
