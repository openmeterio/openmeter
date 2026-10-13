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
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
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
		planCurrency   currencyx.Code
		updated        currencyx.Code
		expected       currencyx.Code
		wantErr        error
		settlementMode productcatalog.SettlementMode
		newItem        bool
		inlinePlan     bool
		unpriced       bool
		pointerPatch   bool
	}{
		{name: "fiat to custom", current: "USD", updated: "CREDITS", expected: "CREDITS"},
		{name: "custom to another custom", current: "CREDITS", updated: "POINTS", expected: "POINTS"},
		{name: "custom to explicit fiat", current: "CREDITS", updated: "USD", expected: "USD"},
		{name: "omitted currency uses fiat plan default", current: "USD", expected: "USD"},
		{name: "omitted currency uses custom plan default", current: "CREDITS", expected: "CREDITS"},
		{name: "new item uses custom plan default", current: "CREDITS", expected: "CREDITS", newItem: true},
		{name: "pointer patch uses custom plan default", current: "CREDITS", expected: "CREDITS", newItem: true, pointerPatch: true},
		{name: "item override does not define edit default", current: "POINTS", planCurrency: "USD", expected: "USD"},
		{name: "inline plan falls back to invoice currency", current: "CREDITS", expected: "USD", inlinePlan: true},
		{name: "unpriced item has no currency", current: "CREDITS", unpriced: true},
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
			if tt.planCurrency != "" {
				rateCard.Currency = lo.ToPtr(currencyRefs[tt.current])
			}

			planInput := subscriptiontestutils.BuildTestPlanInput(t).AddPhase(nil, rateCard).Build()
			planInput.Plan.Currency = currencyRefs[tt.current]
			if tt.planCurrency != "" {
				planInput.Plan.Currency = currencyRefs[tt.planCurrency]
			}

			planInput.Plan.SettlementMode = productcatalog.CreditOnlySettlementMode
			if tt.settlementMode != "" {
				planInput.Plan.SettlementMode = tt.settlementMode
			}

			var plan subscription.Plan
			if tt.inlinePlan {
				plan = &plansubscription.Plan{Plan: planInput.Plan}
			} else {
				plan = deps.PlanHelper.CreatePlan(t, planInput)
			}

			created, err := deps.WorkflowService.CreateFromPlan(t.Context(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
				ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Timing: subscription.Timing{Custom: &start},
				},
				CustomerID: customer.ID,
				Namespace:  subscriptiontestutils.ExampleNamespace,
			}, plan)
			require.NoError(t, err)
			if tt.inlinePlan {
				require.Nil(t, created.Subscription.Currency)
			} else {
				require.NotNil(t, created.Subscription.Currency)
				require.True(t, planInput.Plan.Currency.Equal(*created.Subscription.Currency))
			}

			phaseKey := created.Spec.GetSortedPhases()[0].PhaseKey

			// when: an edit adds or replaces an item, optionally supplying its currency
			editTime := start.Add(24 * time.Hour)
			clock.FreezeTime(editTime)
			defer clock.UnFreeze()
			replacement := rateCard.Clone()
			editedKey := itemKey
			if tt.newItem {
				editedKey = "new-fee"
			}

			require.NoError(t, replacement.ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
				meta.Key = editedKey
				meta.Currency = nil
				if tt.updated != "" {
					meta.Currency = lo.ToPtr(currencies.NewCurrencyReference(tt.updated))
				}

				if tt.unpriced {
					meta.Price = nil
				}

				return meta, nil
			}))
			addPatch := patch.PatchAddItem{
				PhaseKey: phaseKey,
				ItemKey:  editedKey,
				CreateInput: subscription.SubscriptionItemSpec{
					CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
						CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
							PhaseKey: phaseKey,
							ItemKey:  editedKey,
							RateCard: replacement,
						},
					},
				},
			}
			var patches []subscription.Patch
			if !tt.newItem {
				patches = append(patches, patch.PatchRemoveItem{PhaseKey: phaseKey, ItemKey: itemKey})
			}

			if tt.pointerPatch {
				patches = append(patches, &addPatch)
			} else {
				patches = append(patches, addPatch)
			}

			_, err = deps.WorkflowService.EditRunning(t.Context(), created.Subscription.NamespacedID, patches, immediate)

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
			items := phase.ItemsByKey[editedKey]
			if tt.newItem {
				require.Len(t, items, 1)
			} else {
				require.Len(t, items, 2)
				require.NotNil(t, items[0].Spec.RateCard.AsMeta().Currency)
				require.True(t, currencyRefs[tt.current].Equal(*items[0].Spec.RateCard.AsMeta().Currency))
				require.NotNil(t, items[0].SubscriptionItem.ActiveTo)
				require.True(t, items[0].SubscriptionItem.ActiveTo.Equal(editTime))
			}

			added := items[len(items)-1]
			if tt.unpriced {
				require.Nil(t, added.Spec.RateCard.AsMeta().Currency)
			} else {
				require.NotNil(t, added.Spec.RateCard.AsMeta().Currency)
				require.True(t, currencyRefs[tt.expected].Equal(*added.Spec.RateCard.AsMeta().Currency))
			}

			require.True(t, added.SubscriptionItem.ActiveFrom.Equal(editTime))
			require.Equal(t, currencyx.Code("USD"), loaded.Subscription.InvoiceCurrency)
		})
	}
}
