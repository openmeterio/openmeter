package patch_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	psubs "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddItem(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2021-01-01T00:00:01Z")
	clock.SetTime(now)

	pInp := subscriptiontestutils.GetExamplePlanInput(t)
	p := &psubs.Plan{
		Plan: pInp.Plan,
		Ref:  &models.NamespacedID{Namespace: pInp.Namespace, ID: pInp.Key},
	}

	getSpec := func(t *testing.T) *subscription.SubscriptionSpec {
		spec, err := subscription.NewSpecFromPlan(p, subscription.CreateSubscriptionCustomerInput{
			Name:       "Test Plan",
			CustomerId: "test_customer",
			Currency:   currencyx.Code("USD"),
			ActiveFrom: now,
		})
		require.Nil(t, err)

		return &spec
	}

	tt := []struct {
		name            string
		patch           patch.PatchAddItem
		getSpec         func(t *testing.T) *subscription.SubscriptionSpec
		ctx             subscription.ApplyContext
		getExpectedSpec func(t *testing.T) subscription.SubscriptionSpec
		expectedError   error
	}{
		{
			name: "Invalid phase",
			patch: patch.PatchAddItem{
				PhaseKey:    "invalid_phase",
				ItemKey:     "test_item",
				CreateInput: subscription.SubscriptionItemSpec{},
			},
			getSpec: getSpec,
			ctx: subscription.ApplyContext{
				CurrentTime: now,
			},
			expectedError: &subscription.PatchValidationError{Msg: "phase invalid_phase not found"},
		},
		{
			name: "Cannot add item to previous phase",
			patch: patch.PatchAddItem{
				PhaseKey:    p.Phases[0].Key,
				ItemKey:     "new_key",
				CreateInput: subscription.SubscriptionItemSpec{},
			},
			getSpec: func(t *testing.T) *subscription.SubscriptionSpec {
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
			ctx: subscription.ApplyContext{
				// We're doing this edit during the 2nd phase
				CurrentTime: now.AddDate(0, 1, 2),
			},
			expectedError: &subscription.PatchForbiddenError{Msg: fmt.Sprintf("cannot add item to phase %s which starts before current phase", p.Phases[0].Key)},
		},
		{
			name: "Cannot add item to old subscription (where everything is in the past)",
			patch: patch.PatchAddItem{
				PhaseKey:    p.Phases[0].Key,
				ItemKey:     "new_item",
				CreateInput: subscription.SubscriptionItemSpec{},
			},
			getSpec: func(t *testing.T) *subscription.SubscriptionSpec {
				s := getSpec(t)

				s.ActiveTo = lo.ToPtr(s.ActiveFrom.AddDate(0, 6, 0))

				return s
			},
			ctx: subscription.ApplyContext{
				CurrentTime: now.AddDate(1, 0, 0),
			},
			expectedError: &subscription.PatchForbiddenError{Msg: fmt.Sprintf("cannot add item to phase %s which starts before current phase", p.Phases[0].Key)},
		},
		{
			name: "Cannot add item to current phase which would become active in the past",
			patch: patch.PatchAddItem{
				PhaseKey: "test_phase_2",
				ItemKey:  "new_item",
				CreateInput: subscription.SubscriptionItemSpec{
					CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
						CreateSubscriptionItemCustomerInput: subscription.CreateSubscriptionItemCustomerInput{
							ActiveFromOverrideRelativeToPhaseStart: lo.ToPtr(testutils.GetISODuration(t, "P1D")),
						},
					},
				},
			},
			getSpec: func(t *testing.T) *subscription.SubscriptionSpec {
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
			ctx: subscription.ApplyContext{
				CurrentTime: now.AddDate(0, 2, 2),
			},
			expectedError: &subscription.PatchForbiddenError{Msg: fmt.Sprintf("cannot add item to phase %s which would become active in the past at %v", p.Phases[1].Key, now.AddDate(0, 1, 1))},
		},
		{
			name: "Should add item to spec",
			patch: patch.PatchAddItem{
				PhaseKey: "test_phase_3",
				ItemKey:  "new_item_key",
				CreateInput: subscription.SubscriptionItemSpec{
					CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
						CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
							ItemKey:  "new_item_key",
							PhaseKey: "test_phase_3",
							RateCard: subscription.RateCard{
								Name: "New Rate Card",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromInt(100),
									PaymentTerm: productcatalog.InAdvancePaymentTerm,
								}),
							},
						},
					},
				},
			},
			getSpec: func(t *testing.T) *subscription.SubscriptionSpec {
				s := getSpec(t)

				// Let's make sure the spec looks as we expect
				require.GreaterOrEqual(t, len(s.Phases), 3)
				p3, ok := s.Phases["test_phase_3"]
				require.True(t, ok)

				p2strt, _ := p3.StartAfter.AddTo(s.ActiveFrom)
				require.True(t, now.Before(p2strt))

				return s
			},
			ctx: subscription.ApplyContext{
				CurrentTime: now,
			},
			getExpectedSpec: func(t *testing.T) subscription.SubscriptionSpec {
				s := getSpec(t)

				s.Phases["test_phase_3"].ItemsByKey["new_item_key"] = []*subscription.SubscriptionItemSpec{
					{
						CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
							CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
								ItemKey:  "new_item_key",
								PhaseKey: "test_phase_3",
								RateCard: subscription.RateCard{
									Name: "New Rate Card",
									Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
										Amount:      alpacadecimal.NewFromInt(100),
										PaymentTerm: productcatalog.InAdvancePaymentTerm,
									}),
								},
							},
						},
					},
				}

				return *s
			},
			expectedError: nil,
		},
		{
			name: "Should add item to spec and close current item under same key for current phase",
			patch: patch.PatchAddItem{
				PhaseKey: "test_phase_2",
				ItemKey:  "rate-card-2",
				CreateInput: subscription.SubscriptionItemSpec{
					CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
						CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
							ItemKey:  "new_item_key",
							PhaseKey: "rate-card-2",
							RateCard: subscription.RateCard{
								Name: "New Rate Card",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromInt(100),
									PaymentTerm: productcatalog.InAdvancePaymentTerm,
								}),
							},
						},
					},
				},
			},
			getSpec: func(t *testing.T) *subscription.SubscriptionSpec {
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
			ctx: subscription.ApplyContext{
				CurrentTime: now.AddDate(0, 1, 1),
			},
			getExpectedSpec: func(t *testing.T) subscription.SubscriptionSpec {
				s := getSpec(t)

				updatedRc := s.Phases["test_phase_2"].ItemsByKey["rate-card-2"][0]

				// We have to use seconds here as diff resolution will be in seconds
				updatedRc.ActiveToOverrideRelativeToPhaseStart = lo.ToPtr(testutils.GetISODuration(t, "PT86400S"))

				s.Phases["test_phase_2"].ItemsByKey["rate-card-2"] = []*subscription.SubscriptionItemSpec{
					updatedRc,
					{
						CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
							CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
								ItemKey:  "new_item_key",
								PhaseKey: "rate-card-2",
								RateCard: subscription.RateCard{
									Name: "New Rate Card",
									Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
										Amount:      alpacadecimal.NewFromInt(100),
										PaymentTerm: productcatalog.InAdvancePaymentTerm,
									}),
								},
							},
							CreateSubscriptionItemCustomerInput: subscription.CreateSubscriptionItemCustomerInput{
								ActiveFromOverrideRelativeToPhaseStart: lo.ToPtr(testutils.GetISODuration(t, "PT86400S")),
							},
						},
					},
				}

				return *s
			},
			expectedError: nil,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			spec := tc.getSpec(t)
			err := tc.patch.ApplyTo(spec, tc.ctx)

			if tc.expectedError == nil {
				assert.NoError(t, err)
			} else {
				assert.True(t, errors.As(err, lo.ToPtr(tc.expectedError.(any))))
				assert.EqualError(t, err, tc.expectedError.Error())
			}

			if err == nil {
				require.NotNil(t, tc.getExpectedSpec)
				expectedSpec := tc.getExpectedSpec(t)
				if !reflect.DeepEqual(spec, &expectedSpec) {
					t.Errorf("expected: %+v, got: %+v", expectedSpec, spec)
				}
			}
		})
	}
}
