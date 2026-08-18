package subscriptions

import (
	"testing"

	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestToAPIBillingSubscriptionCurrencySemantics(t *testing.T) {
	// given:
	// - a pinned subscription with a materialized invoice fiat and cost-basis pin
	sub := subscription.Subscription{
		InvoiceCurrency: currencyx.Code("USD"),
		CostBasisMode:   subscription.CostBasisModePinned,
		CostBasisPins: []subscription.CostBasisPin{
			{
				CustomCurrencyID: "currency_credits",
				InvoiceCurrency:  currencyx.Code("USD"),
				CostBasis: currencies.CostBasis{
					NamespacedID: models.NamespacedID{ID: "cost_basis_credits_usd"},
				},
			},
		},
	}

	// when:
	// - the subscription's own fields are mapped to the v3 API
	result := subscriptionBaseFields(sub, clock.Now())

	// then:
	// - the invoice currency, mode, and pin resource identities are preserved
	require.Equal(t, "USD", result.InvoiceCurrency)
	require.Equal(t, "pinned", string(result.CostBasisMode))
	require.Equal(t, "currency_credits", result.CostBasisPins[0].CustomCurrencyId)
	require.Equal(t, "USD", result.CostBasisPins[0].InvoiceCurrency)
	require.Equal(t, "cost_basis_credits_usd", result.CostBasisPins[0].CostBasisId)
}

func TestFromAPIBillingSubscriptionCreateCostBasisMode(t *testing.T) {
	// given:
	// - a v3 create request explicitly selecting pinned cost bases
	mode := api.BillingSubscriptionCostBasisMode("pinned")
	request := api.BillingSubscriptionCreate{CostBasisMode: &mode}

	// when:
	// - the request is mapped to the subscription workflow input
	result, err := FromAPIBillingSubscriptionCreate(
		"default",
		customer.CustomerID{ID: "customer_1", Namespace: "default"},
		"Subscription",
		request,
	)

	// then:
	// - the mode survives the API boundary
	require.NoError(t, err)
	require.Equal(t, subscription.CostBasisModePinned, result.CostBasisMode)
}

// TestToAPIBillingSubscriptionViewRoundtrip exercises the full view -> API conversion
// against a real subscription created through the service. Unlike the scalar-field
// test above, this asserts the spec-driven parts the converter resolves from the view
func TestToAPIBillingSubscriptionViewRoundtrip(t *testing.T) {
	ctx := t.Context()

	currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:11Z")
	clock.FreezeTime(currentTime)
	defer clock.UnFreeze()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	defer dbDeps.Cleanup(t)

	deps := subscriptiontestutils.NewService(t, dbDeps)
	service := deps.SubscriptionService

	// given: a subscription created from the example plan, which has phases and rate cards
	cust := deps.CustomerAdapter.CreateExampleCustomer(t)
	_ = deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)
	plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

	spec, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
		CustomerId:      cust.ID,
		InvoiceCurrency: "USD",
		ActiveFrom:      currentTime,
		BillingAnchor:   currentTime,
		Name:            "Test Subscription",
		Annotations:     models.Annotations{},
	})
	require.NoError(t, err)

	sub, err := service.Create(ctx, subscriptiontestutils.ExampleNamespace, spec)
	require.NoError(t, err)

	view, err := service.GetView(ctx, models.NamespacedID{ID: sub.ID, Namespace: sub.Namespace})
	require.NoError(t, err)

	// when: the view is mapped to the v3 API subscription
	result, err := ToAPIBillingSubscription(view)
	require.NoError(t, err)

	// then: the scalar fields survive
	require.Equal(t, sub.ID, result.Id)
	require.Equal(t, cust.ID, result.CustomerId)
	require.Equal(t, "USD", result.InvoiceCurrency)

	// then: the phases roundtrip, each carrying resolved items with rate cards
	require.NotEmpty(t, result.Phases, "expected phases on the expanded subscription")

	foundItem := false
	for _, phase := range result.Phases {
		require.NotEmpty(t, phase.Key)
		require.False(t, phase.ActiveFrom.IsZero())

		for _, item := range phase.Items {
			require.NotEmpty(t, item.RateCard.Key, "resolved item should carry a rate card")
			require.False(t, item.ActiveFrom.IsZero())
			foundItem = true
		}
	}
	require.True(t, foundItem, "expected at least one resolved item across phases")

	// then: the current aligned billing period is populated for the active subscription
	require.NotNil(t, result.CurrentPeriod, "active subscription should expose a current period")
	require.True(t, result.CurrentPeriod.To.After(result.CurrentPeriod.From))
}
