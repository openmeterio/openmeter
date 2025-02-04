package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/registry"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	"github.com/openmeterio/openmeter/openmeter/subscription/service"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestCreateFromPlan(t *testing.T) {
	type testCaseDeps struct {
		Plan            subscription.Plan
		CurrentTime     time.Time
		Customer        customer.Customer
		WorkflowService subscription.WorkflowService
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

				_, err := deps.WorkflowService.CreateFromPlan(ctx, subscription.CreateSubscriptionWorkflowInput{
					ChangeSubscriptionWorkflowInput: subscription.ChangeSubscriptionWorkflowInput{
						ActiveFrom: deps.CurrentTime,
					},
					CustomerID: fmt.Sprintf("nonexistent-customer-%s", deps.Customer.ID),
					Namespace:  subscriptiontestutils.ExampleNamespace,
				}, deps.Plan)

				assert.ErrorAs(t, err, &customer.NotFoundError{}, "expected customer not found error, got %T", err)
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

			services, deps := subscriptiontestutils.NewService(t, dbDeps)
			deps.FeatureConnector.CreateExampleFeature(t)
			plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))
			cust := deps.CustomerAdapter.CreateExampleCustomer(t)
			require.NotNil(t, cust)

			tcDeps.Customer = *cust
			tcDeps.Plan = plan

			tcDeps.DBDeps = dbDeps

			tcDeps.WorkflowService = services.WorkflowService

			tc.Handler(t, tcDeps)
		})
	}
}

func TestEditRunning(t *testing.T) {
	type testCaseDeps struct {
		CurrentTime     time.Time
		SubView         subscription.SubscriptionView
		Customer        customer.Customer
		WorkflowService subscription.WorkflowService
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
				}, nil)

				assert.ErrorAs(t, err, lo.ToPtr(&subscription.NotFoundError{}), "expected subscription not found error, got %T", err)
			},
		},
		{
			Name: "Should do nothing if no patches are provided",
			Handler: func(t *testing.T, deps testCaseDeps) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				subView, err := deps.WorkflowService.EditRunning(ctx, deps.SubView.Subscription.NamespacedID, nil)
				assert.Nil(t, err)

				assert.Equal(t, deps.SubView, subView)
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

				_, err := deps.WorkflowService.EditRunning(ctx, deps.SubView.Subscription.NamespacedID, []subscription.Patch{&invalidPatch})
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

				_, tuDeps := subscriptiontestutils.NewService(t, deps.DBDeps)

				workflowService := service.NewWorkflowService(service.WorkflowServiceConfig{
					Service:            &mSvc,
					CustomerService:    tuDeps.CustomerService,
					TransactionManager: tuDeps.CustomerAdapter,
				})

				_, err := workflowService.EditRunning(ctx, sID, []subscription.Patch{&patch1})
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

			// Let's build the dependencies
			dbDeps := subscriptiontestutils.SetupDBDeps(t)
			require.NotNil(t, dbDeps)
			defer dbDeps.Cleanup(t)

			services, deps := subscriptiontestutils.NewService(t, dbDeps)
			deps.FeatureConnector.CreateExampleFeature(t)
			plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))
			cust := deps.CustomerAdapter.CreateExampleCustomer(t)
			require.NotNil(t, cust)

			// Let's create an example subscription
			sub, err := services.WorkflowService.CreateFromPlan(context.Background(), subscription.CreateSubscriptionWorkflowInput{
				ChangeSubscriptionWorkflowInput: subscription.ChangeSubscriptionWorkflowInput{
					ActiveFrom: tcDeps.CurrentTime,
					Name:       "Example Subscription",
				},
				CustomerID: cust.ID,
				Namespace:  subscriptiontestutils.ExampleNamespace,
			}, plan)
			require.Nil(t, err)

			tcDeps.SubView = sub
			tcDeps.Customer = *cust
			tcDeps.DBDeps = dbDeps
			tcDeps.Service = services.Service
			tcDeps.WorkflowService = services.WorkflowService
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
		WorkflowService subscription.WorkflowService
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
				})
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
				})
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

			services, deps := subscriptiontestutils.NewService(t, dbDeps)
			deps.FeatureConnector.CreateExampleFeature(t)
			plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))
			cust := deps.CustomerAdapter.CreateExampleCustomer(t)
			require.NotNil(t, cust)

			// Let's create an example subscription
			sub, err := services.WorkflowService.CreateFromPlan(context.Background(), subscription.CreateSubscriptionWorkflowInput{
				ChangeSubscriptionWorkflowInput: subscription.ChangeSubscriptionWorkflowInput{
					ActiveFrom: tcDeps.CurrentTime,
					Name:       "Example Subscription",
				},
				CustomerID: cust.ID,
				Namespace:  subscriptiontestutils.ExampleNamespace,
			}, plan)
			require.Nil(t, err)

			tcDeps.SubView = sub
			tcDeps.Customer = *cust
			tcDeps.DBDeps = dbDeps
			tcDeps.Service = services.Service
			tcDeps.WorkflowService = services.WorkflowService
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

	rc1 := productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:         subscriptiontestutils.ExampleFeatureKey,
			Name:        "Rate Card 1",
			Description: lo.ToPtr("Rate Card 1 Description"),
			Feature: &feature.Feature{
				Key: subscriptiontestutils.ExampleFeatureKey,
			},
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
		BillingCadence: &subscriptiontestutils.ISOMonth,
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
				Name:     "Test Plan 2",
				Key:      "test_plan_2",
				Version:  1,
				Currency: currency.USD,
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:         "test_phase_1_new",
						Name:        "Test Phase 1 New",
						Description: lo.ToPtr("Test Phase 1 Description"),
						Duration:    lo.ToPtr(testutils.GetISODuration(t, "P2M")),
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
						Duration:    lo.ToPtr(testutils.GetISODuration(t, "P1M")),
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
		WorkflowService subscription.WorkflowService
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

			services, deps := subscriptiontestutils.NewService(t, dbDeps)
			deps.FeatureConnector.CreateExampleFeature(t)

			// Let's create the two plans
			plan1 := deps.PlanHelper.CreatePlan(t, examplePlanInput1)
			plan2 := deps.PlanHelper.CreatePlan(t, examplePlanInput2)

			cust := deps.CustomerAdapter.CreateExampleCustomer(t)
			require.NotNil(t, cust)

			tcDeps.Customer = *cust
			tcDeps.DBDeps = dbDeps
			tcDeps.Service = services.Service
			tcDeps.WorkflowService = services.WorkflowService
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
			sub, err := deps.WorkflowService.CreateFromPlan(context.Background(), subscription.CreateSubscriptionWorkflowInput{
				ChangeSubscriptionWorkflowInput: subscription.ChangeSubscriptionWorkflowInput{
					ActiveFrom: deps.CurrentTime,
					Name:       "Example Subscription",
				},
				CustomerID: deps.Customer.ID,
				Namespace:  subscriptiontestutils.ExampleNamespace,
			}, deps.Plan1)
			require.Nil(t, err)

			someTimeLater := deps.CurrentTime.AddDate(0, 0, 10)

			changeInput := subscription.ChangeSubscriptionWorkflowInput{
				ActiveFrom: someTimeLater,
				Name:       "New Subscription",
			}

			curr, new, err := deps.WorkflowService.ChangeToPlan(ctx, sub.Subscription.NamespacedID, changeInput, deps.Plan2)
			require.Nil(t, err)

			// Let's do some simple validations
			require.Equal(t, sub.Subscription.NamespacedID, curr.NamespacedID)
			require.NotNil(t, curr.ActiveTo)
			require.Equal(t, someTimeLater, *curr.ActiveTo)

			require.NotNil(t, new.Subscription.PlanRef)
			require.Equal(t, examplePlanInput2.Key, new.Subscription.PlanRef.Key)

			// Let's check that the new plan looks as we expect
			targetSpec, err := subscription.NewSpecFromPlan(deps.Plan2, subscription.CreateSubscriptionCustomerInput{
				Name:           changeInput.Name,
				Description:    changeInput.Description,
				AnnotatedModel: changeInput.AnnotatedModel,
				CustomerId:     curr.CustomerId,
				Currency:       deps.Plan2.Currency(),
				ActiveFrom:     changeInput.ActiveFrom,
				ActiveTo:       nil,
			})
			require.Nil(t, err)

			subscriptiontestutils.ValidateSpecAndView(t, targetSpec, new)
		})
	})
}

func TestEditCombinations(t *testing.T) {
	examplePlanInput1 := subscriptiontestutils.GetExamplePlanInput(t)

	// Let's define what deps a test case needs
	type testCaseDeps struct {
		CurrentTime     time.Time
		Customer        customer.Customer
		WorkflowService subscription.WorkflowService
		Service         subscription.Service
		DBDeps          *subscriptiontestutils.DBDeps
		Plan1           subscription.Plan
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

			services, deps := subscriptiontestutils.NewService(t, dbDeps)
			deps.FeatureConnector.CreateExampleFeature(t)

			// Let's create the plan
			plan1 := deps.PlanHelper.CreatePlan(t, examplePlanInput1)

			cust := deps.CustomerAdapter.CreateExampleCustomer(t)
			require.NotNil(t, cust)

			tcDeps.Customer = *cust
			tcDeps.DBDeps = dbDeps
			tcDeps.Service = services.Service
			tcDeps.WorkflowService = services.WorkflowService
			tcDeps.Plan1 = plan1

			fn(t, tcDeps)
		}
	}

	t.Run("Should be able to cancel an edited subscription", func(t *testing.T) {
		withDeps(t)(func(t *testing.T, deps testCaseDeps) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Let's create an example subscription
			sub, err := deps.WorkflowService.CreateFromPlan(context.Background(), subscription.CreateSubscriptionWorkflowInput{
				ChangeSubscriptionWorkflowInput: subscription.ChangeSubscriptionWorkflowInput{
					ActiveFrom: deps.CurrentTime,
					Name:       "Example Subscription",
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
								RateCard: subscription.RateCard{
									Name:                val.Spec.RateCard.Name,
									Description:         val.Spec.RateCard.Description,
									FeatureKey:          val.Spec.RateCard.FeatureKey,
									EntitlementTemplate: val.Spec.RateCard.EntitlementTemplate,
									TaxConfig:           val.Spec.RateCard.TaxConfig,
									Price:               productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(19)}),
									BillingCadence:      val.Spec.RateCard.BillingCadence,
								},
							},
							CreateSubscriptionItemCustomerInput: subscription.CreateSubscriptionItemCustomerInput{},
						},
					},
				},
			}

			// Let 5 minutes pass
			clock.SetTime(deps.CurrentTime.Add(5 * time.Minute))

			_, err = deps.WorkflowService.EditRunning(ctx, sub.Subscription.NamespacedID, edits)
			require.Nil(t, err)

			// Now let's fetch the view
			view, err := deps.Service.GetView(ctx, sub.Subscription.NamespacedID)
			require.Nil(t, err)

			require.Equal(t, sub.Subscription.NamespacedID, view.Subscription.NamespacedID)

			// Now let's cancel the subscription
			s, err := deps.Service.Cancel(ctx, sub.Subscription.NamespacedID, clock.Now().Add(-time.Minute))
			require.Nil(t, err)

			require.Equal(t, subscription.SubscriptionStatusInactive, s.GetStatusAt(clock.Now()))
		})
	})
}
