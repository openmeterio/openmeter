package service_test

import (
	"context"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

func TestCreation(t *testing.T) {
	t.Run("Should create subscription as specced", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z")
		clock.SetTime(currentTime)

		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup()

		services, deps := subscriptiontestutils.NewService(t, dbDeps)
		service := services.Service

		cust := deps.CustomerAdapter.CreateExampleCustomer(t)
		_ = deps.FeatureConnector.CreateExampleFeature(t)
		plan := deps.PlanAdapter.CreateExamplePlan(t, ctx)

		defaultSpecFromPlan, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
			CustomerId: cust.ID,
			Currency:   "USD",
			ActiveFrom: currentTime,
			Name:       "Test Subscription",
		})
		require.Nil(t, err)

		sub, err := service.Create(ctx, subscriptiontestutils.ExampleNamespace, defaultSpecFromPlan)

		require.Nil(t, err)
		require.Equal(t, lo.ToPtr(plan.GetRef()), sub.PlanRef)
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

			// Test Phases

			foundPhases := found.Phases
			specPhases := defaultSpecFromPlan.GetSortedPhases()

			require.Equal(t, len(specPhases), len(foundPhases))

			for i := range specPhases {
				assert.Equal(t, specPhases[i].PhaseKey, foundPhases[i].SubscriptionPhase.Key)

				expectedStart, _ := specPhases[i].StartAfter.AddTo(foundSub.ActiveFrom)

				assert.Equal(t, expectedStart.UTC(), foundPhases[i].ActiveFrom(foundSub.CadencedModel))

				// Test Rate Cards of Phase
				specPhase := specPhases[i]
				foundPhase := foundPhases[i]

				specItemsByKey := specPhase.ItemsByKey
				foundItemsByKey := foundPhase.ItemsByKey

				require.Equal(t, len(specItemsByKey), len(foundItemsByKey), "item count mismatch for phase %s", specPhase.PhaseKey)

				for specItemsKey := range specItemsByKey {
					specItems, ok := specItemsByKey[specItemsKey]
					require.True(t, ok, "item %s not found in spec phase %s", specItemsKey, specPhase.PhaseKey)
					foundItemsByKey, foundItemsForKey := foundItemsByKey[specItemsKey]
					require.True(t, foundItemsForKey, "item %s not found in found phase %s", specItemsKey, specPhase.PhaseKey)

					require.Equal(t, len(specItems), len(foundItemsByKey), "item count mismatch for item %s in phase %s", specItemsKey, specPhase.PhaseKey)

					for idx, specItem := range specItems {
						foundItem := foundItemsByKey[idx]

						assert.Equal(t, specItem.ItemKey, foundItem.SubscriptionItem.Key)

						pFeatureKey := specItem.RateCard.FeatureKey
						if foundItem.SubscriptionItem.RateCard.FeatureKey != nil {
							require.NotNil(t, pFeatureKey)
							assert.Equal(t, pFeatureKey, foundItem.SubscriptionItem.RateCard.FeatureKey)
						} else {
							assert.Empty(t, pFeatureKey)
						}

						rcInp := specItem.CreateSubscriptionItemPlanInput

						if rcEnt := rcInp.RateCard.EntitlementTemplate; rcEnt != nil {
							ent := foundItem.Entitlement
							exists := ent != nil
							require.True(t, exists)
							entInp := ent.ToScheduleSubscriptionEntitlementInput()
							assert.Equal(t, rcEnt.Type(), entInp.CreateEntitlementInputs.GetType())
							// To simplify here we expect the ExamplePlan to have UsagePeriodISODuration set to 1 month
							up, err := recurrence.FromISODuration(&subscriptiontestutils.ISOMonth, ent.Cadence.ActiveFrom)
							require.Nil(t, err)

							require.Equal(t, entInp.CreateEntitlementInputs.UsagePeriod, lo.ToPtr(entitlement.UsagePeriod(up)))
							assert.Equal(t, recurrence.RecurrencePeriodMonth, ent.Entitlement.UsagePeriod.Interval)
							// Validate that entitlement UsagePeriod matches expected by anchor which is the phase start time
							assert.Equal(t, foundPhase.ActiveFrom(found.Subscription.CadencedModel), ent.Entitlement.UsagePeriod.Anchor)

							// Validate that entitlement activeFrom is the same as the phase activeFrom
							require.NotNil(t, ent.Entitlement.ActiveFrom)
							assert.Equal(t, foundPhase.ActiveFrom(found.Subscription.CadencedModel), *ent.Entitlement.ActiveFrom)

							// Validate that the entitlement is only active until the phase is scheduled to be
							if i < len(specPhases)-1 {
								nextPhase := specPhases[i+1]
								nextPhaseStart, _ := nextPhase.StartAfter.AddTo(foundSub.ActiveFrom)
								require.NotNil(t, ent.Entitlement.ActiveTo)
								assert.Equal(t, nextPhaseStart.UTC(), *ent.Entitlement.ActiveTo)
							}
						}
					}
				}
			}
		})
	})
}

func TestCancellation(t *testing.T) {
	t.Run("Should cancel subscription", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z")
		clock.SetTime(currentTime)

		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup()

		services, deps := subscriptiontestutils.NewService(t, dbDeps)
		service := services.Service

		cust := deps.CustomerAdapter.CreateExampleCustomer(t)
		_ = deps.FeatureConnector.CreateExampleFeature(t)
		plan := deps.PlanAdapter.CreateExamplePlan(t, ctx)

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
		cancelledSub, err := service.Cancel(ctx, sub.NamespacedID, cancelTime)

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
		defer dbDeps.Cleanup()

		services, deps := subscriptiontestutils.NewService(t, dbDeps)
		service := services.Service

		cust := deps.CustomerAdapter.CreateExampleCustomer(t)
		_ = deps.FeatureConnector.CreateExampleFeature(t)
		plan := deps.PlanAdapter.CreateExamplePlan(t, ctx)

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
		_, err = service.Cancel(ctx, sub.NamespacedID, cancelTime)

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
