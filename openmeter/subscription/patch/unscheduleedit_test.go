package patch_test

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
)

func TestUnscheduleEdit(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2021-01-01T00:00:01Z")
	clock.SetTime(now)

	getSpec := func(t *testing.T) *subscription.SubscriptionSpec {
		s, _ := getDefaultSpec(t, now)
		return s
	}

	suite := testsuite[patch.PatchUnscheduleEdit]{
		SystemTime: now,
		TT: []testcase[patch.PatchUnscheduleEdit]{
			{
				Name:  "Should not find phase",
				Patch: patch.PatchUnscheduleEdit{},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s := getSpec(t)
					delete(s.Phases, "test_phase_1")
					return s
				},
				Ctx: subscription.ApplyContext{CurrentTime: now},
				ExpectedError: &subscription.PatchConflictError{
					Msg: "current phase doesn't exist, cannot unschedule edits",
				},
			},
			{
				Name:  "Should unschedule future edits in current phase",
				Patch: patch.PatchUnscheduleEdit{},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s := getSpec(t)

					// Add a future scheduled edit to test_phase_1
					phase := s.Phases["test_phase_1"]
					items := phase.ItemsByKey[subscriptiontestutils.ExampleFeatureKey]

					// Create a future version of the item
					futureItem := *items[0]
					items[0].ActiveToOverrideRelativeToPhaseStart = lo.ToPtr(testutils.GetISODuration(t, "P2D"))

					futureItem.ActiveFromOverrideRelativeToPhaseStart = lo.ToPtr(testutils.GetISODuration(t, "P2D"))
					items = append(items, &futureItem)

					phase.ItemsByKey[subscriptiontestutils.ExampleFeatureKey] = items
					return s
				},
				Ctx: subscription.ApplyContext{CurrentTime: now},
				GetExpectedSpec: func(t *testing.T) subscription.SubscriptionSpec {
					s := getSpec(t)
					// The future edit should be removed, leaving only the original item
					require.Len(t, s.Phases["test_phase_1"].ItemsByKey[subscriptiontestutils.ExampleFeatureKey], 1)
					return *s
				},
			},
			{
				Name:  "Should handle phase with no scheduled edits",
				Patch: patch.PatchUnscheduleEdit{},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s := getSpec(t)
					// No modifications needed as default spec has no scheduled edits
					return s
				},
				Ctx: subscription.ApplyContext{CurrentTime: now},
				GetExpectedSpec: func(t *testing.T) subscription.SubscriptionSpec {
					s := getSpec(t)
					// Spec should remain unchanged
					return *s
				},
			},
			{
				Name:  "Should handle multiple scheduled edits",
				Patch: patch.PatchUnscheduleEdit{},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s := getSpec(t)

					// Add multiple future scheduled edits to test_phase_1
					phase := s.Phases["test_phase_1"]
					items := phase.ItemsByKey[subscriptiontestutils.ExampleFeatureKey]

					// Create two future versions of the item
					futureItem1 := *items[0]
					futureItem1.ActiveFromOverrideRelativeToPhaseStart = lo.ToPtr(testutils.GetISODuration(t, "P2D"))

					futureItem2 := *items[0]
					futureItem2.ActiveFromOverrideRelativeToPhaseStart = lo.ToPtr(testutils.GetISODuration(t, "P3D"))

					items = append(items, &futureItem1, &futureItem2)
					phase.ItemsByKey[subscriptiontestutils.ExampleFeatureKey] = items

					return s
				},
				Ctx: subscription.ApplyContext{CurrentTime: now},
				GetExpectedSpec: func(t *testing.T) subscription.SubscriptionSpec {
					s := getSpec(t)
					// All future edits should be removed, leaving only the original item
					require.Len(t, s.Phases["test_phase_1"].ItemsByKey[subscriptiontestutils.ExampleFeatureKey], 1)
					return *s
				},
			},
			{
				Name:  "Should handle multiple scheduled edits #2",
				Patch: patch.PatchUnscheduleEdit{},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s := getSpec(t)

					// Add multiple future scheduled edits to test_phase_1
					phase := s.Phases["test_phase_1"]
					items := phase.ItemsByKey[subscriptiontestutils.ExampleFeatureKey]

					// Create two future versions of the item
					futureItem1 := *items[0]
					items[0].ActiveToOverrideRelativeToPhaseStart = lo.ToPtr(testutils.GetISODuration(t, "P1D"))

					futureItem1.ActiveFromOverrideRelativeToPhaseStart = lo.ToPtr(testutils.GetISODuration(t, "P1D"))
					futureItem1.ActiveToOverrideRelativeToPhaseStart = lo.ToPtr(testutils.GetISODuration(t, "P3D"))

					futureItem2 := *items[0]
					futureItem2.ActiveFromOverrideRelativeToPhaseStart = lo.ToPtr(testutils.GetISODuration(t, "P3D"))
					futureItem2.ActiveToOverrideRelativeToPhaseStart = lo.ToPtr(testutils.GetISODuration(t, "P4D"))

					futureItem3 := *items[0]
					futureItem3.ActiveFromOverrideRelativeToPhaseStart = lo.ToPtr(testutils.GetISODuration(t, "P4D"))

					items = append(items, &futureItem1, &futureItem2, &futureItem3)
					phase.ItemsByKey[subscriptiontestutils.ExampleFeatureKey] = items

					return s
				},
				Ctx: subscription.ApplyContext{CurrentTime: now.AddDate(0, 0, 2)},
				GetExpectedSpec: func(t *testing.T) subscription.SubscriptionSpec {
					s := getSpec(t)

					// Create two future versions of the item
					phase := s.Phases["test_phase_1"]
					items := phase.ItemsByKey[subscriptiontestutils.ExampleFeatureKey]
					futureItem1 := *items[0]
					items[0].ActiveToOverrideRelativeToPhaseStart = lo.ToPtr(testutils.GetISODuration(t, "P1D"))

					futureItem1.ActiveFromOverrideRelativeToPhaseStart = lo.ToPtr(testutils.GetISODuration(t, "P1D"))

					items = append(items, &futureItem1)
					phase.ItemsByKey[subscriptiontestutils.ExampleFeatureKey] = items

					// All future edits should be removed, leaving only the original item + the future edit
					require.Len(t, s.Phases["test_phase_1"].ItemsByKey[subscriptiontestutils.ExampleFeatureKey], 2)
					return *s
				},
			},
		},
	}

	suite.Run(t)
}
