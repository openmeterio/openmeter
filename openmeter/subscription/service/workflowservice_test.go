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

	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/subscription"
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
		Customer        customerentity.Customer
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

				assert.ErrorAs(t, err, &customerentity.NotFoundError{}, "expected customer not found error, got %T", err)
			},
		},
		// {
		// 	Name: "Should error if plan is not found",
		// 	Handler: func(t *testing.T, deps testCaseDeps) {
		// 		ctx, cancel := context.WithCancel(context.Background())
		// 		defer cancel()

		// 		_, err := deps.WorkflowService.CreateFromPlan(ctx, subscription.CreateSubscriptionWorkflowInput{
		// 			CustomerID: deps.Customer.ID,
		// 			Namespace:  subscriptiontestutils.ExampleNamespace,
		// 			ActiveFrom: deps.CurrentTime,
		// 		}, subscription.PlanRefInput{Key: "nonexistent-plan", Version: lo.ToPtr(1)},
		// 		)

		// 		// assert.ErrorAs does not recognize this error
		// 		_, isErr := lo.ErrorsAs[subscription.PlanNotFoundError](err)
		// 		assert.True(t, isErr, "expected plan not found error, got %T", err)
		// 	},
		// },
		// TODO: validate patches separately
		// {
		// 	Name: "Should error if a patch is invalid",
		// 	Handler: func(t *testing.T, deps testCaseDeps) {
		// 		ctx, cancel := context.WithCancel(context.Background())
		// 		defer cancel()
		// 		errorMsg := "this is an invalid patch"

		// 		invalidPatch := subscriptiontestutils.TestPatch{
		// 			ValdiateFn: func() error {
		// 				return errors.New(errorMsg)
		// 			},
		// 		}

		// 		_, err := deps.WorkflowService.CreateFromPlan(ctx, subscription.CreateFromPlanInput{
		// 			CustomerID:    deps.Customer.ID,
		// 			Namespace:     subscriptiontestutils.ExampleNamespace,
		// 			ActiveFrom:    deps.CurrentTime,
		// 			Currency:      "USD",
		// 			Plan:          subscriptiontestutils.ExamplePlanRef,
		// 			Customization: []subscription.Patch{&invalidPatch},
		// 		})

		// 		assert.ErrorContains(t, err, errorMsg, "expected error message to contain %q, got %v", errorMsg, err)
		// 	},
		// },
		// {
		// 	Name: "Should apply the patch to the specs based on the plan",
		// 	Handler: func(t *testing.T, deps testCaseDeps) {
		// 		ctx, cancel := context.WithCancel(context.Background())
		// 		defer cancel()

		// 		// As we assert for the contextual time, we need to freeze the clock
		// 		clock.FreezeTime(deps.CurrentTime)
		// 		defer clock.UnFreeze()

		// 		errMsg := "this custom patch apply failed"

		// 		expectedPlanSpec, err := subscription.NewSpecFromPlan(subscriptiontestutils.GetExamplePlan(), subscription.CreateSubscriptionCustomerInput{
		// 			CustomerId: deps.Customer.ID,
		// 			Currency:   "USD",
		// 			ActiveFrom: deps.CurrentTime,
		// 		})
		// 		require.Nil(t, err)

		// 		patch1 := subscriptiontestutils.TestPatch{
		// 			ApplyToFn: func(spec *subscription.SubscriptionSpec, c subscription.ApplyContext) error {
		// 				// Let's assert that the correct spec is passed to the patch
		// 				assert.Equal(t, &expectedPlanSpec, spec, "expected spec to be equal to the plan spec")
		// 				assert.Equal(t, subscription.ApplyContext{
		// 					CurrentTime: deps.CurrentTime,
		// 					Operation:   subscription.SpecOperationCreate,
		// 				}, c, "apply context is incorrect")

		// 				// Lets modify the spec to see if its passed to the next
		// 				spec.Plan.Key = "modified-plan"

		// 				return nil
		// 			},
		// 		}

		// 		patch2 := subscriptiontestutils.TestPatch{
		// 			ApplyToFn: func(spec *subscription.SubscriptionSpec, c subscription.ApplyContext) error {
		// 				// Let's see if the modification is passed along
		// 				assert.Equal(t, "modified-plan", spec.Plan.Key, "expected plan key to be modified")

		// 				return nil
		// 			},
		// 		}

		// 		patch3 := subscriptiontestutils.TestPatch{
		// 			ApplyToFn: func(spec *subscription.SubscriptionSpec, c subscription.ApplyContext) error {
		// 				// And let's test if errors are passed correctly
		// 				return errors.New(errMsg)
		// 			},
		// 		}

		// 		_, err = deps.WorkflowService.CreateFromPlan(ctx, subscription.CreateFromPlanInput{
		// 			CustomerID: deps.Customer.ID,
		// 			Namespace:  subscriptiontestutils.ExampleNamespace,
		// 			ActiveFrom: deps.CurrentTime,
		// 			Currency:   "USD",
		// 			Plan:       subscriptiontestutils.ExamplePlanRef,
		// 			Customization: []subscription.Patch{
		// 				&patch1,
		// 				&patch2,
		// 				&patch3,
		// 			},
		// 		})

		// 		// Let's validate the error is surfaced
		// 		assert.ErrorContains(t, err, errMsg, "expected error message to contain %q, got %v", errMsg, err)
		// 	},
		// },
		// {
		// 	Name: "Should use the output of patches without modifications",
		// 	Handler: func(t *testing.T, deps testCaseDeps) {
		// 		ctx, cancel := context.WithCancel(context.Background())
		// 		defer cancel()

		// 		returnedSpec := subscription.SubscriptionSpec{
		// 			CreateSubscriptionPlanInput: subscription.CreateSubscriptionPlanInput{
		// 				Plan: subscription.PlanRef{
		// 					Key:     "returned-plan",
		// 					Version: 1,
		// 				},
		// 			},
		// 			CreateSubscriptionCustomerInput: subscription.CreateSubscriptionCustomerInput{
		// 				CustomerId: "new-customer-id",
		// 			},
		// 			Phases: map[string]*subscription.SubscriptionPhaseSpec{
		// 				"phase-1": {
		// 					CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
		// 						PhaseKey:   "phase-1",
		// 						StartAfter: testutils.GetISODuration(t, "P1D"),
		// 						Name:       "Phase 1",
		// 					},
		// 					ItemsByKey: map[string][]subscription.SubscriptionItemSpec{
		// 						"item-1": {
		// 							{
		// 								CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
		// 									CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
		// 										ItemKey:  "item-1",
		// 										PhaseKey: "phase-1",
		// 										RateCard: subscription.RateCard{
		// 											Name: "rate-card-1",
		// 										},
		// 									},
		// 								},
		// 							},
		// 						},
		// 					},
		// 				},
		// 			},
		// 		}

		// 		patch1 := subscriptiontestutils.TestPatch{
		// 			ApplyToFn: func(spec *subscription.SubscriptionSpec, c subscription.ApplyContext) error {
		// 				// Let's set the new values

		// 				spec.CreateSubscriptionPlanInput = returnedSpec.CreateSubscriptionPlanInput
		// 				spec.CreateSubscriptionCustomerInput = returnedSpec.CreateSubscriptionCustomerInput
		// 				spec.Phases = returnedSpec.Phases

		// 				return nil
		// 			},
		// 		}

		// 		sID := models.NamespacedID{
		// 			Namespace: subscriptiontestutils.ExampleNamespace,
		// 			ID:        "new-subscription-id",
		// 		}

		// 		rView := subscription.SubscriptionView{
		// 			Subscription: subscription.Subscription{
		// 				CustomerId: "bogus-id",
		// 			},
		// 		}

		// 		mSvc := subscriptiontestutils.MockService{
		// 			CreateFn: func(ctx context.Context, namespace string, spec subscription.SubscriptionSpec) (subscription.Subscription, error) {
		// 				// Let's validate that the spec is passed as is
		// 				assert.Equal(t, returnedSpec, spec, "expected spec to be equal to the returned spec")

		// 				return subscription.Subscription{
		// 					NamespacedID: sID,
		// 				}, nil
		// 			},
		// 			GetViewFn: func(ctx context.Context, id models.NamespacedID) (subscription.SubscriptionView, error) {
		// 				assert.Equal(t, sID, id, "expected id to be equal to the returned id")

		// 				return rView, nil
		// 			},
		// 		}

		// 		_, tuDeps := subscriptiontestutils.NewService(t, deps.DBDeps)
		// 		tuDeps.PlanAdapter.AddPlan(t, subscriptiontestutils.GetExamplePlan())

		// 		workflowService := service.NewWorkflowService(service.WorkflowServiceConfig{
		// 			Service:            &mSvc,
		// 			CustomerService:    tuDeps.CustomerService,
		// 			PlanAdapter:        tuDeps.PlanAdapter,
		// 			TransactionManager: tuDeps.CustomerAdapter,
		// 		})

		// 		res, err := workflowService.CreateFromPlan(ctx, subscription.CreateFromPlanInput{
		// 			CustomerID: deps.Customer.ID,
		// 			Namespace:  subscriptiontestutils.ExampleNamespace,
		// 			ActiveFrom: deps.CurrentTime,
		// 			Currency:   "USD",
		// 			Plan:       subscriptiontestutils.ExamplePlanRef,
		// 			Customization: []subscription.Patch{
		// 				&patch1,
		// 			},
		// 		})

		// 		assert.Nil(t, err)

		// 		assert.Equal(t, rView, res)
		// 	},
		// },
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tcDeps := testCaseDeps{
				CurrentTime: testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z"),
			}

			clock.SetTime(tcDeps.CurrentTime)
			dbDeps := subscriptiontestutils.SetupDBDeps(t)
			require.NotNil(t, dbDeps)
			defer dbDeps.Cleanup()

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
		Customer        customerentity.Customer
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
							Operation:   subscription.SpecOperationEdit,
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
			defer dbDeps.Cleanup()

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
	t.Skip("TODO: implement me")
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
		Customer        customerentity.Customer
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
			defer dbDeps.Cleanup()

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
