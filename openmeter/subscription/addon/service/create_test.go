package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestAddonServiceCreate(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2025-04-01T00:00:00Z")

	t.Run("Should error if input is formally invalid", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
			clock.SetTime(now)
			defer clock.ResetTime()

			// Let's create a subscription
			sub := createExampleSubscription(t, deps, now)

			// Let's create an add
			add := deps.AddonService.CreateExampleAddon(t, productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now),
			})

			aRCIDs := lo.Map(add.RateCards, func(rc addon.RateCard, _ int) string {
				return rc.ID
			})
			require.Len(t, aRCIDs, 1)

			// Now, let's create a SubscriptionAddon
			subAddonInp := subscriptionaddon.CreateSubscriptionAddonInput{
				AddonID:        add.ID,
				SubscriptionID: sub.Subscription.ID,
				RateCards:      []subscriptionaddon.CreateSubscriptionAddonRateCardInput{},
				InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
					ActiveFrom: now,
					Quantity:   0,
				},
			}
			expErr := subAddonInp.Validate()
			require.Error(t, expErr)

			_, err := deps.SubscriptionAddonService.Create(context.Background(), subscriptiontestutils.ExampleNamespace, subAddonInp)
			require.Error(t, err)
			require.ErrorContains(t, err, expErr.Error())
		})
	})

	t.Run("Shoul error if addon doesn't exist", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
			clock.SetTime(now)
			defer clock.ResetTime()

			// Let's create a subscription
			sub := createExampleSubscription(t, deps, now)

			// Let's NOT create an addon

			// Now, let's create a SubscriptionAddon
			subAddonInp := subscriptionaddon.CreateSubscriptionAddonInput{
				AddonID:        ulid.Make().String(),
				SubscriptionID: sub.Subscription.ID,
				RateCards: []subscriptionaddon.CreateSubscriptionAddonRateCardInput{
					{
						AddonRateCardID: ulid.Make().String(),

						AffectedSubscriptionItemIDs: []string{sub.Phases[1].ItemsByKey[subscriptiontestutils.ExampleFeatureKey2][0].SubscriptionItem.ID},
					},
				},
				InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
					ActiveFrom: now,
					Quantity:   1,
				},
			}

			_, err := deps.SubscriptionAddonService.Create(context.Background(), subscriptiontestutils.ExampleNamespace, subAddonInp)
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
			add := deps.AddonService.CreateExampleAddon(t, productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now),
			})

			aRCIDs := lo.Map(add.RateCards, func(rc addon.RateCard, _ int) string {
				return rc.ID
			})
			require.Len(t, aRCIDs, 1)

			// Now, let's create a SubscriptionAddon
			subAddonInp := subscriptionaddon.CreateSubscriptionAddonInput{
				AddonID:        add.ID,
				SubscriptionID: ulid.Make().String(),
				RateCards: []subscriptionaddon.CreateSubscriptionAddonRateCardInput{
					{
						AddonRateCardID: aRCIDs[0],

						AffectedSubscriptionItemIDs: []string{ulid.Make().String()},
					},
				},
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

	t.Run("Should error if referenced AddonRateCards don't belong to provided Addon", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
			clock.SetTime(now)
			defer clock.ResetTime()

			// Let's create a subscription
			sub := createExampleSubscription(t, deps, now)

			// Let's create an add-on
			add := deps.AddonService.CreateExampleAddon(t, productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now),
			})

			aRCIDs := lo.Map(add.RateCards, func(rc addon.RateCard, _ int) string {
				return rc.ID
			})
			require.Len(t, aRCIDs, 1)

			// Now, let's create a SubscriptionAddon
			subAddonInp := subscriptionaddon.CreateSubscriptionAddonInput{
				AddonID:        add.ID,
				SubscriptionID: sub.Subscription.ID,
				RateCards: []subscriptionaddon.CreateSubscriptionAddonRateCardInput{
					{
						AddonRateCardID: ulid.Make().String(), // invalid AddonRateCardID

						AffectedSubscriptionItemIDs: []string{sub.Phases[1].ItemsByKey[subscriptiontestutils.ExampleFeatureKey2][0].SubscriptionItem.ID},
					},
				},
				InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
					ActiveFrom: now,
					Quantity:   1,
				},
			}
			_, err := deps.SubscriptionAddonService.Create(context.Background(), subscriptiontestutils.ExampleNamespace, subAddonInp)
			require.Error(t, err)
			require.ErrorAs(t, err, lo.ToPtr(&models.GenericValidationError{}))
		})
	})

	t.Run("Should error if referenced SubscriptionItems don't exist", func(t *testing.T) {
		t.Skip("Conflict error will always precede this so there's no clean way to do this test right now")

		withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
			clock.SetTime(now)
			defer clock.ResetTime()

			// Let's create a subscription
			sub := createExampleSubscription(t, deps, now)

			// Let's create an add
			add := deps.AddonService.CreateExampleAddon(t, productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now),
			})

			aRCIDs := lo.Map(add.RateCards, func(rc addon.RateCard, _ int) string {
				return rc.ID
			})
			require.Len(t, aRCIDs, 1)

			// Now, let's create a SubscriptionAddon
			subAddonInp := subscriptionaddon.CreateSubscriptionAddonInput{
				AddonID:        add.ID,
				SubscriptionID: sub.Subscription.ID,
				RateCards: []subscriptionaddon.CreateSubscriptionAddonRateCardInput{
					{
						AddonRateCardID: aRCIDs[0],

						AffectedSubscriptionItemIDs: []string{ulid.Make().String()},
					},
				},
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

	t.Run("Should error if referenced SubscriptionItems don't belong to provided Subscription", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
			clock.SetTime(now)
			defer clock.ResetTime()

			// Let's create a subscription
			sub := createExampleSubscription(t, deps, now)

			// And let's create another subscription that we'll incorrectly reference

			cust, err := deps.CustomerAdapter.CreateCustomer(context.Background(), customer.CreateCustomerInput{
				Namespace: subscriptiontestutils.ExampleNamespace,
				CustomerMutate: customer.CustomerMutate{
					Name:         "Test Customer 2",
					Key:          lo.ToPtr("test-customer-2"),
					PrimaryEmail: lo.ToPtr("mail2@me.uk"),
					Currency:     lo.ToPtr(currencyx.Code("USD")),
					UsageAttribution: customer.CustomerUsageAttribution{
						SubjectKeys: []string{"john-doe-2"},
					},
				},
			})
			require.Nil(t, err)

			plan, err := deps.PlanService.GetPlan(context.Background(), plan.GetPlanInput{
				Key:           "test_plan",
				IncludeLatest: true,
				NamespacedID: models.NamespacedID{
					Namespace: subscriptiontestutils.ExampleNamespace,
				},
			})
			require.Nil(t, err)

			pp, err := plan.AsProductCatalogPlan(clock.Now())
			require.Nil(t, err)

			subPlan := &plansubscription.Plan{
				Plan: pp,
				Ref:  &plan.NamespacedID,
			}

			spec1, err := subscription.NewSpecFromPlan(subPlan, subscription.CreateSubscriptionCustomerInput{
				CustomerId: cust.ID,
				Currency:   "USD",
				ActiveFrom: now,
				Name:       "Test Subscription",
			})
			require.Nil(t, err)

			sub2, err := deps.SubscriptionService.Create(context.Background(), subscriptiontestutils.ExampleNamespace, spec1)
			require.Nil(t, err)

			view, err := deps.SubscriptionService.GetView(context.Background(), sub2.NamespacedID)
			require.Nil(t, err)

			// Let's create an add
			add := deps.AddonService.CreateExampleAddon(t, productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now),
			})

			aRCIDs := lo.Map(add.RateCards, func(rc addon.RateCard, _ int) string {
				return rc.ID
			})
			require.Len(t, aRCIDs, 1)

			// Now, let's create a SubscriptionAddon
			subAddonInp := subscriptionaddon.CreateSubscriptionAddonInput{
				AddonID:        add.ID,
				SubscriptionID: sub.Subscription.ID,
				InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
					ActiveFrom: now,
					Quantity:   1,
				},
				RateCards: []subscriptionaddon.CreateSubscriptionAddonRateCardInput{
					{
						AddonRateCardID: aRCIDs[0],

						AffectedSubscriptionItemIDs: []string{view.Phases[1].ItemsByKey[subscriptiontestutils.ExampleFeatureKey2][0].SubscriptionItem.ID},
					},
				},
			}
			_, err = deps.SubscriptionAddonService.Create(context.Background(), subscriptiontestutils.ExampleNamespace, subAddonInp)
			require.Error(t, err)
			require.ErrorAs(t, err, lo.ToPtr(&models.GenericConflictError{}))
		})
	})

	t.Run("Should error if addon is single instance but quantity is not 1", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
			clock.SetTime(now)
			defer clock.ResetTime()

			// Let's create a subscription
			sub := createExampleSubscription(t, deps, now)

			// Let's create an addon
			add := deps.AddonService.CreateExampleAddon(t, productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now),
			})

			aRCIDs := lo.Map(add.RateCards, func(rc addon.RateCard, _ int) string {
				return rc.ID
			})
			require.Len(t, aRCIDs, 1)

			// Let's assert that its a single instance addon
			require.Equal(t, add.InstanceType, productcatalog.AddonInstanceTypeSingle)

			// Now, let's create a SubscriptionAddon
			subAddonInp := subscriptionaddon.CreateSubscriptionAddonInput{
				AddonID:        add.ID,
				SubscriptionID: sub.Subscription.ID,
				RateCards: []subscriptionaddon.CreateSubscriptionAddonRateCardInput{
					{
						AddonRateCardID: aRCIDs[0],

						AffectedSubscriptionItemIDs: []string{sub.Phases[1].ItemsByKey[subscriptiontestutils.ExampleFeatureKey2][0].SubscriptionItem.ID},
					},
				},
				InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
					ActiveFrom: now,
					Quantity:   3,
				},
			}
			_, err := deps.SubscriptionAddonService.Create(context.Background(), subscriptiontestutils.ExampleNamespace, subAddonInp)
			require.Error(t, err)
			require.ErrorAs(t, err, lo.ToPtr(&models.GenericValidationError{}))
		})
	})

	t.Run("Should validate Addon can be purchased for plan of subscription", func(t *testing.T) {
		t.Skip("TODO: implement once Addon-Plan linking is implemented")
	})

	t.Run("Should not allow purchasing Addon for a custom Subscription", func(t *testing.T) {
		t.Skip("TODO: implement once Addon-Plan linking is implemented")
	})

	t.Run("Should create and retrieve addon", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
			clock.SetTime(now)
			defer clock.ResetTime()

			// Let's create a subscription
			sub := createExampleSubscription(t, deps, now)

			// Let's create an addon
			add := deps.AddonService.CreateExampleAddon(t, productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now),
			})

			aRCIDs := lo.Map(add.RateCards, func(rc addon.RateCard, _ int) string {
				return rc.ID
			})
			require.Len(t, aRCIDs, 1)

			// Now, let's create a SubscriptionAddon
			subAddonInp := subscriptionaddon.CreateSubscriptionAddonInput{
				AddonID:        add.ID,
				SubscriptionID: sub.Subscription.ID,
				RateCards: []subscriptionaddon.CreateSubscriptionAddonRateCardInput{
					{
						AddonRateCardID: aRCIDs[0],

						AffectedSubscriptionItemIDs: []string{sub.Phases[1].ItemsByKey[subscriptiontestutils.ExampleFeatureKey2][0].SubscriptionItem.ID},
					},
				},
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
				require.Equal(t, subAddonInp.RateCards[0].AddonRateCardID, subAdd1.RateCards[0].AddonRateCard.ID)
				require.Equal(t, subAddonInp.RateCards[0].AffectedSubscriptionItemIDs, subAdd1.RateCards[0].AffectedSubscriptionItemIDs)
			})

			t.Run("Should return same addon on create and then a subsequent get", func(t *testing.T) {
				subscriptiontestutils.SubscriptionAddonsEqual(t, *subAdd1, *subAdd2)
			})
		})
	})
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
