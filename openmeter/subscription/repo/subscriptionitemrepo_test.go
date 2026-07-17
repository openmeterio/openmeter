package repo_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestSubscriptionItemCustomCurrencyPersistence(t *testing.T) {
	// given:
	// - a custom-currency plan whose priced item snapshots the managed currency
	// - an existing fiat subscription phase used only as the persistence owner
	// when:
	// - the materialized item is written and read through the item repository
	// then:
	// - the managed custom-currency identity survives without enabling custom-currency subscriptions
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	require.NotNil(t, dbDeps)
	defer dbDeps.Cleanup(t)

	deps := subscriptiontestutils.NewService(t, dbDeps)
	customer := deps.CustomerAdapter.CreateExampleCustomer(t)

	fiatPlanInput := subscriptiontestutils.BuildTestPlanInput(t).
		AddPhase(nil, subscriptiontestutils.ExampleRateCard2.Clone()).
		Build()
	fiatPlanInput.Key = "subscription-item-currency-persistence-owner"
	fiatPlan := deps.PlanHelper.CreatePlan(t, fiatPlanInput)

	fiatView, err := deps.WorkflowService.CreateFromPlan(t.Context(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{Custom: &now},
		},
		CustomerID: customer.ID,
		Namespace:  subscriptiontestutils.ExampleNamespace,
	}, fiatPlan)
	require.NoError(t, err)

	ownerItems := fiatView.Phases[0].ItemsByKey[subscriptiontestutils.ExampleRateCard2.Key()]
	require.Len(t, ownerItems, 1)
	unmaterializedInput := ownerItems[0].SubscriptionItem.AsEntityInput()
	unmaterializedInput.Key = "unmaterialized-priced-rate-card"
	unmaterializedInput.RateCard = unmaterializedInput.RateCard.Clone()
	require.NoError(t, unmaterializedInput.RateCard.ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
		meta.Key = unmaterializedInput.Key
		meta.Currency = nil
		return meta, nil
	}))
	_, err = deps.ItemRepo.Create(t.Context(), unmaterializedInput)
	require.ErrorContains(t, err, "priced subscription item currency must be materialized")

	managedCurrency, err := deps.CurrencyService.CreateCurrency(t.Context(), currencies.CreateCurrencyInput{
		Namespace: subscriptiontestutils.ExampleNamespace,
		Code:      "CREDITS",
		Name:      "Credits",
		Symbol:    "cr",
	})
	require.NoError(t, err)

	const customRateCardKey = "custom-currency-rate-card"
	customRateCard := subscriptiontestutils.ExampleRateCard2.Clone()
	require.NoError(t, customRateCard.ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
		meta.Key = customRateCardKey
		meta.Name = "Custom currency rate card"
		return meta, nil
	}))

	customCurrency := currencyx.Code(managedCurrency.Code)
	customPlanInput := subscriptiontestutils.BuildTestPlanInput(t).
		AddPhase(nil, customRateCard).
		Build()
	customPlanInput.Key = "custom-currency-item-materialization"
	customPlanInput.Currency = customCurrency
	customPlan := deps.PlanHelper.CreatePlan(t, customPlanInput)

	customSpec, err := subscription.NewSpecFromPlan(customPlan, subscription.CreateSubscriptionCustomerInput{
		CustomerId:      customer.ID,
		InvoiceCurrency: currencyx.Code("USD"),
		ActiveFrom:      now,
		BillingAnchor:   now,
		Name:            customPlan.GetName(),
	})
	require.NoError(t, err)
	require.Equal(t, currencyx.Code("USD"), customSpec.InvoiceCurrency)

	customPhases := customSpec.GetSortedPhases()
	require.Len(t, customPhases, 1)
	customItems := customPhases[0].ItemsByKey[customRateCardKey]
	require.Len(t, customItems, 1)

	materializedCurrency := customItems[0].RateCard.AsMeta().Currency
	require.NotNil(t, materializedCurrency)
	require.Equal(t, customCurrency, materializedCurrency.GetCode())
	materializedManagedCurrency, ok := materializedCurrency.(currencyx.ManagedCurrency)
	require.True(t, ok)
	require.Equal(t, managedCurrency.ID, materializedManagedCurrency.GetID())

	createInput := ownerItems[0].SubscriptionItem.AsEntityInput()
	createInput.Key = customRateCardKey
	createInput.RateCard = customItems[0].RateCard
	createInput.Name = customItems[0].RateCard.AsMeta().Name
	createInput.Description = customItems[0].RateCard.AsMeta().Description
	createInput.EntitlementID = nil

	created, err := deps.ItemRepo.Create(t.Context(), createInput)
	require.NoError(t, err)

	reloaded, err := deps.ItemRepo.GetByID(t.Context(), created.NamespacedID)
	require.NoError(t, err)

	reloadedCurrency := reloaded.RateCard.AsMeta().Currency
	require.NotNil(t, reloadedCurrency)
	require.Equal(t, customCurrency, reloadedCurrency.GetCode())
	reloadedManagedCurrency, ok := reloadedCurrency.(currencyx.ManagedCurrency)
	require.True(t, ok)
	require.Equal(t, managedCurrency.ID, reloadedManagedCurrency.GetID())
}
