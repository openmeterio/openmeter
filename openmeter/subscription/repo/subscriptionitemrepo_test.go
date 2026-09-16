package repo_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionrepo "github.com/openmeterio/openmeter/openmeter/subscription/repo"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/subscription/validators/itemreference"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
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
	fiatItemRow, err := dbDeps.DBClient.SubscriptionItem.Get(t.Context(), ownerItems[0].SubscriptionItem.ID)
	require.NoError(t, err)
	require.NotNil(t, fiatItemRow.Currency)
	require.Equal(t, "USD", *fiatItemRow.Currency)
	require.Nil(t, fiatItemRow.CustomCurrencyID)

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
		CurrencyDetails: currencyx.CurrencyDetails{
			Code:               "CREDITS",
			Name:               "Credits",
			Symbol:             "cr",
			Precision:          0,
			DecimalMark:        ".",
			ThousandsSeparator: ",",
		},
	})
	require.NoError(t, err)

	const customRateCardKey = "custom-currency-rate-card"
	customRateCard := subscriptiontestutils.ExampleRateCard2.Clone()
	require.NoError(t, customRateCard.ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
		meta.Key = customRateCardKey
		meta.Name = "Custom currency rate card"
		return meta, nil
	}))

	customCurrency := managedCurrency.GetCode()
	customPlanInput := subscriptiontestutils.BuildTestPlanInput(t).
		AddPhase(nil, customRateCard).
		Build()
	customPlanInput.Key = "custom-currency-item-materialization"
	customPlanInput.Currency = managedCurrency.Reference()
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
	require.NotNil(t, materializedCurrency.CustomCurrencyID)
	require.Equal(t, managedCurrency.ID, *materializedCurrency.CustomCurrencyID)

	createInput := ownerItems[0].SubscriptionItem.AsEntityInput()
	createInput.Key = customRateCardKey
	createInput.RateCard = customItems[0].RateCard
	createInput.Name = customItems[0].RateCard.AsMeta().Name
	createInput.Description = customItems[0].RateCard.AsMeta().Description
	createInput.EntitlementID = nil

	unresolvedInput := createInput
	unresolvedInput.RateCard = createInput.RateCard.Clone()
	unresolvedCurrencyID := managedCurrency.ID
	require.NoError(t, unresolvedInput.RateCard.ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
		meta.Currency = &currencies.CurrencyReference{
			Code:             managedCurrency.GetCode(),
			CustomCurrencyID: &unresolvedCurrencyID,
		}
		return meta, nil
	}))
	_, err = deps.ItemRepo.Create(t.Context(), unresolvedInput)
	require.ErrorContains(t, err, "must be resolved before persistence")

	foreignCurrency, err := deps.CurrencyService.CreateCurrency(t.Context(), currencies.CreateCurrencyInput{
		Namespace: "subscription-item-currency-persistence-other",
		CurrencyDetails: currencyx.CurrencyDetails{
			Code:               "TOKENS",
			Name:               "Tokens",
			Symbol:             "tok",
			Precision:          0,
			DecimalMark:        ".",
			ThousandsSeparator: ",",
		},
	})
	require.NoError(t, err)

	foreignInput := createInput
	foreignInput.RateCard = createInput.RateCard.Clone()
	foreignReference := foreignCurrency.Reference()
	require.NoError(t, foreignInput.RateCard.ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
		meta.Currency = &foreignReference
		return meta, nil
	}))
	_, err = deps.ItemRepo.Create(t.Context(), foreignInput)
	require.ErrorContains(t, err, "custom currency namespace mismatch")

	createInput.RateCard = createInput.RateCard.Clone()
	resolvedReference := managedCurrency.Reference()
	require.NoError(t, createInput.RateCard.ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
		meta.Currency = &resolvedReference
		return meta, nil
	}))

	created, err := deps.ItemRepo.Create(t.Context(), createInput)
	require.NoError(t, err)
	createdCurrency := created.RateCard.AsMeta().Currency
	require.NotNil(t, createdCurrency)
	require.Equal(t, customCurrency, createdCurrency.GetCode())
	require.NotNil(t, createdCurrency.CustomCurrencyID)
	require.Equal(t, managedCurrency.ID, *createdCurrency.CustomCurrencyID)
	require.True(t, createdCurrency.IsResolved())

	customItemRow, err := dbDeps.DBClient.SubscriptionItem.Get(t.Context(), created.ID)
	require.NoError(t, err)
	require.NotNil(t, customItemRow.Currency)
	require.Equal(t, customCurrency.String(), *customItemRow.Currency)
	require.NotNil(t, customItemRow.CustomCurrencyID)
	require.Equal(t, managedCurrency.ID, *customItemRow.CustomCurrencyID)

	reloaded, err := deps.ItemRepo.GetByID(t.Context(), created.NamespacedID)
	require.NoError(t, err)

	reloadedCurrency := reloaded.RateCard.AsMeta().Currency
	require.NotNil(t, reloadedCurrency)
	require.Equal(t, customCurrency, reloadedCurrency.GetCode())
	require.NotNil(t, reloadedCurrency.CustomCurrencyID)
	require.Equal(t, managedCurrency.ID, *reloadedCurrency.CustomCurrencyID)
	require.True(t, reloadedCurrency.IsResolved())
}

func TestValidateSubscriptionItemReference(t *testing.T) {
	// given:
	// - two materialized subscription item chains in the same namespace
	// when:
	// - structural references are validated across subscription, phase, and item boundaries
	// then:
	// - only one internally consistent chain is accepted, including after archival
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	require.NotNil(t, dbDeps)
	defer dbDeps.Cleanup(t)

	deps := subscriptiontestutils.NewService(t, dbDeps)
	validator, err := itemreference.NewValidator(subscriptionrepo.NewSubscriptionItemRepo(dbDeps.DBClient))
	require.NoError(t, err)
	planInput := subscriptiontestutils.BuildTestPlanInput(t).
		AddPhase(nil, subscriptiontestutils.ExampleRateCard2.Clone()).
		Build()
	planInput.Key = "subscription-item-reference-validation"
	plan := deps.PlanHelper.CreatePlan(t, planInput)

	firstCustomer := deps.CustomerAdapter.CreateExampleCustomer(t)
	firstView, err := deps.WorkflowService.CreateFromPlan(t.Context(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{Custom: &now},
		},
		CustomerID: firstCustomer.ID,
		Namespace:  subscriptiontestutils.ExampleNamespace,
	}, plan)
	require.NoError(t, err)

	secondCustomer := deps.CustomerAdapter.CreateExampleCustomerWithSubject(t, "Jane Doe", "jane-doe")
	secondView, err := deps.WorkflowService.CreateFromPlan(t.Context(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{Custom: &now},
		},
		CustomerID: secondCustomer.ID,
		Namespace:  subscriptiontestutils.ExampleNamespace,
	}, plan)
	require.NoError(t, err)

	firstItem := firstView.Phases[0].ItemsByKey[subscriptiontestutils.ExampleRateCard2.Key()][0].SubscriptionItem
	secondItem := secondView.Phases[0].ItemsByKey[subscriptiontestutils.ExampleRateCard2.Key()][0].SubscriptionItem
	validInput := itemreference.ValidateInput{
		Namespace:      subscriptiontestutils.ExampleNamespace,
		SubscriptionID: firstView.Subscription.ID,
		PhaseID:        firstView.Phases[0].SubscriptionPhase.ID,
		ItemID:         firstItem.ID,
	}

	require.NoError(t, validator.ValidateItemReference(t.Context(), validInput))

	invalidInputs := []itemreference.ValidateInput{
		{
			Namespace:      "other-namespace",
			SubscriptionID: validInput.SubscriptionID,
			PhaseID:        validInput.PhaseID,
			ItemID:         validInput.ItemID,
		},
		{
			Namespace:      validInput.Namespace,
			SubscriptionID: secondView.Subscription.ID,
			PhaseID:        validInput.PhaseID,
			ItemID:         validInput.ItemID,
		},
		{
			Namespace:      validInput.Namespace,
			SubscriptionID: validInput.SubscriptionID,
			PhaseID:        secondView.Phases[0].SubscriptionPhase.ID,
			ItemID:         validInput.ItemID,
		},
		{
			Namespace:      validInput.Namespace,
			SubscriptionID: validInput.SubscriptionID,
			PhaseID:        validInput.PhaseID,
			ItemID:         secondItem.ID,
		},
	}

	for _, input := range invalidInputs {
		err := validator.ValidateItemReference(t.Context(), input)
		require.Error(t, err)
		require.True(t, models.IsGenericPreConditionFailedError(err))
	}

	require.NoError(t, deps.ItemRepo.Delete(t.Context(), firstItem.NamespacedID))
	require.NoError(t, dbDeps.DBClient.SubscriptionPhase.UpdateOneID(validInput.PhaseID).SetDeletedAt(now).Exec(t.Context()))
	require.NoError(t, validator.ValidateItemReference(t.Context(), validInput))
}
