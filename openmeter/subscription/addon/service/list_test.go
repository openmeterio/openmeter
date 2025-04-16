package service_test

import (
	"context"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func TestAddonServiceGet(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2025-04-01T00:00:00Z")
	t.Run("Should use name and description of addon", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
			clock.SetTime(now)
			defer clock.ResetTime()

			// Let's create a subscription
			sub := createExampleSubscription(t, deps, now)

			// Let's create an add
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
				SubscriptionID: sub.Subscription.ID,
				InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
					ActiveFrom: now,
					Quantity:   1,
				},
			}
			subAdd, err := deps.SubscriptionAddonService.Create(context.Background(), subscriptiontestutils.ExampleNamespace, subAddonInp)
			require.Nil(t, err)

			require.Equal(t, add.Name, subAdd.Name)
			require.Equal(t, add.Description, subAdd.Description)
		})
	})
}

func TestAddonServiceList(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2025-04-01T00:00:00Z")

	t.Run("Should error if input is formally invalid", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
			inp := subscriptionaddon.ListSubscriptionAddonsInput{}

			expectedErr := inp.Validate()
			require.Error(t, expectedErr)

			_, err := deps.SubscriptionAddonService.List(context.Background(), subscriptiontestutils.ExampleNamespace, inp)
			require.Error(t, err)
			require.ErrorContains(t, err, expectedErr.Error())
		})
	})

	t.Run("Should return all addons for a subscription", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
			clock.SetTime(now)
			defer clock.ResetTime()

			ctx := context.Background()

			// Let's create a subscription
			sub := createExampleSubscription(t, deps, now)

			// Let's create two addons
			per := productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now),
			}
			add1 := deps.AddonService.CreateTestAddon(t, subscriptiontestutils.GetExampleAddonInput(t, per))

			aRCIDs1 := lo.Map(add1.RateCards, func(rc addon.RateCard, _ int) string {
				return rc.ID
			})
			require.Len(t, aRCIDs1, 1)

			addInp := subscriptiontestutils.GetExampleAddonInput(t, per)
			addInp.Addon.AddonMeta.Key = "addon-2"

			addon2, err := deps.AddonService.CreateAddon(ctx, addInp)
			require.NoError(t, err)

			addon2, err = deps.AddonService.PublishAddon(ctx, addon.PublishAddonInput{
				NamespacedID:    addon2.NamespacedID,
				EffectivePeriod: per,
			})
			require.NoError(t, err)

			aRCIDs2 := lo.Map(addon2.RateCards, func(rc addon.RateCard, _ int) string {
				return rc.ID
			})
			require.Len(t, aRCIDs2, 1)

			// Let's create a SubscriptionAddon for the first addon
			subAddonInp := subscriptionaddon.CreateSubscriptionAddonInput{
				AddonID:        add1.ID,
				SubscriptionID: sub.Subscription.ID,
				InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
					ActiveFrom: now,
					Quantity:   1,
				},
			}
			subAdd1, err := deps.SubscriptionAddonService.Create(ctx, subscriptiontestutils.ExampleNamespace, subAddonInp)
			require.NoError(t, err)

			// Let's create a SubscriptionAddon for the second addon
			subAddonInp2 := subscriptionaddon.CreateSubscriptionAddonInput{
				AddonID:        addon2.ID,
				SubscriptionID: sub.Subscription.ID,
				InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
					ActiveFrom: now,
					Quantity:   1,
				},
			}

			subAdd2, err := deps.SubscriptionAddonService.Create(ctx, subscriptiontestutils.ExampleNamespace, subAddonInp2)
			require.NoError(t, err)

			t.Run("Should return all addons for a subscription", func(t *testing.T) {
				listInp := subscriptionaddon.ListSubscriptionAddonsInput{
					SubscriptionID: sub.Subscription.ID,
				}
				resp, err := deps.SubscriptionAddonService.List(ctx, subscriptiontestutils.ExampleNamespace, listInp)
				require.NoError(t, err)

				require.Len(t, resp.Items, 2)
				require.Equal(t, resp.TotalCount, 2)
				subscriptiontestutils.SubscriptionAddonsEqual(t, *subAdd1, resp.Items[0])
				subscriptiontestutils.SubscriptionAddonsEqual(t, *subAdd2, resp.Items[1])
			})

			t.Run("Should paginate returned addons", func(t *testing.T) {
				listInp := subscriptionaddon.ListSubscriptionAddonsInput{
					SubscriptionID: sub.Subscription.ID,
					Page:           pagination.NewPage(1, 1),
				}
				resp, err := deps.SubscriptionAddonService.List(ctx, subscriptiontestutils.ExampleNamespace, listInp)
				require.NoError(t, err)

				require.Len(t, resp.Items, 1)
				require.Equal(t, resp.TotalCount, 2)
				subscriptiontestutils.SubscriptionAddonsEqual(t, *subAdd1, resp.Items[0])

				// Let's fetch the next page
				listInp.Page = pagination.NewPage(2, 1)
				resp, err = deps.SubscriptionAddonService.List(ctx, subscriptiontestutils.ExampleNamespace, listInp)
				require.NoError(t, err)

				require.Len(t, resp.Items, 1)
				require.Equal(t, resp.TotalCount, 2)
				subscriptiontestutils.SubscriptionAddonsEqual(t, *subAdd2, resp.Items[0])
			})
		})
	})
}
