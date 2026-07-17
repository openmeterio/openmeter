package service_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestSubscriptionCreationPersistsCostBasisPinsByMode(t *testing.T) {
	tests := []struct {
		name         string
		mode         subscription.CostBasisMode
		expectedPins int
	}{
		{
			name:         "pinned stores one deduplicated pair",
			mode:         subscription.CostBasisModePinned,
			expectedPins: 1,
		},
		{
			name:         "dynamic stores no pins",
			mode:         subscription.CostBasisModeDynamic,
			expectedPins: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given:
			// - two priced items sharing one custom-currency to invoice-fiat pair
			// when:
			// - a subscription is created in pinned or dynamic mode
			// then:
			// - only pinned mode stores the exact pair, and it stores it once
			currentTime := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
			clock.FreezeTime(currentTime)
			defer clock.UnFreeze()

			dbDeps := subscriptiontestutils.SetupDBDeps(t)
			defer dbDeps.Cleanup(t)
			deps := subscriptiontestutils.NewService(t, dbDeps)

			customer := deps.CustomerAdapter.CreateExampleCustomer(t)
			customCurrency, err := deps.CurrencyService.CreateCurrency(t.Context(), currencies.CreateCurrencyInput{
				Namespace: subscriptiontestutils.ExampleNamespace,
				Code:      "CREDITS",
				Name:      "Credits",
				Symbol:    "CR",
			})
			require.NoError(t, err)

			costBasisID := ulid.Make().String()
			_, err = dbDeps.DBClient.CurrencyCostBasis.Create().
				SetID(costBasisID).
				SetNamespace(subscriptiontestutils.ExampleNamespace).
				SetCurrencyID(customCurrency.ID).
				SetFiatCode(currencyx.Code("USD")).
				SetRate(alpacadecimal.NewFromFloat(0.5)).
				SetEffectiveFrom(currentTime.Add(-time.Hour)).
				Save(t.Context())
			require.NoError(t, err)

			planInput := subscriptiontestutils.BuildTestPlanInput(t).
				AddPhase(nil, newCostBasisTestRateCard(t, "fee-one", nil), newCostBasisTestRateCard(t, "fee-two", nil)).
				Build()
			planInput.Plan.Currency = customCurrency
			plan := deps.PlanHelper.CreatePlan(t, planInput)

			spec, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
				CustomerId:      customer.ID,
				InvoiceCurrency: currencyx.Code("USD"),
				CostBasisMode:   tt.mode,
				ActiveFrom:      currentTime,
				BillingAnchor:   currentTime,
				Name:            "Cost Basis Subscription",
				Annotations:     models.Annotations{},
			})
			require.NoError(t, err)

			created, err := deps.SubscriptionService.Create(t.Context(), subscriptiontestutils.ExampleNamespace, spec)
			require.NoError(t, err)
			require.Len(t, created.CostBasisPins, tt.expectedPins)

			loaded, err := deps.SubscriptionService.Get(t.Context(), created.NamespacedID)
			require.NoError(t, err)
			require.Len(t, loaded.CostBasisPins, tt.expectedPins)
			if tt.expectedPins == 0 {
				return
			}

			require.Equal(t, customCurrency.ID, loaded.CostBasisPins[0].CustomCurrencyID)
			require.Equal(t, currencyx.Code("USD"), loaded.CostBasisPins[0].InvoiceCurrency)
			require.Equal(t, costBasisID, loaded.CostBasisPins[0].CostBasis.ID)
		})
	}
}

func TestSubscriptionUpdatePinsNewCostBasisPairAtEffectiveTime(t *testing.T) {
	// given:
	// - a pinned subscription with one custom-currency pair
	// - a second custom currency whose cost basis changes exactly at edit time
	// when:
	// - an update introduces the second pair with that explicit effective time
	// then:
	// - the original pin is retained and only the newly effective cost basis is appended
	currentTime := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	editTime := currentTime.Add(time.Hour)
	clock.FreezeTime(currentTime)
	defer clock.UnFreeze()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	defer dbDeps.Cleanup(t)
	deps := subscriptiontestutils.NewService(t, dbDeps)

	customer := deps.CustomerAdapter.CreateExampleCustomer(t)
	creditsCurrency, err := deps.CurrencyService.CreateCurrency(t.Context(), currencies.CreateCurrencyInput{
		Namespace: subscriptiontestutils.ExampleNamespace,
		Code:      "CREDITS",
		Name:      "Credits",
		Symbol:    "CR",
	})
	require.NoError(t, err)
	pointsCurrency, err := deps.CurrencyService.CreateCurrency(t.Context(), currencies.CreateCurrencyInput{
		Namespace: subscriptiontestutils.ExampleNamespace,
		Code:      "POINTS",
		Name:      "Points",
		Symbol:    "PT",
	})
	require.NoError(t, err)

	creditsCostBasisID := ulid.Make().String()
	_, err = dbDeps.DBClient.CurrencyCostBasis.Create().
		SetID(creditsCostBasisID).
		SetNamespace(subscriptiontestutils.ExampleNamespace).
		SetCurrencyID(creditsCurrency.ID).
		SetFiatCode(currencyx.Code("USD")).
		SetRate(alpacadecimal.NewFromFloat(0.5)).
		SetEffectiveFrom(currentTime.Add(-time.Hour)).
		Save(t.Context())
	require.NoError(t, err)

	oldPointsCostBasisID := ulid.Make().String()
	_, err = dbDeps.DBClient.CurrencyCostBasis.Create().
		SetID(oldPointsCostBasisID).
		SetNamespace(subscriptiontestutils.ExampleNamespace).
		SetCurrencyID(pointsCurrency.ID).
		SetFiatCode(currencyx.Code("USD")).
		SetRate(alpacadecimal.NewFromFloat(0.25)).
		SetEffectiveFrom(currentTime.Add(-time.Hour)).
		SetEffectiveTo(editTime).
		Save(t.Context())
	require.NoError(t, err)

	newPointsCostBasisID := ulid.Make().String()
	_, err = dbDeps.DBClient.CurrencyCostBasis.Create().
		SetID(newPointsCostBasisID).
		SetNamespace(subscriptiontestutils.ExampleNamespace).
		SetCurrencyID(pointsCurrency.ID).
		SetFiatCode(currencyx.Code("USD")).
		SetRate(alpacadecimal.NewFromFloat(0.75)).
		SetEffectiveFrom(editTime).
		Save(t.Context())
	require.NoError(t, err)

	planInput := subscriptiontestutils.BuildTestPlanInput(t).
		AddPhase(nil, newCostBasisTestRateCard(t, "credits-fee", nil)).
		Build()
	planInput.Plan.Currency = creditsCurrency
	plan := deps.PlanHelper.CreatePlan(t, planInput)

	spec, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
		CustomerId:      customer.ID,
		InvoiceCurrency: currencyx.Code("USD"),
		CostBasisMode:   subscription.CostBasisModePinned,
		ActiveFrom:      currentTime,
		BillingAnchor:   currentTime,
		Name:            "Pinned Cost Basis Update",
		Annotations:     models.Annotations{},
	})
	require.NoError(t, err)

	created, err := deps.SubscriptionService.Create(t.Context(), subscriptiontestutils.ExampleNamespace, spec)
	require.NoError(t, err)
	require.Len(t, created.CostBasisPins, 1)
	require.Equal(t, creditsCostBasisID, created.CostBasisPins[0].CostBasis.ID)
	originalPinID := created.CostBasisPins[0].ID

	view, err := deps.SubscriptionService.GetView(t.Context(), created.NamespacedID)
	require.NoError(t, err)
	updatedSpec := view.AsSpec()
	phase := updatedSpec.GetSortedPhases()[0]
	phase.ItemsByKey["points-fee"] = []*subscription.SubscriptionItemSpec{
		{
			CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
				CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
					PhaseKey: phase.PhaseKey,
					ItemKey:  "points-fee",
					RateCard: newCostBasisTestRateCard(t, "points-fee", pointsCurrency),
				},
				Annotations: models.Annotations{},
			},
		},
	}

	_, err = deps.SubscriptionService.Update(
		t.Context(),
		created.NamespacedID,
		updatedSpec,
		subscription.WithCostBasisEffectiveAt(editTime),
	)
	require.NoError(t, err)

	loaded, err := deps.SubscriptionService.Get(t.Context(), created.NamespacedID)
	require.NoError(t, err)
	require.Len(t, loaded.CostBasisPins, 2)

	pinsByCustomCurrencyID := map[string]subscription.CostBasisPin{}
	for _, pin := range loaded.CostBasisPins {
		pinsByCustomCurrencyID[pin.CustomCurrencyID] = pin
	}
	require.Equal(t, originalPinID, pinsByCustomCurrencyID[creditsCurrency.ID].ID)
	require.Equal(t, creditsCostBasisID, pinsByCustomCurrencyID[creditsCurrency.ID].CostBasis.ID)
	require.Equal(t, newPointsCostBasisID, pinsByCustomCurrencyID[pointsCurrency.ID].CostBasis.ID)
	require.NotEqual(t, oldPointsCostBasisID, pinsByCustomCurrencyID[pointsCurrency.ID].CostBasis.ID)
}

func newCostBasisTestRateCard(t *testing.T, key string, currencyIdentity currencyx.CurrencyIdentity) productcatalog.RateCard {
	t.Helper()

	billingCadence := datetime.MustParseDuration(t, "P1M")
	return &productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:      key,
			Name:     key,
			Currency: currencyIdentity,
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromInt(10),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
		},
		BillingCadence: &billingCadence,
	}
}
