package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
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
					ID:        subAdd.AddonID,
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

	// Let's create a subscription
	sub := createExampleSubscription(t, deps, now)

	// Let's create an addon
	addon := deps.AddonService.CreateExampleAddon(t, productcatalog.EffectivePeriod{
		EffectiveFrom: lo.ToPtr(now),
	})

	aRCIDs := getRateCardsOfAddon(t, deps, addon)
	require.Len(t, aRCIDs, 1)

	// Now, let's create a SubscriptionAddon
	subAddonInp := subscriptionaddon.CreateSubscriptionAddonInput{
		AddonID:        addon.ID,
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

	return subAdd1
}
