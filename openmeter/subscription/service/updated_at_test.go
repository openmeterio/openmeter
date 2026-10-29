package service_test

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func TestListUpdatedAt(t *testing.T) {
	ctx := t.Context()
	checkpoint := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	clock.FreezeTime(checkpoint.Add(-time.Hour))
	defer clock.UnFreeze()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	defer dbDeps.Cleanup(t)

	deps := subscriptiontestutils.NewService(t, dbDeps)
	service := deps.SubscriptionService

	_ = deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)
	plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

	createSubscription := func(subjectKey string) subscription.Subscription {
		cust, err := deps.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
			Namespace: subscriptiontestutils.ExampleNamespace,
			CustomerMutate: customer.CustomerMutate{
				Name:             subjectKey,
				UsageAttribution: &customer.CustomerUsageAttribution{SubjectKeys: []string{subjectKey}},
			},
		})
		require.NoError(t, err)

		spec, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
			CustomerId:      cust.ID,
			InvoiceCurrency: "USD",
			ActiveFrom:      clock.Now(),
			BillingAnchor:   clock.Now(),
			Name:            "Test Subscription",
		})
		require.NoError(t, err)

		sub, err := service.Create(ctx, subscriptiontestutils.ExampleNamespace, spec)
		require.NoError(t, err)

		return sub
	}

	list := func(updatedAt filter.FilterTime) pagination.Result[subscription.Subscription] {
		result, err := service.List(ctx, subscription.ListSubscriptionsInput{
			Namespaces: []string{subscriptiontestutils.ExampleNamespace},
			Page:       pagination.NewPage(1, 1),
			UpdatedAt:  &updatedAt,
		})
		require.NoError(t, err)

		return result
	}

	ids := func(result pagination.Result[subscription.Subscription]) []string {
		return lo.Map(result.Items, func(s subscription.Subscription, _ int) string { return s.ID })
	}

	// given: a subscription written before a sync checkpoint and one written at it.
	before := createSubscription("before-checkpoint")

	clock.FreezeTime(checkpoint)
	defer clock.UnFreeze()
	atCheckpoint := createSubscription("at-checkpoint")

	// when/then: gte includes the boundary, gt excludes it, and lt selects the
	// earlier subscription; page totals count only the filtered rows.
	gte := list(filter.FilterTime{Gte: &checkpoint})
	require.Equal(t, []string{atCheckpoint.ID}, ids(gte))
	require.Equal(t, 1, gte.TotalCount)

	require.Empty(t, list(filter.FilterTime{Gt: &checkpoint}).Items)

	lt := list(filter.FilterTime{Lt: &checkpoint})
	require.Equal(t, []string{before.ID}, ids(lt))
	require.Equal(t, 1, lt.TotalCount)
}
