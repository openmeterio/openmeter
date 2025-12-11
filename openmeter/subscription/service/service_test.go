package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestCreation(t *testing.T) {
	t.Run("Should create subscription as specced", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:11Z")
		clock.SetTime(currentTime)

		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup(t)

		deps := subscriptiontestutils.NewService(t, dbDeps)
		service := deps.SubscriptionService

		cust := deps.CustomerAdapter.CreateExampleCustomer(t)
		_ = deps.FeatureConnector.CreateExampleFeatures(t)
		plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

		defaultSpecFromPlan, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
			CustomerId:    cust.ID,
			Currency:      "USD",
			ActiveFrom:    currentTime,
			BillingAnchor: currentTime,
			Name:          "Test Subscription",
			Annotations:   models.Annotations{},
		})
		require.Nil(t, err)

		sub, err := service.Create(ctx, subscriptiontestutils.ExampleNamespace, defaultSpecFromPlan)

		require.Nil(t, err)
		require.Equal(t, plan.ToCreateSubscriptionPlanInput().Plan, sub.PlanRef)
		require.Equal(t, subscriptiontestutils.ExampleNamespace, sub.Namespace)
		require.Equal(t, cust.ID, sub.CustomerId)
		require.Equal(t, currencyx.Code("USD"), sub.Currency)
		require.NotNil(t, sub.Annotations)

		t.Run("Should find subscription by ID", func(t *testing.T) {
			found, err := service.Get(ctx, models.NamespacedID{
				ID:        sub.ID,
				Namespace: sub.Namespace,
			})

			assert.Nil(t, err)
			assert.Equal(t, sub.ID, found.ID)
			assert.Equal(t, sub.PlanRef, found.PlanRef)
			assert.Equal(t, sub.Namespace, found.Namespace)
			assert.Equal(t, sub.CustomerId, found.CustomerId)
			assert.Equal(t, sub.Currency, found.Currency)
			// Annotations should be initialized as empty map
			assert.NotNil(t, found.Annotations)
			assert.Equal(t, models.Annotations{}, found.Annotations)
		})

		t.Run("Should create subscription as specced", func(t *testing.T) {
			found, err := service.GetView(ctx, models.NamespacedID{ID: sub.ID, Namespace: sub.Namespace})
			assert.Nil(t, err)

			// Test Sub
			foundSub := found.Subscription

			assert.Equal(t, sub.ID, foundSub.ID)
			assert.Equal(t, sub.PlanRef, foundSub.PlanRef)
			assert.Equal(t, sub.Namespace, foundSub.Namespace)
			assert.Equal(t, sub.CustomerId, foundSub.CustomerId)
			assert.Equal(t, sub.Currency, foundSub.Currency)

			// Let's validate the spec & the view
			subscriptiontestutils.ValidateSpecAndView(t, defaultSpecFromPlan, found)
		})
	})

	t.Run("Should preserve annotations when creating subscription", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:11Z")
		clock.SetTime(currentTime)

		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup(t)

		deps := subscriptiontestutils.NewService(t, dbDeps)
		service := deps.SubscriptionService

		cust := deps.CustomerAdapter.CreateExampleCustomer(t)
		_ = deps.FeatureConnector.CreateExampleFeatures(t)
		plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

		specWithAnnotations, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
			CustomerId:    cust.ID,
			Currency:      "USD",
			ActiveFrom:    currentTime,
			BillingAnchor: currentTime,
			Name:          "Test Subscription with Annotations",
			Annotations:   models.Annotations{"test.key": "test.value", "another.key": float64(123)},
		})
		require.Nil(t, err)

		subWithAnnotations, err := service.Create(ctx, subscriptiontestutils.ExampleNamespace, specWithAnnotations)
		require.Nil(t, err)
		require.Equal(t, models.Annotations{"test.key": "test.value", "another.key": float64(123)}, subWithAnnotations.Annotations)

		// Verify annotations are preserved when retrieving
		found, err := service.Get(ctx, models.NamespacedID{
			ID:        subWithAnnotations.ID,
			Namespace: subWithAnnotations.Namespace,
		})
		require.Nil(t, err)
		assert.Equal(t, models.Annotations{"test.key": "test.value", "another.key": float64(123)}, found.Annotations)

		// Verify annotations are preserved in view
		view, err := service.GetView(ctx, models.NamespacedID{ID: subWithAnnotations.ID, Namespace: subWithAnnotations.Namespace})
		require.Nil(t, err)
		assert.Equal(t, models.Annotations{"test.key": "test.value", "another.key": float64(123)}, view.Subscription.Annotations)
	})

	t.Run("Should not allow creating a subscription with different currency compared to the customer", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:11Z")
		clock.SetTime(currentTime)

		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup(t)

		deps := subscriptiontestutils.NewService(t, dbDeps)
		service := deps.SubscriptionService

		cust := deps.CustomerAdapter.CreateExampleCustomer(t)
		_, err := deps.CustomerService.UpdateCustomer(ctx, customer.UpdateCustomerInput{
			CustomerID: cust.GetID(),
			CustomerMutate: customer.CustomerMutate{
				Name:             cust.Name,
				Description:      cust.Description,
				UsageAttribution: cust.UsageAttribution,
				PrimaryEmail:     cust.PrimaryEmail,
				BillingAddress:   cust.BillingAddress,
				Currency:         lo.ToPtr(currencyx.Code("EUR")),
			},
		})
		require.Nil(t, err)

		_ = deps.FeatureConnector.CreateExampleFeatures(t)
		plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

		defaultSpecFromPlan, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
			CustomerId:    cust.ID,
			Currency:      "USD",
			ActiveFrom:    currentTime,
			BillingAnchor: currentTime,
			Name:          "Test Subscription",
		})
		require.Nil(t, err)

		_, err = service.Create(ctx, subscriptiontestutils.ExampleNamespace, defaultSpecFromPlan)

		require.Error(t, err)
		require.ErrorAs(t, err, lo.ToPtr(&models.GenericValidationError{}))
	})

	t.Run("Should set customer currency based on subscription", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:11Z")
		clock.SetTime(currentTime)

		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup(t)

		deps := subscriptiontestutils.NewService(t, dbDeps)
		service := deps.SubscriptionService

		cust := deps.CustomerAdapter.CreateExampleCustomer(t)
		_, err := deps.CustomerService.UpdateCustomer(ctx, customer.UpdateCustomerInput{
			CustomerID: cust.GetID(),
			CustomerMutate: customer.CustomerMutate{
				Name:             cust.Name,
				Description:      cust.Description,
				UsageAttribution: cust.UsageAttribution,
				PrimaryEmail:     cust.PrimaryEmail,
				BillingAddress:   cust.BillingAddress,
				Currency:         nil,
			},
		})
		require.Nil(t, err)

		_ = deps.FeatureConnector.CreateExampleFeatures(t)
		plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

		defaultSpecFromPlan, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
			CustomerId:    cust.ID,
			Currency:      "USD",
			ActiveFrom:    currentTime,
			BillingAnchor: currentTime,
			Name:          "Test Subscription",
		})
		require.Nil(t, err)

		_, err = service.Create(ctx, subscriptiontestutils.ExampleNamespace, defaultSpecFromPlan)

		require.NoError(t, err)

		c, err := deps.CustomerService.GetCustomer(ctx, customer.GetCustomerInput{
			CustomerID: &customer.CustomerID{
				Namespace: cust.Namespace,
				ID:        cust.ID,
			},
		})
		require.NoError(t, err)

		assert.Equal(t, currencyx.Code("USD"), *c.Currency)
	})
}

func TestCancellation(t *testing.T) {
	t.Run("Should cancel subscription", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z")
		clock.SetTime(currentTime)

		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup(t)

		deps := subscriptiontestutils.NewService(t, dbDeps)
		service := deps.SubscriptionService

		cust := deps.CustomerAdapter.CreateExampleCustomer(t)
		_ = deps.FeatureConnector.CreateExampleFeatures(t)
		plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

		// First, let's create a subscription
		defaultSpecFromPlan, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
			CustomerId:    cust.ID,
			Currency:      "USD",
			ActiveFrom:    currentTime,
			BillingAnchor: currentTime,
			Name:          "Test Subscription",
		})
		require.Nil(t, err)

		sub, err := service.Create(ctx, subscriptiontestutils.ExampleNamespace, defaultSpecFromPlan)

		require.Nil(t, err)

		// Second, let's cancel the subscription
		expectedCancelTime := testutils.GetRFC3339Time(t, "2021-02-01T00:00:00Z")
		cancelledSub, err := service.Cancel(ctx, sub.NamespacedID, subscription.Timing{
			Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
		})

		assert.Nil(t, err, "error canceling subscription: %v", err)
		assert.Equal(t, sub.ID, cancelledSub.ID)
		assert.Equal(t, sub.PlanRef, cancelledSub.PlanRef)
		assert.Equal(t, sub.Namespace, cancelledSub.Namespace)
		assert.Equal(t, sub.CustomerId, cancelledSub.CustomerId)
		assert.Equal(t, sub.Currency, cancelledSub.Currency)
		assert.Equal(t, expectedCancelTime, *cancelledSub.ActiveTo)
		assert.Equal(t, subscription.SubscriptionStatusCanceled, cancelledSub.GetStatusAt(clock.Now()))

		// Third, let's fetch the full view of the subscription and validate that all contents are canceled
		view, err := service.GetView(ctx, sub.NamespacedID)
		assert.Nil(t, err)

		for _, phase := range defaultSpecFromPlan.GetSortedPhases() {
			foundPhase, ok := lo.Find(view.Phases, func(p subscription.SubscriptionPhaseView) bool {
				return p.SubscriptionPhase.Key == phase.PhaseKey
			})
			require.True(t, ok)

			phaseCadence, err := lo.ToPtr(view.AsSpec()).GetPhaseCadence(phase.PhaseKey)
			require.Nil(t, err)

			for itemsKey, itemsByKey := range phase.ItemsByKey {
				foundItems, ok := foundPhase.ItemsByKey[itemsKey]
				require.True(t, ok)
				require.Equal(t, len(itemsByKey), len(foundItems))

				for _, foundItem := range foundItems {
					satisfies := false

					foundItemCadence := foundItem.SubscriptionItem.GetCadence(phaseCadence)

					// All items must have either
					if !foundItemCadence.ActiveFrom.After(expectedCancelTime) {
						// - left in tact if their phase ends before the cancel time
						if phaseCadence.ActiveTo != nil && !phaseCadence.ActiveTo.After(expectedCancelTime) {
							if foundItemCadence.ActiveTo != nil && foundItemCadence.ActiveTo.Equal(*phaseCadence.ActiveTo) {
								satisfies = true
							}
							// - their ActiveTo time set to the cancel time (if they started before the cancel time)
						} else if foundItemCadence.ActiveTo != nil && foundItemCadence.ActiveTo.Equal(expectedCancelTime) {
							satisfies = true
						}
					} else {
						if foundItemCadence.ActiveTo != nil && foundItemCadence.ActiveTo.Equal(foundItemCadence.ActiveFrom) {
							// - or their ActiveTo time set to their ActiveFrom time
							satisfies = true
						}
					}

					assert.True(t, satisfies, "item %+v in phase %s does not satisfy the cancellation criteria", foundItem.SubscriptionItem, phase.PhaseKey)

					// And the same goes for entitlements if present
					if foundItem.Entitlement != nil {
						satisfies := false

						ent := foundItem.Entitlement
						if !ent.Cadence.ActiveFrom.After(expectedCancelTime) {
							if phaseCadence.ActiveTo != nil && !phaseCadence.ActiveTo.After(expectedCancelTime) {
								if ent.Cadence.ActiveTo != nil && ent.Cadence.ActiveTo.Equal(*phaseCadence.ActiveTo) {
									satisfies = true
								}
							} else if ent.Cadence.ActiveTo != nil && ent.Cadence.ActiveTo.Equal(expectedCancelTime) {
								satisfies = true
							}
						} else {
							if ent.Cadence.ActiveTo != nil && ent.Cadence.ActiveTo.Equal(ent.Cadence.ActiveFrom) {
								satisfies = true
							}
						}

						assert.True(t, satisfies, "entitlement %+v for item %s in phase %s does not satisfy the cancellation criteria", ent, foundItem.SubscriptionItem.Key, foundPhase.SubscriptionPhase.Key)
					}
				}
			}
		}
	})
}

func TestContinuing(t *testing.T) {
	t.Run("Should continue canceled subscription", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z")
		clock.SetTime(currentTime)

		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup(t)

		deps := subscriptiontestutils.NewService(t, dbDeps)
		service := deps.SubscriptionService

		cust := deps.CustomerAdapter.CreateExampleCustomer(t)
		_ = deps.FeatureConnector.CreateExampleFeatures(t)
		plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

		// First, let's create a subscription
		defaultSpecFromPlan, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
			CustomerId:    cust.ID,
			Currency:      "USD",
			ActiveFrom:    currentTime,
			BillingAnchor: currentTime,
			Name:          "Test Subscription",
		})
		require.Nil(t, err)

		sub, err := service.Create(ctx, subscriptiontestutils.ExampleNamespace, defaultSpecFromPlan)

		require.Nil(t, err)

		// Second, let's cancel the subscription
		_, err = service.Cancel(ctx, sub.NamespacedID, subscription.Timing{
			Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
		})

		require.Nil(t, err)

		// Let's pass some time
		clock.SetTime(clock.Now().AddDate(0, 0, 1))

		// Third, let's continue the subscription
		_, err = service.Continue(ctx, sub.NamespacedID)
		require.Nil(t, err)

		// Finally, let's fetch the full view of the subscription and validate that all contents are active as should be
		view, err := service.GetView(ctx, sub.NamespacedID)
		assert.Nil(t, err)

		for _, phase := range defaultSpecFromPlan.GetSortedPhases() {
			foundPhase, ok := lo.Find(view.Phases, func(p subscription.SubscriptionPhaseView) bool {
				return p.SubscriptionPhase.Key == phase.PhaseKey
			})
			require.True(t, ok)

			phaseCadence, err := lo.ToPtr(view.AsSpec()).GetPhaseCadence(phase.PhaseKey)
			require.Nil(t, err)

			for itemKey := range phase.ItemsByKey {
				foundItemsByKey, ok := foundPhase.ItemsByKey[itemKey]
				require.True(t, ok)

				for _, foundItem := range foundItemsByKey {
					satisfies := false

					// All items must have their cadence set according to the phase cadence
					if foundItem.SubscriptionItem.GetCadence(phaseCadence).Equal(phaseCadence) {
						satisfies = true
					}

					assert.True(t, satisfies, "item %+v in phase %s does not satisfy the cancellation criteria", foundItem.SubscriptionItem, phase.PhaseKey)

					// And the same goes for entitlements if present
					if foundItem.Entitlement != nil {
						satisfies := false

						if foundItem.Entitlement.Cadence.Equal(phaseCadence) {
							satisfies = true
						}

						assert.True(t, satisfies, "entitlement %+v for item %s in phase %s does not satisfy the cancellation criteria", foundItem.Entitlement, foundItem.SubscriptionItem.Key, foundPhase.SubscriptionPhase.Key)
					}
				}
			}
		}
	})

	t.Run("Should not continue subscription if it would result in a scheduling conflict", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z")
		clock.SetTime(currentTime)

		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup(t)

		deps := subscriptiontestutils.NewService(t, dbDeps)
		service := deps.SubscriptionService

		cust := deps.CustomerAdapter.CreateExampleCustomer(t)
		_ = deps.FeatureConnector.CreateExampleFeatures(t)
		plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

		// First, let's create a subscription
		spec1, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
			CustomerId:    cust.ID,
			Currency:      "USD",
			ActiveFrom:    currentTime,
			BillingAnchor: currentTime,
			Name:          "Test Subscription",
		})
		require.Nil(t, err)

		sub1, err := service.Create(ctx, subscriptiontestutils.ExampleNamespace, spec1)

		require.Nil(t, err)

		// Second, let's cancel the subscription
		expectedCancelTime := testutils.GetRFC3339Time(t, "2021-02-01T00:00:00Z")
		_, err = service.Cancel(ctx, sub1.NamespacedID, subscription.Timing{
			Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
		})

		require.Nil(t, err)

		// Third, let's create another subscription for later
		spec2, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
			CustomerId:    cust.ID,
			Currency:      "USD",
			ActiveFrom:    expectedCancelTime.AddDate(0, 0, 1),
			BillingAnchor: expectedCancelTime.AddDate(0, 0, 1),
			Name:          "Test Subscription",
		})
		require.Nil(t, err)

		_, err = service.Create(ctx, subscriptiontestutils.ExampleNamespace, spec2)
		require.Nil(t, err)

		// Fourth, let's continue the first subscription
		_, err = service.Continue(ctx, sub1.NamespacedID)
		require.Error(t, err)
		issues, err := models.AsValidationIssues(err)
		require.NoError(t, err)
		require.Len(t, issues, 2)
		for _, issue := range issues {
			require.Equal(t, subscription.ErrOnlySingleSubscriptionAllowed.Code(), issue.Code())
		}
	})
}

func TestList(t *testing.T) {
	t.Run("Should list subscription by status", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z")
		clock.SetTime(currentTime)

		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup(t)

		deps := subscriptiontestutils.NewService(t, dbDeps)
		service := deps.SubscriptionService

		cust1, err := deps.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
			Namespace: subscriptiontestutils.ExampleNamespace,
			CustomerMutate: customer.CustomerMutate{
				Name: "Test Customer 1",
				UsageAttribution: &customer.CustomerUsageAttribution{
					SubjectKeys: []string{"subject-1"},
				},
			},
		})
		require.Nil(t, err)

		cust2, err := deps.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
			Namespace: subscriptiontestutils.ExampleNamespace,
			CustomerMutate: customer.CustomerMutate{
				Name: "Test Customer 2",
				UsageAttribution: &customer.CustomerUsageAttribution{
					SubjectKeys: []string{"subject-2"},
				},
			},
		})
		require.Nil(t, err)

		cust3, err := deps.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
			Namespace: subscriptiontestutils.ExampleNamespace,
			CustomerMutate: customer.CustomerMutate{
				Name: "Test Customer 3",
				UsageAttribution: &customer.CustomerUsageAttribution{
					SubjectKeys: []string{"subject-3"},
				},
			},
		})
		require.Nil(t, err)

		cust4, err := deps.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
			Namespace: subscriptiontestutils.ExampleNamespace,
			CustomerMutate: customer.CustomerMutate{
				Name: "Test Customer 4",
				UsageAttribution: &customer.CustomerUsageAttribution{
					SubjectKeys: []string{"subject-4"},
				},
			},
		})
		require.Nil(t, err)

		_ = deps.FeatureConnector.CreateExampleFeatures(t)
		plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

		// Let's create some subscriptions:
		// - One active
		spec1, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
			CustomerId:    cust1.ID,
			Currency:      "USD",
			ActiveFrom:    currentTime,
			BillingAnchor: currentTime,
			Name:          "Test Subscription",
		})
		require.Nil(t, err)

		sub1, err := service.Create(ctx, subscriptiontestutils.ExampleNamespace, spec1)
		require.Nil(t, err)

		// - One canceled
		spec2, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
			CustomerId:    cust2.ID,
			Currency:      "USD",
			ActiveFrom:    currentTime,
			BillingAnchor: currentTime,
			Name:          "Test Subscription",
		})
		require.Nil(t, err)

		sub2, err := service.Create(ctx, subscriptiontestutils.ExampleNamespace, spec2)
		require.Nil(t, err)

		sub2, err = service.Cancel(ctx, sub2.NamespacedID, subscription.Timing{
			Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
		})
		require.Nil(t, err, "error canceling subscription: %v", err)

		// - One inactive
		spec3, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
			CustomerId:    cust3.ID,
			Currency:      "USD",
			ActiveFrom:    currentTime.Add(-1 * time.Minute),
			BillingAnchor: currentTime.Add(-1 * time.Minute),
			Name:          "Test Subscription",
		})
		require.Nil(t, err)

		sub3, err := service.Create(ctx, subscriptiontestutils.ExampleNamespace, spec3)
		require.Nil(t, err)

		sub3, err = service.Cancel(ctx, sub3.NamespacedID, subscription.Timing{
			Enum: lo.ToPtr(subscription.TimingImmediate),
		})
		require.Nil(t, err)

		// - One scheduled
		spec4, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
			CustomerId:    cust4.ID,
			Currency:      "USD",
			ActiveFrom:    currentTime.AddDate(0, 0, 3),
			BillingAnchor: currentTime.AddDate(0, 0, 3),
			Name:          "Test Subscription",
		})
		require.Nil(t, err)

		sub4, err := service.Create(ctx, subscriptiontestutils.ExampleNamespace, spec4)
		require.Nil(t, err)

		// And let's validate the list for each and check that the correct ones are returned and the correct statuses are displayed
		t.Run("Should list active subscriptions", func(t *testing.T) {
			list, err := service.List(ctx, subscription.ListSubscriptionsInput{
				Status: []subscription.SubscriptionStatus{subscription.SubscriptionStatusActive},
			})
			require.Nil(t, err)
			require.Equal(t, 1, len(list.Items))
			require.Equal(t, sub1.ID, list.Items[0].ID)
			require.Equal(t, subscription.SubscriptionStatusActive, list.Items[0].GetStatusAt(clock.Now()))
		})

		t.Run("Should list canceled subscriptions", func(t *testing.T) {
			list, err := service.List(ctx, subscription.ListSubscriptionsInput{
				Status: []subscription.SubscriptionStatus{subscription.SubscriptionStatusCanceled},
			})
			require.Nil(t, err)
			require.Equal(t, 1, len(list.Items))
			require.Equal(t, sub2.ID, list.Items[0].ID)
			require.Equal(t, subscription.SubscriptionStatusCanceled, list.Items[0].GetStatusAt(clock.Now()))
		})

		t.Run("Should list inactive subscriptions", func(t *testing.T) {
			list, err := service.List(ctx, subscription.ListSubscriptionsInput{
				Status: []subscription.SubscriptionStatus{subscription.SubscriptionStatusInactive},
			})
			require.Nil(t, err)
			require.Equal(t, 1, len(list.Items))
			require.Equal(t, sub3.ID, list.Items[0].ID)
			require.Equal(t, subscription.SubscriptionStatusInactive, list.Items[0].GetStatusAt(clock.Now()))
		})

		t.Run("Should list scheduled subscriptions", func(t *testing.T) {
			list, err := service.List(ctx, subscription.ListSubscriptionsInput{
				Status: []subscription.SubscriptionStatus{subscription.SubscriptionStatusScheduled},
			})
			require.Nil(t, err)
			require.Equal(t, 1, len(list.Items))
			require.Equal(t, sub4.ID, list.Items[0].ID)
			require.Equal(t, subscription.SubscriptionStatusScheduled, list.Items[0].GetStatusAt(clock.Now()))
		})
	})
}

func TestSubscriptionChangeTrackingAnnotations(t *testing.T) {
	t.Run("Should set annotations when changing subscription to new plan", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z")
		clock.SetTime(currentTime)

		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup(t)

		deps := subscriptiontestutils.NewService(t, dbDeps)
		service := deps.SubscriptionService
		workflowService := deps.WorkflowService

		cust := deps.CustomerAdapter.CreateExampleCustomer(t)
		_ = deps.FeatureConnector.CreateExampleFeatures(t)
		plan1 := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

		// Create first subscription
		sub1, err := workflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
			ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Timing: subscription.Timing{
					Custom: &currentTime,
				},
				Name: "First Subscription",
			},
			CustomerID: cust.ID,
			Namespace:  subscriptiontestutils.ExampleNamespace,
		}, plan1)
		require.Nil(t, err)

		// Create second plan
		examplePlanInput2 := subscriptiontestutils.GetExamplePlanInput(t)
		examplePlanInput2.Key = "example-plan-2"
		examplePlanInput2.Name = "Example Plan 2"
		plan2 := deps.PlanHelper.CreatePlan(t, examplePlanInput2)

		// Change to new plan
		curr, new, err := workflowService.ChangeToPlan(ctx, sub1.Subscription.NamespacedID, subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{
				Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
			},
			Name: "Second Subscription",
		}, plan2)
		require.Nil(t, err)

		// Verify old subscription has superseding subscription ID
		currView, err := service.GetView(ctx, curr.NamespacedID)
		require.Nil(t, err)
		require.NotNil(t, currView.Subscription.Annotations)
		supersedingID := subscription.AnnotationParser.GetSupersedingSubscriptionID(currView.Subscription.Annotations)
		require.NotNil(t, supersedingID)
		assert.Equal(t, new.Subscription.ID, *supersedingID)

		// Verify new subscription has previous subscription ID
		newView, err := service.GetView(ctx, new.Subscription.NamespacedID)
		require.Nil(t, err)
		require.NotNil(t, newView.Subscription.Annotations)
		previousID := subscription.AnnotationParser.GetPreviousSubscriptionID(newView.Subscription.Annotations)
		require.NotNil(t, previousID)
		assert.Equal(t, curr.ID, *previousID)
	})

	t.Run("Should clean up annotations when deleting subscription with superseding subscription", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z")
		clock.SetTime(currentTime)

		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup(t)

		deps := subscriptiontestutils.NewService(t, dbDeps)
		service := deps.SubscriptionService
		workflowService := deps.WorkflowService

		cust := deps.CustomerAdapter.CreateExampleCustomer(t)
		_ = deps.FeatureConnector.CreateExampleFeatures(t)
		plan1 := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

		// Create first subscription
		sub1, err := workflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
			ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Timing: subscription.Timing{
					Custom: &currentTime,
				},
				Name: "First Subscription",
			},
			CustomerID: cust.ID,
			Namespace:  subscriptiontestutils.ExampleNamespace,
		}, plan1)
		require.Nil(t, err)

		// Create second plan
		examplePlanInput2 := subscriptiontestutils.GetExamplePlanInput(t)
		examplePlanInput2.Key = "example-plan-2"
		examplePlanInput2.Name = "Example Plan 2"
		plan2 := deps.PlanHelper.CreatePlan(t, examplePlanInput2)

		// Change to new plan - this creates sub2 and links sub1->sub2
		curr, new, err := workflowService.ChangeToPlan(ctx, sub1.Subscription.NamespacedID, subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{
				Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
			},
			Name: "Second Subscription",
		}, plan2)
		require.Nil(t, err)

		// Verify sub2 is scheduled (can be deleted)
		sub2View, err := service.GetView(ctx, new.Subscription.NamespacedID)
		require.Nil(t, err)
		require.Equal(t, subscription.SubscriptionStatusScheduled, sub2View.Subscription.GetStatusAt(clock.Now()))

		// Delete sub2 (scheduled subscriptions can be deleted)
		err = service.Delete(ctx, new.Subscription.NamespacedID)
		require.Nil(t, err)

		// Verify sub1 no longer has superseding subscription ID
		sub1View, err := service.GetView(ctx, curr.NamespacedID)
		require.Nil(t, err)
		if sub1View.Subscription.Annotations != nil {
			supersedingID := subscription.AnnotationParser.GetSupersedingSubscriptionID(sub1View.Subscription.Annotations)
			assert.Nil(t, supersedingID)
		}
	})

	t.Run("Should clean up annotations when deleting subscription with previous subscription", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z")
		clock.SetTime(currentTime)

		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup(t)

		deps := subscriptiontestutils.NewService(t, dbDeps)
		service := deps.SubscriptionService
		workflowService := deps.WorkflowService

		cust := deps.CustomerAdapter.CreateExampleCustomer(t)
		_ = deps.FeatureConnector.CreateExampleFeatures(t)
		plan1 := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

		// Create first subscription
		sub1, err := workflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
			ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Timing: subscription.Timing{
					Custom: &currentTime,
				},
				Name: "First Subscription",
			},
			CustomerID: cust.ID,
			Namespace:  subscriptiontestutils.ExampleNamespace,
		}, plan1)
		require.Nil(t, err)

		// Create second plan
		examplePlanInput2 := subscriptiontestutils.GetExamplePlanInput(t)
		examplePlanInput2.Key = "example-plan-2"
		examplePlanInput2.Name = "Example Plan 2"
		plan2 := deps.PlanHelper.CreatePlan(t, examplePlanInput2)

		// Change to new plan - this creates sub2 and links sub1->sub2
		curr, new, err := workflowService.ChangeToPlan(ctx, sub1.Subscription.NamespacedID, subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{
				Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
			},
			Name: "Second Subscription",
		}, plan2)
		require.Nil(t, err)

		// Verify sub2 is scheduled (can be deleted)
		newView, err := service.GetView(ctx, new.Subscription.NamespacedID)
		require.Nil(t, err)
		require.Equal(t, subscription.SubscriptionStatusScheduled, newView.Subscription.GetStatusAt(clock.Now()))

		// Delete new subscription (sub2) - scheduled subscriptions can be deleted
		err = service.Delete(ctx, new.Subscription.NamespacedID)
		require.Nil(t, err)

		// Verify sub1 no longer has superseding subscription ID
		sub1View, err := service.GetView(ctx, curr.NamespacedID)
		require.Nil(t, err)
		if sub1View.Subscription.Annotations != nil {
			supersedingID := subscription.AnnotationParser.GetSupersedingSubscriptionID(sub1View.Subscription.Annotations)
			assert.Nil(t, supersedingID)
		}
	})

	t.Run("Should clean up annotations when deleting subscription without any links", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z")
		clock.SetTime(currentTime)

		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup(t)

		deps := subscriptiontestutils.NewService(t, dbDeps)
		service := deps.SubscriptionService
		workflowService := deps.WorkflowService

		cust := deps.CustomerAdapter.CreateExampleCustomer(t)
		_ = deps.FeatureConnector.CreateExampleFeatures(t)
		plan1 := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

		// Create standalone scheduled subscription (can be deleted)
		futureTime := currentTime.AddDate(0, 1, 0)
		sub1, err := workflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
			ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Timing: subscription.Timing{
					Custom: &futureTime,
				},
				Name: "Standalone Subscription",
			},
			CustomerID: cust.ID,
			Namespace:  subscriptiontestutils.ExampleNamespace,
		}, plan1)
		require.Nil(t, err)

		// Verify subscription is scheduled
		sub1View, err := service.GetView(ctx, sub1.Subscription.NamespacedID)
		require.Nil(t, err)
		require.Equal(t, subscription.SubscriptionStatusScheduled, sub1View.Subscription.GetStatusAt(clock.Now()))

		// Delete subscription - scheduled subscriptions can be deleted
		err = service.Delete(ctx, sub1.Subscription.NamespacedID)
		require.Nil(t, err)
	})

	t.Run("Should handle deleting subscription with nil annotations", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z")
		clock.SetTime(currentTime)

		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup(t)

		deps := subscriptiontestutils.NewService(t, dbDeps)
		service := deps.SubscriptionService
		workflowService := deps.WorkflowService

		cust := deps.CustomerAdapter.CreateExampleCustomer(t)
		_ = deps.FeatureConnector.CreateExampleFeatures(t)
		plan1 := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

		// Create scheduled subscription with nil annotations (can be deleted)
		futureTime := currentTime.AddDate(0, 1, 0)
		sub1, err := workflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
			ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Timing: subscription.Timing{
					Custom: &futureTime,
				},
				Name: "Subscription with Nil Annotations",
			},
			CustomerID:  cust.ID,
			Namespace:   subscriptiontestutils.ExampleNamespace,
			Annotations: nil,
		}, plan1)
		require.Nil(t, err)

		// Verify subscription is scheduled
		sub1View, err := service.GetView(ctx, sub1.Subscription.NamespacedID)
		require.Nil(t, err)
		require.Equal(t, subscription.SubscriptionStatusScheduled, sub1View.Subscription.GetStatusAt(clock.Now()))

		// Delete subscription - scheduled subscriptions can be deleted
		err = service.Delete(ctx, sub1.Subscription.NamespacedID)
		require.Nil(t, err)
	})
}
