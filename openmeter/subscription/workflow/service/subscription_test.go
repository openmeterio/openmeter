package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/registry"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	workflowservice "github.com/openmeterio/openmeter/openmeter/subscription/workflow/service"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/ffx"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestCreateFromPlan(t *testing.T) {
	type testCaseDeps struct {
		Plan            subscription.Plan
		CurrentTime     time.Time
		Customer        customer.Customer
		WorkflowService subscriptionworkflow.Service
		DBDeps          *subscriptiontestutils.DBDeps
	}

	testCases := []struct {
		Name    string
		Handler func(t *testing.T, deps testCaseDeps)
	}{
		{
			Name: "Should error if customer is not found",
			Handler: func(t *testing.T, deps testCaseDeps) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				_, err := deps.WorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
					ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
						Timing: subscription.Timing{
							Custom: &deps.CurrentTime,
						},
					},
					CustomerID: fmt.Sprintf("nonexistent-customer-%s", deps.Customer.ID),
					Namespace:  subscriptiontestutils.ExampleNamespace,
				}, deps.Plan)

				assert.True(t, models.IsGenericNotFoundError(err), "expected customer not found error, got %T", err)
			},
		},
		{
			Name: "Should add annotations to all created items",
			Handler: func(t *testing.T, deps testCaseDeps) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				subView, err := deps.WorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
					ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
						Timing: subscription.Timing{
							Custom: &deps.CurrentTime,
						},
					},
					CustomerID: deps.Customer.ID,
					Namespace:  subscriptiontestutils.ExampleNamespace,
				}, deps.Plan)
				require.Nil(t, err)

				for _, phase := range subView.Phases {
					for _, item := range phase.ItemsByKey {
						for _, i := range item {
							require.NotNil(t, i.SubscriptionItem.Annotations)
							systems := subscription.AnnotationParser.ListOwnerSubSystems(i.SubscriptionItem.Annotations)
							require.Contains(t, systems, subscription.OwnerSubscriptionSubSystem)
						}
					}
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tcDeps := testCaseDeps{
				CurrentTime: testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z"),
			}

			clock.SetTime(tcDeps.CurrentTime)
			dbDeps := subscriptiontestutils.SetupDBDeps(t)
			require.NotNil(t, dbDeps)
			defer dbDeps.Cleanup(t)

			deps := subscriptiontestutils.NewService(t, dbDeps)
			deps.FeatureConnector.CreateExampleFeatures(t)
			plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))
			cust := deps.CustomerAdapter.CreateExampleCustomer(t)
			require.NotNil(t, cust)

			tcDeps.Customer = *cust
			tcDeps.Plan = plan

			tcDeps.DBDeps = dbDeps

			tcDeps.WorkflowService = deps.WorkflowService

			tc.Handler(t, tcDeps)
		})
	}

	t.Run("Should add boolean entitlement count annotations", func(t *testing.T) {
		now := testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z")

		clock.SetTime(now)
		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		require.NotNil(t, dbDeps)
		defer dbDeps.Cleanup(t)

		deps := subscriptiontestutils.NewService(t, dbDeps)
		deps.FeatureConnector.CreateExampleFeatures(t)

		p := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.BuildTestPlanInput(t).
			AddPhase(nil, &subscriptiontestutils.ExampleRateCard3ForAddons, &subscriptiontestutils.ExampleRateCard4ForAddons).
			Build())

		cust := deps.CustomerAdapter.CreateExampleCustomer(t)

		subView, err := deps.WorkflowService.CreateFromPlan(context.Background(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
			ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Timing: subscription.Timing{
					Custom: &now,
				},
			},
			CustomerID: cust.ID,
			Namespace:  subscriptiontestutils.ExampleNamespace,
		}, p)
		require.Nil(t, err)

		nonBoolItem := subView.Phases[0].ItemsByKey[subscriptiontestutils.ExampleRateCard3ForAddons.Key()][0]
		require.NotNil(t, nonBoolItem)
		require.Equal(t, 0, subscription.AnnotationParser.GetBooleanEntitlementCount(nonBoolItem.SubscriptionItem.Annotations))

		boolItem := subView.Phases[0].ItemsByKey[subscriptiontestutils.ExampleRateCard4ForAddons.Key()][0]
		require.NotNil(t, boolItem)
		require.Equal(t, 1, subscription.AnnotationParser.GetBooleanEntitlementCount(boolItem.SubscriptionItem.Annotations))
	})
}

func TestEditRunning(t *testing.T) {
	type testCaseDeps struct {
		CurrentTime     time.Time
		SubView         subscription.SubscriptionView
		Customer        customer.Customer
		WorkflowService subscriptionworkflow.Service
		Service         subscription.Service
		DBDeps          *subscriptiontestutils.DBDeps
		Plan            subscription.Plan
	}

	testCases := []struct {
		Name    string
		Handler func(t *testing.T, deps testCaseDeps)
	}{
		{
			Name: "Should error if subscription is not found",
			Handler: func(t *testing.T, deps testCaseDeps) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				_, err := deps.WorkflowService.EditRunning(ctx, models.NamespacedID{
					ID:        "nonexistent-subscription",
					Namespace: subscriptiontestutils.ExampleNamespace,
				}, nil, immediate)

				assert.ErrorAs(t, err, lo.ToPtr(&subscription.SubscriptionNotFoundError{}), "expected subscription not found error, got %T", err)
			},
		},
		{
			Name: "Should do nothing if no patches are provided",
			Handler: func(t *testing.T, deps testCaseDeps) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				subView, err := deps.WorkflowService.EditRunning(ctx, deps.SubView.Subscription.NamespacedID, nil, immediate)
				assert.Nil(t, err)

				subscriptiontestutils.SpecsEqual(t, deps.SubView.Spec, subView.Spec)
				// cannot be used as function identity always fails (in UsagePeriod.recs[*].GetTime())...
				// assert.Equal(t, deps.SubView, subView)
			},
		},
		{
			Name: "Should validate the provided patches",
			Handler: func(t *testing.T, deps testCaseDeps) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				errorMsg := "this is an invalid patch"

				invalidPatch := subscriptiontestutils.TestPatch{
					ValdiateFn: func() error {
						return errors.New(errorMsg)
					},
				}

				_, err := deps.WorkflowService.EditRunning(ctx, deps.SubView.Subscription.NamespacedID, []subscription.Patch{&invalidPatch}, immediate)
				assert.ErrorContains(t, err, errorMsg, "expected error message to contain %q, got %v", errorMsg, err)
			},
		},
		{
			Name: "Should apply the customizations on the current spec",
			Handler: func(t *testing.T, deps testCaseDeps) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				// Let's increment time by 1 second
				deps.CurrentTime = deps.CurrentTime.Add(time.Second)

				// We have to freeze time here for the assertions
				clock.FreezeTime(deps.CurrentTime)
				defer clock.UnFreeze()

				errMSg := "this custom patch apply failed"

				patch1 := subscriptiontestutils.TestPatch{
					ApplyToFn: func(spec *subscription.SubscriptionSpec, c subscription.ApplyContext) error {
						// Let's assert that the correct spec is passed to the patch
						assert.Equal(t, lo.ToPtr(deps.SubView.AsSpec()), spec, "expected spec to be equal to the current spec")
						assert.Equal(t, subscription.ApplyContext{
							CurrentTime: deps.CurrentTime,
						}, c, "apply context is incorrect")

						// Lets modify the spec to see if its passed to the next
						spec.Name = "modified-name"

						return nil
					},
				}

				patch2 := subscriptiontestutils.TestPatch{
					ApplyToFn: func(spec *subscription.SubscriptionSpec, c subscription.ApplyContext) error {
						// Let's see if the modification is passed along
						assert.Equal(t, "modified-name", spec.Name, "expected name to be modified")

						// Let's return an error to see if it is surfaced
						return errors.New(errMSg)
					},
				}

				_, err := deps.WorkflowService.EditRunning(
					ctx,
					deps.SubView.Subscription.NamespacedID,
					[]subscription.Patch{&patch1, &patch2},
					immediate,
				)
				assert.ErrorContains(t, err, errMSg, "expected error message to contain %q, got %v", errMSg, err)
			},
		},
		{
			Name: "Should use the output of patches without modifications",
			Handler: func(t *testing.T, deps testCaseDeps) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				returnedSpec := deps.SubView.AsSpec()

				pKey := "test_phase_1"

				require.Contains(t, returnedSpec.Phases, pKey, "expected %s to be present in the starting spec", pKey)
				returnedSpec.Phases[pKey].Name = "New Phase 1 Name"
				returnedSpec.Phases[pKey].Description = lo.ToPtr("This is a new description")

				patch1 := subscriptiontestutils.TestPatch{
					ApplyToFn: func(spec *subscription.SubscriptionSpec, c subscription.ApplyContext) error {
						// Let's set the new values

						spec.CreateSubscriptionPlanInput = returnedSpec.CreateSubscriptionPlanInput
						spec.CreateSubscriptionCustomerInput = returnedSpec.CreateSubscriptionCustomerInput
						spec.Phases = returnedSpec.Phases

						return nil
					},
				}

				sID := deps.SubView.Subscription.NamespacedID

				mSvc := subscriptiontestutils.MockService{
					UpdateFn: func(ctx context.Context, id models.NamespacedID, spec subscription.SubscriptionSpec) (subscription.Subscription, error) {
						// Let's validate that the spec is passed as is
						assert.Equal(t, returnedSpec, spec, "expected spec to be equal to the returned spec")

						return deps.Service.Update(ctx, id, spec)
					},
					GetViewFn: func(ctx context.Context, id models.NamespacedID) (subscription.SubscriptionView, error) {
						assert.Equal(t, sID, id, "expected id to be equal to the returned id")

						return deps.Service.GetView(ctx, id)
					},
				}

				tuDeps := subscriptiontestutils.NewService(t, deps.DBDeps)

				lockr, err := lockr.NewLocker(&lockr.LockerConfig{Logger: slog.Default()})
				require.NoError(t, err)
				workflowService := workflowservice.NewWorkflowService(workflowservice.WorkflowServiceConfig{
					Service:            &mSvc,
					CustomerService:    tuDeps.CustomerService,
					TransactionManager: tuDeps.CustomerAdapter,
					AddonService:       tuDeps.SubscriptionAddonService,
					Logger:             slog.Default(),
					Lockr:              lockr,
					FeatureFlags: ffx.NewTestContextService(ffx.AccessConfig{
						subscription.MultiSubscriptionEnabledFF: false,
					}),
				})

				_, err = workflowService.EditRunning(ctx, sID, []subscription.Patch{&patch1}, immediate)
				assert.Nil(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tcDeps := testCaseDeps{
				CurrentTime: testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z"),
			}

			clock.SetTime(tcDeps.CurrentTime)
			defer clock.ResetTime()

			// Let's build the dependencies
			dbDeps := subscriptiontestutils.SetupDBDeps(t)
			require.NotNil(t, dbDeps)
			defer dbDeps.Cleanup(t)

			deps := subscriptiontestutils.NewService(t, dbDeps)
			deps.FeatureConnector.CreateExampleFeatures(t)
			plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))
			cust := deps.CustomerAdapter.CreateExampleCustomer(t)
			require.NotNil(t, cust)

			// Let's create an example subscription
			sub, err := deps.WorkflowService.CreateFromPlan(context.Background(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
				ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Timing: subscription.Timing{
						Custom: &tcDeps.CurrentTime,
					},
					Name: "Example Subscription",
				},
				CustomerID: cust.ID,
				Namespace:  subscriptiontestutils.ExampleNamespace,
			}, plan)
			require.Nil(t, err)

			tcDeps.SubView = sub
			tcDeps.Customer = *cust
			tcDeps.DBDeps = dbDeps
			tcDeps.Service = deps.SubscriptionService
			tcDeps.WorkflowService = deps.WorkflowService
			tcDeps.Plan = plan

			tc.Handler(t, tcDeps)
		})
	}
}

func TestEditingCurrentPhase(t *testing.T) {
	type testCaseDeps struct {
		CurrentTime     time.Time
		SubView         subscription.SubscriptionView
		Customer        customer.Customer
		WorkflowService subscriptionworkflow.Service
		Service         subscription.Service
		ItemRepo        subscription.SubscriptionItemRepository
		DBDeps          *subscriptiontestutils.DBDeps
		Plan            subscription.Plan
		EntReg          *registry.Entitlement
	}

	testCases := []struct {
		Name    string
		Handler func(t *testing.T, deps testCaseDeps)
	}{
		{
			Name: "Should remove item WITHOUT entitlement from the current phase starting now",
			Handler: func(t *testing.T, deps testCaseDeps) {
				second_phase_key := "test_phase_2"
				item_key := "rate-card-2"

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				// Let's assert we have two items in the second phase
				require.GreaterOrEqual(t, len(deps.SubView.Phases), 2, "expected at least two phases")
				require.GreaterOrEqual(t, len(deps.SubView.Phases[1].ItemsByKey), 2, "expected at least two items in the second phase")
				require.Equal(t, second_phase_key, deps.SubView.Phases[1].SubscriptionPhase.Key, "expected the second phase to be of known key")
				itOrigi, ok := deps.SubView.Phases[1].ItemsByKey[item_key]
				require.True(t, ok, "expected item to be present in the second phase")
				require.Len(t, itOrigi, 1, "expected one item to be present")

				// Let's assert the second phase starts when we expect it to
				require.Equal(t, deps.CurrentTime.AddDate(0, 1, 0), deps.SubView.Phases[1].SubscriptionPhase.ActiveFrom, "expected the second phase to start in a month")

				// Let's advance the clock into the 2nd phase where we have two items
				currentTime := deps.CurrentTime.AddDate(0, 1, 1)
				clock.SetTime(currentTime)
				// Let's freeze the time so we can assert properly

				// Let's remove the item without feature & entitlement
				s, err := deps.WorkflowService.EditRunning(ctx, deps.SubView.Subscription.NamespacedID, []subscription.Patch{
					patch.PatchRemoveItem{
						PhaseKey: second_phase_key,
						ItemKey:  item_key,
					},
				}, immediate)
				require.Nil(t, err)
				require.NotNil(t, s)

				// Let's fetch the edited subscription and check that the item was removed effective now
				subView, err := deps.Service.GetView(ctx, deps.SubView.Subscription.NamespacedID)
				require.Nil(t, err)

				// Let's assert that the item is present and has been marked as inactive at the given time
				items, ok := subView.Phases[1].ItemsByKey[item_key]
				require.True(t, ok, "expected item to be present in the second phase")
				assert.Len(t, items, 1, "expected one item to be present")

				tolerance := 5 * time.Second
				testutils.TimeEqualsApproximately(t, currentTime, *items[0].SubscriptionItem.ActiveTo, tolerance)

				// Let's check that the item did get deleted in the background
				// For this, we'll need to do a bit of time travel
				timeBeforeTravel := clock.Now()
				clock.SetTime(currentTime.AddDate(0, 0, -1))

				it, err := deps.ItemRepo.GetByID(ctx, itOrigi[0].SubscriptionItem.NamespacedID)
				require.NoError(t, err)

				testutils.TimeEqualsApproximately(t, currentTime, *it.DeletedAt, tolerance)

				clock.SetTime(timeBeforeTravel)
			},
		},
		{
			Name: "Should remove item WITH entitlement from the current phase starting now",
			Handler: func(t *testing.T, deps testCaseDeps) {
				second_phase_key := "test_phase_2"
				item_key := subscriptiontestutils.ExampleFeatureKey

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				// Let's assert we have two items in the second phase
				require.GreaterOrEqual(t, len(deps.SubView.Phases), 2, "expected at least two phases")
				require.GreaterOrEqual(t, len(deps.SubView.Phases[1].ItemsByKey), 2, "expected at least two items in the second phase")
				require.Equal(t, second_phase_key, deps.SubView.Phases[1].SubscriptionPhase.Key, "expected the second phase to be of known key")
				itOrigi, ok := deps.SubView.Phases[1].ItemsByKey[item_key]
				require.True(t, ok, "expected item to be present in the second phase")
				require.Len(t, itOrigi, 1, "expected one item to be present")

				// Let's assert the second phase starts when we expect it to
				require.Equal(t, deps.CurrentTime.AddDate(0, 1, 0), deps.SubView.Phases[1].SubscriptionPhase.ActiveFrom, "expected the second phase to start in a month")

				// Let's advance the clock into the 2nd phase where we have two items
				currentTime := deps.CurrentTime.AddDate(0, 1, 1)
				clock.SetTime(currentTime)
				// Let's freeze the time so we can assert properly

				// Let's remove the item without
				s, err := deps.WorkflowService.EditRunning(ctx, deps.SubView.Subscription.NamespacedID, []subscription.Patch{
					patch.PatchRemoveItem{
						PhaseKey: second_phase_key,
						ItemKey:  item_key,
					},
				}, immediate)
				require.Nil(t, err)
				require.NotNil(t, s)

				// Let's fetch the edited subscription and check that the item was removed effective now
				subView, err := deps.Service.GetView(ctx, deps.SubView.Subscription.NamespacedID)
				require.Nil(t, err)

				// Let's assert that the item is present and has been marked as inactive at the given time
				items, ok := subView.Phases[1].ItemsByKey[item_key]
				require.True(t, ok, "expected item to be present in the second phase")
				assert.Len(t, items, 1, "expected one item to be present")

				tolerance := 5 * time.Second
				testutils.TimeEqualsApproximately(t, currentTime, *items[0].SubscriptionItem.ActiveTo, tolerance)

				// Let's check that the item & entitlement did get deleted in the background
				// For this, we'll need to do a bit of time travel
				timeBeforeTravel := clock.Now()
				clock.SetTime(currentTime.AddDate(0, 0, -1))

				it, err := deps.ItemRepo.GetByID(ctx, itOrigi[0].SubscriptionItem.NamespacedID)
				require.NoError(t, err)

				testutils.TimeEqualsApproximately(t, currentTime, *it.DeletedAt, tolerance)

				require.NotNil(t, it.EntitlementID)

				ent, err := deps.EntReg.EntitlementRepo.GetEntitlement(ctx, models.NamespacedID{
					Namespace: it.Namespace,
					ID:        *it.EntitlementID,
				})
				require.Nil(t, err)

				testutils.TimeEqualsApproximately(t, currentTime, *ent.DeletedAt, tolerance)

				clock.SetTime(timeBeforeTravel)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tcDeps := testCaseDeps{
				CurrentTime: testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z"),
			}

			clock.SetTime(tcDeps.CurrentTime)

			// Let's build the dependencies
			dbDeps := subscriptiontestutils.SetupDBDeps(t)
			require.NotNil(t, dbDeps)
			defer dbDeps.Cleanup(t)

			deps := subscriptiontestutils.NewService(t, dbDeps)
			deps.FeatureConnector.CreateExampleFeatures(t)
			plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))
			cust := deps.CustomerAdapter.CreateExampleCustomer(t)
			require.NotNil(t, cust)

			// Let's create an example subscription
			sub, err := deps.WorkflowService.CreateFromPlan(context.Background(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
				ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Timing: subscription.Timing{
						Custom: &tcDeps.CurrentTime,
					},
					Name: "Example Subscription",
				},
				CustomerID: cust.ID,
				Namespace:  subscriptiontestutils.ExampleNamespace,
			}, plan)
			require.Nil(t, err)

			tcDeps.SubView = sub
			tcDeps.Customer = *cust
			tcDeps.DBDeps = dbDeps
			tcDeps.Service = deps.SubscriptionService
			tcDeps.WorkflowService = deps.WorkflowService
			tcDeps.Plan = plan
			tcDeps.ItemRepo = deps.ItemRepo
			tcDeps.EntReg = deps.EntitlementRegistry

			tc.Handler(t, tcDeps)
		})
	}
}

func TestEditingWithTiming(t *testing.T) {
	type testCaseDeps struct {
		CurrentTime     time.Time
		SubView         subscription.SubscriptionView
		Customer        customer.Customer
		WorkflowService subscriptionworkflow.Service
		Service         subscription.Service
		ItemRepo        subscription.SubscriptionItemRepository
		DBDeps          *subscriptiontestutils.DBDeps
		Plan            subscription.Plan
		EntReg          *registry.Entitlement
	}
	testCases := []struct {
		Name      string
		IsAligned bool
		Handler   func(t *testing.T, deps testCaseDeps)
	}{
		{
			Name: "Should work with next_billing_cycle in an aligned Subscription",
			Handler: func(t *testing.T, deps testCaseDeps) {
				second_phase_key := "test_phase_2"
				item_key := "rate-card-2"

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				// Let's assert we have two items in the second phase
				require.GreaterOrEqual(t, len(deps.SubView.Phases), 2, "expected at least two phases")
				require.GreaterOrEqual(t, len(deps.SubView.Phases[1].ItemsByKey), 2, "expected at least two items in the second phase")
				require.Equal(t, second_phase_key, deps.SubView.Phases[1].SubscriptionPhase.Key, "expected the second phase to be of known key")
				itOrigi, ok := deps.SubView.Phases[1].ItemsByKey[item_key]
				require.True(t, ok, "expected item to be present in the second phase")
				require.Len(t, itOrigi, 1, "expected one item to be present")

				// Let's assert the second phase starts when we expect it to
				require.Equal(t, deps.CurrentTime.AddDate(0, 1, 0), deps.SubView.Phases[1].SubscriptionPhase.ActiveFrom, "expected the second phase to start in a month")

				// Let's advance the clock into the 2nd phase where we have two items
				currentTime := deps.CurrentTime.AddDate(0, 1, 1)
				clock.SetTime(currentTime)
				defer clock.ResetTime()

				// Let's remove the item without feature & entitlement at the end of the billingcycle
				_, err := deps.WorkflowService.EditRunning(ctx, deps.SubView.Subscription.NamespacedID, []subscription.Patch{
					patch.PatchRemoveItem{
						PhaseKey: second_phase_key,
						ItemKey:  item_key,
					},
				}, subscription.Timing{
					Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
				})
				require.NoError(t, err)
			},
		},
		{
			Name:      "Should error when trying to time to next_billing_cycle when that falls into a different phase (ergo no next billingcycle in the current phase)",
			IsAligned: true,
			Handler: func(t *testing.T, deps testCaseDeps) {
				second_phase_key := "test_phase_2"
				item_key := "rate-card-2"

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				// Let's assert we have two items in the second phase
				require.GreaterOrEqual(t, len(deps.SubView.Phases), 3, "expected at least three phases")
				require.GreaterOrEqual(t, len(deps.SubView.Phases[1].ItemsByKey), 2, "expected at least two items in the second phase")
				require.Equal(t, second_phase_key, deps.SubView.Phases[1].SubscriptionPhase.Key, "expected the second phase to be of known key")
				itOrigi, ok := deps.SubView.Phases[1].ItemsByKey[item_key]
				require.True(t, ok, "expected item to be present in the second phase")
				require.Len(t, itOrigi, 1, "expected one item to be present")

				// Let's assert the second phase starts when we expect it to
				require.Equal(t, deps.CurrentTime.AddDate(0, 1, 0), deps.SubView.Phases[1].SubscriptionPhase.ActiveFrom, "expected the second phase to start in a month")

				// Let's assert that the third phase starts when we expct it to
				require.Equal(t, deps.CurrentTime.AddDate(0, 3, 0), deps.SubView.Phases[2].SubscriptionPhase.ActiveFrom, "expected the third phase to start in 3 months")

				// Let's advance the clock into the 2nd cycle of the 2nd phase where we have two items
				currentTime := deps.CurrentTime.AddDate(0, 2, 1)
				clock.SetTime(currentTime)
				defer clock.ResetTime()

				// Let's remove the item without feature & entitlement at the end of the billingcycle
				_, err := deps.WorkflowService.EditRunning(ctx, deps.SubView.Subscription.NamespacedID, []subscription.Patch{
					patch.PatchRemoveItem{
						PhaseKey: second_phase_key,
						ItemKey:  item_key,
					},
				}, subscription.Timing{
					Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
				})
				require.Error(t, err)

				require.ErrorAs(t, err, lo.ToPtr(&models.GenericValidationError{}), "expected error to be of type models.GenericUserError")
				require.ErrorContains(t, err, "cannot edit to the next billing cycle as it falls into a different phase", "expected error to be about next billing cycle, while it was: %v", err)
			},
		},
		{
			Name:      "Should error when trying to time to a specified time",
			IsAligned: true,
			Handler: func(t *testing.T, deps testCaseDeps) {
				second_phase_key := "test_phase_2"
				item_key := "rate-card-2"

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				// Let's assert we have two items in the second phase
				require.GreaterOrEqual(t, len(deps.SubView.Phases), 3, "expected at least three phases")
				require.GreaterOrEqual(t, len(deps.SubView.Phases[1].ItemsByKey), 2, "expected at least two items in the second phase")
				require.Equal(t, second_phase_key, deps.SubView.Phases[1].SubscriptionPhase.Key, "expected the second phase to be of known key")
				itOrigi, ok := deps.SubView.Phases[1].ItemsByKey[item_key]
				require.True(t, ok, "expected item to be present in the second phase")
				require.Len(t, itOrigi, 1, "expected one item to be present")

				// Let's assert the second phase starts when we expect it to
				require.Equal(t, deps.CurrentTime.AddDate(0, 1, 0), deps.SubView.Phases[1].SubscriptionPhase.ActiveFrom, "expected the second phase to start in a month")

				// Let's assert that the third phase starts when we expct it to
				require.Equal(t, deps.CurrentTime.AddDate(0, 3, 0), deps.SubView.Phases[2].SubscriptionPhase.ActiveFrom, "expected the third phase to start in 3 months")

				// Let's advance the clock into the 2nd cycle of the 2nd phase where we have two items
				currentTime := deps.CurrentTime.AddDate(0, 1, 1)
				clock.SetTime(currentTime)
				defer clock.ResetTime()

				// Let's remove the item without feature & entitlement at the end of the billingcycle
				_, err := deps.WorkflowService.EditRunning(ctx, deps.SubView.Subscription.NamespacedID, []subscription.Patch{
					patch.PatchRemoveItem{
						PhaseKey: second_phase_key,
						ItemKey:  item_key,
					},
				}, subscription.Timing{
					Custom: lo.ToPtr(currentTime.Add(time.Hour)),
				})
				require.Error(t, err)

				require.ErrorAs(t, err, lo.ToPtr(&models.GenericValidationError{}), "expected error to be of type models.GenericUserError")
				require.ErrorContains(t, err, "cannot edit running subscription with custom timing", "expected error to be about custom timing, while it was: %v", err)
			},
		},
		{
			Name:      "Should edit with the start of the next billing cycle",
			IsAligned: true,
			Handler: func(t *testing.T, deps testCaseDeps) {
				second_phase_key := "test_phase_2"
				item_key := "rate-card-2"

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				// Let's assert we have two items in the second phase
				require.GreaterOrEqual(t, len(deps.SubView.Phases), 3, "expected at least three phases")
				require.GreaterOrEqual(t, len(deps.SubView.Phases[1].ItemsByKey), 2, "expected at least two items in the second phase")
				require.Equal(t, second_phase_key, deps.SubView.Phases[1].SubscriptionPhase.Key, "expected the second phase to be of known key")
				itOrigi, ok := deps.SubView.Phases[1].ItemsByKey[item_key]
				require.True(t, ok, "expected item to be present in the second phase")
				require.Len(t, itOrigi, 1, "expected one item to be present")

				// Let's assert the second phase starts when we expect it to
				require.Equal(t, deps.CurrentTime.AddDate(0, 1, 0), deps.SubView.Phases[1].SubscriptionPhase.ActiveFrom, "expected the second phase to start in a month")

				// Let's assert that the third phase starts when we expct it to
				require.Equal(t, deps.CurrentTime.AddDate(0, 3, 0), deps.SubView.Phases[2].SubscriptionPhase.ActiveFrom, "expected the third phase to start in 3 months")

				// Let's advance the clock into the 2nd cycle of the 2nd phase where we have two items
				currentTime := deps.CurrentTime.AddDate(0, 1, 1)
				clock.SetTime(currentTime)
				defer clock.ResetTime()

				// Let's remove the item without feature & entitlement at the end of the billingcycle
				view, err := deps.WorkflowService.EditRunning(ctx, deps.SubView.Subscription.NamespacedID, []subscription.Patch{
					patch.PatchRemoveItem{
						PhaseKey: second_phase_key,
						ItemKey:  item_key,
					},
				}, subscription.Timing{
					Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
				})
				require.NoError(t, err)

				expectedEditTime := deps.CurrentTime.AddDate(0, 2, 0)

				// Let's validate that the removed item is present until that point
				items, ok := view.Phases[1].ItemsByKey[item_key]
				require.True(t, ok, "expected item to be present in the second phase")

				assert.Len(t, items, 1, "expected one item to be present")
				testutils.TimeEqualsApproximately(t, expectedEditTime, *items[0].SubscriptionItem.ActiveTo, time.Second)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tcDeps := testCaseDeps{
				CurrentTime: testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z"),
			}

			clock.SetTime(tcDeps.CurrentTime)

			// Let's build the dependencies
			dbDeps := subscriptiontestutils.SetupDBDeps(t)
			require.NotNil(t, dbDeps)
			defer dbDeps.Cleanup(t)

			deps := subscriptiontestutils.NewService(t, dbDeps)
			deps.FeatureConnector.CreateExampleFeatures(t)
			planInput := subscriptiontestutils.GetExamplePlanInput(t)

			plan := deps.PlanHelper.CreatePlan(t, planInput)
			cust := deps.CustomerAdapter.CreateExampleCustomer(t)
			require.NotNil(t, cust)

			// Let's create an example subscription
			sub, err := deps.WorkflowService.CreateFromPlan(context.Background(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
				ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Timing: subscription.Timing{
						Custom: &tcDeps.CurrentTime,
					},
					Name: "Example Subscription",
				},
				CustomerID: cust.ID,
				Namespace:  subscriptiontestutils.ExampleNamespace,
			}, plan)
			require.Nil(t, err)

			tcDeps.SubView = sub
			tcDeps.Customer = *cust
			tcDeps.DBDeps = dbDeps
			tcDeps.Service = deps.SubscriptionService
			tcDeps.WorkflowService = deps.WorkflowService
			tcDeps.Plan = plan
			tcDeps.ItemRepo = deps.ItemRepo
			tcDeps.EntReg = deps.EntitlementRegistry

			tc.Handler(t, tcDeps)
		})
	}
}

func TestChangeToPlan(t *testing.T) {
	// Let's define two plans. One is the example plan, and the second is a slightly modified version of that
	examplePlanInput1 := subscriptiontestutils.GetExamplePlanInput(t)

	rc1 := productcatalog.UsageBasedRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:         subscriptiontestutils.ExampleFeatureKey,
			Name:        "Rate Card 1",
			Description: lo.ToPtr("Rate Card 1 Description"),
			FeatureKey:  lo.ToPtr(subscriptiontestutils.ExampleFeatureKey),
			FeatureID:   nil,
			EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.MeteredEntitlementTemplate{
				IssueAfterReset: lo.ToPtr(111.0),
				UsagePeriod:     subscriptiontestutils.ISOMonth,
			}),
			TaxConfig: &productcatalog.TaxConfig{
				Stripe: &productcatalog.StripeTaxConfig{
					Code: "txcd_10000001",
				},
			},
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromInt(int64(1)),
			}),
		},
		BillingCadence: subscriptiontestutils.ISOMonth,
	}

	rc2 := productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:         "new-rc-2",
			Name:        "Rate Card 2",
			Description: lo.ToPtr("Rate Card 2 Description"),
			TaxConfig: &productcatalog.TaxConfig{
				Stripe: &productcatalog.StripeTaxConfig{
					Code: "txcd_10000001",
				},
			},
			Price: nil,
		},
		BillingCadence: nil,
	}

	examplePlanInput2 := plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: subscriptiontestutils.ExampleNamespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Test Plan 2",
				Key:            "test_plan_2",
				Version:        1,
				Currency:       currency.USD,
				BillingCadence: datetime.MustParseDuration(t, "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:         "test_phase_1_new",
						Name:        "Test Phase 1 New",
						Description: lo.ToPtr("Test Phase 1 Description"),
						Duration:    lo.ToPtr(datetime.MustParseDuration(t, "P2M")),
					},
					RateCards: productcatalog.RateCards{
						&rc1,
					},
				},
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:         "test_phase_2",
						Name:        "Test Phase 2",
						Description: lo.ToPtr("Test Phase 2 Description"),
						Duration:    lo.ToPtr(datetime.MustParseDuration(t, "P1M")),
					},
					RateCards: productcatalog.RateCards{
						&subscriptiontestutils.ExampleRateCard1,
						&rc2,
					},
				},
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:         "test_phase_3",
						Name:        "Test Phase 3",
						Description: lo.ToPtr("Test Phase 3 Description"),
						Duration:    nil,
					},
					RateCards: productcatalog.RateCards{
						&rc1,
					},
				},
			},
		},
	}

	// Let's define what deps a test case needs
	type testCaseDeps struct {
		CurrentTime     time.Time
		Customer        customer.Customer
		WorkflowService subscriptionworkflow.Service
		Service         subscription.Service
		DBDeps          *subscriptiontestutils.DBDeps
		Plan1           subscription.Plan
		Plan2           subscription.Plan
	}

	withDeps := func(t *testing.T) func(fn func(t *testing.T, deps testCaseDeps)) {
		return func(fn func(t *testing.T, deps testCaseDeps)) {
			tcDeps := testCaseDeps{
				CurrentTime: testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z"),
			}

			clock.SetTime(tcDeps.CurrentTime)

			// Let's build the dependencies
			dbDeps := subscriptiontestutils.SetupDBDeps(t)
			require.NotNil(t, dbDeps)
			defer dbDeps.Cleanup(t)

			deps := subscriptiontestutils.NewService(t, dbDeps)
			deps.FeatureConnector.CreateExampleFeatures(t)

			// Let's create the two plans
			plan1 := deps.PlanHelper.CreatePlan(t, examplePlanInput1)
			plan2 := deps.PlanHelper.CreatePlan(t, examplePlanInput2)

			cust := deps.CustomerAdapter.CreateExampleCustomer(t)
			require.NotNil(t, cust)

			tcDeps.Customer = *cust
			tcDeps.DBDeps = dbDeps
			tcDeps.Service = deps.SubscriptionService
			tcDeps.WorkflowService = deps.WorkflowService
			tcDeps.Plan1 = plan1
			tcDeps.Plan2 = plan2

			fn(t, tcDeps)
		}
	}

	t.Run("Should change to the new plan", func(t *testing.T) {
		withDeps(t)(func(t *testing.T, deps testCaseDeps) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// First, let's create a subscription from the first plan
			// Let's create an example subscription
			sub, err := deps.WorkflowService.CreateFromPlan(context.Background(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
				ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Timing: subscription.Timing{
						Custom: &deps.CurrentTime,
					},
					Name: "Example Subscription",
				},
				CustomerID: deps.Customer.ID,
				Namespace:  subscriptiontestutils.ExampleNamespace,
			}, deps.Plan1)
			require.Nil(t, err)

			changeInput := subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Timing: subscription.Timing{
					Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
				},
				Name: "New Subscription",
			}

			curr, new, err := deps.WorkflowService.ChangeToPlan(ctx, sub.Subscription.NamespacedID, changeInput, deps.Plan2)
			require.Nil(t, err)

			// Let's do some simple validations
			require.Equal(t, sub.Subscription.NamespacedID, curr.NamespacedID)
			require.NotNil(t, curr.ActiveTo)

			require.NotNil(t, new.Subscription.PlanRef)
			require.Equal(t, examplePlanInput2.Key, new.Subscription.PlanRef.Key)

			// Let's check that the new plan looks as we expect
			targetSpec, err := subscription.NewSpecFromPlan(deps.Plan2, subscription.CreateSubscriptionCustomerInput{
				Name:          changeInput.Name,
				Description:   changeInput.Description,
				MetadataModel: changeInput.MetadataModel,
				CustomerId:    curr.CustomerId,
				Currency:      deps.Plan2.Currency(),
				ActiveFrom:    testutils.GetRFC3339Time(t, "2021-02-01T00:00:00Z"),
				BillingAnchor: sub.Subscription.BillingAnchor,
				ActiveTo:      nil,
			})
			require.Nil(t, err)

			subscriptiontestutils.ValidateSpecAndView(t, targetSpec, new)
		})
	})
}

func TestEditCombinations(t *testing.T) {
	// Let's define what deps a test case needs
	type testCaseDeps struct {
		CurrentTime     time.Time
		Customer        customer.Customer
		WorkflowService subscriptionworkflow.Service
		Service         subscription.Service
		DBDeps          *subscriptiontestutils.DBDeps
		Plan1           subscription.Plan
		SubsDeps        *subscriptiontestutils.SubscriptionDependencies
	}

	withDeps := func(t *testing.T) func(fn func(t *testing.T, deps testCaseDeps)) {
		return func(fn func(t *testing.T, deps testCaseDeps)) {
			examplePlanInput1 := subscriptiontestutils.GetExamplePlanInput(t)

			tcDeps := testCaseDeps{
				CurrentTime: testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z"),
			}

			clock.SetTime(tcDeps.CurrentTime)

			// Let's build the dependencies
			dbDeps := subscriptiontestutils.SetupDBDeps(t)
			require.NotNil(t, dbDeps)
			defer dbDeps.Cleanup(t)

			deps := subscriptiontestutils.NewService(t, dbDeps)
			deps.FeatureConnector.CreateExampleFeatures(t)

			// Let's create the plan
			plan1 := deps.PlanHelper.CreatePlan(t, examplePlanInput1)

			cust := deps.CustomerAdapter.CreateExampleCustomer(t)
			require.NotNil(t, cust)

			tcDeps.Customer = *cust
			tcDeps.DBDeps = dbDeps
			tcDeps.Service = deps.SubscriptionService
			tcDeps.WorkflowService = deps.WorkflowService
			tcDeps.Plan1 = plan1
			tcDeps.SubsDeps = &deps

			fn(t, tcDeps)
		}
	}

	t.Run("Should error on edit if subscription has addons purchased", func(t *testing.T) {
		now := testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z")

		clock.SetTime(now)

		// Let's build the dependencies
		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		require.NotNil(t, dbDeps)
		defer dbDeps.Cleanup(t)

		deps := subscriptiontestutils.NewService(t, dbDeps)

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

		cust := deps.CustomerAdapter.CreateExampleCustomer(t)

		subView, err := deps.WorkflowService.CreateFromPlan(context.Background(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
			ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Timing: subscription.Timing{
					Custom: &now,
				},
				Name: "Example Subscription",
			},
			CustomerID: cust.ID,
			Namespace:  subscriptiontestutils.ExampleNamespace,
		}, p)
		require.Nil(t, err)

		_ = subscriptiontestutils.CreateAddonForSubscription(t, &deps, subView.Subscription.NamespacedID, add.NamespacedID, models.CadencedModel{
			ActiveFrom: now,
			ActiveTo:   nil,
		})

		_, err = deps.WorkflowService.EditRunning(context.Background(), subView.Subscription.NamespacedID, []subscription.Patch{
			patch.PatchRemoveItem{
				PhaseKey: "test_phase_1",
				ItemKey:  subscriptiontestutils.ExampleFeatureKey,
			},
		}, immediate)
		require.Error(t, err)
		require.ErrorAs(t, err, lo.ToPtr(&models.GenericForbiddenError{}))
		require.True(t, models.IsGenericForbiddenError(err))
	})

	t.Run("Should add boolean entitlement count annotations", func(t *testing.T) {
		withDeps(t)(func(t *testing.T, deps testCaseDeps) {
			plan := deps.SubsDeps.PlanHelper.CreatePlan(t, subscriptiontestutils.BuildTestPlanInput(t).
				AddPhase(nil, &subscriptiontestutils.ExampleRateCard3ForAddons).
				Build())

			timingNow := subscription.Timing{
				Enum: lo.ToPtr(subscription.TimingImmediate),
			}

			subView, err := deps.WorkflowService.CreateFromPlan(context.Background(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
				Namespace:  deps.Customer.Namespace,
				CustomerID: deps.Customer.ID,
				ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Name:   "test",
					Timing: timingNow,
				},
			}, plan)
			require.NoError(t, err)

			subView, err = deps.WorkflowService.EditRunning(context.Background(), subView.Subscription.NamespacedID, []subscription.Patch{
				patch.PatchAddItem{
					PhaseKey: "test_phase_1",
					ItemKey:  subscriptiontestutils.ExampleRateCard4ForAddons.Key(),
					CreateInput: subscription.SubscriptionItemSpec{
						CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
							CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
								PhaseKey: "test_phase_1",
								ItemKey:  subscriptiontestutils.ExampleRateCard4ForAddons.Key(),
								RateCard: &subscriptiontestutils.ExampleRateCard4ForAddons,
							},
						},
					},
				},
			}, timingNow)
			require.Nil(t, err)

			t.Run("Should properly serialize and deserialize SubscriptionView", func(t *testing.T) {
				viewBytes, err := json.MarshalIndent(subView, "", "  ")
				require.Nil(t, err)

				var targetView subscription.SubscriptionView
				err = json.Unmarshal(viewBytes, &targetView)
				require.Nil(t, err)

				// Lets test the item's ActiveFromOverrideRelativeToPhaseStart
				require.Equal(t, subView.Phases[0].ItemsByKey[subscriptiontestutils.ExampleRateCard4ForAddons.Key()][0].Spec.ActiveFromOverrideRelativeToPhaseStart, targetView.Phases[0].ItemsByKey[subscriptiontestutils.ExampleRateCard4ForAddons.Key()][0].Spec.ActiveFromOverrideRelativeToPhaseStart)
			})

			boolItem := subView.Phases[0].ItemsByKey[subscriptiontestutils.ExampleRateCard4ForAddons.Key()][0]
			require.NotNil(t, boolItem)
			require.Equal(t, 1, subscription.AnnotationParser.GetBooleanEntitlementCount(boolItem.SubscriptionItem.Annotations))
		})
	})

	t.Run("Should be able to cancel an edited subscription", func(t *testing.T) {
		withDeps(t)(func(t *testing.T, deps testCaseDeps) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Let's create an example subscription
			sub, err := deps.WorkflowService.CreateFromPlan(context.Background(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
				ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Timing: subscription.Timing{
						Custom: &deps.CurrentTime,
					},
					Name: "Example Subscription",
				},
				CustomerID: deps.Customer.ID,
				Namespace:  subscriptiontestutils.ExampleNamespace,
			}, deps.Plan1)
			require.Nil(t, err)

			// Let's make sure the sub looks as we expect it to
			require.Equal(t, "test_phase_1", sub.Phases[0].SubscriptionPhase.Key)

			// Let's make sure it has the rate card we're editing
			values, ok := sub.Phases[1].ItemsByKey[subscriptiontestutils.ExampleFeatureKey]
			require.True(t, ok)
			require.Equal(t, 1, len(values))
			val := values[0]

			rc := val.Spec.RateCard
			require.NoError(t, rc.ChangeMeta(func(m productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
				m.Price = productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(19)})
				return m, nil
			}))

			// Let's edit the subscription
			edits := []subscription.Patch{
				// Let's edit an Item that has an Entitlement Associated
				patch.PatchRemoveItem{
					PhaseKey: "test_phase_1",
					ItemKey:  subscriptiontestutils.ExampleFeatureKey,
				},
				patch.PatchAddItem{
					PhaseKey: "test_phase_1",
					ItemKey:  subscriptiontestutils.ExampleFeatureKey,
					CreateInput: subscription.SubscriptionItemSpec{
						CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
							CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
								PhaseKey: "test_phase_1",
								ItemKey:  subscriptiontestutils.ExampleFeatureKey,
								RateCard: rc,
							},
							CreateSubscriptionItemCustomerInput: subscription.CreateSubscriptionItemCustomerInput{},
						},
					},
				},
			}

			// Let 5 minutes pass
			clock.SetTime(deps.CurrentTime.Add(5 * time.Minute))

			_, err = deps.WorkflowService.EditRunning(ctx, sub.Subscription.NamespacedID, edits, immediate)
			require.Nil(t, err)

			// Now let's fetch the view
			view, err := deps.Service.GetView(ctx, sub.Subscription.NamespacedID)
			require.Nil(t, err)

			require.Equal(t, sub.Subscription.NamespacedID, view.Subscription.NamespacedID)

			t.Run("Should add owner annotations", func(t *testing.T) {
				items := view.Phases[0].ItemsByKey[subscriptiontestutils.ExampleFeatureKey]
				addedItem := items[len(items)-1]

				ownerSystems := subscription.AnnotationParser.ListOwnerSubSystems(addedItem.SubscriptionItem.Annotations)
				require.Equal(t, 1, len(ownerSystems))
				require.Equal(t, subscription.OwnerSubscriptionSubSystem, ownerSystems[0])
			})

			// Now let's cancel the subscription
			s, err := deps.Service.Cancel(ctx, sub.Subscription.NamespacedID, subscription.Timing{
				Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
			})
			require.Nil(t, err)

			require.Equal(t, subscription.SubscriptionStatusCanceled, s.GetStatusAt(clock.Now()))
			require.Equal(t, subscription.SubscriptionStatusInactive, s.GetStatusAt(deps.CurrentTime.AddDate(0, 1, 0)))
		})
	})

	t.Run("Let's assert that subscriptions can be cascade deleted", func(t *testing.T) {
		// This will be a bit of a hacky test, but its better see it enforced than to leave it to human testing
		// We'll create a subscription
		// Then we'll delete the subscription
		// Then we'll assert that there are no
		// - phases left
		// - items left
		// - entitlements left
		// - grants left
		withDeps(t)(func(t *testing.T, deps testCaseDeps) {
			// Let's create an example subscription
			_, err := deps.WorkflowService.CreateFromPlan(t.Context(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
				ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Timing: subscription.Timing{
						Custom: &deps.CurrentTime,
					},
					Name: "Example Subscription",
				},
				CustomerID: deps.Customer.ID,
				Namespace:  subscriptiontestutils.ExampleNamespace,
			}, deps.Plan1)
			require.Nil(t, err)

			// Now that we've got a subscription we'll use raw SQL to do the rest, this is the hacky part

			res, err := deps.DBDeps.DBClient.ExecContext(t.Context(), `DELETE FROM subscriptions`)
			require.Nil(t, err)

			affected, err := res.RowsAffected()
			require.Nil(t, err)
			require.Equal(t, int64(1), affected) // deleted the one subscription we had

			// Now let's query the DB, lets encapsulate in inline functions for referential integrity
			func() {
				rows, err := deps.DBDeps.DBClient.QueryContext(t.Context(), `SELECT COUNT(*) FROM subscription_phases`)
				require.Nil(t, err)
				defer rows.Close()

				for rows.Next() {
					var count int
					require.Nil(t, rows.Scan(&count))
					require.Equal(t, 0, count)
				}
			}()

			func() {
				rows, err := deps.DBDeps.DBClient.QueryContext(t.Context(), `SELECT COUNT(*) FROM subscription_items`)
				require.Nil(t, err)
				defer rows.Close()

				for rows.Next() {
					var count int
					require.Nil(t, rows.Scan(&count))
					require.Equal(t, 0, count)
				}
			}()

			func() {
				rows, err := deps.DBDeps.DBClient.QueryContext(t.Context(), `SELECT COUNT(*) FROM entitlements`)
				require.Nil(t, err)
				defer rows.Close()

				for rows.Next() {
					var count int
					require.Nil(t, rows.Scan(&count))
					require.Equal(t, 0, count)
				}
			}()

			func() {
				rows, err := deps.DBDeps.DBClient.QueryContext(t.Context(), `SELECT COUNT(*) FROM grants`)
				require.Nil(t, err)
				defer rows.Close()

				for rows.Next() {
					var count int
					require.Nil(t, rows.Scan(&count))
					require.Equal(t, 0, count)
				}
			}()

			// Just to be safe, let's also assert the things we're not deleting, being
			// - customers
			// - plans
			// - features
			func() {
				rows, err := deps.DBDeps.DBClient.QueryContext(t.Context(), `SELECT COUNT(*) FROM customers`)
				require.Nil(t, err)
				defer rows.Close()

				for rows.Next() {
					var count int
					require.Nil(t, rows.Scan(&count))
					require.NotEqual(t, 0, count)
				}
			}()

			func() {
				rows, err := deps.DBDeps.DBClient.QueryContext(t.Context(), `SELECT COUNT(*) FROM plans`)
				require.Nil(t, err)
				defer rows.Close()

				for rows.Next() {
					var count int
					require.Nil(t, rows.Scan(&count))
					require.NotEqual(t, 0, count)
				}
			}()

			func() {
				rows, err := deps.DBDeps.DBClient.QueryContext(t.Context(), `SELECT COUNT(*) FROM features`)
				require.Nil(t, err)
				defer rows.Close()

				for rows.Next() {
					var count int
					require.Nil(t, rows.Scan(&count))
					require.NotEqual(t, 0, count)
				}
			}()
		})
	})
}

func TestRestore(t *testing.T) {
	// Let's define what deps a test case needs
	type testCaseDeps struct {
		CurrentTime     time.Time
		Customer        customer.Customer
		WorkflowService subscriptionworkflow.Service
		Service         subscription.Service
		DBDeps          *subscriptiontestutils.DBDeps
		Plan1           subscription.Plan
	}

	withDeps := func(t *testing.T) func(fn func(t *testing.T, deps testCaseDeps)) {
		return func(fn func(t *testing.T, deps testCaseDeps)) {
			examplePlanInput1 := subscriptiontestutils.GetExamplePlanInput(t)

			tcDeps := testCaseDeps{
				CurrentTime: testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z"),
			}

			clock.SetTime(tcDeps.CurrentTime)

			// Let's build the dependencies
			dbDeps := subscriptiontestutils.SetupDBDeps(t)
			require.NotNil(t, dbDeps)
			defer dbDeps.Cleanup(t)

			deps := subscriptiontestutils.NewService(t, dbDeps)
			deps.FeatureConnector.CreateExampleFeatures(t)

			// Let's create the plan
			plan1 := deps.PlanHelper.CreatePlan(t, examplePlanInput1)

			cust := deps.CustomerAdapter.CreateExampleCustomer(t)
			require.NotNil(t, cust)

			tcDeps.Customer = *cust
			tcDeps.DBDeps = dbDeps
			tcDeps.Service = deps.SubscriptionService
			tcDeps.WorkflowService = deps.WorkflowService
			tcDeps.Plan1 = plan1

			fn(t, tcDeps)
		}
	}

	t.Run("Should restore a subscription that was canceled", func(t *testing.T) {
		withDeps(t)(func(t *testing.T, deps testCaseDeps) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Let's create an example subscription
			sub, err := deps.WorkflowService.CreateFromPlan(context.Background(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
				ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Timing: subscription.Timing{
						Custom: &deps.CurrentTime,
					},
					Name: "Example Subscription",
				},
				CustomerID: deps.Customer.ID,
				Namespace:  subscriptiontestutils.ExampleNamespace,
			}, deps.Plan1)
			require.Nil(t, err)

			// Let's pass some time
			clock.SetTime(clock.Now().Add(time.Hour))

			// Let's cancel the subscription
			s, err := deps.Service.Cancel(ctx, sub.Subscription.NamespacedID, subscription.Timing{
				Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
			})
			require.Nil(t, err)

			// Let's validate its canceled
			require.Equal(t, subscription.SubscriptionStatusCanceled, s.GetStatusAt(clock.Now()))

			// Let's pass some more time
			clock.SetTime(deps.CurrentTime.AddDate(0, 0, 1))

			// Let's restore the subscription
			restored, err := deps.WorkflowService.Restore(ctx, sub.Subscription.NamespacedID)
			require.Nil(t, err)

			// Let's validate its active
			require.Equal(t, subscription.SubscriptionStatusActive, restored.GetStatusAt(clock.Now()))
		})
	})

	t.Run("Should restore a subscription that was changed to another plan", func(t *testing.T) {
		withDeps(t)(func(t *testing.T, deps testCaseDeps) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Let's create an example subscription
			sub, err := deps.WorkflowService.CreateFromPlan(context.Background(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
				ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Timing: subscription.Timing{
						Custom: &deps.CurrentTime,
					},
					Name: "Example Subscription",
				},
				CustomerID: deps.Customer.ID,
				Namespace:  subscriptiontestutils.ExampleNamespace,
			}, deps.Plan1)
			require.Nil(t, err)

			// Let's pass some time
			clock.SetTime(clock.Now().Add(time.Hour))

			// Let's change to another plan (same plan, but still a change)
			expectedChangeBy := deps.CurrentTime.AddDate(0, 1, 0)
			old, new, err := deps.WorkflowService.ChangeToPlan(ctx, sub.Subscription.NamespacedID, subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Timing: subscription.Timing{
					Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
				},
				Name: "Example Subscription 2",
			}, deps.Plan1)
			require.Nil(t, err)

			// Let's validate the change
			require.Equal(t, subscription.SubscriptionStatusCanceled, old.GetStatusAt(clock.Now()))
			require.Equal(t, subscription.SubscriptionStatusScheduled, new.Subscription.GetStatusAt(clock.Now()))

			require.Equal(t, subscription.SubscriptionStatusInactive, old.GetStatusAt(expectedChangeBy))
			require.Equal(t, subscription.SubscriptionStatusActive, new.Subscription.GetStatusAt(expectedChangeBy))

			// Let's pass some more time
			clock.SetTime(clock.Now().Add(time.Hour))

			// Let's restore the subscription
			restored, err := deps.WorkflowService.Restore(ctx, sub.Subscription.NamespacedID)
			require.Nil(t, err)

			// Let's validate the restored subscription
			require.Equal(t, subscription.SubscriptionStatusActive, restored.GetStatusAt(clock.Now()))
			require.Equal(t, subscription.SubscriptionStatusActive, restored.GetStatusAt(expectedChangeBy))

			// Let's make sure the new sub was deleted
			_, err = deps.Service.GetView(ctx, new.Subscription.NamespacedID)
			require.Error(t, err)
			require.ErrorAs(t, err, lo.ToPtr(&subscription.SubscriptionNotFoundError{}))
		})
	})

	t.Run("Should not allow restore if multi-subscription is enabled", func(t *testing.T) {
		withDeps(t)(func(t *testing.T, deps testCaseDeps) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ctx = ffx.SetAccessOnContext(ctx, ffx.AccessConfig{
				subscription.MultiSubscriptionEnabledFF: true,
			})

			// Let's create an example subscription
			sub, err := deps.WorkflowService.CreateFromPlan(context.Background(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
				ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Timing: subscription.Timing{
						Custom: &deps.CurrentTime,
					},
					Name: "Example Subscription",
				},
				CustomerID: deps.Customer.ID,
				Namespace:  subscriptiontestutils.ExampleNamespace,
			}, deps.Plan1)
			require.Nil(t, err)

			// Let's pass some time
			clock.SetTime(clock.Now().Add(time.Hour))

			// Let's cancel the subscription
			s, err := deps.Service.Cancel(ctx, sub.Subscription.NamespacedID, subscription.Timing{
				Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
			})
			require.Nil(t, err)

			// Let's validate its canceled
			require.Equal(t, subscription.SubscriptionStatusCanceled, s.GetStatusAt(clock.Now()))

			// Let's pass some more time
			clock.SetTime(deps.CurrentTime.AddDate(0, 0, 1))

			// Let's restore the subscription
			_, err = deps.WorkflowService.Restore(ctx, sub.Subscription.NamespacedID)
			require.Error(t, err)
			require.ErrorIs(t, err, subscription.ErrRestoreSubscriptionNotAllowedForMultiSubscription)
		})
	})
}

func TestMultiSubscription(t *testing.T) {
	// Let's define what deps a test case needs
	type testCaseDeps struct {
		CurrentTime     time.Time
		Customer        customer.Customer
		WorkflowService subscriptionworkflow.Service
		Service         subscription.Service
		DBDeps          *subscriptiontestutils.DBDeps
		Plan1           subscription.Plan
		SubsDeps        *subscriptiontestutils.SubscriptionDependencies
	}

	withDeps := func(t *testing.T) func(fn func(t *testing.T, deps testCaseDeps)) {
		return func(fn func(t *testing.T, deps testCaseDeps)) {
			examplePlanInput1 := subscriptiontestutils.GetExamplePlanInput(t)

			tcDeps := testCaseDeps{
				CurrentTime: testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z"),
			}

			clock.SetTime(tcDeps.CurrentTime)

			// Let's build the dependencies
			dbDeps := subscriptiontestutils.SetupDBDeps(t)
			require.NotNil(t, dbDeps)
			defer dbDeps.Cleanup(t)

			deps := subscriptiontestutils.NewService(t, dbDeps)
			deps.FeatureConnector.CreateExampleFeatures(t)

			// Let's create the plan
			plan1 := deps.PlanHelper.CreatePlan(t, examplePlanInput1)

			cust := deps.CustomerAdapter.CreateExampleCustomer(t)
			require.NotNil(t, cust)

			tcDeps.Customer = *cust
			tcDeps.DBDeps = dbDeps
			tcDeps.Service = deps.SubscriptionService
			tcDeps.WorkflowService = deps.WorkflowService
			tcDeps.Plan1 = plan1
			tcDeps.SubsDeps = &deps

			fn(t, tcDeps)
		}
	}

	t.Run("Should only allow one subscription per customer at a given time", func(t *testing.T) {
		withDeps(t)(func(t *testing.T, deps testCaseDeps) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			ctx = ffx.SetAccessOnContext(ctx, ffx.AccessConfig{
				subscription.MultiSubscriptionEnabledFF: false,
			})

			// Let's create an example subscription
			_, err := deps.WorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
				ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Timing: subscription.Timing{
						Custom: &deps.CurrentTime,
					},
					Name: "Example Subscription",
				},
				CustomerID: deps.Customer.ID,
				Namespace:  subscriptiontestutils.ExampleNamespace,
			}, deps.Plan1)
			require.Nil(t, err)

			// Now let's try to create a second subscription
			_, err = deps.WorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
				ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Timing: subscription.Timing{
						Custom: &deps.CurrentTime,
					},
					Name: "Example Subscription 2",
				},
				CustomerID: deps.Customer.ID,
				Namespace:  subscriptiontestutils.ExampleNamespace,
			}, deps.Plan1)

			require.Error(t, err)
			issues, err := models.AsValidationIssues(err)
			require.NoError(t, err)
			require.Len(t, issues, 2)
			for _, issue := range issues {
				require.Equal(t, subscription.ErrOnlySingleSubscriptionAllowed.Code(), issue.Code())
			}
		})
	})

	t.Run("Should still error if multi-subscription when relevant items conflict", func(t *testing.T) {
		withDeps(t)(func(t *testing.T, deps testCaseDeps) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			ctx = ffx.SetAccessOnContext(ctx, ffx.AccessConfig{
				subscription.MultiSubscriptionEnabledFF: true,
			})

			// Let's create an example subscription
			_, err := deps.WorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
				ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Timing: subscription.Timing{
						Custom: &deps.CurrentTime,
					},
					Name: "Example Subscription",
				},
				CustomerID: deps.Customer.ID,
				Namespace:  subscriptiontestutils.ExampleNamespace,
			}, deps.Plan1)
			require.Nil(t, err)

			// Now let's try to create a second subscription with the exact same contents (resulting in a conflict on the entitlement)
			_, err = deps.WorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
				ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Timing: subscription.Timing{
						Custom: &deps.CurrentTime,
					},
					Name: "Example Subscription 2",
				},
				CustomerID: deps.Customer.ID,
				Namespace:  subscriptiontestutils.ExampleNamespace,
			}, deps.Plan1)

			require.Error(t, err)

			issues, err := models.AsValidationIssues(err)
			require.NoError(t, err)

			require.Len(t, issues, 12) // We'll have two issues per conflicting items (one for each side)
			for _, issue := range issues {
				require.Equal(t, subscription.ErrOnlySingleSubscriptionItemAllowedAtATime.Code(), issue.Code())
			}
		})
	})

	t.Run("Should not error if two overlapping subscriptions don't share relevant items", func(t *testing.T) {
		withDeps(t)(func(t *testing.T, deps testCaseDeps) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			ctx = ffx.SetAccessOnContext(ctx, ffx.AccessConfig{
				subscription.MultiSubscriptionEnabledFF: true,
			})

			// Let's set up two plans with different features
			plan1 := deps.SubsDeps.PlanHelper.CreatePlan(t, subscriptiontestutils.BuildTestPlanInput(t).
				AddPhase(nil, &productcatalog.FlatFeeRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Name:       "feature1",
						Key:        subscriptiontestutils.ExampleFeatureKey,
						FeatureKey: lo.ToPtr(subscriptiontestutils.ExampleFeatureKey),
						FeatureID:  nil,
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount: alpacadecimal.NewFromInt(int64(100)),
						}),
					},
				}).
				Build())

			plan2 := deps.SubsDeps.PlanHelper.CreatePlan(t, subscriptiontestutils.BuildTestPlanInput(t).
				AddPhase(nil, &productcatalog.FlatFeeRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Name: "feature2",
						Key:  subscriptiontestutils.ExampleFeatureKey2,
					},
				}).Build())

			// Let's create two subscriptions with the plans
			_, err := deps.WorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
				ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Timing: subscription.Timing{
						Custom: &deps.CurrentTime,
					},
				},
				CustomerID: deps.Customer.ID,
				Namespace:  subscriptiontestutils.ExampleNamespace,
			}, plan1)
			require.Nil(t, err)

			_, err = deps.WorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
				ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Timing: subscription.Timing{
						Custom: &deps.CurrentTime,
					},
				},
				CustomerID: deps.Customer.ID,
				Namespace:  subscriptiontestutils.ExampleNamespace,
			}, plan2)
			require.Nil(t, err)
		})
	})
}

func TestSubscriptionChangeTrackingAnnotations(t *testing.T) {
	// Let's define what deps a test case needs
	type testCaseDeps struct {
		CurrentTime     time.Time
		Customer        customer.Customer
		WorkflowService subscriptionworkflow.Service
		Service         subscription.Service
		DBDeps          *subscriptiontestutils.DBDeps
		Plan1           subscription.Plan
		Plan2           subscription.Plan
		SubsDeps        *subscriptiontestutils.SubscriptionDependencies
	}

	withDeps := func(t *testing.T) func(fn func(t *testing.T, deps testCaseDeps)) {
		return func(fn func(t *testing.T, deps testCaseDeps)) {
			examplePlanInput1 := subscriptiontestutils.GetExamplePlanInput(t)
			examplePlanInput2 := subscriptiontestutils.GetExamplePlanInput(t)
			examplePlanInput2.Key = "example-plan-2"
			examplePlanInput2.Name = "Example Plan 2"

			tcDeps := testCaseDeps{
				CurrentTime: testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z"),
			}

			clock.SetTime(tcDeps.CurrentTime)

			// Let's build the dependencies
			dbDeps := subscriptiontestutils.SetupDBDeps(t)
			require.NotNil(t, dbDeps)
			defer dbDeps.Cleanup(t)

			deps := subscriptiontestutils.NewService(t, dbDeps)
			deps.FeatureConnector.CreateExampleFeatures(t)

			// Let's create the two plans
			plan1 := deps.PlanHelper.CreatePlan(t, examplePlanInput1)
			plan2 := deps.PlanHelper.CreatePlan(t, examplePlanInput2)

			cust := deps.CustomerAdapter.CreateExampleCustomer(t)
			require.NotNil(t, cust)

			tcDeps.Customer = *cust
			tcDeps.DBDeps = dbDeps
			tcDeps.Service = deps.SubscriptionService
			tcDeps.WorkflowService = deps.WorkflowService
			tcDeps.Plan1 = plan1
			tcDeps.Plan2 = plan2
			tcDeps.SubsDeps = &deps

			fn(t, tcDeps)
		}
	}

	t.Run("Should set annotations when changing subscription to new plan", func(t *testing.T) {
		withDeps(t)(func(t *testing.T, deps testCaseDeps) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Create first subscription
			sub1, err := deps.WorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
				ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Timing: subscription.Timing{
						Custom: &deps.CurrentTime,
					},
					Name: "First Subscription",
				},
				CustomerID: deps.Customer.ID,
				Namespace:  subscriptiontestutils.ExampleNamespace,
			}, deps.Plan1)
			require.Nil(t, err)

			// Change to new plan
			curr, new, err := deps.WorkflowService.ChangeToPlan(ctx, sub1.Subscription.NamespacedID, subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Timing: subscription.Timing{
					Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
				},
				Name: "Second Subscription",
			}, deps.Plan2)
			require.Nil(t, err)

			// Verify old subscription has superseding subscription ID
			currView, err := deps.Service.GetView(ctx, curr.NamespacedID)
			require.Nil(t, err)
			require.NotNil(t, currView.Subscription.Annotations)
			supersedingID := subscription.AnnotationParser.GetSupersedingSubscriptionID(currView.Subscription.Annotations)
			require.NotNil(t, supersedingID)
			assert.Equal(t, new.Subscription.ID, *supersedingID)

			// Verify new subscription has previous subscription ID
			newView, err := deps.Service.GetView(ctx, new.Subscription.NamespacedID)
			require.Nil(t, err)
			require.NotNil(t, newView.Subscription.Annotations)
			previousID := subscription.AnnotationParser.GetPreviousSubscriptionID(newView.Subscription.Annotations)
			require.NotNil(t, previousID)
			assert.Equal(t, curr.ID, *previousID)
		})
	})

	t.Run("Should clean up annotations when restoring subscription that superseded another", func(t *testing.T) {
		withDeps(t)(func(t *testing.T, deps testCaseDeps) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Create first subscription
			sub1, err := deps.WorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
				ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Timing: subscription.Timing{
						Custom: &deps.CurrentTime,
					},
					Name: "First Subscription",
				},
				CustomerID: deps.Customer.ID,
				Namespace:  subscriptiontestutils.ExampleNamespace,
			}, deps.Plan1)
			require.Nil(t, err)

			// Change to new plan - this creates sub2 and links sub1->sub2
			curr, new, err := deps.WorkflowService.ChangeToPlan(ctx, sub1.Subscription.NamespacedID, subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Timing: subscription.Timing{
					Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
				},
				Name: "Second Subscription",
			}, deps.Plan2)
			require.Nil(t, err)

			// Verify annotations are set
			currView, err := deps.Service.GetView(ctx, curr.NamespacedID)
			require.Nil(t, err)
			require.NotNil(t, currView.Subscription.Annotations)
			supersedingID := subscription.AnnotationParser.GetSupersedingSubscriptionID(currView.Subscription.Annotations)
			require.NotNil(t, supersedingID)
			assert.Equal(t, new.Subscription.ID, *supersedingID)

			// Restore the original subscription (this deletes sub2)
			clock.SetTime(clock.Now().Add(time.Hour))
			_, err = deps.WorkflowService.Restore(ctx, curr.NamespacedID)
			require.Nil(t, err)

			// Verify sub1 no longer has superseding subscription ID
			currViewAfter, err := deps.Service.GetView(ctx, curr.NamespacedID)
			require.Nil(t, err)
			if currViewAfter.Subscription.Annotations != nil {
				supersedingIDAfter := subscription.AnnotationParser.GetSupersedingSubscriptionID(currViewAfter.Subscription.Annotations)
				assert.Nil(t, supersedingIDAfter)
			}

			// Verify sub2 is deleted
			_, err = deps.Service.GetView(ctx, new.Subscription.NamespacedID)
			require.Error(t, err)
			require.ErrorAs(t, err, lo.ToPtr(&subscription.SubscriptionNotFoundError{}))
		})
	})

	t.Run("Should clean up annotations when restoring subscription that was superseded", func(t *testing.T) {
		withDeps(t)(func(t *testing.T, deps testCaseDeps) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Create first subscription
			sub1, err := deps.WorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
				ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Timing: subscription.Timing{
						Custom: &deps.CurrentTime,
					},
					Name: "First Subscription",
				},
				CustomerID: deps.Customer.ID,
				Namespace:  subscriptiontestutils.ExampleNamespace,
			}, deps.Plan1)
			require.Nil(t, err)

			// Change to new plan - this creates sub2 and links sub1->sub2
			curr, new, err := deps.WorkflowService.ChangeToPlan(ctx, sub1.Subscription.NamespacedID, subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Timing: subscription.Timing{
					Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
				},
				Name: "Second Subscription",
			}, deps.Plan2)
			require.Nil(t, err)

			// Verify sub2 has previous subscription ID
			newView, err := deps.Service.GetView(ctx, new.Subscription.NamespacedID)
			require.Nil(t, err)
			require.NotNil(t, newView.Subscription.Annotations)
			previousID := subscription.AnnotationParser.GetPreviousSubscriptionID(newView.Subscription.Annotations)
			require.NotNil(t, previousID)
			assert.Equal(t, curr.ID, *previousID)

			// Restore the original subscription (this deletes sub2)
			clock.SetTime(clock.Now().Add(time.Hour))
			_, err = deps.WorkflowService.Restore(ctx, curr.NamespacedID)
			require.Nil(t, err)

			// Verify sub2 is deleted
			_, err = deps.Service.GetView(ctx, new.Subscription.NamespacedID)
			require.Error(t, err)
			require.ErrorAs(t, err, lo.ToPtr(&subscription.SubscriptionNotFoundError{}))
		})
	})
}

var immediate = subscription.Timing{
	Enum: lo.ToPtr(subscription.TimingImmediate),
}
