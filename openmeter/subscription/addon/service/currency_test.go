package service_test

import (
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func TestAddonServiceCreateRejectsCustomEffectiveCurrency(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2025-04-01T00:00:00Z")

	tests := []struct {
		name           string
		defaultCustom  bool
		overrideCustom bool
		expectedError  error
	}{
		{
			name: "fiat add-on remains supported",
		},
		{
			name:          "custom add-on default is rejected",
			defaultCustom: true,
			expectedError: subscription.ErrCustomCurrencySubscriptionsNotSupported,
		},
		{
			name:           "custom rate card override is rejected",
			overrideCustom: true,
			expectedError:  subscription.ErrCustomCurrencySubscriptionsNotSupported,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
				// given:
				// - a USD plan with a published, compatible add-on
				// - the add-on is fiat-only or has a valid custom effective currency
				clock.SetTime(now)
				defer clock.ResetTime()

				customCurrency := createCustomCurrencyWithUSDCostBasis(t, deps, "CREDITS")
				rateCardCurrency := (*currencies.CurrencyReference)(nil)
				if tt.overrideCustom {
					rateCardCurrency = lo.ToPtr(currencies.NewCurrencyReference(customCurrency.GetCode()))
				}

				addonInput := newBillableAddonInput(t, now, rateCardCurrency)
				if tt.defaultCustom {
					addonInput.Currency = currencies.NewCurrencyReference(customCurrency.GetCode())
				}

				subscriptionID, add := createSubscriptionForPlanWithAddon(t, deps, now, addonInput)
				input := subscriptionaddon.CreateSubscriptionAddonInput{
					AddonID:        add.ID,
					SubscriptionID: subscriptionID,
					InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
						ActiveFrom: now,
						Quantity:   1,
					},
				}

				// when:
				// - the add-on is attached to the subscription
				created, err := deps.SubscriptionAddonService.Create(t.Context(), subscriptiontestutils.ExampleNamespace, input)

				// then:
				// - fiat pricing remains supported
				// - custom effective currencies fail before an attachment is persisted
				if tt.expectedError == nil {
					require.NoError(t, err)
					require.NotNil(t, created)
					return
				}

				require.ErrorIs(t, err, tt.expectedError)
				require.Nil(t, created)

				result, listErr := deps.SubscriptionAddonService.List(t.Context(), subscriptiontestutils.ExampleNamespace, subscriptionaddon.ListSubscriptionAddonsInput{
					SubscriptionID: subscriptionID,
					Page:           pagination.NewPage(1, 100),
				})
				require.NoError(t, listErr)
				require.Empty(t, result.Items)
			})
		})
	}
}

func TestAddonServiceChangeQuantityRejectsCustomEffectiveCurrency(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2025-04-01T00:00:00Z")

	withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
		// given:
		// - an existing subscription add-on created while its currency is fiat
		// - its persisted add-on definition now represents a legacy custom-currency attachment
		clock.SetTime(now)
		defer clock.ResetTime()

		customCurrency := createCustomCurrencyWithUSDCostBasis(t, deps, "CREDITS")
		subscriptionID, add := createSubscriptionForPlanWithAddon(t, deps, now, newBillableAddonInput(t, now, nil))

		subscriptionAddon, err := deps.SubscriptionAddonService.Create(t.Context(), subscriptiontestutils.ExampleNamespace, subscriptionaddon.CreateSubscriptionAddonInput{
			AddonID:        add.ID,
			SubscriptionID: subscriptionID,
			InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
				ActiveFrom: now,
				Quantity:   1,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, subscriptionAddon)

		_, err = deps.DBDeps.PGDriver.DB().ExecContext(t.Context(), `
			UPDATE addons
			SET currency = $1, custom_currency_id = $2
			WHERE id = $3
		`, customCurrency.GetCode(), customCurrency.ID, add.ID)
		require.NoError(t, err)

		// when:
		// - a quantity change would rematerialize the custom-priced add-on
		updated, err := deps.SubscriptionAddonService.ChangeQuantity(t.Context(), subscriptionAddon.NamespacedID, subscriptionaddon.CreateSubscriptionAddonQuantityInput{
			ActiveFrom: now.Add(24 * time.Hour),
			Quantity:   1,
		})

		// then:
		// - the temporary custom-currency boundary rejects the mutation
		// - no quantity entry is added
		require.ErrorIs(t, err, subscription.ErrCustomCurrencySubscriptionsNotSupported)
		require.Nil(t, updated)

		stored, getErr := deps.SubscriptionAddonService.Get(t.Context(), subscriptionaddon.GetSubscriptionAddonInput{
			NamespacedID: subscriptionAddon.NamespacedID,
		})
		require.NoError(t, getErr)
		require.Len(t, stored.Quantities.GetTimes(), len(subscriptionAddon.Quantities.GetTimes()))
	})
}

func createCustomCurrencyWithUSDCostBasis(
	t *testing.T,
	deps subscriptiontestutils.SubscriptionDependencies,
	code currencyx.Code,
) currencies.Currency {
	t.Helper()

	customCurrency, err := deps.CurrencyService.CreateCurrency(t.Context(), currenciestestutils.NewCreateCurrencyInput(
		subscriptiontestutils.ExampleNamespace,
		code,
		code.String(),
		"cr",
	))
	require.NoError(t, err)

	_, err = deps.CurrencyService.CreateCostBasis(t.Context(), currencies.CreateCostBasisInput{
		Namespace:  subscriptiontestutils.ExampleNamespace,
		CurrencyID: customCurrency.ID,
		FiatCode:   "USD",
		Rate:       decimal.NewFromInt(1),
	})
	require.NoError(t, err)

	return customCurrency
}

func newBillableAddonInput(
	t *testing.T,
	now time.Time,
	currency *currencies.CurrencyReference,
) addon.CreateAddonInput {
	t.Helper()

	rateCard := subscriptiontestutils.ExampleAddonRateCard4.Clone()
	err := rateCard.ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
		meta.Key = "subscription-currency-boundary"
		meta.Name = "Subscription currency boundary"
		meta.FeatureKey = nil
		meta.FeatureID = nil
		meta.Currency = currency

		return meta, nil
	})
	require.NoError(t, err)

	input := subscriptiontestutils.BuildAddonForTesting(
		t,
		productcatalog.EffectivePeriod{EffectiveFrom: lo.ToPtr(now)},
		productcatalog.AddonInstanceTypeSingle,
		rateCard,
	)
	input.Key = "subscription-currency-boundary"

	return input
}

func createSubscriptionForPlanWithAddon(
	t *testing.T,
	deps subscriptiontestutils.SubscriptionDependencies,
	now time.Time,
	addonInput addon.CreateAddonInput,
) (string, addon.Addon) {
	t.Helper()

	p, add := createPlanWithAddon(
		t,
		deps,
		subscriptiontestutils.GetExamplePlanInput(t),
		addonInput,
	)

	customer := deps.CustomerAdapter.CreateExampleCustomer(t)
	spec, err := subscription.NewSpecFromPlan(p, subscription.CreateSubscriptionCustomerInput{
		CustomerId:    customer.ID,
		Currency:      "USD",
		ActiveFrom:    now,
		BillingAnchor: now,
		Name:          "Test Subscription",
	})
	require.NoError(t, err)

	sub, err := deps.SubscriptionService.Create(t.Context(), subscriptiontestutils.ExampleNamespace, spec)
	require.NoError(t, err)
	require.NotNil(t, sub)

	return sub.ID, add
}
