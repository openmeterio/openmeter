package fixtures_test

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/subscription/testutils/fixtures"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func TestScenarioAdjacentSubscriptions(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2025-01-01T00:00:00Z")
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	t.Cleanup(func() { dbDeps.Cleanup(t) })
	deps := subscriptiontestutils.NewService(t, dbDeps)
	scenario := fixtures.New(t, deps)

	// given: one subscription with persisted default customer, plan, and entitlement
	first := scenario.CreateSubscription(fixtures.CreateSubscriptionInput{})
	cust := scenario.Customer()
	feat := scenario.Feature()
	p := scenario.Plan()
	require.Equal(t, cust.ID, scenario.Customer().ID)
	require.Equal(t, feat.ID, scenario.Feature().ID)
	require.Equal(t, p.ToCreateSubscriptionPlanInput().Plan, scenario.Plan().ToCreateSubscriptionPlanInput().Plan)

	// when: explicitly ending the first subscription before creating another for that customer
	canceled, err := deps.SubscriptionService.Cancel(t.Context(), first.Subscription.NamespacedID,
		subscription.Timing{Enum: lo.ToPtr(subscription.TimingNextBillingCycle)})
	require.NoError(t, err)
	require.NotNil(t, canceled.ActiveTo)
	second := scenario.CreateSubscription(fixtures.CreateSubscriptionInput{
		Change: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{Custom: canceled.ActiveTo},
			Name:   "Scheduled successor",
		},
	})

	// then: the subscriptions are distinct and share persisted prerequisites without overlapping
	firstRead, err := deps.SubscriptionService.GetView(t.Context(), first.Subscription.NamespacedID)
	require.NoError(t, err)
	secondRead, err := deps.SubscriptionService.GetView(t.Context(), second.Subscription.NamespacedID)
	require.NoError(t, err)
	require.NotEqual(t, firstRead.Subscription.ID, secondRead.Subscription.ID)
	require.Equal(t, cust.ID, firstRead.Customer.ID)
	require.Equal(t, cust.ID, secondRead.Customer.ID)
	require.Equal(t, firstRead.Subscription.PlanRef, secondRead.Subscription.PlanRef)
	require.NotNil(t, firstRead.Subscription.ActiveTo)
	require.Equal(t, *firstRead.Subscription.ActiveTo, secondRead.Subscription.ActiveFrom)
	require.Equal(t, "Scheduled successor", secondRead.Subscription.Name)
	require.Equal(t, secondRead.Subscription.ActiveFrom, secondRead.Subscription.BillingAnchor)
	requireBooleanEntitlement(t, firstRead, feat.ID)
	requireBooleanEntitlement(t, secondRead, feat.ID)
}

func TestScenarioExplicitPrerequisites(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2025-01-01T00:00:00Z")
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	t.Cleanup(func() { dbDeps.Cleanup(t) })
	deps := subscriptiontestutils.NewService(t, dbDeps)
	scenario := fixtures.New(t, deps)

	// given: a test-authored feature and published plan, plus an explicit customer
	feat, err := deps.FeatureConnector.CreateFeature(t.Context(), feature.CreateFeatureInputs{
		Namespace: subscriptiontestutils.ExampleNamespace,
		Key:       "explicit-feature",
		Name:      "Explicit feature",
	})
	require.NoError(t, err)
	planInput := subscriptiontestutils.BuildTestPlanInput(t).AddPhase(nil, &productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:                 feat.Key,
			Name:                "Explicit access",
			Feature:             productcatalog.NewFeatureReference(&feat.ID, &feat.Key),
			EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.BooleanEntitlementTemplate{}),
		},
		BillingCadence: lo.ToPtr(subscriptiontestutils.ISOMonth),
	}).Build()
	planInput.Plan.Key = "explicit-plan"
	planInput.Plan.Name = "Explicit plan"
	p := scenario.CreatePlan(planInput)
	cust := scenario.CreateCustomer(customer.CustomerMutate{
		Name: "Explicit customer",
		UsageAttribution: &customer.CustomerUsageAttribution{
			SubjectKeys: []string{"explicit-subject"},
		},
	})

	// when: passing both prerequisites directly instead of resolving the default graph
	created := scenario.CreateSubscription(fixtures.CreateSubscriptionInput{
		Customer: cust,
		Plan:     p,
		Change: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Name:          "Explicit subscription",
			BillingAnchor: &now,
		},
	})

	// then: persisted state uses the supplied graph, with no extra default resources
	view, err := deps.SubscriptionService.GetView(t.Context(), created.Subscription.NamespacedID)
	require.NoError(t, err)
	require.Equal(t, cust.ID, view.Customer.ID)
	require.Equal(t, "Explicit customer", view.Customer.Name)
	require.Equal(t, []string{"explicit-subject"}, view.Customer.UsageAttribution.SubjectKeys)
	require.Equal(t, p.ToCreateSubscriptionPlanInput().Plan, view.Subscription.PlanRef)
	require.Equal(t, "Explicit subscription", view.Subscription.Name)
	require.Equal(t, now, view.Subscription.BillingAnchor)
	requireBooleanEntitlement(t, view, feat.ID)

	customers, err := deps.CustomerService.ListCustomers(t.Context(), customer.ListCustomersInput{
		Namespace: subscriptiontestutils.ExampleNamespace,
		Page:      pagination.NewPage(1, 10),
	})
	require.NoError(t, err)
	require.Equal(t, 1, customers.TotalCount)
	plans, err := deps.PlanService.ListPlans(t.Context(), plan.ListPlansInput{
		Namespaces: []string{subscriptiontestutils.ExampleNamespace},
		Page:       pagination.NewPage(1, 10),
	})
	require.NoError(t, err)
	require.Equal(t, 1, plans.TotalCount)
	features, err := deps.FeatureConnector.ListFeatures(t.Context(), feature.ListFeaturesParams{
		Namespace: subscriptiontestutils.ExampleNamespace,
		Page:      pagination.NewPage(1, 10),
	})
	require.NoError(t, err)
	require.Equal(t, 1, features.TotalCount)
}

func TestScenarioIndependentDefaults(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2025-01-01T00:00:00Z")
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	t.Cleanup(func() { dbDeps.Cleanup(t) })
	deps := subscriptiontestutils.NewService(t, dbDeps)

	// given: two fixture scenarios sharing the same services, database, and namespace
	firstScenario := fixtures.New(t, deps)
	secondScenario := fixtures.New(t, deps)

	// when: each scenario resolves its default subscription graph
	first := firstScenario.CreateSubscription(fixtures.CreateSubscriptionInput{})
	second := secondScenario.CreateSubscription(fixtures.CreateSubscriptionInput{})

	// then: persisted defaults are distinct, while each scenario keeps its own cached identities
	firstRead, err := deps.SubscriptionService.GetView(t.Context(), first.Subscription.NamespacedID)
	require.NoError(t, err)
	secondRead, err := deps.SubscriptionService.GetView(t.Context(), second.Subscription.NamespacedID)
	require.NoError(t, err)
	require.Equal(t, firstRead.Subscription.Namespace, secondRead.Subscription.Namespace)
	require.NotEqual(t, firstRead.Subscription.ID, secondRead.Subscription.ID)
	require.NotEqual(t, firstRead.Customer.ID, secondRead.Customer.ID)
	require.NotEqual(t, firstRead.Customer.UsageAttribution.SubjectKeys, secondRead.Customer.UsageAttribution.SubjectKeys)
	require.NotNil(t, firstRead.Subscription.PlanRef)
	require.NotNil(t, secondRead.Subscription.PlanRef)
	require.NotEqual(t, firstRead.Subscription.PlanRef.Id, secondRead.Subscription.PlanRef.Id)
	require.NotEqual(t, firstRead.Subscription.PlanRef.Key, secondRead.Subscription.PlanRef.Key)
	require.Equal(t, firstRead.Customer.ID, firstScenario.Customer().ID)
	require.Equal(t, secondRead.Customer.ID, secondScenario.Customer().ID)
	require.NotEqual(t, firstScenario.Feature().ID, secondScenario.Feature().ID)
	requireBooleanEntitlement(t, firstRead, firstScenario.Feature().ID)
	requireBooleanEntitlement(t, secondRead, secondScenario.Feature().ID)
}

func requireBooleanEntitlement(t *testing.T, view subscription.SubscriptionView, featureID string) {
	t.Helper()
	require.NoError(t, view.Validate(true))
	require.Len(t, view.Phases, 1)
	require.Len(t, view.Phases[0].ItemsByKey, 1)
	for _, items := range view.Phases[0].ItemsByKey {
		require.Len(t, items, 1)
		item := items[0]
		require.NotNil(t, item.Feature)
		require.Equal(t, featureID, item.Feature.ID)
		require.NotNil(t, item.Entitlement)
		require.Equal(t, entitlement.EntitlementTypeBoolean, item.Entitlement.Entitlement.EntitlementType)
		require.Equal(t, featureID, item.Entitlement.Entitlement.FeatureID)
		require.Equal(t, view.Customer.ID, item.Entitlement.Entitlement.Customer.ID)
		require.Equal(t, view.Subscription.ID, item.Entitlement.Entitlement.Annotations[subscription.AnnotationSubscriptionID])
		require.Equal(t, &item.Entitlement.Entitlement.ID, item.SubscriptionItem.EntitlementID)
		require.Equal(t, view.Subscription.ActiveTo, item.Entitlement.Entitlement.ActiveTo)
	}
}
