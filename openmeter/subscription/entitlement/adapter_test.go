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

func TestGetForItemsAtReturnsEntitlementsDeletedAfterAt(t *testing.T) {
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

	// when: an edit deletes their entitlements after viewAt, before the entitlements are read
	clock.FreezeTime(viewAt.Add(time.Minute))
	for _, item := range entItems {
		require.NoError(t, deps.EntitlementAdapter.DeleteByItemID(subscription.NewSubscriptionOperationContext(ctx), item.NamespacedID))
	}

	clock.FreezeTime(viewAt.Add(2 * time.Minute))

	ents, err := deps.EntitlementAdapter.GetForItemsAt(ctx, subscription.GetForItemsAtInput{
		Items: items,
		At:    viewAt,
	})

	// then: the entitlements are still returned as they were at viewAt
	require.NoError(t, err)
	require.ElementsMatch(t,
		lo.Map(entItems, func(i subscription.SubscriptionItem, _ int) string { return *i.EntitlementID }),
		lo.Map(ents, func(e subscription.SubscriptionEntitlement, _ int) string { return e.Entitlement.ID }),
	)
}
