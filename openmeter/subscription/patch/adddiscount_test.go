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
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
)

func TestAddDiscount(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2021-01-01T00:00:01Z")
	clock.SetTime(now)

	s, p := getDefaultSpec(t, now)

	getSpec := func(_ *testing.T) *subscription.SubscriptionSpec {
		return s
	}

	tests := testsuite[patch.PatchAddDiscount]{
		SystemTime: now,
		TT: []testcase[patch.PatchAddDiscount]{
			{
				Name: "Invalid phase",
				Patch: patch.PatchAddDiscount{
					PhaseKey:    "invalid_phase",
					InsertAt:    0,
					CreateInput: subscription.DiscountSpec{},
				},
				GetSpec: getSpec,
				Ctx: subscription.ApplyContext{
					CurrentTime: now,
				},
				ExpectedError: &subscription.PatchConflictError{Msg: "phase invalid_phase not found"},
			},
			{
				Name: "Cannot add discount to previous phase",
				Patch: patch.PatchAddDiscount{
					PhaseKey:    p.Phases[0].Key,
					InsertAt:    0,
					CreateInput: subscription.DiscountSpec{},
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

					return s
				},
				Ctx: subscription.ApplyContext{
					// We're doing this edit during the 2nd phase
					CurrentTime: now.AddDate(0, 1, 2),
				},
				ExpectedError: &subscription.PatchForbiddenError{Msg: fmt.Sprintf("cannot change contents of phase %s which starts before current phase", p.Phases[0].Key)},
			},
			{
				Name: "Cannot add discount to old subscription (where everything is in the past)",
				Patch: patch.PatchAddDiscount{
					PhaseKey:    p.Phases[0].Key,
					InsertAt:    0,
					CreateInput: subscription.DiscountSpec{},
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s := getSpec(t)

					s.ActiveTo = lo.ToPtr(s.ActiveFrom.AddDate(0, 6, 0))

					return s
				},
				Ctx: subscription.ApplyContext{
					CurrentTime: now.AddDate(1, 0, 0),
				},
				ExpectedError: &subscription.PatchForbiddenError{Msg: fmt.Sprintf("cannot change contents of phase %s which starts before current phase", p.Phases[0].Key)},
			},
			{
				Name: "Cannot add discount to current phase which would become active in the past",
				Patch: patch.PatchAddDiscount{
					PhaseKey: "test_phase_2",
					InsertAt: 0,
					CreateInput: subscription.DiscountSpec{
						CadenceOverrideRelativeToPhaseStart: subscription.CadenceOverrideRelativeToPhaseStart{
							ActiveFromOverride: lo.ToPtr(testutils.GetISODuration(t, "P1D")),
						},
					},
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
				ExpectedError: &subscription.PatchForbiddenError{Msg: fmt.Sprintf("cannot add discount to phase %s which would become active in the past at %v", p.Phases[1].Key, now.AddDate(0, 1, 1))},
			},
			{
				Name: "Cannot add discount which references non-existent item",
				Patch: patch.PatchAddDiscount{
					PhaseKey: "test_phase_2",
					InsertAt: 0,
					CreateInput: subscription.DiscountSpec{
						PhaseKey: "test_phase_2",
						Discount: productcatalog.NewDiscountFrom(productcatalog.PercentageDiscount{
							Percentage: alpacadecimal.NewFromInt(10),
							RateCards:  []string{"non_existent_item"},
						}),
					},
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s := getSpec(t)

					// Let's make sure the spec looks as we expect
					require.GreaterOrEqual(t, len(s.Phases), 2)
					p2, ok := s.Phases["test_phase_2"]
					require.True(t, ok)

					// Let's make sure p2 doesn't have the item we're referencing
					require.NotContains(t, p2.ItemsByKey, "non_existent_item")

					return s
				},
				Ctx: subscription.ApplyContext{
					CurrentTime: now,
				},
				ExpectedError: &subscription.PatchConflictError{Msg: "item non_existent_item not found"},
			},
			{
				Name: "Cannot add discount which references non-existent item (among existing items)",
				Patch: patch.PatchAddDiscount{
					PhaseKey: "test_phase_2",
					InsertAt: 0,
					CreateInput: subscription.DiscountSpec{
						PhaseKey: "test_phase_2",
						Discount: productcatalog.NewDiscountFrom(productcatalog.PercentageDiscount{
							Percentage: alpacadecimal.NewFromInt(10),
							RateCards:  []string{subscriptiontestutils.ExampleFeatureKey, "non_existent_item"},
						}),
					},
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s := getSpec(t)

					// Let's make sure the spec looks as we expect
					require.GreaterOrEqual(t, len(s.Phases), 2)
					p2, ok := s.Phases["test_phase_2"]
					require.True(t, ok)

					// Let's make sure p2 doesn't have the item we're referencing
					require.NotContains(t, p2.ItemsByKey, "non_existent_item")
					require.Contains(t, p2.ItemsByKey, subscriptiontestutils.ExampleFeatureKey)

					return s
				},
				Ctx: subscription.ApplyContext{
					CurrentTime: now,
				},
				ExpectedError: &subscription.PatchConflictError{Msg: "item non_existent_item not found"},
			},
			{
				Name: "Should add discount to a phase without discounts",
				Patch: patch.PatchAddDiscount{
					PhaseKey: "test_phase_2",
					InsertAt: 0,
					CreateInput: subscription.DiscountSpec{
						PhaseKey: "test_phase_2",
						Discount: productcatalog.NewDiscountFrom(productcatalog.PercentageDiscount{
							Percentage: alpacadecimal.NewFromInt(10),
							RateCards:  []string{subscriptiontestutils.ExampleFeatureKey},
						}),
					},
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s := getSpec(t)

					return s
				},
				Ctx: subscription.ApplyContext{
					CurrentTime: now,
				},
				GetExpectedSpec: func(t *testing.T) subscription.SubscriptionSpec {
					s := getSpec(t)

					s.Phases["test_phase_2"].Discounts = []subscription.DiscountSpec{
						{
							PhaseKey: "test_phase_2",
							Discount: productcatalog.NewDiscountFrom(productcatalog.PercentageDiscount{
								Percentage: alpacadecimal.NewFromInt(10),
								RateCards:  []string{subscriptiontestutils.ExampleFeatureKey},
							}),
						},
					}

					return *s
				},
			},
			{
				Name: "Should add discount to a phase with discounts - as first discount",
				Patch: patch.PatchAddDiscount{
					PhaseKey: "test_phase_2",
					InsertAt: 0,
					CreateInput: subscription.DiscountSpec{
						PhaseKey: "test_phase_2",
						Discount: productcatalog.NewDiscountFrom(productcatalog.PercentageDiscount{
							Percentage: alpacadecimal.NewFromInt(10),
							RateCards:  []string{subscriptiontestutils.ExampleFeatureKey},
						}),
					},
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s := getSpec(t)

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

					s.Phases["test_phase_2"].Discounts = []subscription.DiscountSpec{
						{
							PhaseKey: "test_phase_2",
							Discount: productcatalog.NewDiscountFrom(productcatalog.PercentageDiscount{
								Percentage: alpacadecimal.NewFromInt(10),
								RateCards:  []string{subscriptiontestutils.ExampleFeatureKey},
							}),
						},
						{
							PhaseKey: "test_phase_2",
							Discount: productcatalog.NewDiscountFrom(productcatalog.PercentageDiscount{
								Percentage: alpacadecimal.NewFromInt(10),
								RateCards:  []string{},
							}),
						},
					}

					return *s
				},
			},
			{
				Name: "Should add discount to a phase with discounts - as last discount",
				Patch: patch.PatchAddDiscount{
					PhaseKey: "test_phase_2",
					InsertAt: 1,
					CreateInput: subscription.DiscountSpec{
						PhaseKey: "test_phase_2",
						Discount: productcatalog.NewDiscountFrom(productcatalog.PercentageDiscount{
							Percentage: alpacadecimal.NewFromInt(10),
							RateCards:  []string{subscriptiontestutils.ExampleFeatureKey},
						}),
					},
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s := getSpec(t)

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

					s.Phases["test_phase_2"].Discounts = []subscription.DiscountSpec{
						{
							PhaseKey: "test_phase_2",
							Discount: productcatalog.NewDiscountFrom(productcatalog.PercentageDiscount{
								Percentage: alpacadecimal.NewFromInt(10),
								RateCards:  []string{},
							}),
						},
						{
							PhaseKey: "test_phase_2",
							Discount: productcatalog.NewDiscountFrom(productcatalog.PercentageDiscount{
								Percentage: alpacadecimal.NewFromInt(10),
								RateCards:  []string{subscriptiontestutils.ExampleFeatureKey},
							}),
						},
					}

					return *s
				},
			},
			{
				Name: "Should add discount to a phase with discounts - as last discount with index too large",
				Patch: patch.PatchAddDiscount{
					PhaseKey: "test_phase_2",
					InsertAt: 100,
					CreateInput: subscription.DiscountSpec{
						PhaseKey: "test_phase_2",
						Discount: productcatalog.NewDiscountFrom(productcatalog.PercentageDiscount{
							Percentage: alpacadecimal.NewFromInt(10),
							RateCards:  []string{subscriptiontestutils.ExampleFeatureKey},
						}),
					},
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s := getSpec(t)

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

					s.Phases["test_phase_2"].Discounts = []subscription.DiscountSpec{
						{
							PhaseKey: "test_phase_2",
							Discount: productcatalog.NewDiscountFrom(productcatalog.PercentageDiscount{
								Percentage: alpacadecimal.NewFromInt(10),
								RateCards:  []string{},
							}),
						},
						{
							PhaseKey: "test_phase_2",
							Discount: productcatalog.NewDiscountFrom(productcatalog.PercentageDiscount{
								Percentage: alpacadecimal.NewFromInt(10),
								RateCards:  []string{subscriptiontestutils.ExampleFeatureKey},
							}),
						},
					}

					return *s
				},
			},
			{
				Name: "Should add discount to a phase with discounts - as an intermediate discount",
				Patch: patch.PatchAddDiscount{
					PhaseKey: "test_phase_2",
					InsertAt: 1,
					CreateInput: subscription.DiscountSpec{
						PhaseKey: "test_phase_2",
						Discount: productcatalog.NewDiscountFrom(productcatalog.PercentageDiscount{
							Percentage: alpacadecimal.NewFromInt(2),
							RateCards:  []string{},
						}),
					},
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s := getSpec(t)

					s.Phases["test_phase_2"].Discounts = []subscription.DiscountSpec{
						{
							PhaseKey: "test_phase_2",
							Discount: productcatalog.NewDiscountFrom(productcatalog.PercentageDiscount{
								Percentage: alpacadecimal.NewFromInt(10),
								RateCards:  []string{},
							}),
						},
						{
							PhaseKey: "test_phase_2",
							Discount: productcatalog.NewDiscountFrom(productcatalog.PercentageDiscount{
								Percentage: alpacadecimal.NewFromInt(5),
								RateCards:  []string{subscriptiontestutils.ExampleFeatureKey},
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

					s.Phases["test_phase_2"].Discounts = []subscription.DiscountSpec{
						{
							PhaseKey: "test_phase_2",
							Discount: productcatalog.NewDiscountFrom(productcatalog.PercentageDiscount{
								Percentage: alpacadecimal.NewFromInt(10),
								RateCards:  []string{},
							}),
						},
						{
							PhaseKey: "test_phase_2",
							Discount: productcatalog.NewDiscountFrom(productcatalog.PercentageDiscount{
								Percentage: alpacadecimal.NewFromInt(2),
								RateCards:  []string{},
							}),
						},
						{
							PhaseKey: "test_phase_2",
							Discount: productcatalog.NewDiscountFrom(productcatalog.PercentageDiscount{
								Percentage: alpacadecimal.NewFromInt(5),
								RateCards:  []string{subscriptiontestutils.ExampleFeatureKey},
							}),
						},
					}

					return *s
				},
			},
		},
	}

	tests.Run(t)
}
