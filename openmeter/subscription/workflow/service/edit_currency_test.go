package service_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestEditRunningItemCurrency(t *testing.T) {
	for _, tt := range []struct {
		name           string
		current        currencyx.Code
		updated        currencyx.Code
		expected       currencyx.Code
		wantErr        error
		settlementMode productcatalog.SettlementMode
	}{
		{name: "fiat to custom", current: "USD", updated: "CREDITS", expected: "CREDITS"},
		{name: "custom to another custom", current: "CREDITS", updated: "POINTS", expected: "POINTS"},
		{name: "custom to explicit fiat", current: "CREDITS", updated: "USD", expected: "USD"},
		{name: "omitted currency uses invoice default", current: "CREDITS", expected: "USD"},
		{name: "reject mismatched fiat", current: "CREDITS", updated: "EUR", wantErr: productcatalog.ErrPlanMultipleFiatCurrencies},
		{name: "reject unknown custom currency", current: "USD", updated: "MISSING", wantErr: productcatalog.ErrCurrencyNotFound},
		{name: "reject missing cost basis", current: "USD", updated: "CREDITS", wantErr: productcatalog.ErrCurrencyCostBasisNotFound, settlementMode: productcatalog.CreditThenInvoiceSettlementMode},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// given: a running subscription with a priced item and managed currencies
			start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
			clock.FreezeTime(start)
			defer clock.UnFreeze()

			dbDeps := subscriptiontestutils.SetupDBDeps(t)
			defer dbDeps.Cleanup(t)
			deps := subscriptiontestutils.NewService(t, dbDeps)
			customer := deps.CustomerAdapter.CreateExampleCustomer(t)

			currencyRefs := map[currencyx.Code]currencies.CurrencyReference{
				"USD": currencies.NewCurrencyReference("USD"),
			}
			for _, code := range []currencyx.Code{"CREDITS", "POINTS"} {
				created, err := deps.CurrencyService.CreateCurrency(t.Context(), currenciestestutils.NewCreateCurrencyInput(
					subscriptiontestutils.ExampleNamespace, code, code.String(), code.String(),
				))
				require.NoError(t, err)
				currencyRefs[code] = created.Reference()
			}

			const itemKey = "fee"
			rateCard := &productcatalog.FlatFeeRateCard{
				RateCardMeta: productcatalog.RateCardMeta{
					Key:  itemKey,
					Name: itemKey,
					Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromInt(10),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
				},
				BillingCadence: lo.ToPtr(subscriptiontestutils.ISOMonth),
			}
			planInput := subscriptiontestutils.BuildTestPlanInput(t).AddPhase(nil, rateCard).Build()
			planInput.Plan.Currency = currencyRefs[tt.current]
			planInput.Plan.SettlementMode = productcatalog.CreditOnlySettlementMode
			if tt.settlementMode != "" {
				planInput.Plan.SettlementMode = tt.settlementMode
			}
			plan := deps.PlanHelper.CreatePlan(t, planInput)
			created, err := deps.WorkflowService.CreateFromPlan(t.Context(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
				ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Timing: subscription.Timing{Custom: &start},
				},
				CustomerID: customer.ID,
				Namespace:  subscriptiontestutils.ExampleNamespace,
			}, plan)
			require.NoError(t, err)
			phaseKey := created.Spec.GetSortedPhases()[0].PhaseKey

			// when: remove/add replaces the same item key with an authored currency
			editTime := start.Add(24 * time.Hour)
			clock.FreezeTime(editTime)
			defer clock.UnFreeze()
			replacement := rateCard.Clone()
			require.NoError(t, replacement.ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
				meta.Currency = nil
				if tt.updated != "" {
					meta.Currency = lo.ToPtr(currencies.NewCurrencyReference(tt.updated))
				}
				return meta, nil
			}))
			_, err = deps.WorkflowService.EditRunning(t.Context(), created.Subscription.NamespacedID, []subscription.Patch{
				patch.PatchRemoveItem{PhaseKey: phaseKey, ItemKey: itemKey},
				patch.PatchAddItem{
					PhaseKey: phaseKey,
					ItemKey:  itemKey,
					CreateInput: subscription.SubscriptionItemSpec{
						CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
							CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
								PhaseKey: phaseKey,
								ItemKey:  itemKey,
								RateCard: replacement,
							},
						},
					},
				},
			}, immediate)

			// then: normal currency validation applies and persisted history is preserved
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				loaded, err := deps.SubscriptionService.GetView(t.Context(), created.Subscription.NamespacedID)
				require.NoError(t, err)
				subscriptiontestutils.SpecsEqual(t, created.Spec, loaded.Spec)
				return
			}
			require.NoError(t, err)
			loaded, err := deps.SubscriptionService.GetView(t.Context(), created.Subscription.NamespacedID)
			require.NoError(t, err)
			phase, ok := loaded.GetPhaseByKey(phaseKey)
			require.True(t, ok)
			items := phase.ItemsByKey[itemKey]
			require.Len(t, items, 2)
			require.NotNil(t, items[0].Spec.RateCard.AsMeta().Currency)
			require.True(t, currencyRefs[tt.current].Equal(*items[0].Spec.RateCard.AsMeta().Currency))
			require.NotNil(t, items[0].SubscriptionItem.ActiveTo)
			require.True(t, items[0].SubscriptionItem.ActiveTo.Equal(editTime))
			require.NotNil(t, items[1].Spec.RateCard.AsMeta().Currency)
			require.True(t, currencyRefs[tt.expected].Equal(*items[1].Spec.RateCard.AsMeta().Currency))
			require.True(t, items[1].SubscriptionItem.ActiveFrom.Equal(editTime))
			require.Equal(t, currencyx.Code("USD"), loaded.Subscription.InvoiceCurrency)
		})
	}
}
