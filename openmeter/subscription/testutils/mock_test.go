package subscriptiontestutils_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestMockServiceUpdateForwardsOptions(t *testing.T) {
	effectiveAt := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	updateCalled := false

	service := subscriptiontestutils.MockService{
		UpdateFn: func(_ context.Context, _ models.NamespacedID, _ subscription.SubscriptionSpec, options ...subscription.UpdateOption) (subscription.Subscription, error) {
			updateCalled = true

			updateOptions := subscription.UpdateOptions{}
			for _, option := range options {
				option(&updateOptions)
			}

			require.Equal(t, effectiveAt, updateOptions.CostBasisEffectiveAt)

			return subscription.Subscription{}, nil
		},
	}

	_, err := service.Update(
		t.Context(),
		models.NamespacedID{},
		subscription.SubscriptionSpec{},
		subscription.WithCostBasisEffectiveAt(effectiveAt),
	)
	require.NoError(t, err)
	require.True(t, updateCalled)
}
