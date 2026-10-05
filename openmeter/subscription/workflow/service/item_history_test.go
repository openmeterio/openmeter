package service_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
)

func TestEditRunningPreservesZeroLengthItemHistory(t *testing.T) {
	for _, tc := range []struct {
		name       string
		editOffset time.Duration
		timing     subscription.TimingEnum
	}{
		{name: "immediate", editOffset: time.Hour, timing: subscription.TimingImmediate},
		{name: "immediate at item start", timing: subscription.TimingImmediate},
		{name: "next billing cycle", editOffset: time.Hour, timing: subscription.TimingNextBillingCycle},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given: a running flat-fee subscription without proration
			start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			clock.FreezeTime(start)
			defer clock.UnFreeze()
			dbDeps := subscriptiontestutils.SetupDBDeps(t)
			defer dbDeps.Cleanup(t)
			deps := subscriptiontestutils.NewService(t, dbDeps)
			cust := deps.CustomerAdapter.CreateExampleCustomer(t)
			const itemKey = "fee"
			rateCard := &productcatalog.FlatFeeRateCard{RateCardMeta: productcatalog.RateCardMeta{
				Key: itemKey, Name: "Initial item",
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount: alpacadecimal.NewFromInt(1), PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
			}}
			planInput := subscriptiontestutils.BuildTestPlanInput(t).AddPhase(nil, rateCard).Build()
			planInput.Plan.ProRatingConfig = productcatalog.ProRatingConfig{}
			plan := deps.PlanHelper.CreatePlan(t, planInput)
			view, err := deps.WorkflowService.CreateFromPlan(t.Context(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
				Namespace: cust.Namespace, CustomerID: cust.ID,
				ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Name: "Subscription", Timing: subscription.Timing{Custom: &start},
				},
			}, plan)
			require.NoError(t, err)
			phaseKey := view.Phases[0].Spec.PhaseKey
			clock.FreezeTime(start.Add(tc.editOffset))
			defer clock.UnFreeze()

			// when: successive remove/add actions replace the item at the same effective time
			expectedNames := []string{"Initial item"}
			for i := 1; i <= 3; i++ {
				name := fmt.Sprintf("Replacement %d", i)
				expectedNames = append(expectedNames, name)
				replacement := rateCard.Clone()
				require.NoError(t, replacement.ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
					meta.Name = name
					return meta, nil
				}))
				view, err = deps.WorkflowService.EditRunning(t.Context(), view.Subscription.NamespacedID, []subscription.Patch{
					patch.PatchRemoveItem{PhaseKey: phaseKey, ItemKey: itemKey},
					patch.PatchAddItem{
						PhaseKey: phaseKey, ItemKey: itemKey,
						CreateInput: subscription.SubscriptionItemSpec{CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
							CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
								PhaseKey: phaseKey, ItemKey: itemKey, RateCard: replacement,
							},
						}},
					},
				}, subscription.Timing{Enum: &tc.timing})
				require.NoError(t, err)
			}

			// then: empty revisions retain their action order, including the unmarked original
			history := view.Phases[0].ItemsByKey[itemKey]
			require.Len(t, history, 4)
			patchIDs := make([]any, len(history))
			for i, item := range history {
				require.Equal(t, expectedNames[i], item.Spec.RateCard.AsMeta().Name)
				patchIDs[i] = item.Spec.Annotations[subscription.AnnotationEditUniqueKey]
				if i > 0 {
					require.NotEmpty(t, patchIDs[i])
				}
				if i == 1 || i == 2 || (i == 0 && tc.editOffset == 0) {
					require.NotNil(t, item.SubscriptionItem.ActiveTo)
					require.True(t, item.SubscriptionItem.AsPeriod().IsEmpty())
				}
			}
			require.Nil(t, patchIDs[0])

			// when: changing each rate card recreates all materialized item rows
			spec := view.AsSpec()
			for i, item := range spec.Phases[phaseKey].ItemsByKey[itemKey] {
				expectedNames[i] += " recreated"
				item.RateCard = item.RateCard.Clone()
				require.NoError(t, item.RateCard.ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
					meta.Name = expectedNames[i]
					return meta, nil
				}))
			}
			_, err = deps.SubscriptionService.Update(t.Context(), view.Subscription.NamespacedID, spec)
			require.NoError(t, err)
			items, err := deps.ItemRepo.GetForSubscriptionsAt(t.Context(), []subscription.GetForSubscriptionAtInput{{
				Namespace: view.Subscription.Namespace, SubscriptionID: view.Subscription.ID, At: clock.Now(),
			}})
			require.NoError(t, err)
			require.Len(t, items, 4)
			// Put the live row first and empty revisions in reverse action order.
			slices.SortFunc(items, func(a, b subscription.SubscriptionItem) int { return strings.Compare(b.Name, a.Name) })
			phases := lo.Map(view.Phases, func(phase subscription.SubscriptionPhaseView, _ int) subscription.SubscriptionPhase {
				return phase.SubscriptionPhase
			})
			rebuilt, err := subscription.NewSubscriptionView(view.Subscription, view.Customer, phases, items, nil, nil, nil)
			require.NoError(t, err)

			// then: logical versions and patch IDs survive row recreation and shuffled reads
			for i, item := range rebuilt.Phases[0].ItemsByKey[itemKey] {
				require.Equal(t, expectedNames[i], item.Spec.RateCard.AsMeta().Name)
				require.Equal(t, patchIDs[i], item.Spec.Annotations[subscription.AnnotationEditUniqueKey])
				require.NotEqual(t, history[i].SubscriptionItem.ID, item.SubscriptionItem.ID)
			}
		})
	}
}
