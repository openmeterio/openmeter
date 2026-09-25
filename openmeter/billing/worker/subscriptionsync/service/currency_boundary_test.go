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
	// - subscription sync runs after the subscription was deleted
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
	subscriptionService := &subscriptionServiceSpy{view: view}
	subscriptionService.onList = func(call int) {
		if call == 2 {
			require.True(t, billingService.withLockCalled, "current state must be read under the billing lock")
		}
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

	err := service.SyncByView(t.Context(), view, time.Now())

	require.NoError(t, err)
	require.True(t, billingService.withLockCalled)
	require.Equal(t, 2, subscriptionService.listCalls)
	require.True(t, reconciler.planCalled)
	require.Len(t, syncAdapter.upserts, 1)
	require.False(t, syncAdapter.upserts[0].HasBillables)
}

type billingServiceSpy struct {
	billing.Service
	withLockCalled bool
}

func (s *billingServiceSpy) WithLock(ctx context.Context, _ customer.CustomerID, fn func(context.Context) error) error {
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
	view      subscription.SubscriptionView
	listCalls int
	onList    func(int)
}

func (s *subscriptionServiceSpy) List(context.Context, subscription.ListSubscriptionsInput) (subscription.SubscriptionList, error) {
	s.listCalls++
	if s.onList != nil {
		s.onList(s.listCalls)
	}
	return subscription.SubscriptionList{Items: []subscription.Subscription{s.view.Subscription}}, nil
}

type subscriptionSyncAdapterSpy struct {
	subscriptionsync.Adapter
	upserts []subscriptionsync.UpsertSyncStateInput
}

func (s *subscriptionSyncAdapterSpy) UpsertSyncState(_ context.Context, input subscriptionsync.UpsertSyncStateInput) error {
	s.upserts = append(s.upserts, input)
	return nil
}

type syncReconcilerSpy struct {
	syncreconciler.Reconciler
	planCalled bool
}

func (s *syncReconcilerSpy) Plan(context.Context, syncreconciler.PlanInput) (*syncreconciler.Plan, error) {
	s.planCalled = true
	return &syncreconciler.Plan{}, nil
}
