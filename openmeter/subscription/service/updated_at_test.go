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
	"github.com/openmeterio/openmeter/pkg/models"
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

	// given: two subscriptions written before a sync checkpoint.
	createSubscription("unchanged")
	changed := createSubscription("changed")

	// when: one of them is updated at the checkpoint.
	clock.FreezeTime(checkpoint)
	defer clock.UnFreeze()

	_, err := service.UpdateAnnotations(ctx, changed.NamespacedID, models.Annotations{"changed": true})
	require.NoError(t, err)

	// then: only the updated subscription matches. Its created_at is before the
	// checkpoint, so this can only match on updated_at.
	result, err := service.List(ctx, subscription.ListSubscriptionsInput{
		Namespaces: []string{subscriptiontestutils.ExampleNamespace},
		Page:       pagination.NewPage(1, 10),
		UpdatedAt:  &filter.FilterTime{Gte: &checkpoint},
	})
	require.NoError(t, err)
	require.Equal(t, []string{changed.ID}, lo.Map(result.Items, func(s subscription.Subscription, _ int) string { return s.ID }))
}
