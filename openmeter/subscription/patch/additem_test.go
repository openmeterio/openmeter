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
	"github.com/openmeterio/openmeter/pkg/datetime"
)

func TestAddItem(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2021-01-01T00:00:01Z")
	clock.SetTime(now)

	s, p := getDefaultSpec(t, now)

	getSpec := func(_ *testing.T) *subscription.SubscriptionSpec {
		return s
	}

	suite := testsuite[patch.PatchAddItem]{
		SystemTime: now,
		TT: []testcase[patch.PatchAddItem]{
			{
				Name: "Invalid phase",
				Patch: patch.PatchAddItem{
					PhaseKey:    "invalid_phase",
					ItemKey:     "test_item",
					CreateInput: subscription.SubscriptionItemSpec{},
				},
				GetSpec: getSpec,
				Ctx: subscription.ApplyContext{
					CurrentTime: now,
				},
				ExpectedError: &subscription.PatchValidationError{Msg: "phase invalid_phase not found"},
			},
			{
				Name: "Cannot add item to previous phase",
				Patch: patch.PatchAddItem{
					PhaseKey:    p.Phases[0].Key,
					ItemKey:     "new_key",
					CreateInput: subscription.SubscriptionItemSpec{},
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
				ExpectedError: &subscription.PatchForbiddenError{Msg: fmt.Sprintf("cannot add item to phase %s which starts before current phase", p.Phases[0].Key)},
			},
			{
				Name: "Cannot add item to old subscription (where everything is in the past)",
				Patch: patch.PatchAddItem{
					PhaseKey:    p.Phases[0].Key,
					ItemKey:     "new_item",
					CreateInput: subscription.SubscriptionItemSpec{},
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s := getSpec(t)

					s.ActiveTo = lo.ToPtr(s.ActiveFrom.AddDate(0, 6, 0))

					return s
				},
				Ctx: subscription.ApplyContext{
					CurrentTime: now.AddDate(1, 0, 0),
				},
				ExpectedError: &subscription.PatchForbiddenError{Msg: fmt.Sprintf("cannot add item to phase %s which starts before current phase", p.Phases[0].Key)},
			},
			{
				Name: "Cannot add item to current phase which would become active in the past",
				Patch: patch.PatchAddItem{
					PhaseKey: "test_phase_2",
					ItemKey:  "new_item",
					CreateInput: subscription.SubscriptionItemSpec{
						CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
							CreateSubscriptionItemCustomerInput: subscription.CreateSubscriptionItemCustomerInput{
								ActiveFromOverrideRelativeToPhaseStart: lo.ToPtr(datetime.MustParseDuration(t, "P1D")),
							},
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
				ExpectedError: &subscription.PatchForbiddenError{Msg: fmt.Sprintf("cannot add item to phase %s which would become active in the past at %v", p.Phases[1].Key, now.AddDate(0, 1, 1))},
			},
			{
				Name: "Should add item to spec",
				Patch: patch.PatchAddItem{
					PhaseKey: "test_phase_3",
					ItemKey:  "new_item_key",
					CreateInput: subscription.SubscriptionItemSpec{
						CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
							CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
								ItemKey:  "new_item_key",
								PhaseKey: "test_phase_3",
								RateCard: &productcatalog.FlatFeeRateCard{
									RateCardMeta: productcatalog.RateCardMeta{
										Name: "New Rate Card",
										Key:  "new_item_key",
										Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
											Amount:      alpacadecimal.NewFromInt(100),
											PaymentTerm: productcatalog.InAdvancePaymentTerm,
										}),
									},
								},
							},
						},
					},
				},
				GetSpec: func(t *testing.T) *subscription.SubscriptionSpec {
					s := getSpec(t)

					// Let's make sure the spec looks as we expect
					require.GreaterOrEqual(t, len(s.Phases), 3)
					p3, ok := s.Phases["test_phase_3"]
					require.True(t, ok)

					p2strt, _ := p3.StartAfter.AddTo(s.ActiveFrom)
					require.True(t, now.Before(p2strt))

					return s
				},
				Ctx: subscription.ApplyContext{
					CurrentTime: now,
				},
				GetExpectedSpec: func(t *testing.T) subscription.SubscriptionSpec {
					s := getSpec(t)

					s.Phases["test_phase_3"].ItemsByKey["new_item_key"] = []*subscription.SubscriptionItemSpec{
						{
							CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
								CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
									ItemKey:  "new_item_key",
									PhaseKey: "test_phase_3",
									RateCard: &productcatalog.FlatFeeRateCard{
										RateCardMeta: productcatalog.RateCardMeta{
											Name: "New Rate Card",
											Key:  "new_item_key",
											Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
												Amount:      alpacadecimal.NewFromInt(100),
												PaymentTerm: productcatalog.InAdvancePaymentTerm,
											}),
										},
									},
								},
							},
						},
					}

					return *s
				},
				ExpectedError: nil,
			},
			{
				Name: "Should add item to spec and close current item under same key for current phase",
				Patch: patch.PatchAddItem{
					PhaseKey: "test_phase_2",
					ItemKey:  "rate-card-2",
					CreateInput: subscription.SubscriptionItemSpec{
						CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
							CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
								ItemKey:  "new_item_key",
								PhaseKey: "rate-card-2",
								RateCard: &productcatalog.FlatFeeRateCard{
									RateCardMeta: productcatalog.RateCardMeta{
										Name: "New Rate Card",
										Key:  "new_item_key",
										Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
											Amount:      alpacadecimal.NewFromInt(100),
											PaymentTerm: productcatalog.InAdvancePaymentTerm,
										}),
									},
								},
							},
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

					currTime := now.AddDate(0, 1, 1)

					p2strt, _ := p2.StartAfter.AddTo(s.ActiveFrom)
					require.True(t, currTime.After(p2strt))

					p3strt, _ := p3.StartAfter.AddTo(s.ActiveFrom)
					require.True(t, currTime.Before(p3strt))

					v, ok := s.Phases["test_phase_2"].ItemsByKey["rate-card-2"]
					require.True(t, ok)
					require.Len(t, v, 1)

					return s
				},
				Ctx: subscription.ApplyContext{
					CurrentTime: now.AddDate(0, 1, 1),
				},
				GetExpectedSpec: func(t *testing.T) subscription.SubscriptionSpec {
					s := getSpec(t)

					updatedRc := s.Phases["test_phase_2"].ItemsByKey["rate-card-2"][0]

					// We have to use seconds here as diff resolution will be in seconds
					updatedRc.ActiveToOverrideRelativeToPhaseStart = lo.ToPtr(datetime.MustParseDuration(t, "PT86400S"))

					s.Phases["test_phase_2"].ItemsByKey["rate-card-2"] = []*subscription.SubscriptionItemSpec{
						updatedRc,
						{
							CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
								CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
									ItemKey:  "new_item_key",
									PhaseKey: "rate-card-2",
									RateCard: &productcatalog.FlatFeeRateCard{
										RateCardMeta: productcatalog.RateCardMeta{
											Name: "New Rate Card",
											Key:  "new_item_key",
											Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
												Amount:      alpacadecimal.NewFromInt(100),
												PaymentTerm: productcatalog.InAdvancePaymentTerm,
											}),
										},
									},
								},
								CreateSubscriptionItemCustomerInput: subscription.CreateSubscriptionItemCustomerInput{
									ActiveFromOverrideRelativeToPhaseStart: lo.ToPtr(datetime.MustParseDuration(t, "PT86400S")),
								},
							},
						},
					}

					return *s
				},
				ExpectedError: nil,
			},
		},
	}

	suite.Run(t)
}
