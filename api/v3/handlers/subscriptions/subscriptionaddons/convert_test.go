package subscriptionaddons

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestMapCreateSubscriptionAddonRequestToInput(t *testing.T) {
	t.Run("maps immediate timing and labels", func(t *testing.T) {
		var timing apiv3.BillingSubscriptionEditTiming
		require.NoError(t, timing.FromBillingSubscriptionEditTimingEnum(apiv3.BillingSubscriptionEditTimingEnum("immediate")))

		labels := apiv3.Labels{"team": "billing"}
		req := apiv3.CreateSubscriptionAddonRequest{
			Addon:    apiv3.AddonReference{Id: "01J8GFKQ0000000000000000"},
			Labels:   &labels,
			Quantity: 2,
			Timing:   timing,
		}

		input, err := mapCreateSubscriptionAddonRequestToInput(req)
		require.NoError(t, err)

		assert.Equal(t, "01J8GFKQ0000000000000000", input.AddonID)
		assert.Equal(t, 2, input.InitialQuantity)
		require.NotNil(t, input.Timing.Enum)
		assert.Equal(t, subscription.TimingImmediate, *input.Timing.Enum)
		assert.Equal(t, models.Metadata{"team": "billing"}, input.Metadata)
	})

	t.Run("maps next_billing_cycle timing", func(t *testing.T) {
		var timing apiv3.BillingSubscriptionEditTiming
		require.NoError(t, timing.FromBillingSubscriptionEditTimingEnum(apiv3.BillingSubscriptionEditTimingEnum("next_billing_cycle")))

		req := apiv3.CreateSubscriptionAddonRequest{
			Addon:    apiv3.AddonReference{Id: "addon-id"},
			Quantity: 1,
			Timing:   timing,
		}

		input, err := mapCreateSubscriptionAddonRequestToInput(req)
		require.NoError(t, err)
		require.NotNil(t, input.Timing.Enum)
		assert.Equal(t, subscription.TimingNextBillingCycle, *input.Timing.Enum)
		assert.Nil(t, input.Metadata)
	})

	t.Run("fails on invalid timing string", func(t *testing.T) {
		var timing apiv3.BillingSubscriptionEditTiming
		require.NoError(t, timing.FromBillingSubscriptionEditTimingEnum(apiv3.BillingSubscriptionEditTimingEnum("sometime_later")))

		req := apiv3.CreateSubscriptionAddonRequest{
			Addon:    apiv3.AddonReference{Id: "addon-id"},
			Quantity: 1,
			Timing:   timing,
		}

		_, err := mapCreateSubscriptionAddonRequestToInput(req)
		require.Error(t, err)
	})
}

// newTestSubscriptionAddon builds a SubscriptionAddon with a single quantity at activeFrom,
// no rate cards, and no soft-delete. The view stays empty — toAPISubscriptionAddon's only
// dependency on the view is through GetAffectedItemIDs, which returns an empty map for an
// empty view, exercising the nil-slice fallback path on rate cards.
func newTestSubscriptionAddon(activeFrom time.Time, qty int) subscriptionaddon.SubscriptionAddon {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	return subscriptionaddon.SubscriptionAddon{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01J8GFKQ0000000000000000"},
		ManagedModel: models.ManagedModel{CreatedAt: now, UpdatedAt: now},
		Name:         "Test addon",
		Quantities: timeutil.NewTimeline([]timeutil.Timed[subscriptionaddon.SubscriptionAddonQuantity]{
			subscriptionaddon.SubscriptionAddonQuantity{
				ActiveFrom: activeFrom,
				Quantity:   qty,
			}.AsTimed(),
		}),
	}
}

func TestToAPISubscriptionAddon(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	t.Run("active instance maps quantity from current instance", func(t *testing.T) {
		addon := newTestSubscriptionAddon(now.Add(-time.Hour), 3)

		got, err := toAPISubscriptionAddon(subscription.SubscriptionView{}, addon)
		require.NoError(t, err)

		assert.Equal(t, 3, got.Quantity)
		assert.Equal(t, now, got.QuantityAt)
		assert.Empty(t, got.RateCards)
		require.Len(t, got.Timeline, 1)
		assert.Equal(t, 3, got.Timeline[0].Quantity)
	})

	t.Run("future-scheduled instance returns quantity 0 instead of error", func(t *testing.T) {
		// Reproduces the next_billing_cycle case where active_from is in the future:
		// the previous code returned NotFound; the fix surfaces quantity 0.
		addon := newTestSubscriptionAddon(now.Add(time.Hour), 5)

		got, err := toAPISubscriptionAddon(subscription.SubscriptionView{}, addon)
		require.NoError(t, err)

		assert.Equal(t, 0, got.Quantity)
		assert.Equal(t, now, got.QuantityAt)
		require.Len(t, got.Timeline, 1)
		assert.Equal(t, 5, got.Timeline[0].Quantity)
		assert.Equal(t, now.Add(time.Hour), got.ActiveFrom)
	})

	t.Run("errors when addon has no instances", func(t *testing.T) {
		addon := subscriptionaddon.SubscriptionAddon{
			NamespacedID: models.NamespacedID{Namespace: "ns", ID: "id"},
			Quantities:   timeutil.NewTimeline([]timeutil.Timed[subscriptionaddon.SubscriptionAddonQuantity]{}),
		}

		_, err := toAPISubscriptionAddon(subscription.SubscriptionView{}, addon)
		require.Error(t, err)
	})

	t.Run("preserves description and labels round-trip", func(t *testing.T) {
		addon := newTestSubscriptionAddon(now.Add(-time.Hour), 1)
		addon.Description = lo.ToPtr("with desc")
		addon.Metadata = models.Metadata{"team": "billing"}

		got, err := toAPISubscriptionAddon(subscription.SubscriptionView{}, addon)
		require.NoError(t, err)

		assert.Equal(t, lo.ToPtr("with desc"), got.Description)
		require.NotNil(t, got.Labels)
		assert.Equal(t, "billing", (*got.Labels)["team"])
	})
}
