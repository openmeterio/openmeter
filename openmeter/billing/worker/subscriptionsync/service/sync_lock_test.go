package service

import (
	"context"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/invoicing/legacy/splitlinegroup"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	syncreconciler "github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestSyncReloadsDeletionAfterBillingLock(t *testing.T) {
	// given: a sync reads an active subscription, then deletion cleanup finishes
	// before this sync acquires the customer's billing lock
	now := time.Now()
	beforeLockRevision := ulid.Make().String()
	lockedRevision := ulid.Make().String()
	laterRevision := ulid.Make().String()
	subscriptionService := &subscriptionServiceSpy{view: subscription.SubscriptionView{
		Subscription: subscription.Subscription{
			NamespacedID:    models.NamespacedID{Namespace: "namespace", ID: "subscription-id"},
			CustomerId:      "customer-id",
			InvoiceCurrency: currencyx.Code("USD"),
		},
	}}
	syncAdapter := &subscriptionSyncAdapterSpy{
		states: []subscriptionsync.SyncStateManaged{{
			SyncState: subscriptionsync.SyncState{
				LastSyncedSubsRevision: beforeLockRevision,
				CurrentSubsRevision:    beforeLockRevision,
			},
		}},
	}
	billingService := &billingServiceSpy{
		beforeLock: func() {
			subscriptionService.view.Subscription.DeletedAt = &now
			syncAdapter.states[0].CurrentSubsRevision = lockedRevision
		},
	}
	reconciler := &syncReconcilerSpy{
		onPlan: func() {
			syncAdapter.states[0].CurrentSubsRevision = laterRevision
		},
	}
	service := &Service{
		billingService:          billingService,
		splitLineGroupService:   &splitLineGroupServiceSpy{},
		reconciler:              reconciler,
		subscriptionService:     subscriptionService,
		subscriptionSyncAdapter: syncAdapter,
		logger:                  testutils.NewDiscardLogger(t),
		tracer:                  noop.NewTracerProvider().Tracer("test"),
	}

	// when: the waiting sync resumes
	err := service.SyncByID(t.Context(), subscriptionService.view.Subscription.NamespacedID, now, subscriptionsync.WithInvoicePendingLines())

	// then: it uses the deleted root and cannot recreate lines from an active view
	require.NoError(t, err)
	require.Zero(t, subscriptionService.getViewCalls)
	require.True(t, billingService.withLockCalled)
	require.Zero(t, billingService.invoicePendingLinesCalls)
	require.True(t, reconciler.planCalled)
	require.Empty(t, reconciler.planInput.Target.Items)
	require.Len(t, syncAdapter.recordedResults, 1)
	require.Equal(t, subscriptionsync.SyncStateStateSubscriptionDeleted, syncAdapter.recordedResults[0].LastSuccessState)
	// Acknowledge what was read after locking, not a revision changed during planning.
	require.Equal(t, lockedRevision, syncAdapter.recordedResults[0].LastSyncedSubsRevision)
	require.Equal(t, 1, syncAdapter.getStatesCalls)
}

func TestOldSubscriptionSkipUsesStateAfterBillingLock(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	// Given a subscription end and revision change committed while sync waits for the billing lock.
	end := now.Add(-31 * 24 * time.Hour)
	observedRevision := ulid.Make().String()
	subscriptionService := &subscriptionServiceSpy{view: subscription.SubscriptionView{
		Subscription: subscription.Subscription{
			NamespacedID: models.NamespacedID{Namespace: "namespace", ID: "subscription-id"},
			CustomerId:   "customer-id",
		},
	}}
	syncAdapter := &subscriptionSyncAdapterSpy{
		states: []subscriptionsync.SyncStateManaged{{
			SyncState: subscriptionsync.SyncState{CurrentSubsRevision: ulid.Make().String()},
		}},
	}
	billingService := &billingServiceSpy{
		beforeLock: func() {
			subscriptionService.view.Subscription.ActiveTo = &end
			syncAdapter.states[0].CurrentSubsRevision = observedRevision
		},
	}
	splitLineGroupService := &splitLineGroupServiceSpy{}
	reconciler := &syncReconcilerSpy{}
	service := &Service{
		billingService:          billingService,
		splitLineGroupService:   splitLineGroupService,
		reconciler:              reconciler,
		subscriptionService:     subscriptionService,
		subscriptionSyncAdapter: syncAdapter,
		logger:                  testutils.NewDiscardLogger(t),
		tracer:                  noop.NewTracerProvider().Tracer("test"),
	}

	// When sync resumes, it uses the reloaded end to skip old billing work.
	err := service.SyncByID(t.Context(), subscriptionService.view.Subscription.NamespacedID, now, subscriptionsync.WithInvoicePendingLines())
	require.NoError(t, err)

	// Then only the observed revision's skipped result is written, without loading billing artifacts.
	require.True(t, billingService.withLockCalled)
	require.Zero(t, subscriptionService.getViewCalls)
	require.Zero(t, billingService.getLinesCalls)
	require.Zero(t, splitLineGroupService.getLinesCalls)
	require.Zero(t, billingService.invoicePendingLinesCalls)
	require.False(t, reconciler.planCalled)
	require.Len(t, syncAdapter.recordedResults, 1)
	require.Equal(t, subscriptionsync.SyncStateStateTooOld, syncAdapter.recordedResults[0].LastSuccessState)
	require.Equal(t, observedRevision, syncAdapter.recordedResults[0].LastSyncedSubsRevision)
}

func TestSyncEventSkipsSubscriptionSyncedSinceSelection(t *testing.T) {
	selectedAt := time.Now().Add(-time.Hour)
	// An end this old takes the cheapest sync path: it only records the skipped
	// result, which shows whether the sync went ahead.
	end := time.Now().Add(-31 * 24 * time.Hour)

	for _, tc := range []struct {
		name          string
		lastSuccessAt time.Time
		changedSince  bool
		// skipIfSyncedSince is what the event carries.
		skipIfSyncedSince *time.Time
		wantSynced        bool
	}{
		{
			name:              "another sync succeeded after selection",
			lastSuccessAt:     selectedAt.Add(time.Minute),
			skipIfSyncedSince: &selectedAt,
		},
		{
			name:              "the subscription changed after that sync",
			lastSuccessAt:     selectedAt.Add(time.Minute),
			changedSince:      true,
			skipIfSyncedSince: &selectedAt,
			wantSynced:        true,
		},
		{
			name:              "the last sync predates selection",
			lastSuccessAt:     selectedAt.Add(-time.Minute),
			skipIfSyncedSince: &selectedAt,
			wantSynced:        true,
		},
		{
			name:          "the event carries no selection time",
			lastSuccessAt: selectedAt.Add(time.Minute),
			wantSynced:    true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given: a subscription with a recorded sync
			sub := subscription.Subscription{
				NamespacedID: models.NamespacedID{Namespace: "namespace", ID: "subscription-id"},
				CustomerId:   "customer-id",
			}
			sub.ActiveTo = &end
			state := subscriptionsync.SyncState{
				LastSuccessState:       subscriptionsync.SyncStateStateTooOld,
				LastSuccessAt:          &tc.lastSuccessAt,
				LastSyncedSubsRevision: ulid.Make().String(),
			}
			state.CurrentSubsRevision = state.LastSyncedSubsRevision
			if tc.changedSince {
				state.CurrentSubsRevision = ulid.Make().String()
			}

			syncAdapter := &subscriptionSyncAdapterSpy{states: []subscriptionsync.SyncStateManaged{{SyncState: state}}}
			billingService := &billingServiceSpy{}
			service := &Service{
				billingService:          billingService,
				splitLineGroupService:   &splitLineGroupServiceSpy{},
				reconciler:              &syncReconcilerSpy{},
				subscriptionService:     &subscriptionServiceSpy{view: subscription.SubscriptionView{Subscription: sub}},
				subscriptionSyncAdapter: syncAdapter,
				logger:                  testutils.NewDiscardLogger(t),
				tracer:                  noop.NewTracerProvider().Tracer("test"),
			}

			// when: the periodic sync event for it is handled
			err := service.HandleSubscriptionSyncEvent(t.Context(), &subscription.SubscriptionSyncEvent{
				Subscription:      sub,
				SkipIfSyncedSince: tc.skipIfSyncedSince,
			})

			// then: it is synced unless another sync already covered this selection
			require.NoError(t, err)
			require.True(t, billingService.withLockCalled, "the decision is made under the billing lock")
			if tc.wantSynced {
				require.Len(t, syncAdapter.recordedResults, 1)
			} else {
				require.Empty(t, syncAdapter.recordedResults)
			}
		})
	}
}

type billingServiceSpy struct {
	billing.Service
	beforeLock               func()
	withLockCalled           bool
	invoicePendingLinesCalls int
	getLinesCalls            int
}

func (s *billingServiceSpy) GetCustomerBillingProfile(ctx context.Context, input billing.GetCustomerBillingProfileInput) (billing.CustomerBillingProfile, error) {
	if s.Service != nil {
		return s.Service.GetCustomerBillingProfile(ctx, input)
	}

	return billing.CustomerBillingProfile{Customer: &customer.Customer{}}, nil
}

func (s *billingServiceSpy) InvoicePendingLines(ctx context.Context, input billing.InvoicePendingLinesInput, opts ...billing.InvoicePendingLinesOption) ([]billing.StandardInvoice, error) {
	s.invoicePendingLinesCalls++
	return s.Service.InvoicePendingLines(ctx, input, opts...)
}

func (s *billingServiceSpy) WithLock(ctx context.Context, customerID customer.CustomerID, fn func(context.Context) error) error {
	if s.beforeLock != nil {
		s.beforeLock()
	}

	s.withLockCalled = true
	if s.Service != nil {
		return s.Service.WithLock(ctx, customerID, fn)
	}

	return fn(ctx)
}

func (s *billingServiceSpy) GetStandardLinesForSubscription(context.Context, billing.GetLinesForSubscriptionInput) (billing.StandardLines, error) {
	s.getLinesCalls++
	return nil, nil
}

func (s *billingServiceSpy) GetGatheringLinesForSubscription(context.Context, billing.GetLinesForSubscriptionInput) (billing.GatheringLines, error) {
	s.getLinesCalls++
	return nil, nil
}

type splitLineGroupServiceSpy struct {
	splitlinegroup.Service
	getLinesCalls int
}

func (s *splitLineGroupServiceSpy) GetSplitLineGroupsForSubscription(context.Context, splitlinegroup.GetSplitLineGroupsForSubscriptionInput) ([]splitlinegroup.SplitLineHierarchy, error) {
	s.getLinesCalls++
	return nil, nil
}

type subscriptionServiceSpy struct {
	subscription.Service
	view         subscription.SubscriptionView
	getViewCalls int
}

func (s *subscriptionServiceSpy) List(context.Context, subscription.ListSubscriptionsInput) (subscription.SubscriptionList, error) {
	return subscription.SubscriptionList{Items: []subscription.Subscription{s.view.Subscription}}, nil
}

func (s *subscriptionServiceSpy) GetView(context.Context, models.NamespacedID) (subscription.SubscriptionView, error) {
	s.getViewCalls++
	return s.view, nil
}

type subscriptionSyncAdapterSpy struct {
	subscriptionsync.Adapter
	recordedResults []subscriptionsync.RecordSyncResultInput
	states          []subscriptionsync.SyncStateManaged
	getStatesCalls  int
}

func (s *subscriptionSyncAdapterSpy) GetSyncStates(context.Context, subscriptionsync.GetSyncStatesInput) ([]subscriptionsync.SyncStateManaged, error) {
	s.getStatesCalls++
	return s.states, nil
}

func (s *subscriptionSyncAdapterSpy) RecordSyncResult(_ context.Context, input subscriptionsync.RecordSyncResultInput) error {
	s.recordedResults = append(s.recordedResults, input)
	return nil
}

type syncReconcilerSpy struct {
	syncreconciler.Reconciler
	planCalled bool
	planInput  syncreconciler.PlanInput
	onPlan     func()
}

func (s *syncReconcilerSpy) Plan(_ context.Context, input syncreconciler.PlanInput) (*syncreconciler.Plan, error) {
	s.planCalled = true
	s.planInput = input
	if s.onPlan != nil {
		s.onPlan()
	}

	return &syncreconciler.Plan{}, nil
}
