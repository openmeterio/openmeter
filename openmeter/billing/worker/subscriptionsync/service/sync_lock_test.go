package service

import (
	"context"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/billing"
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
		statesV2: []subscriptionsync.SyncStateV2Managed{{
			SyncStateV2: subscriptionsync.SyncStateV2{
				LastSyncedSubsRevision: beforeLockRevision,
				CurrentSubsRevision:    beforeLockRevision,
			},
		}},
	}
	billingService := &billingServiceSpy{
		beforeLock: func() {
			subscriptionService.view.Subscription.DeletedAt = &now
			syncAdapter.statesV2[0].CurrentSubsRevision = lockedRevision
		},
	}
	reconciler := &syncReconcilerSpy{
		onPlan: func() {
			syncAdapter.statesV2[0].CurrentSubsRevision = laterRevision
		},
	}
	service := &Service{
		billingService:          billingService,
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
	require.Len(t, syncAdapter.recordedResultsV2, 1)
	require.Equal(t, subscriptionsync.SyncStateV2StateSubscriptionDeleted, syncAdapter.recordedResultsV2[0].LastSuccessState)
	// Acknowledge what was read after locking, not a revision changed during planning.
	require.Equal(t, lockedRevision, syncAdapter.recordedResultsV2[0].LastSyncedSubsRevision)
	require.Equal(t, 1, syncAdapter.getStatesV2Calls)
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
		statesV2: []subscriptionsync.SyncStateV2Managed{{
			SyncStateV2: subscriptionsync.SyncStateV2{CurrentSubsRevision: ulid.Make().String()},
		}},
	}
	billingService := &billingServiceSpy{
		beforeLock: func() {
			subscriptionService.view.Subscription.ActiveTo = &end
			syncAdapter.statesV2[0].CurrentSubsRevision = observedRevision
		},
	}
	reconciler := &syncReconcilerSpy{}
	service := &Service{
		billingService:          billingService,
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
	require.Zero(t, billingService.invoicePendingLinesCalls)
	require.False(t, reconciler.planCalled)
	require.Len(t, syncAdapter.recordedResultsV2, 1)
	require.Equal(t, subscriptionsync.SyncStateV2StateTooOld, syncAdapter.recordedResultsV2[0].LastSuccessState)
	require.Equal(t, observedRevision, syncAdapter.recordedResultsV2[0].LastSyncedSubsRevision)
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

func (s *billingServiceSpy) GetSplitLineGroupsForSubscription(context.Context, billing.GetLinesForSubscriptionInput) ([]billing.SplitLineHierarchy, error) {
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
	recordedResultsV2 []subscriptionsync.RecordSyncResultV2Input
	statesV2          []subscriptionsync.SyncStateV2Managed
	getStatesV2Calls  int
}

func (s *subscriptionSyncAdapterSpy) GetSyncStatesV2(context.Context, subscriptionsync.GetSyncStatesV2Input) ([]subscriptionsync.SyncStateV2Managed, error) {
	s.getStatesV2Calls++
	return s.statesV2, nil
}

func (s *subscriptionSyncAdapterSpy) RecordSyncResultV2(_ context.Context, input subscriptionsync.RecordSyncResultV2Input) error {
	s.recordedResultsV2 = append(s.recordedResultsV2, input)
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
