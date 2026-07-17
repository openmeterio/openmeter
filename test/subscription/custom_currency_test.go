package subscription_test

import (
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	pcsubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestCustomCurrencySubscriptionLifecycle(t *testing.T) {
	// given:
	// - a managed custom currency and a published plan priced entirely in it
	// - customers whose billing currency is USD
	// when:
	// - subscriptions are started in each supported settlement and cost-basis mode
	// then:
	// - the persisted subscription keeps USD for invoicing and the managed currency on its item
	// - conversion eligibility and billing's temporary safety boundary are enforced end to end
	const namespace = "test-namespace"

	now := time.Now().UTC().Truncate(time.Second)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	deps := setup(t, setupConfig{})
	defer deps.cleanup(t)

	customCode := currencyx.Code("CREDITS")
	customCurrency, err := deps.CurrencyService.CreateCurrency(t.Context(), currencies.CreateCurrencyInput{
		Namespace: namespace,
		Code:      customCode.String(),
		Name:      "Credits",
		Symbol:    "CR",
	})
	require.NoError(t, err)

	month := datetime.MustParseDuration(t, "P1M")
	customPlan, err := deps.PlanService.CreatePlan(t.Context(), plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{Namespace: namespace},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Key:            "custom-currency-subscriptions",
				Name:           "Custom currency subscriptions",
				Currency:       customCode,
				BillingCadence: month,
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{Key: "default", Name: "Default"},
					RateCards: productcatalog.RateCards{
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "credits",
								Name: "Credits",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      decimal.NewFromInt(25),
									PaymentTerm: productcatalog.InAdvancePaymentTerm,
								}),
							},
							BillingCadence: &month,
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	customPlan, err = deps.PlanService.PublishPlan(t.Context(), plan.PublishPlanInput{
		NamespacedID: customPlan.NamespacedID,
		EffectivePeriod: productcatalog.EffectivePeriod{
			EffectiveFrom: lo.ToPtr(now.Add(-time.Second)),
		},
	})
	require.NoError(t, err)

	startsAt := now.Add(time.Hour)

	t.Run("credit only materializes currency without a cost basis", func(t *testing.T) {
		// given:
		// - no USD cost basis exists for the plan's custom currency
		// when:
		// - a credit-only subscription is started for a USD customer
		// then:
		// - it persists with USD as invoice currency and the managed custom identity on its item
		cus := createUSDSubscriptionCustomer(t, deps, namespace, "credit-only")
		sub, err := createCustomCurrencySubscription(
			t,
			deps,
			*customPlan,
			cus.ID,
			startsAt,
			productcatalog.CreditOnlySettlementMode,
			subscription.CostBasisModeDynamic,
		)
		require.NoError(t, err)

		view, err := deps.subscriptionService.GetView(t.Context(), sub.NamespacedID)
		require.NoError(t, err)
		require.NoError(t, view.Validate(true))
		require.Equal(t, currencyx.Code("USD"), view.Subscription.InvoiceCurrency)
		require.Empty(t, view.Subscription.CostBasisPins)

		phase, ok := view.GetPhaseByKey("default")
		require.True(t, ok)
		require.Len(t, phase.ItemsByKey["credits"], 1)

		itemCurrency := phase.ItemsByKey["credits"][0].SubscriptionItem.RateCard.AsMeta().Currency
		managedCurrency, ok := itemCurrency.(currencyx.ManagedCurrency)
		require.True(t, ok, "persisted custom item currency must reload as a managed resource")
		require.Equal(t, customCurrency.ID, managedCurrency.GetID())

		// Billing conversion is deliberately outside this PR. Explicit callers see the
		// unsupported boundary; automatic reconciliation skips the whole subscription.
		err = deps.subscriptionSyncService.SyncByView(t.Context(), view, startsAt)
		require.ErrorIs(t, err, subscriptionsync.ErrCustomCurrencyBillingNotSupported)
		require.True(t, models.IsGenericConflictError(err))

		err = deps.subscriptionSyncService.SyncByView(
			t.Context(),
			view,
			startsAt,
			subscriptionsync.SkipCustomCurrencySubscriptions(),
		)
		require.NoError(t, err)
	})

	t.Run("credit then invoice resolves dynamic and pinned cost bases", func(t *testing.T) {
		// given:
		// - a USD customer and no effective custom-currency to USD cost basis
		// when:
		// - a credit-then-invoice subscription is started
		// then:
		// - creation is rejected until a cost basis exists
		dynamicCustomer := createUSDSubscriptionCustomer(t, deps, namespace, "dynamic-cost-basis")
		_, err := createCustomCurrencySubscription(
			t,
			deps,
			*customPlan,
			dynamicCustomer.ID,
			startsAt,
			productcatalog.CreditThenInvoiceSettlementMode,
			subscription.CostBasisModeDynamic,
		)
		require.Error(t, err)
		require.True(t, models.IsGenericValidationError(err))
		require.ErrorIs(t, err, productcatalog.ErrCurrencyCostBasisNotFound)

		costBasis, err := deps.CurrencyService.CreateCostBasis(t.Context(), currencies.CreateCostBasisInput{
			Namespace:  namespace,
			CurrencyID: customCurrency.ID,
			FiatCode:   "USD",
			Rate:       decimal.NewFromInt(2),
		})
		require.NoError(t, err)

		// Dynamic mode validates the pair at creation time but does not persist a pin.
		dynamicSub, err := createCustomCurrencySubscription(
			t,
			deps,
			*customPlan,
			dynamicCustomer.ID,
			startsAt,
			productcatalog.CreditThenInvoiceSettlementMode,
			subscription.CostBasisModeDynamic,
		)
		require.NoError(t, err)

		dynamicView, err := deps.subscriptionService.GetView(t.Context(), dynamicSub.NamespacedID)
		require.NoError(t, err)
		require.Equal(t, subscription.CostBasisModeDynamic, dynamicView.Subscription.CostBasisMode)
		require.Empty(t, dynamicView.Subscription.CostBasisPins)

		// Pinned mode reloads the exact managed cost-basis resource selected at start.
		pinnedCustomer := createUSDSubscriptionCustomer(t, deps, namespace, "pinned-cost-basis")
		pinnedSub, err := createCustomCurrencySubscription(
			t,
			deps,
			*customPlan,
			pinnedCustomer.ID,
			startsAt,
			productcatalog.CreditThenInvoiceSettlementMode,
			subscription.CostBasisModePinned,
		)
		require.NoError(t, err)

		pinnedView, err := deps.subscriptionService.GetView(t.Context(), pinnedSub.NamespacedID)
		require.NoError(t, err)
		require.NoError(t, pinnedView.Validate(true))
		require.Equal(t, subscription.CostBasisModePinned, pinnedView.Subscription.CostBasisMode)
		require.Len(t, pinnedView.Subscription.CostBasisPins, 1)

		pin := pinnedView.Subscription.CostBasisPins[0]
		require.Equal(t, customCurrency.ID, pin.CustomCurrencyID)
		require.Equal(t, currencyx.Code("USD"), pin.InvoiceCurrency)
		require.Equal(t, costBasis.ID, pin.CostBasis.ID)
		require.Equal(t, float64(2), pin.CostBasis.Rate.InexactFloat64())
	})
}

func createUSDSubscriptionCustomer(t *testing.T, deps testDeps, namespace, key string) *customer.Customer {
	t.Helper()

	cus, err := deps.CustomerService.CreateCustomer(t.Context(), customer.CreateCustomerInput{
		Namespace: namespace,
		CustomerMutate: customer.CustomerMutate{
			Key:      lo.ToPtr(key),
			Name:     key,
			Currency: lo.ToPtr(currencyx.Code("USD")),
		},
	})
	require.NoError(t, err)

	return cus
}

func createCustomCurrencySubscription(
	t *testing.T,
	deps testDeps,
	customPlan plan.Plan,
	customerID string,
	startsAt time.Time,
	settlementMode productcatalog.SettlementMode,
	costBasisMode subscription.CostBasisMode,
) (subscription.Subscription, error) {
	t.Helper()

	planInput := &pcsubscription.PlanInput{}
	planInput.FromRef(&pcsubscription.PlanRefInput{
		Key:     customPlan.Key,
		Version: &customPlan.Version,
	})

	return deps.pcSubscriptionService.Create(t.Context(), pcsubscription.CreateSubscriptionRequest{
		WorkflowInput: subscriptionworkflow.CreateSubscriptionWorkflowInput{
			Namespace:  customPlan.Namespace,
			CustomerID: customerID,
			ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Timing:        subscription.Timing{Custom: &startsAt},
				Name:          customerID,
				CostBasisMode: costBasisMode,
			},
		},
		PlanInput:      *planInput,
		SettlementMode: &settlementMode,
	})
}
