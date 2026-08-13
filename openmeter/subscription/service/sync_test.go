package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestEdit(t *testing.T) {
	type TDeps struct {
		CurrentTime time.Time
		Customer    customer.Customer
		ExamplePlan subscription.Plan
		ServiceDeps subscriptiontestutils.SubscriptionDependencies
		Service     subscription.Service
	}

	tt := []struct {
		Name    string
		Handler func(t *testing.T, deps TDeps)
	}{
		{
			Name: "Should do nothing if no changes",
			Handler: func(t *testing.T, deps TDeps) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				spec, err := subscription.NewSpecFromPlan(deps.ExamplePlan, subscription.CreateSubscriptionCustomerInput{
					Name:            "Test",
					CustomerId:      deps.Customer.ID,
					InvoiceCurrency: currencyx.Code("USD"),
					ActiveFrom:      deps.CurrentTime,
					BillingAnchor:   deps.CurrentTime,
				})
				require.Nil(t, err)

				sub, err := deps.Service.Create(ctx, deps.Customer.Namespace, spec)
				require.Nil(t, err)

				before, err := deps.Service.GetView(ctx, sub.NamespacedID)
				require.NoError(t, err)

				_, err = deps.Service.Update(ctx, sub.NamespacedID, spec)
				require.Nil(t, err)

				after, err := deps.Service.GetView(ctx, sub.NamespacedID)
				require.NoError(t, err)
				require.Len(t, after.Phases, len(before.Phases))

				for phaseIdx, beforePhase := range before.Phases {
					afterPhase := after.Phases[phaseIdx]
					require.Equal(t, beforePhase.SubscriptionPhase.ID, afterPhase.SubscriptionPhase.ID)

					for itemKey, beforeItems := range beforePhase.ItemsByKey {
						afterItems := afterPhase.ItemsByKey[itemKey]
						require.Len(t, afterItems, len(beforeItems))

						for itemIdx, beforeItem := range beforeItems {
							afterItem := afterItems[itemIdx]
							require.Equal(t, beforeItem.SubscriptionItem.ID, afterItem.SubscriptionItem.ID)

							if beforeItem.Entitlement == nil {
								require.Nil(t, afterItem.Entitlement)
								continue
							}

							require.NotNil(t, afterItem.Entitlement)
							require.Equal(t, beforeItem.Entitlement.Entitlement.ID, afterItem.Entitlement.Entitlement.ID)
						}
					}
				}
			},
		},
		{
			Name: "Should error if plan changes",
			Handler: func(t *testing.T, deps TDeps) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				spec, err := subscription.NewSpecFromPlan(deps.ExamplePlan, subscription.CreateSubscriptionCustomerInput{
					Name:            "Test",
					CustomerId:      deps.Customer.ID,
					InvoiceCurrency: currencyx.Code("USD"),
					ActiveFrom:      deps.CurrentTime,
					BillingAnchor:   deps.CurrentTime,
				})
				require.Nil(t, err)

				sub, err := deps.Service.Create(ctx, deps.Customer.Namespace, spec)
				require.Nil(t, err)

				require.NotNil(t, spec.Plan)
				// Let's remove the plan reference
				spec.Plan = nil

				_, err = deps.Service.Update(ctx, sub.NamespacedID, spec)
				assert.Error(t, err)
				assert.ErrorContains(t, err, "cannot change plan")
			},
		},
		{
			Name: "Should error if customer changes",
			Handler: func(t *testing.T, deps TDeps) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				spec, err := subscription.NewSpecFromPlan(deps.ExamplePlan, subscription.CreateSubscriptionCustomerInput{
					Name:            "Test",
					CustomerId:      deps.Customer.ID,
					InvoiceCurrency: currencyx.Code("USD"),
					ActiveFrom:      deps.CurrentTime,
					BillingAnchor:   deps.CurrentTime,
				})
				require.Nil(t, err)

				sub, err := deps.Service.Create(ctx, deps.Customer.Namespace, spec)
				require.Nil(t, err)

				// Let's change the customer reference
				spec.CustomerId = "new-customer-id"

				_, err = deps.Service.Update(ctx, sub.NamespacedID, spec)
				assert.Error(t, err)
				assert.ErrorContains(t, err, "cannot change customer")
			},
		},
		{
			Name: "Should error if subscription start changes",
			Handler: func(t *testing.T, deps TDeps) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				spec, err := subscription.NewSpecFromPlan(deps.ExamplePlan, subscription.CreateSubscriptionCustomerInput{
					Name:            "Test",
					CustomerId:      deps.Customer.ID,
					InvoiceCurrency: currencyx.Code("USD"),
					ActiveFrom:      deps.CurrentTime,
					BillingAnchor:   deps.CurrentTime,
				})
				require.Nil(t, err)

				sub, err := deps.Service.Create(ctx, deps.Customer.Namespace, spec)
				require.Nil(t, err)

				// Let's change the start time of the subscription
				spec.ActiveFrom = spec.ActiveFrom.Add(time.Hour)

				_, err = deps.Service.Update(ctx, sub.NamespacedID, spec)
				assert.Error(t, err)
				assert.ErrorContains(t, err, "cannot change subscription start")
			},
		},
		{
			Name: "Should error if settlement mode changes",
			Handler: func(t *testing.T, deps TDeps) {
				ctx := t.Context()

				spec, err := subscription.NewSpecFromPlan(deps.ExamplePlan, subscription.CreateSubscriptionCustomerInput{
					Name:            "Test",
					CustomerId:      deps.Customer.ID,
					InvoiceCurrency: currencyx.Code("USD"),
					ActiveFrom:      deps.CurrentTime,
					BillingAnchor:   deps.CurrentTime,
				})
				require.NoError(t, err)

				sub, err := deps.Service.Create(ctx, deps.Customer.Namespace, spec)
				require.NoError(t, err)

				spec.SettlementMode = productcatalog.CreditOnlySettlementMode

				_, err = deps.Service.Update(ctx, sub.NamespacedID, spec)
				require.ErrorContains(t, err, "cannot change settlement mode")
			},
		},
		{
			Name: "Should update contents of future phases when phase start changes",
			Handler: func(t *testing.T, deps TDeps) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				spec, err := subscription.NewSpecFromPlan(deps.ExamplePlan, subscription.CreateSubscriptionCustomerInput{
					Name:            "Test",
					CustomerId:      deps.Customer.ID,
					InvoiceCurrency: currencyx.Code("USD"),
					ActiveFrom:      deps.CurrentTime,
					BillingAnchor:   deps.CurrentTime,
				})
				require.Nil(t, err)

				sub, err := deps.Service.Create(ctx, deps.Customer.Namespace, spec)
				require.Nil(t, err)

				_, err = deps.Service.GetView(ctx, sub.NamespacedID)
				require.Nil(t, err)

				// Let's change the start time of the last phase so phase contents must change for both the last and one before last phase
				require.Equal(t, 3, len(spec.Phases))

				pKey := "test_phase_3"
				_, ok := spec.Phases[pKey]
				require.True(t, ok)

				// Let's make sure that the phase we're changing is actually in the future
				st, _ := spec.Phases[pKey].StartAfter.AddTo(sub.ActiveFrom)
				require.True(t, st.After(clock.Now()))

				// Let's make it start one month later
				spec.Phases[pKey].StartAfter, err = spec.Phases[pKey].StartAfter.Add(datetime.MustParseDuration(t, "P1M"))
				require.Nil(t, err)

				_, err = deps.Service.Update(ctx, sub.NamespacedID, spec)
				require.Nil(t, err)

				v2, err := deps.Service.GetView(ctx, sub.NamespacedID)
				require.Nil(t, err)

				// Let's validate the update
				subscriptiontestutils.ValidateSpecAndView(t, spec, v2)
			},
		},
		{
			Name: "Should delete item from future phase",
			Handler: func(t *testing.T, deps TDeps) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				spec, err := subscription.NewSpecFromPlan(deps.ExamplePlan, subscription.CreateSubscriptionCustomerInput{
					Name:            "Test",
					CustomerId:      deps.Customer.ID,
					InvoiceCurrency: currencyx.Code("USD"),
					ActiveFrom:      deps.CurrentTime,
					BillingAnchor:   deps.CurrentTime,
				})
				require.Nil(t, err)

				sub, err := deps.Service.Create(ctx, deps.Customer.Namespace, spec)
				require.Nil(t, err)

				_, err = deps.Service.GetView(ctx, sub.NamespacedID)
				require.Nil(t, err)

				// Let's delete an item from the 2nd phase
				require.Equal(t, 3, len(spec.Phases))

				pKey := "test_phase_2"
				_, ok := spec.Phases[pKey]
				require.True(t, ok)

				// Let's make sure it has the item we want to delete
				iKey := subscriptiontestutils.ExampleFeatureKey

				v, ok := spec.Phases[pKey].ItemsByKey[iKey]
				require.True(t, ok)
				require.Greater(t, len(v), 0)

				// Let's delete the item
				delete(spec.Phases[pKey].ItemsByKey, iKey)

				_, err = deps.Service.Update(ctx, sub.NamespacedID, spec)
				require.Nil(t, err)

				v2, err := deps.Service.GetView(ctx, sub.NamespacedID)
				require.Nil(t, err)

				// Let's validate the update
				subscriptiontestutils.ValidateSpecAndView(t, spec, v2)
			},
		},
		{
			Name: "Should add item to future phase",
			Handler: func(t *testing.T, deps TDeps) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				spec, err := subscription.NewSpecFromPlan(deps.ExamplePlan, subscription.CreateSubscriptionCustomerInput{
					Name:            "Test",
					CustomerId:      deps.Customer.ID,
					InvoiceCurrency: currencyx.Code("USD"),
					ActiveFrom:      deps.CurrentTime,
					BillingAnchor:   deps.CurrentTime,
				})
				require.Nil(t, err)

				sub, err := deps.Service.Create(ctx, deps.Customer.Namespace, spec)
				require.Nil(t, err)

				_, err = deps.Service.GetView(ctx, sub.NamespacedID)
				require.Nil(t, err)

				// Let's add an extra item to the last phase
				require.Equal(t, 3, len(spec.Phases))

				pKey := "test_phase_3"
				_, ok := spec.Phases[pKey]
				require.True(t, ok)

				// Let's make sure it doesn't have the item we want to add
				iKey := subscriptiontestutils.ExampleRateCard2.Key()

				_, ok = spec.Phases[pKey].ItemsByKey[iKey]
				require.False(t, ok)

				rc := subscriptiontestutils.ExampleRateCard2

				// Let's add the item
				spec.Phases[pKey].ItemsByKey[iKey] = []*subscription.SubscriptionItemSpec{
					{
						CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
							CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
								PhaseKey: pKey,
								ItemKey:  iKey,
								RateCard: &rc,
							},
							CreateSubscriptionItemCustomerInput: subscription.CreateSubscriptionItemCustomerInput{},
						},
					},
				}

				_, err = deps.Service.Update(ctx, sub.NamespacedID, spec)
				require.Nil(t, err)

				v2, err := deps.Service.GetView(ctx, sub.NamespacedID)
				require.Nil(t, err)

				// Let's validate the update
				subscriptiontestutils.ValidateSpecAndView(t, spec, v2)
			},
		},
		{
			Name: "Should update item entitlement",
			Handler: func(t *testing.T, deps TDeps) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				spec, err := subscription.NewSpecFromPlan(deps.ExamplePlan, subscription.CreateSubscriptionCustomerInput{
					Name:            "Test",
					CustomerId:      deps.Customer.ID,
					InvoiceCurrency: currencyx.Code("USD"),
					ActiveFrom:      deps.CurrentTime,
					BillingAnchor:   deps.CurrentTime,
				})
				require.Nil(t, err)

				sub, err := deps.Service.Create(ctx, deps.Customer.Namespace, spec)
				require.Nil(t, err)

				v1, err := deps.Service.GetView(ctx, sub.NamespacedID)
				require.Nil(t, err)
				require.NotEmpty(t, v1.Subscription.ID)

				// Let's validate we have an item with an entitlement template
				require.Equal(t, 3, len(spec.Phases))

				pKey := "test_phase_1"
				_, ok := spec.Phases[pKey]
				require.True(t, ok)

				// Let's make sure it has the item we want to change
				iKey := subscriptiontestutils.ExampleRateCard1.Key()

				v, ok := spec.Phases[pKey].ItemsByKey[iKey]
				require.True(t, ok)
				require.Greater(t, len(v), 0)

				item := v[0]

				// Let's unset the entitlement template
				require.NoError(t, item.RateCard.ChangeMeta(func(m productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
					m.EntitlementTemplate = nil
					return m, nil
				}))

				// Let's add the item
				spec.Phases[pKey].ItemsByKey[iKey] = []*subscription.SubscriptionItemSpec{
					item,
				}

				u, err := deps.Service.Update(ctx, sub.NamespacedID, spec)
				require.Nil(t, err)
				require.NotEmpty(t, u.ID)

				v2, err := deps.Service.GetView(ctx, sub.NamespacedID)
				require.Nil(t, err)

				// Let's validate the update
				subscriptiontestutils.ValidateSpecAndView(t, spec, v2)
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z")
			clock.SetTime(currentTime)

			dbDeps := subscriptiontestutils.SetupDBDeps(t)
			defer dbDeps.Cleanup(t)

			deps := subscriptiontestutils.NewService(t, dbDeps)
			service := deps.SubscriptionService

			cust := deps.CustomerAdapter.CreateExampleCustomer(t)
			require.NotNil(t, cust)

			_ = deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)
			plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

			tc.Handler(t, TDeps{
				CurrentTime: currentTime,
				Customer:    *cust,
				ExamplePlan: plan,
				ServiceDeps: deps,
				Service:     service,
			})
		})
	}
}

func TestSyncReconciliation(t *testing.T) {
	const (
		firstPhaseKey  = "test_phase_1"
		secondPhaseKey = "test_phase_2"
		thirdPhaseKey  = "test_phase_3"
		addedPhaseKey  = "added_phase"
	)

	type testCase struct {
		name    string
		prepare func(t *testing.T, spec *subscription.SubscriptionSpec)
		mutate  func(t *testing.T, spec *subscription.SubscriptionSpec)
		assert  func(t *testing.T, before, after subscription.SubscriptionView)
	}

	tests := []testCase{
		{
			name: "changing a phase persists its properties and child links",
			mutate: func(t *testing.T, spec *subscription.SubscriptionSpec) {
				spec.Phases[firstPhaseKey].Name = "Changed phase"
			},
			assert: func(t *testing.T, before, after subscription.SubscriptionView) {
				beforePhase := requireSyncPhase(t, before, firstPhaseKey)
				afterPhase := requireSyncPhase(t, after, firstPhaseKey)

				require.Equal(t, "Changed phase", afterPhase.SubscriptionPhase.Name)
				require.Len(t, afterPhase.ItemsByKey, len(beforePhase.ItemsByKey))
				for _, items := range afterPhase.ItemsByKey {
					for _, item := range items {
						require.Equal(t, afterPhase.SubscriptionPhase.ID, item.SubscriptionItem.PhaseId)
					}
				}
			},
		},
		{
			name: "removing a phase deletes it and reconciles the preceding phase cadence",
			mutate: func(t *testing.T, spec *subscription.SubscriptionSpec) {
				delete(spec.Phases, thirdPhaseKey)
			},
			assert: func(t *testing.T, before, after subscription.SubscriptionView) {
				require.Len(t, after.Phases, len(before.Phases)-1)
				_, found := lo.Find(after.Phases, func(phase subscription.SubscriptionPhaseView) bool {
					return phase.SubscriptionPhase.Key == thirdPhaseKey
				})
				require.False(t, found)

				phase := requireSyncPhase(t, after, secondPhaseKey)
				cadence, err := after.Spec.GetPhaseCadence(secondPhaseKey)
				require.NoError(t, err)
				require.Nil(t, cadence.ActiveTo)
				for _, items := range phase.ItemsByKey {
					for _, item := range items {
						require.Nil(t, item.SubscriptionItem.ActiveTo)
						if item.Entitlement != nil {
							require.Nil(t, item.Entitlement.Cadence.ActiveTo)
						}
					}
				}
			},
		},
		{
			name: "adding a phase creates it and reconciles the preceding phase cadence",
			mutate: func(t *testing.T, spec *subscription.SubscriptionSpec) {
				rateCard := subscriptiontestutils.ExampleRateCard2.Clone()
				itemKey := rateCard.Key()
				spec.Phases[addedPhaseKey] = &subscription.SubscriptionPhaseSpec{
					CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
						PhaseKey:   addedPhaseKey,
						StartAfter: datetime.MustParseDuration(t, "P4M"),
						Name:       "Added phase",
					},
					ItemsByKey: map[string][]*subscription.SubscriptionItemSpec{
						itemKey: {
							{
								CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
									CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
										PhaseKey: addedPhaseKey,
										ItemKey:  itemKey,
										RateCard: rateCard,
									},
								},
							},
						},
					},
				}
			},
			assert: func(t *testing.T, before, after subscription.SubscriptionView) {
				require.Len(t, after.Phases, len(before.Phases)+1)
				addedPhase := requireSyncPhase(t, after, addedPhaseKey)
				require.Equal(t, "Added phase", addedPhase.SubscriptionPhase.Name)
				require.Len(t, requireSyncItems(t, addedPhase, subscriptiontestutils.ExampleRateCard2.Key()), 1)

				precedingPhase := requireSyncPhase(t, after, thirdPhaseKey)
				precedingCadence, err := after.Spec.GetPhaseCadence(thirdPhaseKey)
				require.NoError(t, err)
				require.NotNil(t, precedingCadence.ActiveTo)
				require.True(t, precedingCadence.ActiveTo.Equal(addedPhase.SubscriptionPhase.ActiveFrom))
				for _, items := range precedingPhase.ItemsByKey {
					for _, item := range items {
						require.NotNil(t, item.SubscriptionItem.ActiveTo)
						require.True(t, item.SubscriptionItem.ActiveTo.Equal(addedPhase.SubscriptionPhase.ActiveFrom))
						if item.Entitlement != nil {
							require.NotNil(t, item.Entitlement.Cadence.ActiveTo)
							require.True(t, item.Entitlement.Cadence.ActiveTo.Equal(addedPhase.SubscriptionPhase.ActiveFrom))
						}
					}
				}
			},
		},
		{
			name: "removing an item key deletes every version and keeps sibling items",
			prepare: func(t *testing.T, spec *subscription.SubscriptionSpec) {
				rateCard := subscriptiontestutils.ExampleRateCard2.Clone()
				itemKey := rateCard.Key()
				spec.Phases[firstPhaseKey].ItemsByKey[itemKey] = []*subscription.SubscriptionItemSpec{{
					CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
						CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
							PhaseKey: firstPhaseKey,
							ItemKey:  itemKey,
							RateCard: rateCard,
						},
					},
				}}
			},
			mutate: func(t *testing.T, spec *subscription.SubscriptionSpec) {
				delete(spec.Phases[firstPhaseKey].ItemsByKey, subscriptiontestutils.ExampleFeatureKey)
			},
			assert: func(t *testing.T, before, after subscription.SubscriptionView) {
				beforePhase := requireSyncPhase(t, before, firstPhaseKey)
				afterPhase := requireSyncPhase(t, after, firstPhaseKey)
				require.Len(t, beforePhase.ItemsByKey, 2)
				require.Len(t, afterPhase.ItemsByKey, 1)

				_, found := afterPhase.ItemsByKey[subscriptiontestutils.ExampleFeatureKey]
				require.False(t, found)
				require.Len(t, requireSyncItems(t, afterPhase, subscriptiontestutils.ExampleRateCard2.Key()), 1)
			},
		},
		{
			name: "adding an item key creates it and keeps existing items",
			mutate: func(t *testing.T, spec *subscription.SubscriptionSpec) {
				rateCard := subscriptiontestutils.ExampleRateCard2.Clone()
				itemKey := rateCard.Key()
				spec.Phases[firstPhaseKey].ItemsByKey[itemKey] = []*subscription.SubscriptionItemSpec{{
					CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
						CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
							PhaseKey: firstPhaseKey,
							ItemKey:  itemKey,
							RateCard: rateCard,
						},
					},
				}}
			},
			assert: func(t *testing.T, before, after subscription.SubscriptionView) {
				beforePhase := requireSyncPhase(t, before, firstPhaseKey)
				afterPhase := requireSyncPhase(t, after, firstPhaseKey)
				require.Len(t, beforePhase.ItemsByKey, 1)
				require.Len(t, afterPhase.ItemsByKey, 2)
				require.Len(t, requireSyncItems(t, afterPhase, subscriptiontestutils.ExampleFeatureKey), 1)

				addedItems := requireSyncItems(t, afterPhase, subscriptiontestutils.ExampleRateCard2.Key())
				require.Len(t, addedItems, 1)
				require.Nil(t, addedItems[0].Entitlement)
			},
		},
		{
			name: "adding a trailing item version creates contiguous item and entitlement cadences",
			prepare: func(t *testing.T, spec *subscription.SubscriptionSpec) {
				activeTo := datetime.MustParseDuration(t, "P15D")
				spec.Phases[firstPhaseKey].ItemsByKey[subscriptiontestutils.ExampleFeatureKey][0].ActiveToOverrideRelativeToPhaseStart = &activeTo
			},
			mutate: func(t *testing.T, spec *subscription.SubscriptionSpec) {
				items := spec.Phases[firstPhaseKey].ItemsByKey[subscriptiontestutils.ExampleFeatureKey]
				activeFrom := datetime.MustParseDuration(t, "P15D")
				secondVersion := *items[0]
				secondVersion.RateCard = items[0].RateCard.Clone()
				secondVersion.ActiveFromOverrideRelativeToPhaseStart = &activeFrom
				secondVersion.ActiveToOverrideRelativeToPhaseStart = nil
				spec.Phases[firstPhaseKey].ItemsByKey[subscriptiontestutils.ExampleFeatureKey] = append(items, &secondVersion)
			},
			assert: func(t *testing.T, before, after subscription.SubscriptionView) {
				beforeItems := requireSyncItems(t, requireSyncPhase(t, before, firstPhaseKey), subscriptiontestutils.ExampleFeatureKey)
				afterItems := requireSyncItems(t, requireSyncPhase(t, after, firstPhaseKey), subscriptiontestutils.ExampleFeatureKey)
				require.Len(t, beforeItems, 1)
				require.Len(t, afterItems, 2)

				require.NotNil(t, afterItems[0].SubscriptionItem.ActiveTo)
				require.Equal(t, *afterItems[0].SubscriptionItem.ActiveTo, afterItems[1].SubscriptionItem.ActiveFrom)
				require.NotNil(t, afterItems[0].Entitlement)
				require.NotNil(t, afterItems[1].Entitlement)
				require.NotNil(t, afterItems[0].Entitlement.Cadence.ActiveTo)
				require.Equal(t, *afterItems[0].Entitlement.Cadence.ActiveTo, afterItems[1].Entitlement.Cadence.ActiveFrom)
			},
		},
		{
			name: "removing a trailing item version leaves only the earlier cadence",
			prepare: func(t *testing.T, spec *subscription.SubscriptionSpec) {
				items := spec.Phases[firstPhaseKey].ItemsByKey[subscriptiontestutils.ExampleFeatureKey]
				boundary := datetime.MustParseDuration(t, "P15D")
				items[0].ActiveToOverrideRelativeToPhaseStart = &boundary

				secondVersion := *items[0]
				secondVersion.RateCard = items[0].RateCard.Clone()
				secondVersion.ActiveFromOverrideRelativeToPhaseStart = &boundary
				secondVersion.ActiveToOverrideRelativeToPhaseStart = nil
				spec.Phases[firstPhaseKey].ItemsByKey[subscriptiontestutils.ExampleFeatureKey] = append(items, &secondVersion)
			},
			mutate: func(t *testing.T, spec *subscription.SubscriptionSpec) {
				items := spec.Phases[firstPhaseKey].ItemsByKey[subscriptiontestutils.ExampleFeatureKey]
				spec.Phases[firstPhaseKey].ItemsByKey[subscriptiontestutils.ExampleFeatureKey] = items[:1]
			},
			assert: func(t *testing.T, before, after subscription.SubscriptionView) {
				beforeItems := requireSyncItems(t, requireSyncPhase(t, before, firstPhaseKey), subscriptiontestutils.ExampleFeatureKey)
				afterItems := requireSyncItems(t, requireSyncPhase(t, after, firstPhaseKey), subscriptiontestutils.ExampleFeatureKey)
				require.Len(t, beforeItems, 2)
				require.Len(t, afterItems, 1)
				require.NotNil(t, afterItems[0].SubscriptionItem.ActiveTo)
				require.NotNil(t, afterItems[0].Entitlement)
				require.Equal(t, afterItems[0].SubscriptionItem.ActiveTo, afterItems[0].Entitlement.Cadence.ActiveTo)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a fully materialized subscription built from the example plan.
			currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z")
			clock.FreezeTime(currentTime)
			defer clock.UnFreeze()

			dbDeps := subscriptiontestutils.SetupDBDeps(t)
			defer dbDeps.Cleanup(t)

			deps := subscriptiontestutils.NewService(t, dbDeps)
			cust := deps.CustomerAdapter.CreateExampleCustomer(t)
			require.NotNil(t, cust)
			_ = deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)
			plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

			spec, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
				Name:            "Test",
				CustomerId:      cust.ID,
				InvoiceCurrency: currencyx.Code("USD"),
				ActiveFrom:      currentTime,
				BillingAnchor:   currentTime,
			})
			require.NoError(t, err)

			if tt.prepare != nil {
				tt.prepare(t, &spec)
			}

			sub, err := deps.SubscriptionService.Create(t.Context(), cust.Namespace, spec)
			require.NoError(t, err)
			beforeView, err := deps.SubscriptionService.GetView(t.Context(), sub.NamespacedID)
			require.NoError(t, err)
			subscriptiontestutils.ValidateSpecAndView(t, spec, beforeView)

			// When the complete target spec changes one reconciliation boundary.
			tt.mutate(t, &spec)
			_, err = deps.SubscriptionService.Update(t.Context(), sub.NamespacedID, spec)
			require.NoError(t, err)

			// Then the persisted graph and derived entitlements exactly match the target.
			afterView, err := deps.SubscriptionService.GetView(t.Context(), sub.NamespacedID)
			require.NoError(t, err)
			tt.assert(t, beforeView, afterView)
			subscriptiontestutils.ValidateSpecAndView(t, spec, afterView)
		})
	}
}

func requireSyncPhase(t *testing.T, view subscription.SubscriptionView, phaseKey string) subscription.SubscriptionPhaseView {
	t.Helper()

	phase, found := lo.Find(view.Phases, func(phase subscription.SubscriptionPhaseView) bool {
		return phase.SubscriptionPhase.Key == phaseKey
	})
	require.True(t, found, "phase %s not found", phaseKey)

	return phase
}

func requireSyncItems(t *testing.T, phase subscription.SubscriptionPhaseView, itemKey string) []subscription.SubscriptionItemView {
	t.Helper()

	items, found := phase.ItemsByKey[itemKey]
	require.True(t, found, "item %s not found in phase %s", itemKey, phase.SubscriptionPhase.Key)

	return items
}

func TestDeleteScheduledDowngradeCanExpandDeletedSubscriptionForSync(t *testing.T) {
	ctx := t.Context()
	currentTime := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(currentTime)
	defer clock.UnFreeze()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	defer dbDeps.Cleanup(t)

	deps := subscriptiontestutils.NewService(t, dbDeps)
	customerEntity := deps.CustomerAdapter.CreateExampleCustomer(t)
	_ = deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)

	billingAnchor := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	premiumPlanInput := subscriptiontestutils.GetExamplePlanInput(t)
	premiumPlanInput.Key = "premium-plan"
	premiumPlanInput.Name = "Premium"
	premiumPlan := deps.PlanHelper.CreatePlan(t, premiumPlanInput)

	basicPlanInput := subscriptiontestutils.GetExamplePlanInput(t)
	basicPlanInput.Key = "basic-plan"
	basicPlanInput.Name = "Basic"
	basicPlan := deps.PlanHelper.CreatePlan(t, basicPlanInput)

	var premiumSubscription subscription.SubscriptionView
	var scheduledSubscription subscription.SubscriptionView

	t.Run("given a customer with a premium subscription", func(t *testing.T) {
		// given:
		// - a customer with a premium subscription anchored to the first day of the month
		// - a lower-priced basic plan available for the next billing cycle
		// when:
		// - the premium subscription is created from its plan
		// then:
		// - the active subscription exists
		var err error
		premiumSubscription, err = deps.WorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
			ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Timing: subscription.Timing{
					Custom: &currentTime,
				},
			},
			Namespace:     subscriptiontestutils.ExampleNamespace,
			CustomerID:    customerEntity.ID,
			BillingAnchor: &billingAnchor,
		}, premiumPlan)
		require.NoError(t, err)
		require.NotNil(t, premiumSubscription)
	})

	t.Run("when a downgrade is scheduled for the next billing cycle", func(t *testing.T) {
		// given:
		// - an active premium subscription
		// when:
		// - the customer schedules a downgrade to the basic plan
		// then:
		// - the current subscription is capped
		// - a scheduled basic subscription is created
		currentSubscription, nextSubscription, err := deps.WorkflowService.ChangeToPlan(ctx, premiumSubscription.Subscription.NamespacedID, subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{
				Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
			},
		}, basicPlan)
		require.NoError(t, err)
		require.NotNil(t, currentSubscription.ActiveTo)
		require.NotNil(t, nextSubscription)

		scheduledSubscription = nextSubscription
		require.Equal(t, subscription.SubscriptionStatusScheduled, scheduledSubscription.Subscription.GetStatusAt(clock.Now()))
	})

	t.Run("when the scheduled downgrade is deleted", func(t *testing.T) {
		// given:
		// - a scheduled downgrade subscription
		// when:
		// - the customer cancels the scheduled downgrade
		// then:
		// - the default read path no longer returns the deleted subscription
		// - the deleted subscription can still be listed and expanded for billing-side cleanup
		require.NoError(t, deps.SubscriptionService.Delete(ctx, scheduledSubscription.Subscription.NamespacedID))

		_, err := deps.SubscriptionService.GetView(ctx, scheduledSubscription.Subscription.NamespacedID)
		require.Error(t, err)

		deletedSubscription := getSubscriptionViewIncludingDeleted(t, ctx, deps.SubscriptionService, scheduledSubscription.Subscription.NamespacedID)
		require.Equal(t, scheduledSubscription.Subscription.ID, deletedSubscription.Subscription.ID)
		require.NotNil(t, deletedSubscription.Subscription.DeletedAt)
	})
}

func getSubscriptionViewIncludingDeleted(t *testing.T, ctx context.Context, service subscription.Service, subscriptionID models.NamespacedID) subscription.SubscriptionView {
	t.Helper()

	subscriptions, err := service.List(ctx, subscription.ListSubscriptionsInput{
		Namespaces:     []string{subscriptionID.Namespace},
		ID:             &filter.FilterULID{FilterString: filter.FilterString{Eq: &subscriptionID.ID}},
		IncludeDeleted: true,
	})
	require.NoError(t, err)
	require.Len(t, subscriptions.Items, 1)

	views, err := service.ExpandViews(ctx, subscriptions.Items)
	require.NoError(t, err)
	require.Len(t, views, 1)

	return views[0]
}
