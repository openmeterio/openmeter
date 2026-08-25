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
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestAddonWorkflowSupportsCustomEffectiveCurrency(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2025-04-01T00:00:00Z")

	tests := []struct {
		name           string
		defaultCustom  bool
		overrideCustom bool
	}{
		{
			name:          "custom add-on default is materialized",
			defaultCustom: true,
		},
		{
			name:           "custom rate card override is materialized",
			overrideCustom: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
				// given:
				// - a USD plan with a published, compatible add-on
				// - the add-on has a valid custom effective currency
				clock.FreezeTime(now.Add(time.Millisecond))
				defer func() {
					clock.UnFreeze()
					clock.ResetTime()
				}()

				customCurrency := createCustomCurrencyWithUSDCostBasis(t, deps, "CREDITS")
				rateCardCurrency := (*currencies.CurrencyReference)(nil)
				if tt.overrideCustom {
					rateCardCurrency = lo.ToPtr(currencies.NewCurrencyReference(customCurrency.GetCode()))
				}

				addonInput := newBillableAddonInput(t, now, productcatalog.AddonInstanceTypeMultiple, rateCardCurrency)
				if tt.defaultCustom {
					addonInput.Currency = currencies.NewCurrencyReference(customCurrency.GetCode())
				}

				subscriptionID, add := createSubscriptionForPlanWithAddon(t, deps, now, addonInput)
				addonStart := clock.Now()

				// when:
				// - the add-on is attached to the subscription through the workflow
				view, subscriptionAddon, err := deps.WorkflowService.AddAddon(t.Context(), subscriptionID, subscriptionworkflow.AddAddonWorkflowInput{
					AddonID:         add.ID,
					InitialQuantity: 1,
					Timing: subscription.Timing{
						Custom: &addonStart,
					},
				})

				// then:
				// - the attachment succeeds
				// - every materialized add-on item retains the managed custom identity
				require.NoError(t, err)
				require.NotEmpty(t, subscriptionAddon.ID)
				assertCustomCurrencyOnAddonItems(t, view, customCurrency)
			})
		})
	}
}

func TestAddonWorkflowChangesCustomCurrencyAddonQuantity(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2025-04-01T00:00:00Z")

	withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
		// given:
		// - a custom-currency add-on attached to a USD subscription
		clock.FreezeTime(now.Add(time.Millisecond))
		defer func() {
			clock.UnFreeze()
			clock.ResetTime()
		}()

		customCurrency := createCustomCurrencyWithUSDCostBasis(t, deps, "CREDITS")
		addonInput := newBillableAddonInput(t, now, productcatalog.AddonInstanceTypeMultiple, nil)
		addonInput.Currency = currencies.NewCurrencyReference(customCurrency.GetCode())
		subscriptionID, add := createSubscriptionForPlanWithAddon(t, deps, now, addonInput)
		addonStart := clock.Now()

		view, subscriptionAddon, err := deps.WorkflowService.AddAddon(t.Context(), subscriptionID, subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add.ID,
			InitialQuantity: 1,
			Timing: subscription.Timing{
				Custom: &addonStart,
			},
		})
		require.NoError(t, err)
		assertCustomCurrencyOnAddonItems(t, view, customCurrency)

		// when:
		// - the add-on quantity changes through the workflow
		changeTime := now.Add(24 * time.Hour)
		view, subscriptionAddon, err = deps.WorkflowService.ChangeAddonQuantity(t.Context(), subscriptionID, subscriptionworkflow.ChangeAddonQuantityWorkflowInput{
			SubscriptionAddonID: subscriptionAddon.NamespacedID,
			Quantity:            2,
			Timing: subscription.Timing{
				Custom: lo.ToPtr(changeTime),
			},
		})

		// then:
		// - the quantity history is extended
		// - the custom identity remains on the resulting subscription items
		require.NoError(t, err)
		require.Len(t, subscriptionAddon.Quantities.GetTimes(), 2)
		require.Equal(t, 2, subscriptionAddon.Quantities.GetAt(1).GetValue().Quantity)
		assertCustomCurrencyOnAddonItems(t, view, customCurrency)
	})
}

func assertCustomCurrencyOnAddonItems(t *testing.T, view subscription.SubscriptionView, customCurrency currencies.Currency) {
	t.Helper()

	itemCount := 0
	for _, phase := range view.Phases {
		for _, item := range phase.ItemsByKey["subscription-currency-boundary"] {
			itemCount++
			reference := item.Spec.RateCard.AsMeta().Currency
			require.NotNil(t, reference)
			require.Equal(t, customCurrency.GetCode(), reference.GetCode())
			require.Equal(t, lo.ToPtr(customCurrency.ID), reference.CustomCurrencyID)
		}
	}

	require.NotZero(t, itemCount)
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
	instanceType productcatalog.AddonInstanceType,
	currency *currencies.CurrencyReference,
) addon.CreateAddonInput {
	t.Helper()

	rateCard := subscriptiontestutils.ExampleAddonRateCard4.Clone()
	err := rateCard.ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
		meta.Key = "subscription-currency-boundary"
		meta.Name = "Subscription currency boundary"
		meta.Feature = nil
		meta.Currency = currency

		return meta, nil
	})
	require.NoError(t, err)

	input := subscriptiontestutils.BuildAddonForTesting(
		t,
		productcatalog.EffectivePeriod{EffectiveFrom: lo.ToPtr(now)},
		instanceType,
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
) (models.NamespacedID, addon.Addon) {
	t.Helper()

	p, add := createPlanWithAddon(
		t,
		deps,
		subscriptiontestutils.GetExamplePlanInput(t),
		addonInput,
	)

	customer := deps.CustomerAdapter.CreateExampleCustomer(t)
	spec, err := subscription.NewSpecFromPlan(p, subscription.CreateSubscriptionCustomerInput{
		CustomerId:      customer.ID,
		InvoiceCurrency: "USD",
		ActiveFrom:      now,
		BillingAnchor:   now,
		Name:            "Test Subscription",
	})
	require.NoError(t, err)

	sub, err := deps.SubscriptionService.Create(t.Context(), subscriptiontestutils.ExampleNamespace, spec)
	require.NoError(t, err)
	require.NotNil(t, sub)

	return sub.NamespacedID, add
}
