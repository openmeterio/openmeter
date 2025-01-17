package patch_test

import (
	"fmt"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
)

func TestRemoveDiscount(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2021-01-01T00:00:01Z")
	clock.SetTime(now)

	s, p := getDefaultSpec(t, now)

	getSpec := func(_ *testing.T) *subscription.SubscriptionSpec {
		return s
	}

	tests := testsuite[patch.PatchRemoveDiscount]{
		SystemTime: now,
		TT: []testcase[patch.PatchRemoveDiscount]{
			{
				Name: "Invalid phase",
				Patch: patch.PatchRemoveDiscount{
					PhaseKey:    "invalid_phase",
					RemoveAtIdx: 0,
				},
				GetSpec: getSpec,
				Ctx: subscription.ApplyContext{
					CurrentTime: now,
				},
				ExpectedError: &subscription.PatchConflictError{Msg: "phase invalid_phase not found"},
			},
			{
				Name: "Cannot remove discount from previous phase",
				Patch: patch.PatchRemoveDiscount{
					PhaseKey:    p.Phases[0].Key,
					RemoveAtIdx: 0,
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s := getSpec(t)

					// Let's make sure the spec looks as we expect
					require.Equal(t, "test_phase_1", p.Phases[0].Key)
					require.GreaterOrEqual(t, len(s.Phases), 2)
					p2, ok := s.Phases["test_phase_2"]
					require.True(t, ok)

					pstrt, _ := p2.StartAfter.AddTo(s.ActiveFrom)
					require.True(t, now.AddDate(0, 1, 1).After(pstrt))

					pKey := p.Phases[0].Key
					s.Phases[pKey].Discounts = []subscription.DiscountSpec{
						{
							PhaseKey: pKey,
							Discount: productcatalog.NewDiscountFrom(productcatalog.PercentageDiscount{
								Percentage: alpacadecimal.NewFromInt(10),
								RateCards:  []string{},
							}),
						},
					}

					return s
				},
				Ctx: subscription.ApplyContext{
					// We're doing this edit during the 2nd phase
					CurrentTime: now.AddDate(0, 1, 2),
				},
				ExpectedError: &subscription.PatchForbiddenError{Msg: fmt.Sprintf("cannot change contents of phase %s which starts before current phase", p.Phases[0].Key)},
			},
			{
				Name: "Cannot remove discount from old subscription (where everything is in the past)",
				Patch: patch.PatchRemoveDiscount{
					PhaseKey:    p.Phases[0].Key,
					RemoveAtIdx: 0,
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s := getSpec(t)

					pKey := p.Phases[0].Key
					s.Phases[pKey].Discounts = []subscription.DiscountSpec{
						{
							PhaseKey: pKey,
							Discount: productcatalog.NewDiscountFrom(productcatalog.PercentageDiscount{
								Percentage: alpacadecimal.NewFromInt(10),
								RateCards:  []string{},
							}),
						},
					}

					s.ActiveTo = lo.ToPtr(s.ActiveFrom.AddDate(0, 6, 0))

					return s
				},
				Ctx: subscription.ApplyContext{
					CurrentTime: now.AddDate(1, 0, 0),
				},
				ExpectedError: &subscription.PatchForbiddenError{Msg: fmt.Sprintf("cannot change contents of phase %s which starts before current phase", p.Phases[0].Key)},
			},
			{
				Name: "Cannot remove discount from current phase that doesnt have discounts",
				Patch: patch.PatchRemoveDiscount{
					PhaseKey:    "test_phase_2",
					RemoveAtIdx: 0,
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s := getSpec(t)

					// Let's make sure the spec looks as we expect
					require.GreaterOrEqual(t, len(s.Phases), 3)
					p2, ok := s.Phases["test_phase_2"]
					require.True(t, ok)

					p3, ok := s.Phases["test_phase_3"]
					require.True(t, ok)

					p2strt, _ := p2.StartAfter.AddTo(s.ActiveFrom)
					require.True(t, now.AddDate(0, 1, 1).After(p2strt))

					p3strt, _ := p3.StartAfter.AddTo(s.ActiveFrom)
					require.True(t, now.AddDate(0, 2, 2).Before(p3strt))

					return s
				},
				Ctx: subscription.ApplyContext{
					CurrentTime: now.AddDate(0, 2, 2),
				},
				ExpectedError: &subscription.PatchValidationError{Msg: "index 0 out of bounds for 0 items"},
			},
			{
				Name: "Cannot remove non-existent discount from current phase",
				Patch: patch.PatchRemoveDiscount{
					PhaseKey:    "test_phase_2",
					RemoveAtIdx: 2,
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s := getSpec(t)

					// Let's make sure the spec looks as we expect
					require.GreaterOrEqual(t, len(s.Phases), 3)
					p2, ok := s.Phases["test_phase_2"]
					require.True(t, ok)

					p3, ok := s.Phases["test_phase_3"]
					require.True(t, ok)

					p2strt, _ := p2.StartAfter.AddTo(s.ActiveFrom)
					require.True(t, now.AddDate(0, 1, 1).After(p2strt))

					p3strt, _ := p3.StartAfter.AddTo(s.ActiveFrom)
					require.True(t, now.AddDate(0, 2, 2).Before(p3strt))

					// Let's add a discount to the phase
					s.Phases["test_phase_2"].Discounts = []subscription.DiscountSpec{
						{
							PhaseKey: "test_phase_2",
							Discount: productcatalog.NewDiscountFrom(productcatalog.PercentageDiscount{
								Percentage: alpacadecimal.NewFromInt(10),
								RateCards:  []string{},
							}),
						},
					}

					return s
				},
				Ctx: subscription.ApplyContext{
					CurrentTime: now.AddDate(0, 2, 2),
				},
				ExpectedError: &subscription.PatchValidationError{Msg: "index 2 out of bounds for 1 items"},
			},
			{
				Name: "Cannot remove non-existent discount from current phase",
				Patch: patch.PatchRemoveDiscount{
					PhaseKey:    "test_phase_2",
					RemoveAtIdx: 2,
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s := getSpec(t)

					// Let's make sure the spec looks as we expect
					require.GreaterOrEqual(t, len(s.Phases), 3)
					p2, ok := s.Phases["test_phase_2"]
					require.True(t, ok)

					p3, ok := s.Phases["test_phase_3"]
					require.True(t, ok)

					p2strt, _ := p2.StartAfter.AddTo(s.ActiveFrom)
					require.True(t, now.AddDate(0, 1, 1).After(p2strt))

					p3strt, _ := p3.StartAfter.AddTo(s.ActiveFrom)
					require.True(t, now.AddDate(0, 2, 2).Before(p3strt))

					// Let's add a discount to the phase
					s.Phases["test_phase_2"].Discounts = []subscription.DiscountSpec{
						{
							PhaseKey: "test_phase_2",
							Discount: productcatalog.NewDiscountFrom(productcatalog.PercentageDiscount{
								Percentage: alpacadecimal.NewFromInt(10),
								RateCards:  []string{},
							}),
						},
					}

					return s
				},
				Ctx: subscription.ApplyContext{
					CurrentTime: now.AddDate(0, 2, 2),
				},
				ExpectedError: &subscription.PatchValidationError{Msg: "index 2 out of bounds for 1 items"},
			},
			{
				Name: "Should remove discount from future phase",
				Patch: patch.PatchRemoveDiscount{
					PhaseKey:    "test_phase_2",
					RemoveAtIdx: 0,
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s := getSpec(t)

					// Let's make sure the spec looks as we expect
					p2, ok := s.Phases["test_phase_2"]
					require.True(t, ok)

					p2strt, _ := p2.StartAfter.AddTo(s.ActiveFrom)
					require.True(t, now.Before(p2strt))

					// Let's add a discount to the phase
					s.Phases["test_phase_2"].Discounts = []subscription.DiscountSpec{
						{
							PhaseKey: "test_phase_2",
							Discount: productcatalog.NewDiscountFrom(productcatalog.PercentageDiscount{
								Percentage: alpacadecimal.NewFromInt(10),
								RateCards:  []string{},
							}),
						},
					}

					return s
				},
				Ctx: subscription.ApplyContext{
					CurrentTime: now,
				},
				GetExpectedSpec: func(t *testing.T) subscription.SubscriptionSpec {
					s := getSpec(t)

					// Note we're not adding anything = we expect it to be removed

					return *s
				},
			},
			{
				Name: "Should remove discount from current phase",
				Patch: patch.PatchRemoveDiscount{
					PhaseKey:    "test_phase_2",
					RemoveAtIdx: 0,
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s := getSpec(t)

					// Let's make sure the spec looks as we expect
					p2, ok := s.Phases["test_phase_2"]
					require.True(t, ok)

					p2strt, _ := p2.StartAfter.AddTo(s.ActiveFrom)
					require.True(t, now.Before(p2strt))

					// Let's add a discount to the phase
					s.Phases["test_phase_2"].Discounts = []subscription.DiscountSpec{
						{
							PhaseKey: "test_phase_2",
							Discount: productcatalog.NewDiscountFrom(productcatalog.PercentageDiscount{
								Percentage: alpacadecimal.NewFromInt(10),
								RateCards:  []string{},
							}),
						},
					}

					return s
				},
				Ctx: subscription.ApplyContext{
					CurrentTime: now.AddDate(0, 2, 2),
				},
				GetExpectedSpec: func(t *testing.T) subscription.SubscriptionSpec {
					s := getSpec(t)

					// Note we're not adding anything = we expect it to be removed

					return *s
				},
			},
		},
	}

	tests.Run(t)
}
