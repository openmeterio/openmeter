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

func TestRemoveItem(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2021-01-01T00:00:01Z")
	clock.SetTime(now)

	getSpec := func(t *testing.T) *subscription.SubscriptionSpec {
		s, _ := getDefaultSpec(t, now)
		return s
	}

	suite := testsuite[patch.PatchRemoveItem]{
		SystemTime: now,
		TT: []testcase[patch.PatchRemoveItem]{
			{
				Name: "Should not find phase",
				Patch: patch.PatchRemoveItem{
					PhaseKey: "notfound",
					ItemKey:  "item1",
				},
				GetSpec: getSpec,
				Ctx:     subscription.ApplyContext{CurrentTime: now},
				ExpectedError: &subscription.PatchValidationError{
					Msg: "phase notfound not found",
				},
			},
			{
				Name: "Should not find item",
				Patch: patch.PatchRemoveItem{
					PhaseKey: "test_phase_1",
					ItemKey:  "invalid",
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s, _ := getDefaultSpec(t, now)

					// Lets validate the spec looks as we expect it
					require.GreaterOrEqual(t, len(s.Phases), 1)
					ph, ok := s.Phases["test_phase_1"]
					require.True(t, ok)

					_, ok = ph.ItemsByKey["invalid"]
					require.False(t, ok)

					return s
				},
				Ctx: subscription.ApplyContext{CurrentTime: now},
				ExpectedError: &subscription.PatchValidationError{
					Msg: "items for key invalid doesn't exists in phase test_phase_1",
				},
			},
			{
				Name: "Should not remove item from past phase",
				Patch: patch.PatchRemoveItem{
					PhaseKey: "test_phase_1",
					ItemKey:  subscriptiontestutils.ExampleFeatureKey,
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s, _ := getDefaultSpec(t, now)

					// Lets validate the spec looks as we expect it
					require.GreaterOrEqual(t, len(s.Phases), 3)

					ph2, ok := s.Phases["test_phase_2"]
					require.True(t, ok)

					ph3, ok := s.Phases["test_phase_3"]
					require.True(t, ok)

					ts := now.AddDate(0, 1, 1)

					// Let's make sure we're in the second phase
					ph2st, _ := ph2.StartAfter.AddTo(s.ActiveFrom)
					require.True(t, ph2st.Before(ts))

					ph3st, _ := ph3.StartAfter.AddTo(s.ActiveFrom)
					require.True(t, ph3st.After(ts))

					// Let's make sure the 1st phase has the item we're removing
					ph1, ok := s.Phases["test_phase_1"]
					require.True(t, ok)

					v, ok := ph1.ItemsByKey[subscriptiontestutils.ExampleFeatureKey]
					require.True(t, ok)
					require.Greater(t, len(v), 0)

					return s
				},
				Ctx: subscription.ApplyContext{
					CurrentTime: now.AddDate(0, 1, 1), // same as ts above
				},
				ExpectedError: &subscription.PatchForbiddenError{
					Msg: "cannot remove item from phase test_phase_1 which starts before current phase",
				},
			},
			{
				Name: "Should not remove item from inactive subscription",
				Patch: patch.PatchRemoveItem{
					PhaseKey: "test_phase_1",
					ItemKey:  subscriptiontestutils.ExampleFeatureKey,
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s, _ := getDefaultSpec(t, now)

					s.ActiveTo = lo.ToPtr(s.ActiveFrom.AddDate(0, 9, 0))

					return s
				},
				Ctx: subscription.ApplyContext{
					CurrentTime: now.AddDate(1, 0, 0), // We're far in the future
				},
				ExpectedError: &subscription.PatchForbiddenError{
					Msg: "cannot remove item from phase test_phase_1 which starts before current phase",
				},
			},
			{
				Name: "Should remove item from future phase",
				Patch: patch.PatchRemoveItem{
					PhaseKey: "test_phase_3",
					ItemKey:  subscriptiontestutils.ExampleFeatureKey,
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s, _ := getDefaultSpec(t, now)

					// Lets validate the spec looks as we expect it
					require.GreaterOrEqual(t, len(s.Phases), 3)

					ph2, ok := s.Phases["test_phase_2"]
					require.True(t, ok)

					ph3, ok := s.Phases["test_phase_3"]
					require.True(t, ok)

					ts := now.AddDate(0, 1, 1)

					// Let's make sure we're in the second phase
					ph2st, _ := ph2.StartAfter.AddTo(s.ActiveFrom)
					require.True(t, ph2st.Before(ts))

					ph3st, _ := ph3.StartAfter.AddTo(s.ActiveFrom)
					require.True(t, ph3st.After(ts))

					// Let's make sure the 3rd phase has the item we're removing
					v, ok := ph3.ItemsByKey[subscriptiontestutils.ExampleFeatureKey]
					require.True(t, ok)
					require.Greater(t, len(v), 0)

					return s
				},
				Ctx: subscription.ApplyContext{
					CurrentTime: now.AddDate(0, 1, 1), // same as ts above
				},
				GetExpectedSpec: func(t *testing.T) subscription.SubscriptionSpec {
					s, _ := getDefaultSpec(t, now)

					// Let's make sure that's the only item in the 3rd phase
					require.Equal(t, 1, len(s.Phases["test_phase_3"].ItemsByKey))

					// Let's remove the item from the 3rd phase
					s.Phases["test_phase_3"].ItemsByKey = map[string][]*subscription.SubscriptionItemSpec{}

					return *s
				},
			},
			{
				Name: "Should remove item from current phase by closing the previous version of it",
				Patch: patch.PatchRemoveItem{
					PhaseKey: "test_phase_2",
					ItemKey:  subscriptiontestutils.ExampleFeatureKey,
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s, _ := getDefaultSpec(t, now)

					// Lets validate the spec looks as we expect it
					require.GreaterOrEqual(t, len(s.Phases), 3)

					ph2, ok := s.Phases["test_phase_2"]
					require.True(t, ok)

					ph3, ok := s.Phases["test_phase_3"]
					require.True(t, ok)

					ts := now.AddDate(0, 1, 1)

					// Let's make sure we're in the second phase
					ph2st, _ := ph2.StartAfter.AddTo(s.ActiveFrom)
					require.True(t, ph2st.Before(ts))

					ph3st, _ := ph3.StartAfter.AddTo(s.ActiveFrom)
					require.True(t, ph3st.After(ts))

					// Let's make sure the 2nd phase has the item we're removing
					v, ok := ph2.ItemsByKey[subscriptiontestutils.ExampleFeatureKey]
					require.True(t, ok)
					require.Greater(t, len(v), 0)

					return s
				},
				Ctx: subscription.ApplyContext{
					CurrentTime: now.AddDate(0, 1, 1), // same as ts above
				},
				GetExpectedSpec: func(t *testing.T) subscription.SubscriptionSpec {
					s, _ := getDefaultSpec(t, now)

					// Let's make sure that's the only item in the 3rd phase
					require.GreaterOrEqual(t, len(s.Phases["test_phase_2"].ItemsByKey), 1)
					v, ok := s.Phases["test_phase_2"].ItemsByKey[subscriptiontestutils.ExampleFeatureKey]
					require.True(t, ok)
					require.Equal(t, len(v), 1)

					// When removing an item from the current phase it should close that item with the current timestamp
					// We have to use seconds due to how duration is managed
					s.Phases["test_phase_2"].ItemsByKey[subscriptiontestutils.ExampleFeatureKey][0].CadenceOverrideRelativeToPhaseStart.ActiveToOverride = lo.ToPtr(testutils.GetISODuration(t, "PT86400S"))

					return *s
				},
			},
		},
	}

	suite.Run(t)
}
