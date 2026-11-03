package subscriptionentitlement_test

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestGetForItemsAt(t *testing.T) {
	ctx := t.Context()

	viewAt := testutils.GetRFC3339Time(t, "2021-01-01T00:00:11Z")
	clock.FreezeTime(viewAt)
	defer clock.UnFreeze()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	defer dbDeps.Cleanup(t)

	deps := subscriptiontestutils.NewService(t, dbDeps)

	_ = deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)
	plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))
	cust := deps.CustomerAdapter.CreateExampleCustomer(t)

	spec, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
		CustomerId:      cust.ID,
		InvoiceCurrency: "USD",
		ActiveFrom:      viewAt,
		BillingAnchor:   viewAt,
		Name:            "Test Subscription",
		Annotations:     models.Annotations{},
	})
	require.NoError(t, err)

	sub, err := deps.SubscriptionService.Create(ctx, subscriptiontestutils.ExampleNamespace, spec)
	require.NoError(t, err)

	// given: items read for a view pinned at viewAt
	items, err := deps.ItemRepo.GetForSubscriptionsAt(ctx, []subscription.GetForSubscriptionAtInput{{
		Namespace:      sub.Namespace,
		SubscriptionID: sub.ID,
		At:             viewAt,
	}})
	require.NoError(t, err)

	entItems := lo.Filter(items, func(i subscription.SubscriptionItem, _ int) bool { return i.EntitlementID != nil })
	require.NotEmpty(t, entItems)

	entIDs := func(ents []subscription.SubscriptionEntitlement) []string {
		return lo.Map(ents, func(e subscription.SubscriptionEntitlement, _ int) string { return e.Entitlement.ID })
	}

	t.Run("returns only the given items' entitlements", func(t *testing.T) {
		// given: a subscription with several entitlement-bearing items
		require.Greater(t, len(entItems), 1)

		// when: resolving entitlements for just one of them
		ents, err := deps.EntitlementAdapter.GetForItemsAt(ctx, subscription.GetForItemsAtInput{
			Items: entItems[:1],
			At:    viewAt,
		})

		// then: the other items' entitlements are not pulled in
		require.NoError(t, err)
		require.Equal(t, []string{*entItems[0].EntitlementID}, entIDs(ents))
	})

	t.Run("returns entitlements deleted after At but not those deleted by At", func(t *testing.T) {
		require.Greater(t, len(entItems), 1)

		subCtx := subscription.NewSubscriptionOperationContext(ctx)

		// given: one entitlement deleted at viewAt, and an edit deleting the rest after viewAt
		require.NoError(t, deps.EntitlementAdapter.DeleteByItemID(subCtx, entItems[0].NamespacedID))

		clock.FreezeTime(viewAt.Add(time.Minute))
		defer clock.UnFreeze()
		for _, item := range entItems[1:] {
			require.NoError(t, deps.EntitlementAdapter.DeleteByItemID(subCtx, item.NamespacedID))
		}

		clock.FreezeTime(viewAt.Add(2 * time.Minute))
		defer clock.UnFreeze()

		// when: resolving them for the view pinned at viewAt
		ents, err := deps.EntitlementAdapter.GetForItemsAt(ctx, subscription.GetForItemsAtInput{
			Items: items,
			At:    viewAt,
		})

		// then: only the entitlements still live at viewAt are returned
		require.NoError(t, err)
		require.ElementsMatch(t,
			lo.Map(entItems[1:], func(i subscription.SubscriptionItem, _ int) string { return *i.EntitlementID }),
			entIDs(ents),
		)
	})
}
