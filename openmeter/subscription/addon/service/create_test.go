package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
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

func TestAddonServiceCreate(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2025-04-01T00:00:00Z")

	t.Run("Should error if input is formally invalid", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
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

			// Now, let's create a SubscriptionAddon
			subAddonInp := subscriptionaddon.CreateSubscriptionAddonInput{
				AddonID:        add.ID,
				SubscriptionID: sub.NamespacedID.ID,
				InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
					ActiveFrom: now,
					Quantity:   0,
				},
			}
			expErr := subAddonInp.Validate()
			require.Error(t, expErr)

			_, err = deps.SubscriptionAddonService.Create(context.Background(), subscriptiontestutils.ExampleNamespace, subAddonInp)
			require.Error(t, err)
			require.ErrorContains(t, err, expErr.Error())
		})
	})

	t.Run("Should error if addon doesn't exist", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
			clock.SetTime(now)
			defer clock.ResetTime()

			p, _ := createPlanWithAddon(
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

			// Now, let's create a SubscriptionAddon
			subAddonInp := subscriptionaddon.CreateSubscriptionAddonInput{
				AddonID:        ulid.Make().String(),
				SubscriptionID: sub.NamespacedID.ID,
				InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
					ActiveFrom: now,
					Quantity:   1,
				},
			}

			_, err = deps.SubscriptionAddonService.Create(context.Background(), subscriptiontestutils.ExampleNamespace, subAddonInp)
			require.Error(t, err)
			require.True(t, models.IsGenericNotFoundError(err))
		})
	})

	t.Run("Should error if subscription doesn't exist", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
			clock.SetTime(now)
			defer clock.ResetTime()

			// Let's NOT create a subscription
			_ = deps.FeatureConnector.CreateExampleFeatures(t)

			// Let's create an add
			add := deps.AddonService.CreateTestAddon(t, subscriptiontestutils.GetExampleAddonInput(t, productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now),
			}))

			// Now, let's create a SubscriptionAddon
			subAddonInp := subscriptionaddon.CreateSubscriptionAddonInput{
				AddonID:        add.ID,
				SubscriptionID: ulid.Make().String(),
				InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
					ActiveFrom: now,
					Quantity:   1,
				},
			}

			_, err := deps.SubscriptionAddonService.Create(context.Background(), subscriptiontestutils.ExampleNamespace, subAddonInp)
			require.Error(t, err)
			require.ErrorAs(t, err, lo.ToPtr(&models.GenericNotFoundError{}))
		})
	})

	t.Run("Should error if addon is single instance but quantity is not 1", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
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

			// Let's assert that its a single instance addon
			require.Equal(t, add.InstanceType, productcatalog.AddonInstanceTypeSingle)

			// Now, let's create a SubscriptionAddon
			subAddonInp := subscriptionaddon.CreateSubscriptionAddonInput{
				AddonID:        add.ID,
				SubscriptionID: sub.NamespacedID.ID,
				InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
					ActiveFrom: now,
					Quantity:   3,
				},
			}
			_, err = deps.SubscriptionAddonService.Create(context.Background(), subscriptiontestutils.ExampleNamespace, subAddonInp)
			require.Error(t, err)
			require.ErrorAs(t, err, lo.ToPtr(&models.GenericValidationError{}))
		})
	})

	t.Run("Should validate Addon can be purchased for plan of subscription", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
			clock.SetTime(now)
			defer clock.ResetTime()

			p, _ := createPlanWithAddon(
				t,
				deps,
				subscriptiontestutils.GetExamplePlanInput(t),
				subscriptiontestutils.GetExampleAddonInput(t, productcatalog.EffectivePeriod{
					EffectiveFrom: lo.ToPtr(now),
				}),
			)

			addonInp := subscriptiontestutils.GetExampleAddonInput(t, productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now),
			})

			addonInp.Key = "unlinked"

			add := deps.AddonService.CreateTestAddon(t, addonInp)

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

			// Now, let's create a SubscriptionAddon
			subAddonInp := subscriptionaddon.CreateSubscriptionAddonInput{
				AddonID:        add.ID,
				SubscriptionID: sub.NamespacedID.ID,
				InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
					ActiveFrom: now,
					Quantity:   1,
				},
			}
			_, err = deps.SubscriptionAddonService.Create(context.Background(), subscriptiontestutils.ExampleNamespace, subAddonInp)
			require.Error(t, err)
			require.ErrorAs(t, err, lo.ToPtr(&models.GenericValidationError{}))
			require.ErrorContains(t, err, fmt.Sprintf("addon %s@%d is not linked to the plan %s@%d", add.Key, add.Version, p.ToCreateSubscriptionPlanInput().Plan.Key, p.ToCreateSubscriptionPlanInput().Plan.Version))
		})
	})

	t.Run("Should validate addon can be purchased in target phase", func(t *testing.T) {
		t.Skip("TODO: Implement this")
	})

	t.Run("Should validate addon quantity after purchase doesnt exceed maximum quantity", func(t *testing.T) {
		t.Skip("TODO: Implement this")
	})

	t.Run("Should not allow purchasing Addon for a custom Subscription", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
			clock.SetTime(now)
			defer clock.ResetTime()

			// Let's create a subscription without a plan reference
			cust := deps.CustomerAdapter.CreateExampleCustomer(t)
			_ = deps.FeatureConnector.CreateExampleFeatures(t)
			plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

			spec1, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
				CustomerId: cust.ID,
				Currency:   "USD",
				ActiveFrom: now,
				Name:       "Test Subscription",
			})
			require.Nil(t, err)

			spec1.Plan = nil

			sub, err := deps.SubscriptionService.Create(context.Background(), subscriptiontestutils.ExampleNamespace, spec1)
			require.Nil(t, err)

			// Let's create an addon
			add := deps.AddonService.CreateTestAddon(t, subscriptiontestutils.GetExampleAddonInput(t, productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now),
			}))

			aRCIDs := lo.Map(add.RateCards, func(rc addon.RateCard, _ int) string {
				return rc.ID
			})
			require.Len(t, aRCIDs, 1)

			// Now, let's create a SubscriptionAddon
			subAddonInp := subscriptionaddon.CreateSubscriptionAddonInput{
				AddonID:        add.ID,
				SubscriptionID: sub.NamespacedID.ID,
				InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
					ActiveFrom: now,
					Quantity:   1,
				},
			}

			_, err = deps.SubscriptionAddonService.Create(context.Background(), subscriptiontestutils.ExampleNamespace, subAddonInp)
			require.Error(t, err)
			require.ErrorAs(t, err, lo.ToPtr(&models.GenericValidationError{}))
			require.ErrorContains(t, err, "cannot add addon to a custom subscription")
		})
	})

	t.Run("Should create and retrieve addon", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
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

			// Now, let's fetch the subscription addon
			subAdd2, err := deps.SubscriptionAddonService.Get(context.Background(), subAdd1.NamespacedID)
			require.Nil(t, err)

			t.Run("Should create addon as specified", func(t *testing.T) {
				require.Equal(t, subAddonInp.AddonID, subAdd1.Addon.ID)
				require.Equal(t, subAddonInp.SubscriptionID, subAdd1.SubscriptionID)
				require.Len(t, subAdd1.RateCards, 1)
				require.Equal(t, aRCIDs[0], subAdd1.RateCards[0].AddonRateCard.ID)
			})

			t.Run("Should return same addon on create and then a subsequent get", func(t *testing.T) {
				subscriptiontestutils.SubscriptionAddonsEqual(t, *subAdd1, *subAdd2)
			})
		})
	})
}

func createPlanWithAddon(
	t *testing.T,
	deps subscriptiontestutils.SubscriptionDependencies,
	planInp plan.CreatePlanInput,
	addonInp addon.CreateAddonInput,
) (subscription.Plan, addon.Addon) {
	t.Helper()

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

	return &plansubscription.Plan{
		Plan: p.AsProductCatalogPlan(),
		Ref:  &p.NamespacedID,
	}, add
}

func createExampleSubscription(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies, currentTime time.Time) subscription.SubscriptionView {
	cust := deps.CustomerAdapter.CreateExampleCustomer(t)
	_ = deps.FeatureConnector.CreateExampleFeatures(t)
	plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

	spec1, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
		CustomerId: cust.ID,
		Currency:   "USD",
		ActiveFrom: currentTime,
		Name:       "Test Subscription",
	})
	require.Nil(t, err)

	sub, err := deps.SubscriptionService.Create(context.Background(), subscriptiontestutils.ExampleNamespace, spec1)
	require.Nil(t, err)

	view, err := deps.SubscriptionService.GetView(context.Background(), sub.NamespacedID)
	require.Nil(t, err)

	return view
}

func withDeps(t *testing.T, fn func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies)) {
	dbDeps := subscriptiontestutils.SetupDBDeps(t)

	deps := subscriptiontestutils.NewService(t, dbDeps)

	defer dbDeps.Cleanup(t)

	fn(t, deps)
}
