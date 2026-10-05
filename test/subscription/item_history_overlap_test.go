package subscription_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
)

func TestSubscriptionViewAllowsAPIProducedZeroLengthFlatFeeRevisionWithoutProration(t *testing.T) {
	// Given a running subscription with one in-advance flat-fee item and proration disabled.
	startedAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	clock.SetTime(startedAt)
	defer clock.ResetTime()

	deps := setup(t, setupConfig{})
	defer deps.cleanup(t)

	const (
		phaseKey = "test_phase_1"
		itemKey  = "item"
	)

	planInput := subscriptiontestutils.BuildTestPlanInput(t).
		AddPhase(nil, &productcatalog.FlatFeeRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:  itemKey,
				Name: "Initial item",
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      decimal.NewFromInt(1),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
			},
		}).
		Build()
	planInput.Plan.ProRatingConfig = productcatalog.ProRatingConfig{}
	plan := deps.PlanHelper.CreatePlan(t, planInput)
	customerModel, err := deps.CustomerService.CreateCustomer(t.Context(), customer.CreateCustomerInput{
		Namespace: subscriptiontestutils.ExampleNamespace,
		CustomerMutate: customer.CustomerMutate{
			Name: "Customer",
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{"item-history-overlap"},
			},
		},
	})
	require.NoError(t, err)

	created, err := deps.WorkflowService.CreateFromPlan(t.Context(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
		Namespace:  subscriptiontestutils.ExampleNamespace,
		CustomerID: customerModel.ID,
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Name:   "Subscription",
			Timing: subscription.Timing{Custom: &startedAt},
		},
	}, plan)
	require.NoError(t, err)
	require.False(t, created.Subscription.ProRatingConfig.Enabled)

	// When an API client replaces the same item twice at one effective time.
	replacedAt := startedAt.Add(time.Hour)
	clock.FreezeTime(replacedAt)
	defer clock.UnFreeze()
	immediate := subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)}

	firstReplacement, err := deps.WorkflowService.EditRunning(t.Context(), created.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchRemoveItem{PhaseKey: phaseKey, ItemKey: itemKey},
		patch.PatchAddItem{
			PhaseKey: phaseKey,
			ItemKey:  itemKey,
			CreateInput: subscription.SubscriptionItemSpec{CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
				CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
					PhaseKey: phaseKey,
					ItemKey:  itemKey,
					RateCard: &productcatalog.FlatFeeRateCard{RateCardMeta: productcatalog.RateCardMeta{
						Key:  itemKey,
						Name: "First replacement",
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      decimal.NewFromInt(2),
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						}),
					}},
				},
			}},
		},
	}, immediate)
	require.NoError(t, err)

	secondReplacement, err := deps.WorkflowService.EditRunning(t.Context(), firstReplacement.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchRemoveItem{PhaseKey: phaseKey, ItemKey: itemKey},
		patch.PatchAddItem{
			PhaseKey: phaseKey,
			ItemKey:  itemKey,
			CreateInput: subscription.SubscriptionItemSpec{CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
				CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
					PhaseKey: phaseKey,
					ItemKey:  itemKey,
					RateCard: &productcatalog.FlatFeeRateCard{RateCardMeta: productcatalog.RateCardMeta{
						Key:  itemKey,
						Name: "Second replacement",
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      decimal.NewFromInt(3),
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						}),
					}},
				},
			}},
		},
	}, immediate)
	require.NoError(t, err)

	// Then persistence contains the API-produced history: [start,T), [T,T), and [T,...).
	items, err := deps.ItemRepo.GetForSubscriptionsAt(t.Context(), []subscription.GetForSubscriptionAtInput{{
		Namespace:      secondReplacement.Subscription.Namespace,
		SubscriptionID: secondReplacement.Subscription.ID,
		At:             replacedAt,
	}})
	require.NoError(t, err)
	require.Len(t, items, 3)

	previous, found := lo.Find(items, func(item subscription.SubscriptionItem) bool {
		return item.ActiveFrom.Equal(startedAt)
	})
	require.True(t, found)
	require.NotNil(t, previous.ActiveTo)
	require.Equal(t, replacedAt, *previous.ActiveTo)

	empty, found := lo.Find(items, func(item subscription.SubscriptionItem) bool {
		return item.ActiveFrom.Equal(replacedAt) && item.ActiveTo != nil && item.ActiveTo.Equal(replacedAt)
	})
	require.True(t, found)
	current, found := lo.Find(items, func(item subscription.SubscriptionItem) bool {
		return item.ActiveFrom.Equal(replacedAt) && item.ActiveTo == nil
	})
	require.True(t, found)

	// Repository reads have no ordering contract. Reproduce the valid observed order deterministically:
	// the nonempty revision precedes the empty revision with the same start time.
	itemsInObservedOrder := slices.Clone(items)
	slices.SortStableFunc(itemsInObservedOrder, func(a, b subscription.SubscriptionItem) int {
		if diff := a.ActiveFrom.Compare(b.ActiveFrom); diff != 0 {
			return diff
		}

		aEmpty := a.ActiveTo != nil && a.ActiveTo.Equal(a.ActiveFrom)
		bEmpty := b.ActiveTo != nil && b.ActiveTo.Equal(b.ActiveFrom)
		switch {
		case aEmpty && !bEmpty:
			return 1
		case !aEmpty && bEmpty:
			return -1
		default:
			return strings.Compare(a.ID, b.ID)
		}
	})
	require.Equal(t, current.ID, itemsInObservedOrder[1].ID)
	require.Equal(t, empty.ID, itemsInObservedOrder[2].ID)

	view, err := subscription.NewSubscriptionView(
		secondReplacement.Subscription,
		secondReplacement.Customer,
		lo.Map(secondReplacement.Phases, func(phase subscription.SubscriptionPhaseView, _ int) subscription.SubscriptionPhase {
			return phase.SubscriptionPhase
		}),
		itemsInObservedOrder,
		nil,
		nil,
		nil,
	)

	require.NoError(t, err)
	require.Equal(t, []string{previous.ID, current.ID, empty.ID}, lo.Map(view.Phases[0].ItemsByKey[itemKey], func(item subscription.SubscriptionItemView, _ int) string {
		return item.SubscriptionItem.ID
	}))
}
