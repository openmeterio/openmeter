package service_test

import (
	"context"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
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

		services, deps := subscriptiontestutils.NewService(t, dbDeps)
		service := services.Service

		cust := deps.CustomerAdapter.CreateExampleCustomer(t)
		_ = deps.FeatureConnector.CreateExampleFeature(t)
		plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

		defaultSpecFromPlan, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
			CustomerId: cust.ID,
			Currency:   "USD",
			ActiveFrom: currentTime,
			Name:       "Test Subscription",
		})
		require.Nil(t, err)

		sub, err := service.Create(ctx, subscriptiontestutils.ExampleNamespace, defaultSpecFromPlan)

		require.Nil(t, err)
		require.Equal(t, plan.ToCreateSubscriptionPlanInput().Plan, sub.PlanRef)
		require.Equal(t, subscriptiontestutils.ExampleNamespace, sub.Namespace)
		require.Equal(t, cust.ID, sub.CustomerId)
		require.Equal(t, currencyx.Code("USD"), sub.Currency)

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

	t.Run("Should not allow creating a subscription with different currency compared to the customer", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:11Z")
		clock.SetTime(currentTime)

		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup(t)

		services, deps := subscriptiontestutils.NewService(t, dbDeps)
		service := services.Service

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

		_ = deps.FeatureConnector.CreateExampleFeature(t)
		plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

		defaultSpecFromPlan, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
			CustomerId: cust.ID,
			Currency:   "USD",
			ActiveFrom: currentTime,
			Name:       "Test Subscription",
		})
		require.Nil(t, err)

		_, err = service.Create(ctx, subscriptiontestutils.ExampleNamespace, defaultSpecFromPlan)

		require.Error(t, err)
		require.ErrorAs(t, err, lo.ToPtr(&models.GenericUserError{}))
	})

	t.Run("Should set customer currency based on subscription", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:11Z")
		clock.SetTime(currentTime)

		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup(t)

		services, deps := subscriptiontestutils.NewService(t, dbDeps)
		service := services.Service

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

		_ = deps.FeatureConnector.CreateExampleFeature(t)
		plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

		defaultSpecFromPlan, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
			CustomerId: cust.ID,
			Currency:   "USD",
			ActiveFrom: currentTime,
			Name:       "Test Subscription",
		})
		require.Nil(t, err)

		_, err = service.Create(ctx, subscriptiontestutils.ExampleNamespace, defaultSpecFromPlan)

		require.NoError(t, err)

		c, err := deps.CustomerService.GetCustomer(ctx, customer.GetCustomerInput{
			Namespace: cust.Namespace,
			ID:        cust.ID,
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

		services, deps := subscriptiontestutils.NewService(t, dbDeps)
		service := services.Service

		cust := deps.CustomerAdapter.CreateExampleCustomer(t)
		_ = deps.FeatureConnector.CreateExampleFeature(t)
		plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

		// First, let's create a subscription
		defaultSpecFromPlan, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
			CustomerId: cust.ID,
			Currency:   "USD",
			ActiveFrom: currentTime,
			Name:       "Test Subscription",
		})
		require.Nil(t, err)

		sub, err := service.Create(ctx, subscriptiontestutils.ExampleNamespace, defaultSpecFromPlan)

		require.Nil(t, err)

		// Second, let's cancel the subscription
		cancelTime := testutils.GetRFC3339Time(t, "2021-04-01T00:00:00Z")
		cancelledSub, err := service.Cancel(ctx, sub.NamespacedID, subscription.Timing{
			Custom: &cancelTime,
		})

		assert.Nil(t, err)
		assert.Equal(t, sub.ID, cancelledSub.ID)
		assert.Equal(t, sub.PlanRef, cancelledSub.PlanRef)
		assert.Equal(t, sub.Namespace, cancelledSub.Namespace)
		assert.Equal(t, sub.CustomerId, cancelledSub.CustomerId)
		assert.Equal(t, sub.Currency, cancelledSub.Currency)
		assert.Equal(t, cancelTime, *cancelledSub.ActiveTo)
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
					if !foundItemCadence.ActiveFrom.After(cancelTime) {
						// - left in tact if their phase ends before the cancel time
						if phaseCadence.ActiveTo != nil && !phaseCadence.ActiveTo.After(cancelTime) {
							if foundItemCadence.ActiveTo != nil && foundItemCadence.ActiveTo.Equal(*phaseCadence.ActiveTo) {
								satisfies = true
							}
							// - their ActiveTo time set to the cancel time (if they started before the cancel time)
						} else if foundItemCadence.ActiveTo != nil && foundItemCadence.ActiveTo.Equal(cancelTime) {
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
						if !ent.Cadence.ActiveFrom.After(cancelTime) {
							if phaseCadence.ActiveTo != nil && !phaseCadence.ActiveTo.After(cancelTime) {
								if ent.Cadence.ActiveTo != nil && ent.Cadence.ActiveTo.Equal(*phaseCadence.ActiveTo) {
									satisfies = true
								}
							} else if ent.Cadence.ActiveTo != nil && ent.Cadence.ActiveTo.Equal(cancelTime) {
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

		services, deps := subscriptiontestutils.NewService(t, dbDeps)
		service := services.Service

		cust := deps.CustomerAdapter.CreateExampleCustomer(t)
		_ = deps.FeatureConnector.CreateExampleFeature(t)
		plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

		// First, let's create a subscription
		defaultSpecFromPlan, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
			CustomerId: cust.ID,
			Currency:   "USD",
			ActiveFrom: currentTime,
			Name:       "Test Subscription",
		})
		require.Nil(t, err)

		sub, err := service.Create(ctx, subscriptiontestutils.ExampleNamespace, defaultSpecFromPlan)

		require.Nil(t, err)

		// Second, let's cancel the subscription
		cancelTime := testutils.GetRFC3339Time(t, "2021-04-01T00:00:00Z")
		_, err = service.Cancel(ctx, sub.NamespacedID, subscription.Timing{
			Custom: &cancelTime,
		})

		require.Nil(t, err)

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
}
