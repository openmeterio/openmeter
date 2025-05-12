package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
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
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestAddAddon(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2025-04-01T00:00:00Z")

	type testCaseDeps struct {
		deps subscriptiontestutils.SubscriptionDependencies
	}

	runWithDeps := func(fn func(t *testing.T, deps testCaseDeps)) func(t *testing.T) {
		return func(t *testing.T) {
			clock.SetTime(now)
			defer clock.ResetTime()

			dbDeps := subscriptiontestutils.SetupDBDeps(t)
			defer dbDeps.Cleanup(t)

			deps := subscriptiontestutils.NewService(t, dbDeps)
			fn(t, testCaseDeps{deps: deps})
		}
	}

	t.Run("Should error on invalid input", runWithDeps(func(t *testing.T, deps testCaseDeps) {
		p, add := subscriptiontestutils.CreatePlanWithAddon(
			t,
			deps.deps,
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

		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps.deps, p, now)

		addonInp := subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add.ID,
			InitialQuantity: 0,
			Timing: subscription.Timing{
				Custom: &now,
			},
		}

		expectedErr := addonInp.Validate()
		require.NotNil(t, expectedErr)

		_, _, err := deps.deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, addonInp)
		require.Error(t, err)

		require.True(t, models.IsGenericValidationError(err))
		require.ErrorContains(t, err, expectedErr.Error())
	}))

	t.Run("Should error if the subscription is inactive or in wrong state", runWithDeps(func(t *testing.T, deps testCaseDeps) {
		p, add := subscriptiontestutils.CreatePlanWithAddon(
			t,
			deps.deps,
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
		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps.deps, p, now)

		// Let's cancel the subscription
		_, err := deps.deps.SubscriptionService.Cancel(context.Background(), subView.Subscription.NamespacedID, subscription.Timing{
			Custom: lo.ToPtr(now.AddDate(0, 1, 0)),
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

		_, _, err = deps.deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, addonInp)
		require.Error(t, err)
		require.ErrorAs(t, err, lo.ToPtr(&models.GenericForbiddenError{}))
		require.True(t, models.IsGenericForbiddenError(err))
		require.ErrorContains(t, err, "state canceled not allowed")
	}))

	t.Run("Should add a new addon to a subscription that already has a different addon", runWithDeps(func(t *testing.T, deps testCaseDeps) {
		_ = deps.deps.FeatureConnector.CreateExampleFeatures(t)

		// Let's create a plan
		p, err := deps.deps.PlanService.CreatePlan(context.Background(), subscriptiontestutils.BuildTestPlan(t).
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

		add1 := deps.deps.AddonService.CreateTestAddon(t, addonInp)

		addonInp.Key = "some-new-key"

		add2 := deps.deps.AddonService.CreateTestAddon(t, addonInp)

		// Let's link both addons to the plan
		_, err = deps.deps.PlanAddonService.CreatePlanAddon(context.Background(), planaddon.CreatePlanAddonInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: subscriptiontestutils.ExampleNamespace,
			},
			PlanID:        p.ID,
			AddonID:       add1.ID,
			FromPlanPhase: p.Phases[0].Key,
		})
		require.Nil(t, err, "received error: %s", err)

		_, err = deps.deps.PlanAddonService.CreatePlanAddon(context.Background(), planaddon.CreatePlanAddonInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: subscriptiontestutils.ExampleNamespace,
			},
			PlanID:        p.ID,
			AddonID:       add2.ID,
			FromPlanPhase: p.Phases[0].Key,
		})
		require.Nil(t, err, "received error: %s", err)

		// Let's publish the plan

		p, err = deps.deps.PlanService.PublishPlan(context.Background(), plan.PublishPlanInput{
			NamespacedID: p.NamespacedID,
			EffectivePeriod: productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(clock.Now()),
				EffectiveTo:   lo.ToPtr(testutils.GetRFC3339Time(t, "2099-01-01T00:00:00Z")),
			},
		})
		require.Nil(t, err, "received error: %s", err)

		// Let's create a subscription from the plan

		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps.deps, &plansubscription.Plan{
			Plan: p.AsProductCatalogPlan(),
			Ref:  &p.NamespacedID,
		}, now)

		// Let's add the first addon
		_, _, err = deps.deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add1.ID,
			InitialQuantity: 1,
			Timing: subscription.Timing{
				Custom: &now,
			},
		})
		require.NoError(t, err)

		// Now let's add the second addon
		subView, _, err = deps.deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, subscriptionworkflow.AddAddonWorkflowInput{
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

	t.Run("Should sync subscription with new addons contents", runWithDeps(func(t *testing.T, deps testCaseDeps) {
		p, add := subscriptiontestutils.CreatePlanWithAddon(
			t,
			deps.deps,
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
		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps.deps, p, now)

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

		subView, subAdd, err := deps.deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, addonInp)
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

	t.Run("Should return conflict error if subscription already has that addon purchased", runWithDeps(func(t *testing.T, deps testCaseDeps) {
		p, add := subscriptiontestutils.CreatePlanWithAddon(
			t,
			deps.deps,
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

		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps.deps, p, now)

		addonInp := subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add.ID,
			InitialQuantity: 1,
			Timing: subscription.Timing{
				Custom: &now,
			},
		}

		_, _, err := deps.deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, addonInp)
		require.NoError(t, err)

		_, _, err = deps.deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, addonInp)
		require.Error(t, err)
		require.ErrorAs(t, err, lo.ToPtr(&models.GenericConflictError{}))
		require.True(t, models.IsGenericConflictError(err))
	}))

	t.Run("Should work if subscription addon associated with quantity already changed", runWithDeps(func(t *testing.T, deps testCaseDeps) {
		// We'll set up a plan with two addons
		// We'll create the sub, add one addon to it and play with it
		// We'll then add the second addon too and play with it

		// Let's set up a plan with two compatible addons
		planInp := subscriptiontestutils.BuildTestPlan(t).
			SetMeta(productcatalog.PlanMeta{
				Name:     "Test Plan",
				Key:      "test_plan",
				Version:  1,
				Currency: currency.USD,
				Alignment: productcatalog.Alignment{
					BillablesMustAlign: true,
				},
			}).
			AddPhase(nil, &subscriptiontestutils.ExampleRateCard1).
			Build()

		addonInp1 := subscriptiontestutils.BuildAddonForTesting(t,
			productcatalog.EffectivePeriod{
				EffectiveFrom: &now,
				EffectiveTo:   nil,
			},
			productcatalog.AddonInstanceTypeMultiple,
			subscriptiontestutils.ExampleAddonRateCard5.Clone(),
		)

		addonInp2 := addon.CreateAddonInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: subscriptiontestutils.ExampleNamespace,
			},
			Addon: productcatalog.Addon{
				AddonMeta: productcatalog.AddonMeta{
					Name:        "Test Addon",
					Description: lo.ToPtr("Test Addon Description"),
					EffectivePeriod: productcatalog.EffectivePeriod{
						EffectiveFrom: &now,
						EffectiveTo:   nil,
					},
					Key:          "test-addon-2",
					Version:      1,
					Currency:     currency.USD,
					InstanceType: productcatalog.AddonInstanceTypeMultiple,
					Metadata: models.NewMetadata(map[string]string{
						"test": "test",
					}),
				},
				RateCards: []productcatalog.RateCard{subscriptiontestutils.ExampleAddonRateCard5.Clone()},
			},
		}

		_ = deps.deps.FeatureConnector.CreateExampleFeatures(t)

		p, err := deps.deps.PlanService.CreatePlan(context.Background(), planInp)
		require.Nil(t, err)
		require.NotNil(t, p)

		add1 := deps.deps.AddonService.CreateTestAddon(t, addonInp1)
		add2 := deps.deps.AddonService.CreateTestAddon(t, addonInp2)

		_, err = deps.deps.PlanAddonService.CreatePlanAddon(context.Background(), planaddon.CreatePlanAddonInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: subscriptiontestutils.ExampleNamespace,
			},
			PlanID:        p.ID,
			AddonID:       add1.ID,
			FromPlanPhase: p.Phases[0].Key,
		})
		require.Nil(t, err, "received error: %s", err)

		_, err = deps.deps.PlanAddonService.CreatePlanAddon(context.Background(), planaddon.CreatePlanAddonInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: subscriptiontestutils.ExampleNamespace,
			},
			PlanID:        p.ID,
			AddonID:       add2.ID,
			FromPlanPhase: p.Phases[0].Key,
		})
		require.Nil(t, err, "received error: %s", err)

		p, err = deps.deps.PlanService.PublishPlan(context.Background(), plan.PublishPlanInput{
			NamespacedID: p.NamespacedID,
			EffectivePeriod: productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(clock.Now()),
				EffectiveTo:   lo.ToPtr(testutils.GetRFC3339Time(t, "2099-01-01T00:00:00Z")),
			},
		})
		require.Nil(t, err, "received error: %s", err)

		sp := &plansubscription.Plan{
			Plan: p.AsProductCatalogPlan(),
			Ref:  &p.NamespacedID,
		}

		// Now let's create a subscription from the plan
		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps.deps, sp, now)

		// Now let's add both addons
		subsAddInp1 := subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add1.ID,
			InitialQuantity: 1,
			Timing: subscription.Timing{
				Custom: &now,
			},
		}

		subView, subsAdd1, err := deps.deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, subsAddInp1)
		require.NoError(t, err)
		subAddons, err := deps.deps.SubscriptionAddonService.List(context.Background(), subView.Subscription.Namespace, subscriptionaddon.ListSubscriptionAddonsInput{
			SubscriptionID: subView.Subscription.ID,
		})
		require.NoError(t, err)
		require.Len(t, subAddons.Items, 1)
		require.True(t, lo.ContainsBy(subAddons.Items, func(item subscriptionaddon.SubscriptionAddon) bool {
			return item.ID == subsAdd1.ID
		}))

		// Now let's try and change the quantity of the first addon to two
		changeInp := subscriptionworkflow.ChangeAddonQuantityWorkflowInput{
			SubscriptionAddonID: subsAdd1.NamespacedID,
			Quantity:            2,
			Timing: subscription.Timing{
				Custom: lo.ToPtr(now.Add(time.Hour * 2)),
			},
		}

		subView, subsAdd1, err = deps.deps.WorkflowService.ChangeAddonQuantity(context.Background(), subView.Subscription.NamespacedID, changeInp)
		require.NoError(t, err)

		// Let's advance some time
		now := now.Add(time.Hour * 4)
		clock.SetTime(now)
		defer clock.ResetTime()

		// Let's add the second addon

		subsAddInp2 := subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add2.ID,
			InitialQuantity: 1,
			Timing: subscription.Timing{
				Custom: &now,
			},
		}

		subView, subsAdd2, err := deps.deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, subsAddInp2)
		require.NoError(t, err)
		subAddons, err = deps.deps.SubscriptionAddonService.List(context.Background(), subView.Subscription.Namespace, subscriptionaddon.ListSubscriptionAddonsInput{
			SubscriptionID: subView.Subscription.ID,
		})
		require.NoError(t, err)
		require.Len(t, subAddons.Items, 2)
		require.True(t, lo.ContainsBy(subAddons.Items, func(item subscriptionaddon.SubscriptionAddon) bool {
			return item.ID == subsAdd2.ID
		}))

		// Let's change the quantity of the second addon to three
		changeInp2 := subscriptionworkflow.ChangeAddonQuantityWorkflowInput{
			SubscriptionAddonID: subsAdd2.NamespacedID,
			Quantity:            3,
			Timing: subscription.Timing{
				Custom: &now,
			},
		}

		subView, subsAdd2, err = deps.deps.WorkflowService.ChangeAddonQuantity(context.Background(), subView.Subscription.NamespacedID, changeInp2)
		require.NoError(t, err)
	}))
}

func TestChangeAddonQuantity(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2025-04-01T00:00:00Z")

	type testCaseDeps struct {
		deps subscriptiontestutils.SubscriptionDependencies
	}

	runWithDeps := func(fn func(t *testing.T, deps testCaseDeps)) func(t *testing.T) {
		return func(t *testing.T) {
			clock.SetTime(now)
			defer clock.ResetTime()

			dbDeps := subscriptiontestutils.SetupDBDeps(t)
			defer dbDeps.Cleanup(t)

			deps := subscriptiontestutils.NewService(t, dbDeps)
			fn(t, testCaseDeps{deps: deps})
		}
	}

	t.Run("Should error if SubscriptionAddon and Subscription are in different namespaces", runWithDeps(func(t *testing.T, deps testCaseDeps) {
		p, add := subscriptiontestutils.CreatePlanWithAddon(
			t,
			deps.deps,
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

		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps.deps, p, now)

		addonInp := subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add.ID,
			InitialQuantity: 1,
			Timing: subscription.Timing{
				Custom: &now,
			},
		}

		subView, subAdd, err := deps.deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, addonInp)
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

		_, _, err = deps.deps.WorkflowService.ChangeAddonQuantity(context.Background(), subView.Subscription.NamespacedID, changeInp)
		require.Error(t, err)
		require.ErrorAs(t, err, lo.ToPtr(&models.GenericValidationError{}))
		require.True(t, models.IsGenericValidationError(err))
	}))

	t.Run("Should error if SubscriptionAddon doesn't belong to Subscription", runWithDeps(func(t *testing.T, deps testCaseDeps) {
		p, add := subscriptiontestutils.CreatePlanWithAddon(
			t,
			deps.deps,
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

		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps.deps, p, now)

		addonInp := subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add.ID,
			InitialQuantity: 1,
			Timing: subscription.Timing{
				Custom: &now,
			},
		}

		cust2, err := deps.deps.CustomerAdapter.CreateCustomer(context.Background(), customer.CreateCustomerInput{
			Namespace: subscriptiontestutils.ExampleNamespace,
			CustomerMutate: customer.CustomerMutate{
				Name:         "another",
				PrimaryEmail: lo.ToPtr("mail@me.uk"),
				Currency:     lo.ToPtr(currencyx.Code("USD")),
				UsageAttribution: customer.CustomerUsageAttribution{
					SubjectKeys: []string{"another"},
				},
			},
		})
		require.NoError(t, err)

		subView2, err := deps.deps.WorkflowService.CreateFromPlan(context.Background(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
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

		subView, _, err = deps.deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, addonInp)
		require.NoError(t, err)

		subView2, subAdd2, err := deps.deps.WorkflowService.AddAddon(context.Background(), subView2.Subscription.NamespacedID, addonInp)
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

		_, _, err = deps.deps.WorkflowService.ChangeAddonQuantity(context.Background(), subView.Subscription.NamespacedID, changeInp)
		require.Error(t, err)
		require.ErrorAs(t, err, lo.ToPtr(&models.GenericValidationError{}))
		require.True(t, models.IsGenericValidationError(err))
	}))

	t.Run("Should error if the subscription is inactive or in wrong state", runWithDeps(func(t *testing.T, deps testCaseDeps) {
		p, add := subscriptiontestutils.CreatePlanWithAddon(
			t,
			deps.deps,
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
		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps.deps, p, now)

		// Let's add an addon to the subscription
		addonInp := subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add.ID,
			InitialQuantity: 1,
			Timing: subscription.Timing{
				Custom: &now,
			},
		}

		_, subAdd, err := deps.deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, addonInp)
		require.NoError(t, err)

		// Let's cancel the subscription
		_, err = deps.deps.SubscriptionService.Cancel(context.Background(), subView.Subscription.NamespacedID, subscription.Timing{
			Custom: lo.ToPtr(now.AddDate(0, 1, 0)),
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

		_, _, err = deps.deps.WorkflowService.ChangeAddonQuantity(context.Background(), subView.Subscription.NamespacedID, changeInp)
		require.Error(t, err)
		require.ErrorAs(t, err, lo.ToPtr(&models.GenericForbiddenError{}))
		require.True(t, models.IsGenericForbiddenError(err))
		require.ErrorContains(t, err, "state canceled not allowed")
	}))

	t.Run("Should update the quantity of the addon", runWithDeps(func(t *testing.T, deps testCaseDeps) {
		p, add := subscriptiontestutils.CreatePlanWithAddon(
			t,
			deps.deps,
			subscriptiontestutils.BuildTestPlan(t).
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

		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps.deps, p, now)

		addonInp := subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add.ID,
			InitialQuantity: 1,
			Timing: subscription.Timing{
				Custom: &now,
			},
		}

		subView, subAdd, err := deps.deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, addonInp)
		require.NoError(t, err)

		changeTime := now.AddDate(0, 0, 1)

		changeInp := subscriptionworkflow.ChangeAddonQuantityWorkflowInput{
			SubscriptionAddonID: subAdd.NamespacedID,
			Quantity:            0,
			Timing: subscription.Timing{
				Custom: &changeTime,
			},
		}

		subView, subAdd, err = deps.deps.WorkflowService.ChangeAddonQuantity(context.Background(), subView.Subscription.NamespacedID, changeInp)
		require.NoError(t, err)

		require.Len(t, subAdd.Quantities.GetTimes(), 2)
		require.Equal(t, subAdd.Quantities.GetAt(0).GetValue().Quantity, 1)
		require.Equal(t, subAdd.Quantities.GetAt(1).GetValue().Quantity, 0)

		require.Len(t, subView.Phases[0].ItemsByKey[subscriptiontestutils.ExampleRateCard1.Key()], 2)
	}))

	t.Run("Should not combine two subsequent identical items (present due to an edit) when changing the quantity after an addon is already purchased", runWithDeps(func(t *testing.T, deps testCaseDeps) {
		twoMonths := testutils.GetISODuration(t, "P2M")

		twoMonthsFromNow, _ := twoMonths.AddTo(now)

		p, add := subscriptiontestutils.CreatePlanWithAddon(
			t,
			deps.deps,
			subscriptiontestutils.BuildTestPlan(t).
				SetMeta(productcatalog.PlanMeta{
					Name:     "Test Plan",
					Key:      "test_plan",
					Version:  1,
					Currency: currency.USD,
					Alignment: productcatalog.Alignment{
						BillablesMustAlign: true,
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

		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps.deps, p, now)

		// Now let's edit the sub
		ogItem := subView.Phases[0].ItemsByKey[subscriptiontestutils.ExampleRateCard1.Key()][0].Spec

		timing := subscription.Timing{
			Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
		}

		subView, err := deps.deps.WorkflowService.EditRunning(context.Background(), subView.Subscription.NamespacedID, []subscription.Patch{
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

		subView, subAdd, err := deps.deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, addonInp)
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

		subView, subAdd, err = deps.deps.WorkflowService.ChangeAddonQuantity(context.Background(), subView.Subscription.NamespacedID, changeInp)
		require.NoError(t, err)

		// Now it should be three items: original two with addon included + 3rd without the addon after 3 months
		require.Len(t, subView.Phases[0].ItemsByKey[subscriptiontestutils.ExampleRateCard1.Key()], 3)
	}))

	t.Run("Should work if subscription has multiple addons associated already", runWithDeps(func(t *testing.T, deps testCaseDeps) {
		// We'll set up a plan with two addons
		// We'll create the sub, add both addons to it
		// We'll then play around with them

		// Let's set up a plan with two compatible addons
		planInp := subscriptiontestutils.BuildTestPlan(t).
			SetMeta(productcatalog.PlanMeta{
				Name:     "Test Plan",
				Key:      "test_plan",
				Version:  1,
				Currency: currency.USD,
				Alignment: productcatalog.Alignment{
					BillablesMustAlign: true,
				},
			}).
			AddPhase(nil, &subscriptiontestutils.ExampleRateCard1).
			Build()

		addonInp1 := subscriptiontestutils.BuildAddonForTesting(t,
			productcatalog.EffectivePeriod{
				EffectiveFrom: &now,
				EffectiveTo:   nil,
			},
			productcatalog.AddonInstanceTypeMultiple,
			subscriptiontestutils.ExampleAddonRateCard5.Clone(),
		)

		addonInp2 := addon.CreateAddonInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: subscriptiontestutils.ExampleNamespace,
			},
			Addon: productcatalog.Addon{
				AddonMeta: productcatalog.AddonMeta{
					Name:        "Test Addon",
					Description: lo.ToPtr("Test Addon Description"),
					EffectivePeriod: productcatalog.EffectivePeriod{
						EffectiveFrom: &now,
						EffectiveTo:   nil,
					},
					Key:          "test-addon-2",
					Version:      1,
					Currency:     currency.USD,
					InstanceType: productcatalog.AddonInstanceTypeMultiple,
					Metadata: models.NewMetadata(map[string]string{
						"test": "test",
					}),
				},
				RateCards: []productcatalog.RateCard{subscriptiontestutils.ExampleAddonRateCard5.Clone()},
			},
		}

		_ = deps.deps.FeatureConnector.CreateExampleFeatures(t)

		p, err := deps.deps.PlanService.CreatePlan(context.Background(), planInp)
		require.Nil(t, err)
		require.NotNil(t, p)

		add1 := deps.deps.AddonService.CreateTestAddon(t, addonInp1)
		add2 := deps.deps.AddonService.CreateTestAddon(t, addonInp2)

		_, err = deps.deps.PlanAddonService.CreatePlanAddon(context.Background(), planaddon.CreatePlanAddonInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: subscriptiontestutils.ExampleNamespace,
			},
			PlanID:        p.ID,
			AddonID:       add1.ID,
			FromPlanPhase: p.Phases[0].Key,
		})
		require.Nil(t, err, "received error: %s", err)

		_, err = deps.deps.PlanAddonService.CreatePlanAddon(context.Background(), planaddon.CreatePlanAddonInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: subscriptiontestutils.ExampleNamespace,
			},
			PlanID:        p.ID,
			AddonID:       add2.ID,
			FromPlanPhase: p.Phases[0].Key,
		})
		require.Nil(t, err, "received error: %s", err)

		p, err = deps.deps.PlanService.PublishPlan(context.Background(), plan.PublishPlanInput{
			NamespacedID: p.NamespacedID,
			EffectivePeriod: productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(clock.Now()),
				EffectiveTo:   lo.ToPtr(testutils.GetRFC3339Time(t, "2099-01-01T00:00:00Z")),
			},
		})
		require.Nil(t, err, "received error: %s", err)

		sp := &plansubscription.Plan{
			Plan: p.AsProductCatalogPlan(),
			Ref:  &p.NamespacedID,
		}

		// Now let's create a subscription from the plan
		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps.deps, sp, now)

		// Now let's add both addons
		subsAddInp1 := subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add1.ID,
			InitialQuantity: 1,
			Timing: subscription.Timing{
				Custom: &now,
			},
		}

		subView, subsAdd1, err := deps.deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, subsAddInp1)
		require.NoError(t, err)
		subAddons, err := deps.deps.SubscriptionAddonService.List(context.Background(), subView.Subscription.Namespace, subscriptionaddon.ListSubscriptionAddonsInput{
			SubscriptionID: subView.Subscription.ID,
		})
		require.NoError(t, err)
		require.Len(t, subAddons.Items, 1)
		require.True(t, lo.ContainsBy(subAddons.Items, func(item subscriptionaddon.SubscriptionAddon) bool {
			return item.ID == subsAdd1.ID
		}))

		subsAddInp2 := subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add2.ID,
			InitialQuantity: 1,
			Timing: subscription.Timing{
				Custom: &now,
			},
		}

		subView, subsAdd2, err := deps.deps.WorkflowService.AddAddon(context.Background(), subView.Subscription.NamespacedID, subsAddInp2)
		require.NoError(t, err)
		subAddons, err = deps.deps.SubscriptionAddonService.List(context.Background(), subView.Subscription.Namespace, subscriptionaddon.ListSubscriptionAddonsInput{
			SubscriptionID: subView.Subscription.ID,
		})
		require.NoError(t, err)
		require.Len(t, subAddons.Items, 2)
		require.True(t, lo.ContainsBy(subAddons.Items, func(item subscriptionaddon.SubscriptionAddon) bool {
			return item.ID == subsAdd2.ID
		}))

		// Now let's try and change the quantity of the first addon to two
		changeInp := subscriptionworkflow.ChangeAddonQuantityWorkflowInput{
			SubscriptionAddonID: subsAdd1.NamespacedID,
			Quantity:            2,
			Timing: subscription.Timing{
				Custom: lo.ToPtr(now.Add(time.Hour * 2)),
			},
		}

		subView, subsAdd1, err = deps.deps.WorkflowService.ChangeAddonQuantity(context.Background(), subView.Subscription.NamespacedID, changeInp)
		require.NoError(t, err)

		// Let's advance some time
		now := now.Add(time.Hour * 4)
		clock.SetTime(now)
		defer clock.ResetTime()

		// Let's change the quantity of the second addon to three
		changeInp2 := subscriptionworkflow.ChangeAddonQuantityWorkflowInput{
			SubscriptionAddonID: subsAdd2.NamespacedID,
			Quantity:            3,
			Timing: subscription.Timing{
				Custom: &now,
			},
		}

		subView, subsAdd2, err = deps.deps.WorkflowService.ChangeAddonQuantity(context.Background(), subView.Subscription.NamespacedID, changeInp2)
		require.NoError(t, err)
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
