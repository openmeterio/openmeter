package service_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestMigrateToPlanAddons(t *testing.T) {
	for _, scenario := range []string{"unchanged addon", "changed base", "incompatible", "quantity limit"} {
		t.Run(scenario, func(t *testing.T) {
			// given an addon that extends a base item and already has a future quantity change
			start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			clock.FreezeTime(start.Add(time.Millisecond))
			defer clock.UnFreeze()
			ctx := t.Context()
			db := subscriptiontestutils.SetupDBDeps(t)
			defer db.Cleanup(t)
			deps := subscriptiontestutils.NewService(t, db)
			planInput := subscriptiontestutils.BuildTestPlanInput(t).AddPhase(nil,
				subscriptiontestutils.ExampleRateCard3ForAddons.Clone(), subscriptiontestutils.ExampleRateCard2.Clone()).Build()
			p1, addon := subscriptiontestutils.CreatePlanWithAddon(t, deps, planInput,
				subscriptiontestutils.BuildAddonForTesting(t, productcatalog.EffectivePeriod{EffectiveFrom: &start},
					productcatalog.AddonInstanceTypeMultiple, subscriptiontestutils.ExampleAddonRateCard4.Clone()))
			before := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps, p1, start)
			addonAt := start.Add(2 * 24 * time.Hour)
			clock.FreezeTime(addonAt)
			before, purchase, err := deps.WorkflowService.AddAddon(ctx, before.Subscription.NamespacedID, subscriptionworkflow.AddAddonWorkflowInput{
				AddonID: addon.ID, InitialQuantity: 1, Timing: subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)},
			})
			require.NoError(t, err)
			quantityAt := start.Add(19 * 24 * time.Hour)
			before, purchase, err = deps.WorkflowService.ChangeAddonQuantity(ctx, before.Subscription.NamespacedID, subscriptionworkflow.ChangeAddonQuantityWorkflowInput{
				SubscriptionAddonID: purchase.NamespacedID, Quantity: 2, Timing: subscription.Timing{Custom: &quantityAt},
			})
			require.NoError(t, err)
			at := start.Add(9 * 24 * time.Hour)
			clock.FreezeTime(at)
			nextInput := subscriptiontestutils.BuildTestPlanInput(t).AddPhase(nil,
				subscriptiontestutils.ExampleRateCard3ForAddons.Clone(), subscriptiontestutils.ExampleRateCard2.Clone(), subscriptiontestutils.ExampleRateCard1.Clone()).Build()
			if scenario == "changed base" {
				require.NoError(t, nextInput.Phases[0].RateCards[0].ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
					meta.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{Amount: alpacadecimal.NewFromInt(200), PaymentTerm: productcatalog.InAdvancePaymentTerm})
					return meta, nil
				}))
			}
			p2, err := deps.PlanService.CreatePlan(ctx, nextInput)
			require.NoError(t, err)
			if scenario != "incompatible" {
				var maxQuantity *int
				if scenario == "quantity limit" {
					maxQuantity = lo.ToPtr(1)
				}
				_, err := deps.PlanAddonService.CreatePlanAddon(ctx, planaddon.CreatePlanAddonInput{
					NamespacedModel: models.NamespacedModel{Namespace: addon.Namespace}, PlanID: p2.ID, AddonID: addon.ID,
					FromPlanPhase: "test_phase_1", MaxQuantity: maxQuantity,
				})
				require.NoError(t, err)
			}
			p2, err = deps.PlanService.PublishPlan(ctx, plan.PublishPlanInput{
				NamespacedID:    p2.NamespacedID,
				EffectivePeriod: productcatalog.EffectivePeriod{EffectiveFrom: lo.ToPtr(at)},
			})
			require.NoError(t, err)
			// when migrating with the existing addon purchases
			after, err := deps.WorkflowService.MigrateToPlan(ctx, subscriptionworkflow.MigrateSubscriptionWorkflowInput{
				SubscriptionID: before.Subscription.NamespacedID,
				Plan:           &plansubscription.Plan{Plan: p2.AsProductCatalogPlan(), Ref: &p2.NamespacedID},
				Timing:         subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)},
			})
			// then incompatible current or future purchases reject the whole operation
			if scenario == "incompatible" || scenario == "quantity limit" {
				require.Error(t, err)
				persisted, err := deps.SubscriptionService.GetView(ctx, before.Subscription.NamespacedID)
				require.NoError(t, err)
				beforeJSON, err := json.Marshal(before)
				require.NoError(t, err)
				persistedJSON, err := json.Marshal(persisted)
				require.NoError(t, err)
				require.JSONEq(t, string(beforeJSON), string(persistedJSON))
				return
			}
			require.NoError(t, err)
			key := subscriptiontestutils.ExampleFeatureKey2
			require.Equal(t, before.Subscription.ID, after.Subscription.ID)
			require.Equal(t, before.Phases[0].ItemsByKey["rate-card-2"], after.Phases[0].ItemsByKey["rate-card-2"])
			if scenario == "unchanged addon" {
				require.Len(t, after.Phases[0].ItemsByKey[key], len(before.Phases[0].ItemsByKey[key]))
				for idx, item := range before.Phases[0].ItemsByKey[key] {
					actual := after.Phases[0].ItemsByKey[key][idx].SubscriptionItem
					require.Equal(t, item.SubscriptionItem.ID, actual.ID)
					require.Equal(t, item.SubscriptionItem.EntitlementID, actual.EntitlementID)
					require.Equal(t, item.SubscriptionItem.CadencedModel, actual.CadencedModel)
				}
			}
			// when removing the addon later, restoration must retain the migrated base price
			removeAt := start.Add(24 * 24 * time.Hour)
			clock.FreezeTime(removeAt)
			removed, _, err := deps.WorkflowService.ChangeAddonQuantity(ctx, after.Subscription.NamespacedID, subscriptionworkflow.ChangeAddonQuantityWorkflowInput{
				SubscriptionAddonID: purchase.NamespacedID, Quantity: 0, Timing: subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)},
			})
			require.NoError(t, err)
			expected := 100.0
			if scenario == "changed base" {
				expected = 200
			}
			active, ok := lo.Find(removed.Phases[0].ItemsByKey[key], func(item subscription.SubscriptionItemView) bool {
				return item.SubscriptionItem.IsActiveAt(removeAt)
			})
			require.True(t, ok)
			price, err := active.Spec.RateCard.AsMeta().Price.AsFlat()
			require.NoError(t, err)
			require.Equal(t, expected, price.Amount.InexactFloat64())
			require.NoError(t, removed.Validate(true))
		})
	}
}
