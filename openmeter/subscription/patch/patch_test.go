package patch_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	psubs "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestRemoveAdd(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2021-01-01T00:00:01Z")
	clock.SetTime(now)

	t.Run("Can remove then add an item in a future phase", func(t *testing.T) {
		s, _ := getDefaultSpec(t, now)

		// Let's validate the spec looks as expected
		require.GreaterOrEqual(t, len(s.Phases), 3)

		p3, ok := s.Phases["test_phase_3"]
		require.True(t, ok)

		require.GreaterOrEqual(t, len(p3.ItemsByKey), 1)

		v, ok := p3.ItemsByKey[subscriptiontestutils.ExampleFeatureKey]
		require.True(t, ok)
		require.GreaterOrEqual(t, len(v), 1)

		// Let's remove an item from the last phase
		rmP := &patch.PatchRemoveItem{
			PhaseKey: "test_phase_3",
			ItemKey:  subscriptiontestutils.ExampleFeatureKey,
		}

		// Then add it back with changes
		nSpec := *v[0]
		nSpec.CreateSubscriptionItemPlanInput.RateCard.Name = "new_name"

		assert.NotEqual(t, "new_name", s.Phases["test_phase_3"].ItemsByKey[subscriptiontestutils.ExampleFeatureKey][0].RateCard.Name)

		addP := &patch.PatchAddItem{
			PhaseKey:    "test_phase_3",
			ItemKey:     subscriptiontestutils.ExampleFeatureKey,
			CreateInput: nSpec,
		}

		err := s.ApplyPatches(lo.Map([]subscription.Patch{rmP, addP}, subscription.ToApplies), subscription.ApplyContext{
			CurrentTime: now,
		})
		require.NoError(t, err)

		// Let's validate that the new version of the item is present
		found := s.Phases["test_phase_3"].ItemsByKey[subscriptiontestutils.ExampleFeatureKey][0]

		assert.Equal(t, "new_name", found.RateCard.Name)
	})

	t.Run("Can remove then add an item in the current phase", func(t *testing.T) {
		s, _ := getDefaultSpec(t, now)

		now := now.AddDate(0, 1, 1)
		latestNowWeGoTu := now.Add(time.Hour * 3)

		// Let's validate the spec looks as expected
		require.GreaterOrEqual(t, len(s.Phases), 3)

		// Let's make sure we are in the second phase
		p2, ok := s.Phases["test_phase_2"]
		require.True(t, ok)
		p3, ok := s.Phases["test_phase_3"]
		require.True(t, ok)

		p2st, _ := p2.StartAfter.AddTo(s.ActiveFrom)
		require.True(t, now.After(p2st))
		require.True(t, latestNowWeGoTu.After(p2st))
		p3st, _ := p3.StartAfter.AddTo(s.ActiveFrom)
		require.True(t, now.Before(p3st))
		require.True(t, latestNowWeGoTu.Before(p3st))

		// Let's make sure the phase has the item we're trying to remove
		require.GreaterOrEqual(t, len(p2.ItemsByKey), 1)
		v, ok := p2.ItemsByKey[subscriptiontestutils.ExampleFeatureKey]
		require.True(t, ok)
		require.GreaterOrEqual(t, len(v), 1)

		// Let's remove an item from the second phase
		rmP := &patch.PatchRemoveItem{
			PhaseKey: "test_phase_2",
			ItemKey:  subscriptiontestutils.ExampleFeatureKey,
		}

		// Then add it back with changes
		nSpec := *v[0]
		nSpec.CreateSubscriptionItemPlanInput.RateCard.Name = "new_name"

		assert.NotEqual(t, "new_name", s.Phases["test_phase_2"].ItemsByKey[subscriptiontestutils.ExampleFeatureKey][0].RateCard.Name)

		addP := &patch.PatchAddItem{
			PhaseKey:    "test_phase_2",
			ItemKey:     subscriptiontestutils.ExampleFeatureKey,
			CreateInput: nSpec,
		}

		err := s.ApplyPatches(lo.Map([]subscription.Patch{rmP, addP}, subscription.ToApplies), subscription.ApplyContext{
			CurrentTime: now,
		})
		require.NoError(t, err)

		// Let's validate that the old version is kept expiring now, and the new version is added starting now
		found := s.Phases["test_phase_2"].ItemsByKey[subscriptiontestutils.ExampleFeatureKey]

		assert.Len(t, found, 2)
		assert.Equal(t, lo.ToPtr(testutils.GetISODuration(t, "PT86400S")), found[0].ActiveToOverrideRelativeToPhaseStart)
		assert.Equal(t, lo.ToPtr(testutils.GetISODuration(t, "PT86400S")), found[1].ActiveFromOverrideRelativeToPhaseStart)

		// Now lets simulate some time passing
		now = now.Add(time.Hour * 1)
		require.True(t, now.Before(latestNowWeGoTu))

		// And lets repeat the same process
		err = s.ApplyPatches(lo.Map([]subscription.Patch{rmP, addP}, subscription.ToApplies), subscription.ApplyContext{
			CurrentTime: now,
		})
		require.NoError(t, err)

		// Let's validate that the previous one was closed and the new one is added
		found = s.Phases["test_phase_2"].ItemsByKey[subscriptiontestutils.ExampleFeatureKey]

		assert.Len(t, found, 3)
		assert.Equal(t, lo.ToPtr(testutils.GetISODuration(t, "PT86400S")), found[0].ActiveToOverrideRelativeToPhaseStart)
		assert.Equal(t, lo.ToPtr(testutils.GetISODuration(t, "PT86400S")), found[1].ActiveFromOverrideRelativeToPhaseStart)
		// 90000s = 25h = 1d + 1h
		assert.Equal(t, lo.ToPtr(testutils.GetISODuration(t, "PT90000S")), found[1].ActiveToOverrideRelativeToPhaseStart)
		assert.Equal(t, lo.ToPtr(testutils.GetISODuration(t, "PT90000S")), found[2].ActiveFromOverrideRelativeToPhaseStart)
	})
}

// utils
type testcase[T subscription.Applies] struct {
	Name            string
	Patch           T
	GetSpec         func(t *testing.T) *subscription.SubscriptionSpec
	Ctx             subscription.ApplyContext
	GetExpectedSpec func(t *testing.T) subscription.SubscriptionSpec
	ExpectedError   error
}

type testsuite[T subscription.Applies] struct {
	SystemTime time.Time
	TT         []testcase[T]
}

func (ts *testsuite[T]) Run(t *testing.T) {
	for _, tc := range ts.TT {
		t.Run(tc.Name, func(t *testing.T) {
			spec := tc.GetSpec(t)
			err := tc.Patch.ApplyTo(spec, tc.Ctx)

			if tc.ExpectedError == nil {
				assert.NoError(t, err)
			} else {
				assert.True(t, errors.As(err, lo.ToPtr(tc.ExpectedError.(any))))
				assert.EqualError(t, err, tc.ExpectedError.Error())
			}

			if err == nil {
				require.NotNil(t, tc.GetExpectedSpec)
				expectedSpec := tc.GetExpectedSpec(t)
				if !reflect.DeepEqual(spec, &expectedSpec) {
					t.Errorf("expected: %+v, got: %+v", expectedSpec, spec)
				}
			}
		})
	}
}

func getDefaultSpec(t *testing.T, activeFrom time.Time) (*subscription.SubscriptionSpec, *psubs.Plan) {
	pInp := subscriptiontestutils.GetExamplePlanInput(t)
	p := &psubs.Plan{
		Plan: pInp.Plan,
		Ref:  &models.NamespacedID{Namespace: pInp.Namespace, ID: pInp.Key},
	}

	spec, err := subscription.NewSpecFromPlan(p, subscription.CreateSubscriptionCustomerInput{
		Name:       "Test Plan",
		CustomerId: "test_customer",
		Currency:   currencyx.Code("USD"),
		ActiveFrom: activeFrom,
	})
	require.Nil(t, err)

	return &spec, p
}
