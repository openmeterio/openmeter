package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	addondiff "github.com/openmeterio/openmeter/openmeter/subscription/addon/diff"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestAddAddon(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2025-04-01T00:00:00Z")

	runWithDeps := func(fn func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies)) func(t *testing.T) {
		return func(t *testing.T) {
			clock.SetTime(now)
			defer clock.ResetTime()

			dbDeps := subscriptiontestutils.SetupDBDeps(t)
			defer dbDeps.Cleanup(t)

			deps := subscriptiontestutils.NewService(t, dbDeps)
			fn(t, deps)
		}
	}

	t.Run("Should error on invalid input", runWithDeps(func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
		p, add := subscriptiontestutils.CreatePlanWithAddon(
			t,
			deps,
			subscriptiontestutils.GetExamplePlanInput(t),
			subscriptiontestutils.BuildAddonForTesting(t,
				productcatalog.EffectivePeriod{
					EffectiveFrom: &now,
					EffectiveTo:   nil,
				},
				productcatalog.AddonInstanceTypeSingle,
				subscriptiontestutils.ExampleAddonRateCard2.Clone(),
				subscriptiontestutils.ExampleAddonRateCard4.Clone(),
			),
		)

		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps, p, now)

		addonInp := subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add.ID,
			InitialQuantity: 0,
			Timing: subscription.Timing{
				Custom: &now,
			},
		}

		expectedErr := addonInp.Validate()
		require.NotNil(t, expectedErr)

		_, _, err := deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, addonInp)
		require.Error(t, err)

		require.True(t, models.IsGenericValidationError(err))
		require.ErrorContains(t, err, expectedErr.Error())
	}))

	t.Run("Should error if the subscription is inactive or in wrong state", runWithDeps(func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
		p, add := subscriptiontestutils.CreatePlanWithAddon(
			t,
			deps,
			subscriptiontestutils.GetExamplePlanInput(t),
			subscriptiontestutils.BuildAddonForTesting(t,
				productcatalog.EffectivePeriod{
					EffectiveFrom: &now,
					EffectiveTo:   nil,
				},
				productcatalog.AddonInstanceTypeSingle,
				subscriptiontestutils.ExampleAddonRateCard2.Clone(), // This will add a new item
				subscriptiontestutils.ExampleAddonRateCard4.Clone(), // This will extend existing items
			),
		)

		// Let's create a subscription from the plan
		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps, p, now)

		// Let's cancel the subscription
		_, err := deps.SubscriptionService.Cancel(context.Background(), subView.Subscription.NamespacedID, subscription.Timing{
			Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
		})
		require.NoError(t, err)

		// Let's add an addon to the subscription
		addonInp := subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add.ID,
			InitialQuantity: 1,
			Timing: subscription.Timing{
				Custom: &now,
			},
		}

		_, _, err = deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, addonInp)
		require.Error(t, err)
		require.ErrorAs(t, err, lo.ToPtr(&models.GenericForbiddenError{}))
		require.True(t, models.IsGenericForbiddenError(err))
		require.ErrorContains(t, err, "state canceled not allowed")
	}))

	t.Run("Should add a new addon to a subscription that already has a different addon", runWithDeps(func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
		_ = deps.FeatureConnector.CreateExampleFeatures(t)

		// Let's create a plan
		p, err := deps.PlanService.CreatePlan(context.Background(), subscriptiontestutils.BuildTestPlanInput(t).
			AddPhase(nil, &subscriptiontestutils.ExampleRateCard1).
			Build())
		require.Nil(t, err)
		require.NotNil(t, p)

		// Let's create two addons that are compatible
		addonInp := subscriptiontestutils.BuildAddonForTesting(t,
			productcatalog.EffectivePeriod{
				EffectiveFrom: &now,
				EffectiveTo:   nil,
			},
			productcatalog.AddonInstanceTypeSingle,
			subscriptiontestutils.ExampleAddonRateCard5.Clone(),
		)

		add1 := deps.AddonService.CreateTestAddon(t, addonInp)

		addonInp.Key = "some-new-key"

		add2 := deps.AddonService.CreateTestAddon(t, addonInp)

		// Let's link both addons to the plan
		_, err = deps.PlanAddonService.CreatePlanAddon(context.Background(), planaddon.CreatePlanAddonInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: subscriptiontestutils.ExampleNamespace,
			},
			PlanID:        p.ID,
			AddonID:       add1.ID,
			FromPlanPhase: p.Phases[0].Key,
		})
		require.Nil(t, err, "received error: %s", err)

		_, err = deps.PlanAddonService.CreatePlanAddon(context.Background(), planaddon.CreatePlanAddonInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: subscriptiontestutils.ExampleNamespace,
			},
			PlanID:        p.ID,
			AddonID:       add2.ID,
			FromPlanPhase: p.Phases[0].Key,
		})
		require.Nil(t, err, "received error: %s", err)

		// Let's publish the plan

		p, err = deps.PlanService.PublishPlan(context.Background(), plan.PublishPlanInput{
			NamespacedID: p.NamespacedID,
			EffectivePeriod: productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(clock.Now()),
				EffectiveTo:   lo.ToPtr(testutils.GetRFC3339Time(t, "2099-01-01T00:00:00Z")),
			},
		})
		require.Nil(t, err, "received error: %s", err)

		// Let's create a subscription from the plan

		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps, &plansubscription.Plan{
			Plan: p.AsProductCatalogPlan(),
			Ref:  &p.NamespacedID,
		}, now)

		// Let's add the first addon
		_, _, err = deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add1.ID,
			InitialQuantity: 1,
			Timing: subscription.Timing{
				Custom: &now,
			},
		})
		require.NoError(t, err)

		// Now let's add the second addon
		subView, _, err = deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add2.ID,
			InitialQuantity: 1,
			Timing: subscription.Timing{
				Custom: &now,
			},
		})
		require.NoError(t, err)

		// And lets validate that both of them updated the entitlement limit
		item := subView.Phases[0].ItemsByKey[subscriptiontestutils.ExampleRateCard1.Key()][0]
		require.Equal(t, 200.0, *item.Entitlement.Entitlement.IssueAfterReset)
	}))

	t.Run("Should sync subscription with new addons contents", runWithDeps(func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
		p, add := subscriptiontestutils.CreatePlanWithAddon(
			t,
			deps,
			subscriptiontestutils.GetExamplePlanInput(t),
			subscriptiontestutils.BuildAddonForTesting(t,
				productcatalog.EffectivePeriod{
					EffectiveFrom: &now,
					EffectiveTo:   nil,
				},
				productcatalog.AddonInstanceTypeSingle,
				subscriptiontestutils.ExampleAddonRateCard2.Clone(), // This will add a new item
				subscriptiontestutils.ExampleAddonRateCard4.Clone(), // This will extend existing items
			),
		)

		// Let's create a subscription from the plan
		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps, p, now)

		ogView := subView
		require.NotNil(t, ogView)

		spec := subView.AsSpec()

		addonInp := subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add.ID,
			InitialQuantity: 1,
			Timing: subscription.Timing{
				Custom: &now,
			},
		}

		subView, subAdd, err := deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, addonInp)
		require.NoError(t, err)

		// Let's figure out what the expected spec should be
		{
			diff, err := addondiff.GetDiffableFromAddon(subView, subAdd)
			require.NoError(t, err)

			require.NoError(t, spec.Apply(diff.GetApplies(), subscription.ApplyContext{
				CurrentTime: now,
			}))
		}

		newSpec := subView.AsSpec()

		// Due to not knowing the FeatureIDs before the subscription is updated, we cannot use subscriptiontestutils.SpecsEqual properly
		// We'll strip all FeatureIDs from the comparison, which OPENS UP silent errors but this is the best we can do for now
		stripFeatureIDs(&spec)
		stripFeatureIDs(&newSpec)

		subscriptiontestutils.SpecsEqual(t, newSpec, spec)
	}))

	t.Run("Should return conflict error if subscription already has that addon purchased", runWithDeps(func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
		p, add := subscriptiontestutils.CreatePlanWithAddon(
			t,
			deps,
			subscriptiontestutils.GetExamplePlanInput(t),
			subscriptiontestutils.BuildAddonForTesting(t,
				productcatalog.EffectivePeriod{
					EffectiveFrom: &now,
					EffectiveTo:   nil,
				},
				productcatalog.AddonInstanceTypeSingle,
				subscriptiontestutils.ExampleAddonRateCard2.Clone(), // This will add a new item
				subscriptiontestutils.ExampleAddonRateCard4.Clone(), // This will extend existing items
			),
		)

		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps, p, now)

		addonInp := subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add.ID,
			InitialQuantity: 1,
			Timing: subscription.Timing{
				Custom: &now,
			},
		}

		_, _, err := deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, addonInp)
		require.NoError(t, err)

		_, _, err = deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, addonInp)
		require.Error(t, err)
		require.ErrorAs(t, err, lo.ToPtr(&models.GenericConflictError{}))
		require.True(t, models.IsGenericConflictError(err))
	}))
}

func TestChangeAddonQuantity(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2025-04-01T00:00:00Z")

	runWithDeps := func(fn func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies)) func(t *testing.T) {
		return func(t *testing.T) {
			clock.SetTime(now)
			defer clock.ResetTime()

			dbDeps := subscriptiontestutils.SetupDBDeps(t)
			defer dbDeps.Cleanup(t)

			deps := subscriptiontestutils.NewService(t, dbDeps)
			fn(t, deps)
		}
	}

	t.Run("Should error if SubscriptionAddon and Subscription are in different namespaces", runWithDeps(func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
		p, add := subscriptiontestutils.CreatePlanWithAddon(
			t,
			deps,
			subscriptiontestutils.GetExamplePlanInput(t),
			subscriptiontestutils.BuildAddonForTesting(t,
				productcatalog.EffectivePeriod{
					EffectiveFrom: &now,
					EffectiveTo:   nil,
				},
				productcatalog.AddonInstanceTypeSingle,
				subscriptiontestutils.ExampleAddonRateCard2.Clone(), // This will add a new item
				subscriptiontestutils.ExampleAddonRateCard4.Clone(), // This will extend existing items
			),
		)

		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps, p, now)

		addonInp := subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add.ID,
			InitialQuantity: 1,
			Timing: subscription.Timing{
				Custom: &now,
			},
		}

		subView, subAdd, err := deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, addonInp)
		require.NoError(t, err)

		changeTime := now.AddDate(0, 0, 1)

		changeInp := subscriptionworkflow.ChangeAddonQuantityWorkflowInput{
			SubscriptionAddonID: models.NamespacedID{
				ID:        subAdd.ID,
				Namespace: "some-other-namespace",
			},
			Quantity: 0,
			Timing: subscription.Timing{
				Custom: &changeTime,
			},
		}

		_, _, err = deps.WorkflowService.ChangeAddonQuantity(context.Background(), subView.Subscription.NamespacedID, changeInp)
		require.Error(t, err)
		require.ErrorAs(t, err, lo.ToPtr(&models.GenericValidationError{}))
		require.True(t, models.IsGenericValidationError(err))
	}))

	t.Run("Should error if SubscriptionAddon doesn't belong to Subscription", runWithDeps(func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
		p, add := subscriptiontestutils.CreatePlanWithAddon(
			t,
			deps,
			subscriptiontestutils.GetExamplePlanInput(t),
			subscriptiontestutils.BuildAddonForTesting(t,
				productcatalog.EffectivePeriod{
					EffectiveFrom: &now,
					EffectiveTo:   nil,
				},
				productcatalog.AddonInstanceTypeSingle,
				subscriptiontestutils.ExampleAddonRateCard2.Clone(), // This will add a new item
				subscriptiontestutils.ExampleAddonRateCard4.Clone(), // This will extend existing items
			),
		)

		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps, p, now)

		addonInp := subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add.ID,
			InitialQuantity: 1,
			Timing: subscription.Timing{
				Custom: &now,
			},
		}

		cust2, err := deps.CustomerService.CreateCustomer(context.Background(), customer.CreateCustomerInput{
			Namespace: subscriptiontestutils.ExampleNamespace,
			CustomerMutate: customer.CustomerMutate{
				Name:         "another",
				PrimaryEmail: lo.ToPtr("mail@me.uk"),
				Currency:     lo.ToPtr(currencyx.Code("USD")),
				UsageAttribution: &customer.CustomerUsageAttribution{
					SubjectKeys: []string{"another"},
				},
			},
		})
		require.NoError(t, err)

		subView2, err := deps.WorkflowService.CreateFromPlan(context.Background(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
			Namespace:  cust2.Namespace,
			CustomerID: cust2.ID,
			ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Name: "test",
				Timing: subscription.Timing{
					Custom: &now,
				},
			},
		}, p)
		require.NoError(t, err)

		subView, _, err = deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, addonInp)
		require.NoError(t, err)

		subView2, subAdd2, err := deps.WorkflowService.AddAddon(context.Background(), subView2.Subscription.NamespacedID, addonInp)
		require.NoError(t, err)

		changeTime := now.AddDate(0, 0, 1)

		// Now let's reference a SubsAddon from sub2 while providing sub1 below
		changeInp := subscriptionworkflow.ChangeAddonQuantityWorkflowInput{
			SubscriptionAddonID: subAdd2.NamespacedID,
			Quantity:            0,
			Timing: subscription.Timing{
				Custom: &changeTime,
			},
		}

		_, _, err = deps.WorkflowService.ChangeAddonQuantity(context.Background(), subView.Subscription.NamespacedID, changeInp)
		require.Error(t, err)
		require.ErrorAs(t, err, lo.ToPtr(&models.GenericValidationError{}))
		require.True(t, models.IsGenericValidationError(err))
	}))

	t.Run("Should error if the subscription is inactive or in wrong state", runWithDeps(func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
		p, add := subscriptiontestutils.CreatePlanWithAddon(
			t,
			deps,
			subscriptiontestutils.GetExamplePlanInput(t),
			subscriptiontestutils.BuildAddonForTesting(t,
				productcatalog.EffectivePeriod{
					EffectiveFrom: &now,
					EffectiveTo:   nil,
				},
				productcatalog.AddonInstanceTypeMultiple,
				subscriptiontestutils.ExampleAddonRateCard2.Clone(), // This will add a new item
				subscriptiontestutils.ExampleAddonRateCard4.Clone(), // This will extend existing items
			),
		)

		// Let's create a subscription from the plan
		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps, p, now)

		// Let's add an addon to the subscription
		addonInp := subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add.ID,
			InitialQuantity: 1,
			Timing: subscription.Timing{
				Custom: &now,
			},
		}

		_, subAdd, err := deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, addonInp)
		require.NoError(t, err)

		// Let's cancel the subscription
		_, err = deps.SubscriptionService.Cancel(context.Background(), subView.Subscription.NamespacedID, subscription.Timing{
			Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
		})
		require.NoError(t, err)

		// Now let's try to change the quantity of the addon
		changeInp := subscriptionworkflow.ChangeAddonQuantityWorkflowInput{
			SubscriptionAddonID: subAdd.NamespacedID,
			Quantity:            2,
			Timing: subscription.Timing{
				Custom: lo.ToPtr(now.AddDate(0, 0, 1)),
			},
		}

		_, _, err = deps.WorkflowService.ChangeAddonQuantity(context.Background(), subView.Subscription.NamespacedID, changeInp)
		require.Error(t, err)
		require.ErrorAs(t, err, lo.ToPtr(&models.GenericForbiddenError{}))
		require.True(t, models.IsGenericForbiddenError(err))
		require.ErrorContains(t, err, "state canceled not allowed")
	}))

	t.Run("Should update the quantity of the addon", runWithDeps(func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
		p, add := subscriptiontestutils.CreatePlanWithAddon(
			t,
			deps,
			subscriptiontestutils.BuildTestPlanInput(t).
				AddPhase(nil, &subscriptiontestutils.ExampleRateCard1).
				Build(),
			subscriptiontestutils.BuildAddonForTesting(t,
				productcatalog.EffectivePeriod{
					EffectiveFrom: &now,
					EffectiveTo:   nil,
				},
				productcatalog.AddonInstanceTypeMultiple,
				subscriptiontestutils.ExampleAddonRateCard5.Clone(),
			),
		)

		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps, p, now)

		addonInp := subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add.ID,
			InitialQuantity: 1,
			Timing: subscription.Timing{
				Custom: &now,
			},
		}

		subView, subAdd, err := deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, addonInp)
		require.NoError(t, err)

		changeTime := now.AddDate(0, 0, 1)

		changeInp := subscriptionworkflow.ChangeAddonQuantityWorkflowInput{
			SubscriptionAddonID: subAdd.NamespacedID,
			Quantity:            0,
			Timing: subscription.Timing{
				Custom: &changeTime,
			},
		}

		subView, subAdd, err = deps.WorkflowService.ChangeAddonQuantity(context.Background(), subView.Subscription.NamespacedID, changeInp)
		require.NoError(t, err)

		require.Len(t, subAdd.Quantities.GetTimes(), 2)
		require.Equal(t, subAdd.Quantities.GetAt(0).GetValue().Quantity, 1)
		require.Equal(t, subAdd.Quantities.GetAt(1).GetValue().Quantity, 0)

		require.Len(t, subView.Phases[0].ItemsByKey[subscriptiontestutils.ExampleRateCard1.Key()], 2)
	}))

	t.Run("Should not combine two subsequent identical items (present due to an edit) when changing the quantity after an addon is already purchased", runWithDeps(func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
		twoMonths := datetime.MustParseDuration(t, "P2M")

		twoMonthsFromNow, _ := twoMonths.AddTo(now)

		p, add := subscriptiontestutils.CreatePlanWithAddon(
			t,
			deps,
			subscriptiontestutils.BuildTestPlanInput(t).
				SetMeta(productcatalog.PlanMeta{
					Name:           "Test Plan",
					Key:            "test_plan",
					Version:        1,
					Currency:       currency.USD,
					BillingCadence: datetime.MustParseDuration(t, "P1M"),
					ProRatingConfig: productcatalog.ProRatingConfig{
						Enabled: true,
						Mode:    productcatalog.ProRatingModeProratePrices,
					},
				}).
				AddPhase(nil, &subscriptiontestutils.ExampleRateCard1).
				Build(),
			subscriptiontestutils.BuildAddonForTesting(t,
				productcatalog.EffectivePeriod{
					EffectiveFrom: &now,
					EffectiveTo:   nil,
				},
				productcatalog.AddonInstanceTypeMultiple,
				subscriptiontestutils.ExampleAddonRateCard5.Clone(),
			),
		)

		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps, p, now)

		// Now let's edit the sub
		ogItem := subView.Phases[0].ItemsByKey[subscriptiontestutils.ExampleRateCard1.Key()][0].Spec

		timing := subscription.Timing{
			Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
		}

		subView, err := deps.WorkflowService.EditRunning(context.Background(), subView.Subscription.NamespacedID, []subscription.Patch{
			patch.PatchRemoveItem{
				PhaseKey: ogItem.PhaseKey,
				ItemKey:  ogItem.ItemKey,
			},
			patch.PatchAddItem{
				PhaseKey:    ogItem.PhaseKey,
				ItemKey:     ogItem.ItemKey,
				CreateInput: ogItem, // This will be identical to the original but will still cause a split
			},
		}, timing)
		require.NoError(t, err)

		require.Len(t, subView.Phases[0].ItemsByKey[subscriptiontestutils.ExampleRateCard1.Key()], 2)

		// Now let's add the addon

		addonInp := subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add.ID,
			InitialQuantity: 1,
			Timing: subscription.Timing{
				Custom: &now,
			},
		}

		subView, subAdd, err := deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, addonInp)
		require.NoError(t, err)

		// assert that its still two items
		require.Len(t, subView.Phases[0].ItemsByKey[subscriptiontestutils.ExampleRateCard1.Key()], 2)

		// Now let's change the quantity of the addon
		changeInp := subscriptionworkflow.ChangeAddonQuantityWorkflowInput{
			SubscriptionAddonID: subAdd.NamespacedID,
			Quantity:            0,
			Timing: subscription.Timing{
				Custom: &twoMonthsFromNow,
			},
		}

		subView, subAdd, err = deps.WorkflowService.ChangeAddonQuantity(context.Background(), subView.Subscription.NamespacedID, changeInp)
		require.NoError(t, err)

		// Now it should be three items: original two with addon included + 3rd without the addon after 3 months
		require.Len(t, subView.Phases[0].ItemsByKey[subscriptiontestutils.ExampleRateCard1.Key()], 3)
	}))
}

// Instead of stripping them, we could also populate them with the correct values
func stripFeatureIDs(spec *subscription.SubscriptionSpec) {
	for _, phase := range spec.Phases {
		for _, items := range phase.ItemsByKey {
			for _, item := range items {
				_ = item.RateCard.ChangeMeta(func(m productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
					m.FeatureID = nil
					return m, nil
				})
			}
		}
	}
}

func TestAddonCombinations(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2025-04-01T00:00:00Z")

	runWithDeps := func(fn func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies)) func(t *testing.T) {
		return func(t *testing.T) {
			clock.SetTime(now)
			defer clock.ResetTime()

			dbDeps := subscriptiontestutils.SetupDBDeps(t)
			defer dbDeps.Cleanup(t)

			deps := subscriptiontestutils.NewService(t, dbDeps)
			fn(t, deps)
		}
	}

	t.Run("Should handle repeated add/remove of single instance addon with dynamic price", runWithDeps(func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
		// Create a plan with mixed rate cards
		planInput := subscriptiontestutils.BuildTestPlanInput(t).
			AddPhase(nil,
				&subscriptiontestutils.ExampleRateCard1, // Flat price
				&productcatalog.UsageBasedRateCard{ // Dynamic price
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  "dynamic_rc_1",
						Name: "Dynamic Rate Card 1",
						Price: productcatalog.NewPriceFrom(productcatalog.TieredPrice{
							Mode: productcatalog.VolumeTieredPrice,
							Tiers: []productcatalog.PriceTier{
								{
									UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(100)),
									FlatPrice: &productcatalog.PriceTierFlatPrice{
										Amount: alpacadecimal.NewFromInt(10),
									},
								},
								{
									// UpToAmount: nil for infinity
									UnitPrice: &productcatalog.PriceTierUnitPrice{
										Amount: alpacadecimal.NewFromInt(1),
									},
								},
							},
						}),
					},
					BillingCadence: subscriptiontestutils.ISOMonth,
				},
			).
			Build()
		p, add := subscriptiontestutils.CreatePlanWithAddon(
			t,
			deps,
			planInput,
			subscriptiontestutils.BuildAddonForTesting(t,
				productcatalog.EffectivePeriod{
					EffectiveFrom: &now,
					EffectiveTo:   nil,
				},
				productcatalog.AddonInstanceTypeSingle,
				&productcatalog.UsageBasedRateCard{ // Dynamic price for addon
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  "addon_dynamic_rc",
						Name: "Addon Dynamic Rate Card",
						Price: productcatalog.NewPriceFrom(productcatalog.TieredPrice{
							Mode: productcatalog.VolumeTieredPrice,
							Tiers: []productcatalog.PriceTier{
								{
									// UpToAmount: nil for infinity
									UnitPrice: &productcatalog.PriceTierUnitPrice{
										Amount: alpacadecimal.NewFromFloat(0.5),
									},
								},
							},
						}),
					},
					BillingCadence: subscriptiontestutils.ISOMonth,
				},
			),
		)

		// Create a subscription from the plan
		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps, p, now)

		// Add the addon initially
		addTime := now
		var subAdd subscriptionaddon.SubscriptionAddon
		var err error
		subView, subAdd, err = deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add.ID,
			InitialQuantity: 1,
			Timing: subscription.Timing{
				Custom: &addTime,
			},
		})
		require.NoError(t, err, "failed to add addon initially")
		require.NotNil(t, subAdd)

		// Now, repeatedly change quantity to 0 and then back to 1
		// Let's pass time
		clock.SetTime(clock.Now().Add(time.Minute))

		for i := 0; i < 3; i++ {
			// Change quantity to 0 (remove)
			changeInpZero := subscriptionworkflow.ChangeAddonQuantityWorkflowInput{
				SubscriptionAddonID: subAdd.NamespacedID,
				Quantity:            0,
				Timing: subscription.Timing{
					Custom: lo.ToPtr(clock.Now()),
				},
			}
			subView, _, err = deps.WorkflowService.ChangeAddonQuantity(context.Background(), subView.Subscription.NamespacedID, changeInpZero)
			require.NoError(t, err, "failed to change addon quantity to 0 on iteration %d", i)

			// Let's pass time
			clock.SetTime(clock.Now().Add(time.Minute))

			// Change quantity back to 1 (re-add)
			changeInpOne := subscriptionworkflow.ChangeAddonQuantityWorkflowInput{
				SubscriptionAddonID: subAdd.NamespacedID,
				Quantity:            1,
				Timing: subscription.Timing{
					Custom: lo.ToPtr(clock.Now()),
				},
			}
			subView, subAdd, err = deps.WorkflowService.ChangeAddonQuantity(context.Background(), subView.Subscription.NamespacedID, changeInpOne)
			require.NoError(t, err, "failed to change addon quantity to 1 on iteration %d", i)

			// Let's pass time
			clock.SetTime(clock.Now().Add(time.Minute))
		}
	}))
}
