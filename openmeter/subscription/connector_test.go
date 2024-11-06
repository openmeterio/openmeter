package subscription_test

import (
	"context"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionentitlement "github.com/openmeterio/openmeter/openmeter/subscription/entitlement"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/recurrence"
	"github.com/openmeterio/openmeter/pkg/timezone"
)

func TestCreation(t *testing.T) {
	t.Run("Should create subscription without customizations", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z")
		clock.SetTime(currentTime)

		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup()

		command, query, deps := subscriptiontestutils.NewCommandAndQuery(t, dbDeps)

		deps.PlanAdapter.AddPlan(subscriptiontestutils.ExamplePlan)
		cust := deps.CustomerAdapter.CreateExampleCustomer(t)
		_ = deps.FeatureConnector.CreateExampleFeature(t)

		sub, err := command.Create(ctx, subscription.NewSubscriptionRequest{
			Plan:       subscriptiontestutils.ExamplePlanRef,
			Namespace:  subscriptiontestutils.ExampleNamespace,
			ActiveFrom: currentTime,
			CustomerID: cust.ID,
			Currency:   "USD",
		})

		require.Nil(t, err)
		require.Equal(t, subscriptiontestutils.ExamplePlanRef, sub.Plan)
		require.Equal(t, subscriptiontestutils.ExampleNamespace, sub.Namespace)
		require.Equal(t, cust.ID, sub.CustomerId)
		require.Equal(t, currencyx.Code("USD"), sub.Currency)

		t.Run("Should find subscription by ID", func(t *testing.T) {
			found, err := query.Get(ctx, models.NamespacedID{
				ID:        sub.ID,
				Namespace: sub.Namespace,
			})

			assert.Nil(t, err)
			assert.Equal(t, sub.ID, found.ID)
			assert.Equal(t, sub.Plan, found.Plan)
			assert.Equal(t, sub.Namespace, found.Namespace)
			assert.Equal(t, sub.CustomerId, found.CustomerId)
			assert.Equal(t, sub.Currency, found.Currency)
		})

		t.Run("Should create subscription according to plan", func(t *testing.T) {
			found, err := query.Expand(ctx, models.NamespacedID{ID: sub.ID, Namespace: sub.Namespace})
			assert.Nil(t, err)

			// Test Sub
			foundSub := found.Sub()

			assert.Equal(t, sub.ID, foundSub.ID)
			assert.Equal(t, sub.Plan, foundSub.Plan)
			assert.Equal(t, sub.Namespace, foundSub.Namespace)
			assert.Equal(t, sub.CustomerId, foundSub.CustomerId)
			assert.Equal(t, sub.Currency, foundSub.Currency)

			// Test Phases

			plan, err := deps.PlanAdapter.GetVersion(ctx, sub.Plan.Key, sub.Plan.Version)
			require.Nil(t, err)

			planPhases := plan.GetPhases()
			foundPhases := found.Phases()

			require.Equal(t, len(planPhases), len(foundPhases))

			for i := range planPhases {
				assert.Equal(t, planPhases[i].GetKey(), foundPhases[i].Key())
				assert.Equal(t, planPhases[i].ToCreateSubscriptionPhasePlanInput().PhaseKey, foundPhases[i].Key())

				expectedStart, _ := planPhases[i].ToCreateSubscriptionPhasePlanInput().StartAfter.AddTo(foundSub.ActiveFrom)

				assert.Equal(t, expectedStart.UTC(), foundPhases[i].ActiveFrom())

				// Test Rate Cards of Phase
				planPhase := planPhases[i]
				foundPhase := foundPhases[i]

				planRateCards := planPhase.GetRateCards()
				foundRateCards := foundPhase.Items()

				require.Equal(t, len(planRateCards), len(foundRateCards), "rate card count mismatch for phase %s", planPhase.GetKey())

				for j := range planRateCards {
					assert.Equal(t, planRateCards[j].GetKey(), foundRateCards[j].Key())
					assert.Equal(t, planRateCards[j].ToCreateSubscriptionItemPlanInput().ItemKey, foundRateCards[j].Key())

					featureKey, hasFeatureKey := foundRateCards[j].FeatureKey()

					if hasFeatureKey {
						pFeatureKey := planRateCards[j].ToCreateSubscriptionItemPlanInput().FeatureKey
						require.NotNil(t, pFeatureKey)
						assert.Equal(t, *pFeatureKey, featureKey)
					} else {
						assert.Nil(t, planRateCards[j].ToCreateSubscriptionItemPlanInput().FeatureKey)
					}

					entSpec := planRateCards[j].ToCreateSubscriptionItemPlanInput().CreateEntitlementInput
					if entSpec != nil {
						ent, exists := foundRateCards[j].Entitlement()
						require.True(t, exists)
						assert.Equal(t, entSpec.EntitlementType, ent.Entitlement.EntitlementType)
						// To simplify here we expect the ExamplePlan to have UsagePeriodISODuration set to 1 month
						require.Equal(t, entSpec.UsagePeriodISODuration, &subscriptiontestutils.ISOMonth)
						assert.Equal(t, recurrence.RecurrencePeriodMonth, ent.Entitlement.UsagePeriod.Interval)
						// Validate that entitlement UsagePeriod matches expected by anchor which is the phase start time
						assert.Equal(t, foundPhase.ActiveFrom(), ent.Entitlement.UsagePeriod.Anchor)

						// Validate that entitlement activeFrom is the same as the phase activeFrom
						require.NotNil(t, ent.Entitlement.ActiveFrom)
						assert.Equal(t, foundPhase.ActiveFrom(), *ent.Entitlement.ActiveFrom)

						// Validate that the entitlement is only active until the phase is scheduled to be
						if i < len(planPhases)-1 {
							nextPhase := planPhases[i+1]
							nextPhaseStart, _ := nextPhase.ToCreateSubscriptionPhasePlanInput().StartAfter.AddTo(foundSub.ActiveFrom)
							require.NotNil(t, ent.Entitlement.ActiveTo)
							assert.Equal(t, nextPhaseStart.UTC(), *ent.Entitlement.ActiveTo)
						}
					}

					priceSpec := planRateCards[j].ToCreateSubscriptionItemPlanInput().CreatePriceInput
					if priceSpec != nil {
						price, exists := foundRateCards[j].Price()
						require.True(t, exists)
						assert.Equal(t, priceSpec.Key, price.Key)
						assert.Equal(t, priceSpec.Value, price.Value)
						// Validate that price activeFrom is the same as the phase activeFrom
						assert.Equal(t, foundPhase.ActiveFrom(), price.ActiveFrom)

						// Validate that the price is only active until the phase is scheduled to be
						if i < len(planPhases)-1 {
							nextPhase := planPhases[i+1]
							nextPhaseStart, _ := nextPhase.ToCreateSubscriptionPhasePlanInput().StartAfter.AddTo(foundSub.ActiveFrom)
							require.NotNil(t, price.ActiveTo)
							assert.Equal(t, nextPhaseStart.UTC(), *price.ActiveTo)
						}
					}
				}
			}
		})
	})

	t.Run("Should create subscription with customizations", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z")
		clock.SetTime(currentTime)

		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup()

		command, query, deps := subscriptiontestutils.NewCommandAndQuery(t, dbDeps)

		deps.PlanAdapter.AddPlan(subscriptiontestutils.ExamplePlan)
		_ = deps.FeatureConnector.CreateExampleFeature(t)

		// All subs will use the same Plan as a baseline but they will affect different customers
		createCustomerInfoByName := func(name string) customerentity.Customer {
			return customerentity.Customer{
				ManagedResource: models.ManagedResource{
					Name: name,
				},
				PrimaryEmail: lo.ToPtr(lo.CamelCase(name) + "@fake.com"),
				Currency:     lo.ToPtr(currencyx.Code("USD")),
				Timezone:     lo.ToPtr(timezone.Timezone("America/Los_Angeles")),
				UsageAttribution: customerentity.CustomerUsageAttribution{
					SubjectKeys: []string{lo.CamelCase(name)},
				},
			}
		}

		t.Run("Should disallow adding a phase during subscription creation", func(t *testing.T) {
			cust, err := deps.CustomerAdapter.CreateCustomer(ctx, customerentity.CreateCustomerInput{
				Namespace: subscriptiontestutils.ExampleNamespace,
				Customer:  createCustomerInfoByName("Johnas Doe"),
			})
			require.Nil(t, err)

			_, err = command.Create(ctx, subscription.NewSubscriptionRequest{
				Plan:       subscriptiontestutils.ExamplePlanRef,
				Namespace:  subscriptiontestutils.ExampleNamespace,
				ActiveFrom: currentTime,
				CustomerID: cust.ID,
				Currency:   "USD",
				ItemCustomization: []subscription.Patch{
					subscription.PatchAddPhase{
						PhaseKey: "extra-phase",
						CreateInput: subscription.CreateSubscriptionPhaseInput{
							CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
								PhaseKey:   "extra-phase",
								StartAfter: testutils.GetISODuration(t, "P9M"),
							},
							CreateSubscriptionPhaseCustomerInput: subscription.CreateSubscriptionPhaseCustomerInput{
								// TODO: implement discounts
								CreateDiscountInput: nil,
							},
						},
					},
				},
			})
			assert.NotNil(t, err)
			assert.ErrorAs(t, err, lo.ToPtr(&subscription.PatchForbiddenError{}), "expected error to be of type PatchForbiddenError, got %T", err)
			assert.ErrorContains(t, err, "you can only add a phase in edit")
		})

		t.Run("Should disallow extending a phase during subscription creation", func(t *testing.T) {
			cust, err := deps.CustomerAdapter.CreateCustomer(ctx, customerentity.CreateCustomerInput{
				Namespace: subscriptiontestutils.ExampleNamespace,
				Customer:  createCustomerInfoByName("Jane Doe"),
			})
			require.Nil(t, err)

			_, err = command.Create(ctx, subscription.NewSubscriptionRequest{
				Plan:       subscriptiontestutils.ExamplePlanRef,
				Namespace:  subscriptiontestutils.ExampleNamespace,
				ActiveFrom: currentTime,
				CustomerID: cust.ID,
				Currency:   "USD",
				ItemCustomization: []subscription.Patch{
					// Let's extend the first phase by 1 month
					subscription.PatchExtendPhase{
						PhaseKey: "test-phase-1",
						Duration: testutils.GetISODuration(t, "P1M"),
					},
				},
			})

			assert.NotNil(t, err)
			assert.ErrorAs(t, err, lo.ToPtr(&subscription.PatchForbiddenError{}), "expected error to be of type PatchForbiddenError, got %T", err)
			assert.ErrorContains(t, err, "you can only extend a phase in edit")
		})

		t.Run("Should remove RateCard from phase", func(t *testing.T) {
			cust, err := deps.CustomerAdapter.CreateCustomer(ctx, customerentity.CreateCustomerInput{
				Namespace: subscriptiontestutils.ExampleNamespace,
				Customer:  createCustomerInfoByName("Joshua Doe"),
			})
			require.Nil(t, err)

			// Let's validate that the RateCard we're removing exists
			plan, err := deps.PlanAdapter.GetVersion(ctx, subscriptiontestutils.ExamplePlanRef.Key, subscriptiontestutils.ExamplePlanRef.Version)
			require.Nil(t, err)

			require.GreaterOrEqual(t, len(plan.GetPhases()), 2, "example plan should have at least 2 phases")
			require.Equal(t, "test-phase-2", plan.GetPhases()[1].GetKey(), "example plan's second phase should have known key")
			require.GreaterOrEqual(t, len(plan.GetPhases()[1].GetRateCards()), 1, "example plan's second phase should have at least 1 rate card")
			require.Equal(t, "test-rate-card-1", plan.GetPhases()[1].GetRateCards()[0].GetKey(), "example plan's second phase should have a rate card with known key")

			// Let's validate that the RateCard we're removing creates an entitlement
			require.NotNil(t, plan.GetPhases()[1].GetRateCards()[0].ToCreateSubscriptionItemPlanInput().CreateEntitlementInput, "example plan's second phase's rate card should create an entitlement")

			sub, err := command.Create(ctx, subscription.NewSubscriptionRequest{
				Plan:       subscriptiontestutils.ExamplePlanRef,
				Namespace:  subscriptiontestutils.ExampleNamespace,
				ActiveFrom: currentTime,
				CustomerID: cust.ID,
				Currency:   "USD",
				ItemCustomization: []subscription.Patch{
					subscription.PatchRemoveItem{
						PhaseKey: "test-phase-2",
						ItemKey:  "test-rate-card-1",
					},
				},
			})
			assert.Nil(t, err)

			// Let's validate that the RateCard was removed
			found, err := query.Expand(ctx, models.NamespacedID{ID: sub.ID, Namespace: sub.Namespace})
			require.Nil(t, err)

			assert.GreaterOrEqual(t, len(found.Phases()), 2, "subscription should have at least 2 phases")
			assert.Equal(t, "test-phase-2", found.Phases()[1].Key(), "subscription's second phase should have known key")
			assert.Equal(t, len(plan.GetPhases()[1].GetRateCards())-1, len(found.Phases()[1].Items()), "subscription's second phase should have one less rate card")

			for _, item := range found.Phases()[1].Items() {
				assert.NotEqual(t, "test-rate-card-1", item.Key(), "subscription's second phase should not have the removed rate card")
			}

			// Let's validate that the RateCard we removed did not create an entitlement
			_, err = deps.EntitlementAdapter.GetForItem(ctx, sub.Namespace, subscription.SubscriptionItemRef{
				SubscriptionId: sub.ID,
				PhaseKey:       "test-phase-2",
				ItemKey:        "test-rate-card-1",
			}, clock.Now())

			assert.NotNil(t, err)
			assert.ErrorAs(t, err, lo.ToPtr(&subscriptionentitlement.NotFoundError{}), "expected error to be of type EntitlementNotFoundError, got %T", err)
		})

		t.Run("Should add RateCard to phase", func(t *testing.T) {
			cust, err := deps.CustomerAdapter.CreateCustomer(ctx, customerentity.CreateCustomerInput{
				Namespace: subscriptiontestutils.ExampleNamespace,
				Customer:  createCustomerInfoByName("Jasmine Doe"),
			})
			require.Nil(t, err)

			// Let's validate that the phase exists and it doesn't have a RateCard with the same key
			subPlan, err := deps.PlanAdapter.GetVersion(ctx, subscriptiontestutils.ExamplePlanRef.Key, subscriptiontestutils.ExamplePlanRef.Version)
			require.Nil(t, err)

			require.GreaterOrEqual(t, len(subPlan.GetPhases()), 2, "example plan should have at least 2 phases")
			require.Equal(t, "test-phase-2", subPlan.GetPhases()[1].GetKey(), "example plan's second phase should have known key")
			for _, rateCard := range subPlan.GetPhases()[1].GetRateCards() {
				require.NotEqual(t, "added-ratecard", rateCard.GetKey(), "example plan's second phase should not have a rate card with known key")
			}

			sub, err := command.Create(ctx, subscription.NewSubscriptionRequest{
				Plan:       subscriptiontestutils.ExamplePlanRef,
				Namespace:  subscriptiontestutils.ExampleNamespace,
				ActiveFrom: currentTime,
				CustomerID: cust.ID,
				Currency:   "USD",
				ItemCustomization: []subscription.Patch{
					subscription.PatchAddItem{
						PhaseKey: "test-phase-2",
						ItemKey:  "added-ratecard",
						CreateInput: subscription.SubscriptionItemSpec{
							CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
								PhaseKey: "test-phase-2",
								ItemKey:  "added-ratecard",
								CreatePriceInput: &subscription.CreatePriceInput{
									PhaseKey: "test-phase-2",
									ItemKey:  "added-ratecard",
									Value:    subscriptiontestutils.GetFlatPrice(100),
									Key:      "added-ratecard",
								},
							},
						},
					},
				},
			})
			assert.Nil(t, err)

			// Let's validate that the RateCard was added
			found, err := query.Expand(ctx, models.NamespacedID{ID: sub.ID, Namespace: sub.Namespace})
			require.Nil(t, err)

			assert.GreaterOrEqual(t, len(found.Phases()), 2, "subscription should have at least 2 phases")
			assert.Equal(t, "test-phase-2", found.Phases()[1].Key(), "subscription's second phase should have known key")
			assert.Equal(t, len(subPlan.GetPhases()[1].GetRateCards())+1, len(found.Phases()[1].Items()), "subscription's second phase should have one more rate card")

			// Let's find our new Item
			var foundItem subscription.SubscriptionItemView
			for _, item := range found.Phases()[1].Items() {
				if item.Key() == "added-ratecard" {
					foundItem = item
					break
				}
			}

			require.NotNil(t, foundItem, "subscription's second phase should have the added rate card")

			// Let's validate that the RateCard we added created a price
			price, exists := foundItem.Price()
			require.True(t, exists)

			assert.Equal(t, "added-ratecard", price.Key)
			assert.Equal(t, subscriptiontestutils.GetFlatPrice(100), price.Value)
			assert.Equal(t, found.Phases()[1].ActiveFrom(), price.ActiveFrom)
		})
	})
}
