package subscription_test

import (
	"context"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/models"
)

var oneMonthISO, _ = datex.ISOString("P1M").Parse()

func TestEdit(t *testing.T) {
	t.Run("Should edit subscription of ExamplePlan", func(t *testing.T) {
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

		t.Run("Should work fine if no patches were provided", func(t *testing.T) {
			_, err := command.Edit(ctx, models.NamespacedID{
				ID:        sub.ID,
				Namespace: sub.Namespace,
			}, []subscription.Patch{})
			require.Nil(t, err)
		})

		t.Run("Should add new items to an existing phase", func(t *testing.T) {
			// Let's assert that the base Subscription looks as we believe
			subView, err := query.Expand(ctx, models.NamespacedID{
				ID:        sub.ID,
				Namespace: sub.Namespace,
			})
			require.Nil(t, err)

			// Let's add items to the last phase

			sub := subView.Sub()
			phases := subView.Phases()
			phaseKey := phases[len(phases)-1].Key()

			// Let's create a new featue so we can add it to the Subscription
			feat, err := deps.FeatureConnector.CreateFeature(ctx, feature.CreateFeatureInputs{
				Name:      "New Feature",
				Key:       "new-feature",
				Namespace: subscriptiontestutils.ExampleNamespace,
				MeterSlug: &subscriptiontestutils.ExampleFeatureMeterSlug,
			})
			require.Nil(t, err)

			_, err = command.Edit(ctx, models.NamespacedID{
				ID:        sub.ID,
				Namespace: sub.Namespace,
			}, []subscription.Patch{
				// Let's add a new Item with a Price
				subscription.PatchAddItem{
					PhaseKey: phaseKey,
					ItemKey:  "new-item-1",
					CreateInput: subscription.SubscriptionItemSpec{
						CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
							PhaseKey: phaseKey,
							ItemKey:  "new-item-1",
							CreatePriceInput: &subscription.CreatePriceInput{
								PhaseKey: phaseKey,
								ItemKey:  "new-item-1",
								Value:    "100",
								Key:      "new-item-1",
							},
						},
					},
				},
				// Let's add a new Item with an Entitlement
				subscription.PatchAddItem{
					PhaseKey: phaseKey,
					ItemKey:  "new-item-2",
					CreateInput: subscription.SubscriptionItemSpec{
						CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
							PhaseKey:   phaseKey,
							ItemKey:    "new-item-2",
							FeatureKey: &feat.Key,
							CreateEntitlementInput: &subscription.CreateSubscriptionEntitlementInput{
								EntitlementType:        entitlement.EntitlementTypeMetered,
								IssueAfterReset:        lo.ToPtr(100.0),
								UsagePeriodISODuration: &oneMonthISO,
							},
						},
					},
				},
			})
			require.Nil(t, err)

			// Let's assert that the Subscription now has the new Items
			subView, err = query.Expand(ctx, models.NamespacedID{
				ID:        sub.ID,
				Namespace: sub.Namespace,
			})
			require.Nil(t, err)

			lastPhase := subView.Phases()[len(phases)-1]

			items := lastPhase.Items()
			assert.True(t, lo.SomeBy(items, func(i subscription.SubscriptionItemView) bool {
				if i.Key() == "new-item-1" {
					p, has := i.Price()
					assert.True(t, has)
					assert.Equal(t, "100", p.Value)
					assert.Equal(t, lastPhase.ActiveFrom(), p.ActiveFrom)
					assert.Nil(t, p.ActiveTo)
					return true
				}
				return false
			}))
			assert.True(t, lo.SomeBy(items, func(i subscription.SubscriptionItemView) bool {
				if i.Key() == "new-item-2" {
					e, has := i.Entitlement()
					assert.True(t, has)
					assert.Equal(t, entitlement.EntitlementTypeMetered, e.Entitlement.EntitlementType)
					assert.Equal(t, lo.ToPtr(100.0), e.Entitlement.IssueAfterReset)
					assert.Equal(t, lastPhase.ActiveFrom(), e.Cadence.ActiveFrom)
					assert.Nil(t, e.Cadence.ActiveTo)
					return true
				}
				return false
			}))
		})

		t.Run("Should add new empty phase to end of subscription", func(t *testing.T) {
			// Let's assert that the base Subscription looks as we believe
			subView, err := query.Expand(ctx, models.NamespacedID{
				ID:        sub.ID,
				Namespace: sub.Namespace,
			})
			require.Nil(t, err)

			sub := subView.Sub()
			phases := subView.Phases()
			oldPhaseCount := len(phases)

			lastPhase := phases[oldPhaseCount-1]

			newPhaseKey := "new-last-phase"
			newPhaseStartAfter, err := lastPhase.AsSpec().CreateSubscriptionPhaseInput.StartAfter.Add(datex.MustParse(t, "P1M"))
			require.Nil(t, err)

			_, err = command.Edit(ctx, models.NamespacedID{ID: sub.ID, Namespace: sub.Namespace}, []subscription.Patch{
				subscription.PatchAddPhase{
					PhaseKey: newPhaseKey,
					CreateInput: subscription.CreateSubscriptionPhaseInput{
						CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
							PhaseKey:   newPhaseKey,
							StartAfter: newPhaseStartAfter,
						},
						CreateSubscriptionPhaseCustomerInput: subscription.CreateSubscriptionPhaseCustomerInput{
							CreateDiscountInput: nil, // TODO: implement
						},
						Duration: datex.MustParse(t, "P3M"),
					},
				},
			})
			require.Nil(t, err)

			// Let's re-fetch the subscription
			// Let's assert that the Subscription now has the new Items
			subView, err = query.Expand(ctx, models.NamespacedID{
				ID:        sub.ID,
				Namespace: sub.Namespace,
			})
			require.Nil(t, err)

			require.Equal(t, oldPhaseCount+1, len(subView.Phases()))

			newLastPhase := subView.Phases()[oldPhaseCount]
			currentOldLastPhase := subView.Phases()[oldPhaseCount-1]

			require.Equal(t, lastPhase.Key(), subView.Phases()[oldPhaseCount-1].Key())

			expectedActiveFrom, _ := newPhaseStartAfter.AddTo(sub.ActiveFrom)

			// Should add new phase
			assert.Equal(t, newPhaseKey, newLastPhase.Key())
			assert.Equal(t, expectedActiveFrom, newLastPhase.ActiveFrom())

			// Should ignore duration when adding last phase
			// TODO: validate it gets ignored

			// Should close entitlement and price of previous phase
			for _, item := range currentOldLastPhase.Items() {
				if ent, ok := item.Entitlement(); ok {
					assert.NotNil(t, ent.Cadence.ActiveTo)
					assert.Equal(t, lo.ToPtr(newLastPhase.ActiveFrom()), ent.Cadence.ActiveTo)
				}
				if price, ok := item.Price(); ok {
					assert.NotNil(t, price.ActiveTo)
					assert.Equal(t, lo.ToPtr(newLastPhase.ActiveFrom()), price.ActiveTo)
				}
			}
		})

		t.Run("Should add new phase with item and delay subsequent phases in subscription", func(t *testing.T) {
			// Let's assert that the base Subscription looks as we believe
			subView, err := query.Expand(ctx, models.NamespacedID{
				ID:        sub.ID,
				Namespace: sub.Namespace,
			})
			require.Nil(t, err)

			sub := subView.Sub()
			phases := subView.Phases()
			oldPhaseCount := len(phases)

			require.Equal(t, 4, oldPhaseCount) // 3 original in ExamplePlan + 1 new phase added above

			expected2ndPhaseStart, _ := datex.MustParse(t, "P2M").AddTo(sub.ActiveFrom)
			expected3rdPhaseStart, _ := datex.MustParse(t, "P6M").AddTo(sub.ActiveFrom)

			// Lets assert these two phases start when we believe they do
			require.Equal(t, expected2ndPhaseStart, phases[1].ActiveFrom())
			require.Equal(t, expected3rdPhaseStart, phases[2].ActiveFrom())

			// Let's add a new phase in between them
			newPhaseKey := "in-between-phase"
			newPhaseStartAfter := datex.MustParse(t, "P4M")
			newPhaseDuration := datex.MustParse(t, "P3M")

			_, err = command.Edit(ctx, models.NamespacedID{ID: sub.ID, Namespace: sub.Namespace}, []subscription.Patch{
				subscription.PatchAddPhase{
					PhaseKey: newPhaseKey,
					CreateInput: subscription.CreateSubscriptionPhaseInput{
						CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
							PhaseKey:   newPhaseKey,
							StartAfter: newPhaseStartAfter,
						},
						CreateSubscriptionPhaseCustomerInput: subscription.CreateSubscriptionPhaseCustomerInput{
							CreateDiscountInput: nil, // TODO: implement
						},
						Duration: newPhaseDuration,
					},
				},
				subscription.PatchAddItem{
					PhaseKey: newPhaseKey,
					ItemKey:  "new-item-1",
					CreateInput: subscription.SubscriptionItemSpec{
						CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
							PhaseKey: newPhaseKey,
							ItemKey:  "new-item-1",
							CreatePriceInput: &subscription.CreatePriceInput{
								PhaseKey: newPhaseKey,
								ItemKey:  "new-item-1",
								Value:    "100",
								Key:      "new-item-1",
							},
						},
					},
				},
			})
			require.Nil(t, err)
			oldPhases := phases

			// Let's re-fetch the subscription
			subView, err = query.Expand(ctx, models.NamespacedID{
				ID:        sub.ID,
				Namespace: sub.Namespace,
			})
			require.Nil(t, err)

			// Lets assert that the phase starts when specified
			newPhase := subView.Phases()[2]
			require.Equal(t, newPhaseKey, newPhase.Key())
			expectedActiveFrom, _ := newPhaseStartAfter.AddTo(sub.ActiveFrom)
			require.Equal(t, expectedActiveFrom, newPhase.ActiveFrom())

			// Let's assert that it has a new Item with a price that starts and ends when expected
			items := newPhase.Items()
			assert.Len(t, items, 1)
			assert.True(t, lo.SomeBy(items, func(i subscription.SubscriptionItemView) bool {
				if i.Key() == "new-item-1" {
					p, has := i.Price()
					assert.True(t, has)
					assert.Equal(t, "100", p.Value)
					assert.Equal(t, newPhase.ActiveFrom(), p.ActiveFrom)

					aTo, _ := newPhaseDuration.AddTo(newPhase.ActiveFrom())

					assert.Equal(t, lo.ToPtr(aTo), p.ActiveTo)
					return true
				}
				return false
			}))

			// Lets assert that the next and all subsequent phases were delayed by on month
			// 1 month, because it used to start after 6 months, no it starts after 4 + 3 = 7 months
			for i, phase := range subView.Phases() {
				if i < 3 {
					continue
				}

				var opv subscription.SubscriptionPhaseView
				for _, op := range oldPhases {
					if op.Key() == phase.Key() {
						opv = op
						break
					}
				}
				require.NotNil(t, opv)

				expectedStart, _ := datex.MustParse(t, "P1M").AddTo(opv.ActiveFrom())
				assert.Equal(t, expectedStart, phase.ActiveFrom())
			}

			// Instead of checking that each item has been drifted, we can just validate the view that everything aligns
			err = subView.Validate(true)
			require.Nil(t, err)
		})

		t.Run("Should remove phase and all items in it", func(t *testing.T) {
			// Let's assert that the base Subscription looks as we believe
			subView, err := query.Expand(ctx, models.NamespacedID{
				ID:        sub.ID,
				Namespace: sub.Namespace,
			})
			require.Nil(t, err)

			sub := subView.Sub()
			phases := subView.Phases()
			oldPhaseCount := len(phases)

			require.Equal(t, 5, oldPhaseCount) // 3 original in ExamplePlan + 2 new phase added above
			phaseToDelete := phases[2]
			oldPrevPhase := phases[1]
			oldNextPhase := phases[3]
			require.Equal(t, "in-between-phase", phaseToDelete.Key())
			require.Equal(t, 1, len(phaseToDelete.Items()))

			// Let's delete the phase we added above
			_, err = command.Edit(ctx, models.NamespacedID{ID: sub.ID, Namespace: sub.Namespace}, []subscription.Patch{
				subscription.PatchRemovePhase{
					PhaseKey: phaseToDelete.Key(),
					RemoveInput: subscription.RemoveSubscriptionPhaseInput{
						Shift: subscription.RemoveSubscriptionPhaseShiftPrev,
					},
				},
			})
			require.Nil(t, err)

			// Let's re-fetch the subscription
			subView, err = query.Expand(ctx, models.NamespacedID{
				ID:        sub.ID,
				Namespace: sub.Namespace,
			})
			require.Nil(t, err)

			// Lets assert that the phase is gone
			require.Equal(t, oldPhaseCount-1, len(subView.Phases()))

			// Let's assert that the prev phase was extended to last until the end of the deleted phase
			prevPhase := subView.Phases()[1]
			assert.Equal(t, oldPrevPhase.Key(), prevPhase.Key())
			assert.Equal(t, oldPrevPhase.ActiveFrom(), prevPhase.ActiveFrom())
			nextPhase := subView.Phases()[2]
			assert.Equal(t, oldNextPhase.Key(), nextPhase.Key())
			assert.Equal(t, oldNextPhase.ActiveFrom(), nextPhase.ActiveFrom())

			// Let's assert that the items of the prev phase were extended
			for _, item := range prevPhase.Items() {
				if ent, ok := item.Entitlement(); ok {
					assert.Equal(t, lo.ToPtr(nextPhase.ActiveFrom()), ent.Cadence.ActiveTo)
				}
				if price, ok := item.Price(); ok {
					assert.Equal(t, lo.ToPtr(nextPhase.ActiveFrom()), price.ActiveTo)
				}
			}
		})

		t.Run("Let's remove last phase", func(t *testing.T) {
			// Let's assert that the base Subscription looks as we believe
			subView, err := query.Expand(ctx, models.NamespacedID{
				ID:        sub.ID,
				Namespace: sub.Namespace,
			})
			require.Nil(t, err)

			sub := subView.Sub()
			phases := subView.Phases()
			oldPhaseCount := len(phases)

			require.Equal(t, 4, oldPhaseCount) // 3 original in ExamplePlan + 1 new phase added above
			require.Equal(t, "new-last-phase", phases[3].Key())

			// Let's delete the last phase
			_, err = command.Edit(ctx, models.NamespacedID{ID: sub.ID, Namespace: sub.Namespace}, []subscription.Patch{
				subscription.PatchRemovePhase{
					PhaseKey: "new-last-phase",
					RemoveInput: subscription.RemoveSubscriptionPhaseInput{
						Shift: subscription.RemoveSubscriptionPhaseShiftPrev,
					},
				},
			})
			require.Nil(t, err)

			// Let's re-fetch the subscription
			subView, err = query.Expand(ctx, models.NamespacedID{
				ID:        sub.ID,
				Namespace: sub.Namespace,
			})
			require.Nil(t, err)

			// Lets assert that the phase is gone
			require.Equal(t, oldPhaseCount-1, len(subView.Phases()))

			// Let's assert that the items of the prev phase were extended
			prevPhase := subView.Phases()[2]
			for _, item := range prevPhase.Items() {
				if ent, ok := item.Entitlement(); ok {
					assert.Nil(t, ent.Cadence.ActiveTo)
				}
				if price, ok := item.Price(); ok {
					assert.Nil(t, price.ActiveTo)
				}
			}
		})
	})
}
