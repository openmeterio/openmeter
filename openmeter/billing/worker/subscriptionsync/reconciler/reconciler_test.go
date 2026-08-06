package reconciler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestReconcileSubscriptionSkipsUnsupportedCustomCurrencies(t *testing.T) {
	// given:
	// - periodic reconciliation delegates to subscription sync
	// when:
	// - a subscription is reconciled
	// then:
	// - it opts into the temporary custom-currency skip behavior
	spy := &subscriptionSyncSpy{}
	reconciler := Reconciler{subscriptionSync: spy}

	err := reconciler.ReconcileSubscription(t.Context(), models.NamespacedID{
		Namespace: "namespace",
		ID:        "subscription-id",
	})

	require.NoError(t, err)
	require.True(t, spy.options.SkipCustomCurrencySubscriptions)
}

type subscriptionSyncSpy struct {
	subscriptionsync.Service
	options subscriptionsync.SynchronizeSubscriptionOptions
}

func (s *subscriptionSyncSpy) SyncByID(_ context.Context, _ models.NamespacedID, _ time.Time, opts ...subscriptionsync.SynchronizeSubscriptionOption) error {
	for _, opt := range opts {
		opt(&s.options)
	}

	return nil
}
