package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	syncreconciler "github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestDeletedSubscriptionReconcilesBillingArtifacts(t *testing.T) {
	// given:
	// - a deleted subscription whose billing artifacts still need cleanup
	// when:
	// - subscription sync reloads the deleted subscription by ID
	// then:
	// - it plans an empty target and records that no future billables remain
	deletedAt := time.Now()
	view := subscription.SubscriptionView{Subscription: subscription.Subscription{
		NamespacedID:    models.NamespacedID{Namespace: "namespace", ID: "subscription-id"},
		ManagedModel:    models.ManagedModel{DeletedAt: &deletedAt},
		CustomerId:      "customer-id",
		InvoiceCurrency: currencyx.Code("USD"),
	}}

	billingService := &billingServiceSpy{}
	syncAdapter := &subscriptionSyncAdapterSpy{}
	reconciler := &syncReconcilerSpy{}
	service := &Service{
		billingService:          billingService,
		reconciler:              reconciler,
		subscriptionService:     &subscriptionServiceSpy{view: view},
		subscriptionSyncAdapter: syncAdapter,
		logger:                  testutils.NewDiscardLogger(t),
		tracer:                  noop.NewTracerProvider().Tracer("test"),
	}

	err := service.SyncByID(t.Context(), view.Subscription.NamespacedID, time.Now(), subscriptionsync.WithInvoicePendingLines())

	require.NoError(t, err)
	require.True(t, billingService.withLockCalled)
	require.Zero(t, billingService.invoicePendingLinesCalls)
	require.True(t, reconciler.planCalled)
	require.Empty(t, reconciler.planInput.Target.Items)
	require.Len(t, syncAdapter.upsertsV2, 1)
	require.Equal(t, view.Subscription.NamespacedID, syncAdapter.upsertsV2[0].SubscriptionID)
	require.Equal(t, subscriptionsync.SyncStateV2StateSubscriptionDeleted, syncAdapter.upsertsV2[0].LastSuccessState)
	require.Len(t, syncAdapter.upserts, 1)
	require.False(t, syncAdapter.upserts[0].HasBillables)
}

func TestSyncReloadsDeletionAfterBillingLock(t *testing.T) {
	// given: a sync reads an active subscription, then deletion cleanup finishes
	// before this sync acquires the customer's billing lock
	now := time.Now()
	subscriptionService := &subscriptionServiceSpy{view: subscription.SubscriptionView{
		Subscription: subscription.Subscription{
			NamespacedID:    models.NamespacedID{Namespace: "namespace", ID: "subscription-id"},
			CustomerId:      "customer-id",
			InvoiceCurrency: currencyx.Code("USD"),
		},
	}}
	billingService := &billingServiceSpy{
		beforeLock: func() {
			subscriptionService.view.Subscription.DeletedAt = &now
		},
	}
	syncAdapter := &subscriptionSyncAdapterSpy{}
	reconciler := &syncReconcilerSpy{}
	service := &Service{
		billingService:          billingService,
		reconciler:              reconciler,
		subscriptionService:     subscriptionService,
		subscriptionSyncAdapter: syncAdapter,
		logger:                  testutils.NewDiscardLogger(t),
		tracer:                  noop.NewTracerProvider().Tracer("test"),
	}

	// when: the waiting sync resumes
	err := service.SyncByID(t.Context(), subscriptionService.view.Subscription.NamespacedID, now)

	// then: it uses the deleted root and cannot recreate lines from an active view
	require.NoError(t, err)
	require.Zero(t, subscriptionService.getViewCalls)
	require.True(t, reconciler.planCalled)
	require.Empty(t, reconciler.planInput.Target.Items)
	require.Len(t, syncAdapter.upsertsV2, 1)
}

type billingServiceSpy struct {
	billing.Service
	beforeLock               func()
	withLockCalled           bool
	invoicePendingLinesCalls int
}

func (s *billingServiceSpy) InvoicePendingLines(ctx context.Context, input billing.InvoicePendingLinesInput, opts ...billing.InvoicePendingLinesOption) ([]billing.StandardInvoice, error) {
	s.invoicePendingLinesCalls++
	return s.Service.InvoicePendingLines(ctx, input, opts...)
}

func (s *billingServiceSpy) WithLock(ctx context.Context, _ customer.CustomerID, fn func(context.Context) error) error {
	if s.beforeLock != nil {
		s.beforeLock()
	}

	s.withLockCalled = true

	return fn(ctx)
}

func (s *billingServiceSpy) GetStandardLinesForSubscription(context.Context, billing.GetLinesForSubscriptionInput) (billing.StandardLines, error) {
	return nil, nil
}

func (s *billingServiceSpy) GetGatheringLinesForSubscription(context.Context, billing.GetLinesForSubscriptionInput) (billing.GatheringLines, error) {
	return nil, nil
}

func (s *billingServiceSpy) GetSplitLineGroupsForSubscription(context.Context, billing.GetLinesForSubscriptionInput) ([]billing.SplitLineHierarchy, error) {
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
	upserts   []subscriptionsync.UpsertSyncStateInput
	upsertsV2 []subscriptionsync.UpsertSyncStateV2Input
}

func (s *subscriptionSyncAdapterSpy) UpsertSyncState(_ context.Context, input subscriptionsync.UpsertSyncStateInput) error {
	s.upserts = append(s.upserts, input)
	return nil
}

func (s *subscriptionSyncAdapterSpy) UpsertSyncStateV2(_ context.Context, input subscriptionsync.UpsertSyncStateV2Input) error {
	s.upsertsV2 = append(s.upsertsV2, input)
	return nil
}

type syncReconcilerSpy struct {
	syncreconciler.Reconciler
	planCalled bool
	planInput  syncreconciler.PlanInput
}

func (s *syncReconcilerSpy) Plan(_ context.Context, input syncreconciler.PlanInput) (*syncreconciler.Plan, error) {
	s.planCalled = true
	s.planInput = input

	return &syncreconciler.Plan{}, nil
}
