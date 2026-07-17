package service

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	syncreconciler "github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestCustomCurrencySubscriptionBillingBoundary(t *testing.T) {
	view := newCustomCurrencySubscriptionView()
	syncAdapter := &subscriptionSyncAdapterSpy{}
	service := &Service{
		subscriptionService:     &subscriptionServiceSpy{view: view},
		subscriptionSyncAdapter: syncAdapter,
		logger:                  testutils.NewDiscardLogger(t),
		tracer:                  noop.NewTracerProvider().Tracer("test"),
	}

	t.Run("explicit sync fails before billing mutations", func(t *testing.T) {
		// given:
		// - a subscription with a priced custom-currency item
		// when:
		// - an explicit caller requests billing sync
		// then:
		// - sync fails with a conflict before requiring any billing dependency
		err := service.SyncByView(t.Context(), view, time.Now())

		require.ErrorIs(t, err, subscriptionsync.ErrCustomCurrencyBillingNotSupported)
		require.True(t, models.IsGenericConflictError(err))
	})

	t.Run("automatic sync skips the whole subscription", func(t *testing.T) {
		// given:
		// - a subscription with a priced custom-currency item
		// when:
		// - an automatic caller requests sync and customer invoicing
		// then:
		// - the subscription is ignored and customer invoicing is not attempted
		err := service.SyncByViewAndInvoiceCustomer(
			t.Context(),
			view,
			time.Now(),
			subscriptionsync.SkipCustomCurrencySubscriptions(),
		)

		require.NoError(t, err)
		require.Empty(t, syncAdapter.upserts)
	})

	t.Run("explicit sync rejects a stale fiat view when current state is custom", func(t *testing.T) {
		// given:
		// - an event-carried fiat-only view older than the persisted custom-currency state
		// when:
		// - an explicit caller synchronizes the stale view
		// then:
		// - persisted state keeps the custom-currency boundary from being bypassed
		staleView := newFiatCurrencySubscriptionView()
		err := service.SyncByView(t.Context(), staleView, time.Now())

		require.ErrorIs(t, err, subscriptionsync.ErrCustomCurrencyBillingNotSupported)
		require.True(t, models.IsGenericConflictError(err))
		require.Empty(t, syncAdapter.upserts)
	})

	t.Run("automatic sync skips a stale fiat view when current state is custom", func(t *testing.T) {
		// given:
		// - an event-carried fiat-only view older than the persisted custom-currency state
		// when:
		// - automatic reconciliation synchronizes the stale view
		// then:
		// - the whole subscription is skipped without billing mutations
		staleView := newFiatCurrencySubscriptionView()
		err := service.SyncByViewAndInvoiceCustomer(
			t.Context(),
			staleView,
			time.Now(),
			subscriptionsync.SkipCustomCurrencySubscriptions(),
		)

		require.NoError(t, err)
		require.Empty(t, syncAdapter.upserts)
	})

	t.Run("scheduled sync event opts into skipping", func(t *testing.T) {
		// given:
		// - an automatic subscription-sync event for a custom-currency subscription
		// when:
		// - the event handler loads and synchronizes the current view
		// then:
		// - it skips without surfacing an error or mutating sync state
		event := subscription.NewSubscriptionSyncEvent(t.Context(), view.Subscription)

		err := service.HandleSubscriptionSyncEvent(t.Context(), &event)

		require.NoError(t, err)
		require.Empty(t, syncAdapter.upserts)
	})

	t.Run("deleted subscriptions still reconcile", func(t *testing.T) {
		// given:
		// - a deleted subscription whose last view contained a custom-currency item
		// when:
		// - automatic deletion cleanup runs with custom-currency skipping enabled
		// then:
		// - the deleted entity bypasses the guard and reaches persisted-state reconciliation
		deletedAt := time.Now()
		deletedView := view
		deletedView.Subscription.DeletedAt = &deletedAt
		deletedView.Subscription.InvoiceCurrency = currencyx.Code("USD")

		billingService := &billingServiceSpy{}
		syncAdapter := &subscriptionSyncAdapterSpy{}
		reconciler := &syncReconcilerSpy{}
		service := &Service{
			billingService:          billingService,
			reconciler:              reconciler,
			subscriptionService:     &subscriptionServiceSpy{view: deletedView},
			subscriptionSyncAdapter: syncAdapter,
			logger:                  testutils.NewDiscardLogger(t),
			tracer:                  noop.NewTracerProvider().Tracer("test"),
		}

		err := service.SyncByView(
			t.Context(),
			deletedView,
			time.Now(),
			subscriptionsync.SkipCustomCurrencySubscriptions(),
		)

		require.NoError(t, err)
		require.True(t, billingService.withLockCalled)
		require.True(t, reconciler.planCalled)
		require.Len(t, syncAdapter.upserts, 1)
		require.False(t, syncAdapter.upserts[0].HasBillables)
	})
}

func newCustomCurrencySubscriptionView() subscription.SubscriptionView {
	customCurrency := &currencies.Currency{
		NamespacedID: models.NamespacedID{
			Namespace: "namespace",
			ID:        "custom-currency-id",
		},
		Code: "CREDITS",
		Name: "Credits",
	}
	rateCard := &productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:      "custom-fee",
			Name:     "Custom fee",
			Currency: customCurrency,
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount: alpacadecimal.NewFromInt(1),
			}),
		},
	}

	subs := subscription.Subscription{
		NamespacedID: models.NamespacedID{
			Namespace: "namespace",
			ID:        "subscription-id",
		},
		CustomerId: "customer-id",
	}

	return subscription.SubscriptionView{
		Subscription: subs,
		Spec: subscription.SubscriptionSpec{
			Phases: map[string]*subscription.SubscriptionPhaseSpec{
				"phase": {
					ItemsByKey: map[string][]*subscription.SubscriptionItemSpec{
						"custom-fee": {
							{
								CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
									CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
										PhaseKey: "phase",
										ItemKey:  "custom-fee",
										RateCard: rateCard,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func newFiatCurrencySubscriptionView() subscription.SubscriptionView {
	view := newCustomCurrencySubscriptionView()
	view.Subscription.InvoiceCurrency = currencyx.Code("USD")
	view.Spec.InvoiceCurrency = currencyx.Code("USD")

	rateCard := view.Spec.Phases["phase"].ItemsByKey["custom-fee"][0].RateCard.(*productcatalog.FlatFeeRateCard)
	rateCard.RateCardMeta.Currency = currencyx.Code("USD")

	return view
}

type billingServiceSpy struct {
	billing.Service
	withLockCalled bool
}

func (s *billingServiceSpy) WithLock(ctx context.Context, _ customer.CustomerID, fn func(context.Context) error) error {
	s.withLockCalled = true
	return fn(ctx)
}

func (s *billingServiceSpy) GetLinesForSubscription(context.Context, billing.GetLinesForSubscriptionInput) ([]billing.LineOrHierarchy, error) {
	return nil, nil
}

type subscriptionServiceSpy struct {
	subscription.Service
	view subscription.SubscriptionView
}

func (s *subscriptionServiceSpy) List(context.Context, subscription.ListSubscriptionsInput) (subscription.SubscriptionList, error) {
	return subscription.SubscriptionList{
		Items: []subscription.Subscription{s.view.Subscription},
	}, nil
}

func (s *subscriptionServiceSpy) GetView(context.Context, models.NamespacedID) (subscription.SubscriptionView, error) {
	return s.view, nil
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
