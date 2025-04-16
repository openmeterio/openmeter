package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
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
					Name:       "Test",
					CustomerId: deps.Customer.ID,
					Currency:   currencyx.Code("USD"),
					ActiveFrom: deps.CurrentTime,
				})
				require.Nil(t, err)

				sub, err := deps.Service.Create(ctx, deps.Customer.Namespace, spec)
				require.Nil(t, err)

				_, err = deps.Service.Update(ctx, sub.NamespacedID, spec)
				require.Nil(t, err)
			},
		},
		{
			Name: "Should error if plan changes",
			Handler: func(t *testing.T, deps TDeps) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				spec, err := subscription.NewSpecFromPlan(deps.ExamplePlan, subscription.CreateSubscriptionCustomerInput{
					Name:       "Test",
					CustomerId: deps.Customer.ID,
					Currency:   currencyx.Code("USD"),
					ActiveFrom: deps.CurrentTime,
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
					Name:       "Test",
					CustomerId: deps.Customer.ID,
					Currency:   currencyx.Code("USD"),
					ActiveFrom: deps.CurrentTime,
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
					Name:       "Test",
					CustomerId: deps.Customer.ID,
					Currency:   currencyx.Code("USD"),
					ActiveFrom: deps.CurrentTime,
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
			Name: "Should update contents of future phases when phase start changes",
			Handler: func(t *testing.T, deps TDeps) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				spec, err := subscription.NewSpecFromPlan(deps.ExamplePlan, subscription.CreateSubscriptionCustomerInput{
					Name:       "Test",
					CustomerId: deps.Customer.ID,
					Currency:   currencyx.Code("USD"),
					ActiveFrom: deps.CurrentTime,
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
				spec.Phases[pKey].StartAfter, err = spec.Phases[pKey].StartAfter.Add(testutils.GetISODuration(t, "P1M"))
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
					Name:       "Test",
					CustomerId: deps.Customer.ID,
					Currency:   currencyx.Code("USD"),
					ActiveFrom: deps.CurrentTime,
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
					Name:       "Test",
					CustomerId: deps.Customer.ID,
					Currency:   currencyx.Code("USD"),
					ActiveFrom: deps.CurrentTime,
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
					Name:       "Test",
					CustomerId: deps.Customer.ID,
					Currency:   currencyx.Code("USD"),
					ActiveFrom: deps.CurrentTime,
				})
				require.Nil(t, err)

				sub, err := deps.Service.Create(ctx, deps.Customer.Namespace, spec)
				require.Nil(t, err)

				_, err = deps.Service.GetView(ctx, sub.NamespacedID)
				require.Nil(t, err)

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

				_, err = deps.Service.Update(ctx, sub.NamespacedID, spec)
				require.Nil(t, err)

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

			_ = deps.FeatureConnector.CreateExampleFeatures(t)
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
