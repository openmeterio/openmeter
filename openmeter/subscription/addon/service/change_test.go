package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestAddonServiceChangeQuantity(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2025-04-01T00:00:00Z")

	t.Run("Should error if input is invalid", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
			clock.SetTime(now)
			defer clock.ResetTime()

			subAdd := createExampleSubscriptionAddon(t, deps, now)

			createInp := subscriptionaddon.CreateSubscriptionAddonQuantityInput{
				ActiveFrom: now,
				Quantity:   -1,
			}
			expectedErr := createInp.Validate()
			require.Error(t, expectedErr)

			_, err := deps.SubscriptionAddonService.ChangeQuantity(context.Background(), subAdd.NamespacedID, createInp)
			require.Error(t, err)
			require.ErrorContains(t, err, expectedErr.Error())
		})
	})

	t.Run("Should error if quantity is greater than one for single instance addon", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
			clock.SetTime(now)
			defer clock.ResetTime()

			subAdd := createExampleSubscriptionAddon(t, deps, now)
			add, err := deps.AddonService.GetAddon(context.Background(), addon.GetAddonInput{
				NamespacedID: models.NamespacedID{
					Namespace: subAdd.Namespace,
					ID:        subAdd.Addon.ID,
				},
			})
			require.NoError(t, err)
			require.Equal(t, productcatalog.AddonInstanceTypeSingle, add.InstanceType)

			createInp := subscriptionaddon.CreateSubscriptionAddonQuantityInput{
				ActiveFrom: now,
				Quantity:   2,
			}
			_, err = deps.SubscriptionAddonService.ChangeQuantity(context.Background(), subAdd.NamespacedID, createInp)
			require.Error(t, err)
			require.ErrorAs(t, err, lo.ToPtr(&models.GenericValidationError{}))
		})
	})

	t.Run("Should validate addon quantity after purchase doesnt exceed maximum quantity", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
			clock.SetTime(now)
			defer clock.ResetTime()

			planInp := subscriptiontestutils.GetExamplePlanInput(t)

			addonInp := subscriptiontestutils.BuildAddonForTesting(t, productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now),
			}, productcatalog.AddonInstanceTypeMultiple, &subscriptiontestutils.ExampleAddonRateCard1)

			_ = deps.FeatureConnector.CreateExampleFeatures(t)

			p, err := deps.PlanService.CreatePlan(context.Background(), planInp)
			require.Nil(t, err)
			require.NotNil(t, p)

			add := deps.AddonService.CreateTestAddon(t, addonInp)

			_, err = deps.PlanAddonService.CreatePlanAddon(context.Background(), planaddon.CreatePlanAddonInput{
				NamespacedModel: models.NamespacedModel{
					Namespace: subscriptiontestutils.ExampleNamespace,
				},
				PlanID:        p.ID,
				AddonID:       add.ID,
				FromPlanPhase: p.Phases[0].Key,
				MaxQuantity:   lo.ToPtr(2),
			})
			require.Nil(t, err, "received error: %s", err)

			p, err = deps.PlanService.PublishPlan(context.Background(), plan.PublishPlanInput{
				NamespacedID: p.NamespacedID,
				EffectivePeriod: productcatalog.EffectivePeriod{
					EffectiveFrom: lo.ToPtr(clock.Now()),
					EffectiveTo:   lo.ToPtr(testutils.GetRFC3339Time(t, "2099-01-01T00:00:00Z")),
				},
			})
			require.Nil(t, err, "received error: %s", err)

			// Let's create a subscription
			cust := deps.CustomerAdapter.CreateExampleCustomer(t)
			spec1, err := subscription.NewSpecFromPlan(&plansubscription.Plan{
				Plan: p.AsProductCatalogPlan(),
				Ref:  &p.NamespacedID,
			}, subscription.CreateSubscriptionCustomerInput{
				CustomerId: cust.ID,
				Currency:   "USD",
				ActiveFrom: now,
				Name:       "Test Subscription",
			})
			require.Nil(t, err)

			sub, err := deps.SubscriptionService.Create(context.Background(), subscriptiontestutils.ExampleNamespace, spec1)
			require.Nil(t, err)

			aRCIDs := lo.Map(add.RateCards, func(rc addon.RateCard, _ int) string {
				return rc.ID
			})
			require.Len(t, aRCIDs, 1)

			// Now, let's create a SubscriptionAddon
			subAddonInp := subscriptionaddon.CreateSubscriptionAddonInput{
				AddonID:        add.ID,
				SubscriptionID: sub.ID,
				InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
					ActiveFrom: now,
					Quantity:   1,
				},
			}
			subAdd, err := deps.SubscriptionAddonService.Create(context.Background(), subscriptiontestutils.ExampleNamespace, subAddonInp)
			require.Nil(t, err)

			changeTime := testutils.GetRFC3339Time(t, "2025-04-02T00:00:00Z")

			createInp := subscriptionaddon.CreateSubscriptionAddonQuantityInput{
				ActiveFrom: changeTime,
				Quantity:   5,
			}

			_, err = deps.SubscriptionAddonService.ChangeQuantity(context.Background(), subAdd.NamespacedID, createInp)
			require.Error(t, err)
			require.ErrorAs(t, err, lo.ToPtr(&models.GenericValidationError{}))
			require.ErrorContains(t, err, fmt.Sprintf("addon %s@%d can be added a maximum of %d times", add.Key, add.Version, 2))
		})
	})

	t.Run("Should update quantity", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
			clock.SetTime(now)
			defer clock.ResetTime()

			subAdd := createExampleSubscriptionAddon(t, deps, now)

			changeTime := testutils.GetRFC3339Time(t, "2025-04-02T00:00:00Z")

			createInp := subscriptionaddon.CreateSubscriptionAddonQuantityInput{
				ActiveFrom: changeTime,
				Quantity:   1,
			}

			subAddUpdated, err := deps.SubscriptionAddonService.ChangeQuantity(context.Background(), subAdd.NamespacedID, createInp)
			require.NoError(t, err)

			require.Equal(t, len(subAdd.Quantities.GetTimes())+1, len(subAddUpdated.Quantities.GetTimes()))
			last := subAddUpdated.Quantities.GetAt(len(subAddUpdated.Quantities.GetTimes()) - 1)

			require.True(t, changeTime.Equal(last.GetTime()))
			require.Equal(t, 1, last.GetValue().Quantity)
		})
	})
}

func createExampleSubscriptionAddon(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies, now time.Time) *subscriptionaddon.SubscriptionAddon {
	clock.SetTime(now)
	defer clock.ResetTime()

	p, add := createPlanWithAddon(
		t,
		deps,
		subscriptiontestutils.GetExamplePlanInput(t),
		subscriptiontestutils.GetExampleAddonInput(t, productcatalog.EffectivePeriod{
			EffectiveFrom: lo.ToPtr(now),
		}),
	)

	// Let's create a subscription
	cust := deps.CustomerAdapter.CreateExampleCustomer(t)
	spec1, err := subscription.NewSpecFromPlan(p, subscription.CreateSubscriptionCustomerInput{
		CustomerId: cust.ID,
		Currency:   "USD",
		ActiveFrom: now,
		Name:       "Test Subscription",
	})
	require.Nil(t, err)

	sub, err := deps.SubscriptionService.Create(context.Background(), subscriptiontestutils.ExampleNamespace, spec1)
	require.Nil(t, err)

	aRCIDs := lo.Map(add.RateCards, func(rc addon.RateCard, _ int) string {
		return rc.ID
	})
	require.Len(t, aRCIDs, 1)

	// Now, let's create a SubscriptionAddon
	subAddonInp := subscriptionaddon.CreateSubscriptionAddonInput{
		AddonID:        add.ID,
		SubscriptionID: sub.ID,
		InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
			ActiveFrom: now,
			Quantity:   1,
		},
	}
	subAdd1, err := deps.SubscriptionAddonService.Create(context.Background(), subscriptiontestutils.ExampleNamespace, subAddonInp)
	require.Nil(t, err)

	return subAdd1
}
