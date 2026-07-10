package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

// TestV3SubscriptionAddonAttach exercises POST /subscriptions/{id}/addons end to end:
// build a published plan + published addon, create a subscription, attach the addon,
// verify the response shape (rate_cards/timeline arrays, never null), then confirm
// the conflict path returns 409 when the same addon is attached twice.
func TestV3SubscriptionAddonAttach(t *testing.T) {
	c := newV3Client(t)

	// --- Fixture: customer ---

	customerKey := uniqueKey("sub_addon_customer")
	customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
		Key:          customerKey,
		Name:         "Subscription Addon Test Customer",
		Currency:     lo.ToPtr("USD"),
		PrimaryEmail: lo.ToPtr("test-" + customerKey + "@test.com"),
		UsageAttribution: &v3sdk.CustomerUsageAttribution{
			SubjectKeys: []string{customerKey},
		},
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, customer)

	// --- Fixture: draft plan + published addon, attach addon, then publish plan ---
	// Order matters: addons can only be attached to a plan while it is still in draft,
	// and the addon must be published before attach.

	planBody := validPlanRequest("sub_addon_plan")
	plan, err := c.Plans.Create(t.Context(), planBody)
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, plan)
	require.NotEmpty(t, plan.Phases, "plan must have at least one phase to attach an addon")

	addonBody := validAddonRequest("sub_addon")
	addon, err := c.Addons.Create(t.Context(), addonBody)
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, addon)

	_, err = c.Addons.Publish(t.Context(), addon.ID)
	c.requireStatus(http.StatusOK, err)

	_, err = c.PlanAddons.Create(t.Context(), plan.ID, validPlanAddonRequest(plan.Phases[0].Key, addon.ID))
	c.requireStatus(http.StatusCreated, err)

	_, err = c.Plans.Publish(t.Context(), plan.ID)
	c.requireStatus(http.StatusOK, err)

	// --- Fixture: subscription on the published plan ---

	subBody := v3sdk.SubscriptionCreate{
		Customer: v3sdk.SubscriptionChangeCustomer{ID: &customer.ID},
		Plan:     v3sdk.SubscriptionChangePlan{ID: &plan.ID},
	}

	sub, err := c.Subscriptions.Create(t.Context(), subBody)
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, sub)
	subscriptionID := sub.ID

	// --- Test: attach addon ---

	var subAddonID string

	t.Run("Should attach addon with immediate timing and return 201", func(t *testing.T) {
		timing := lo.Must(v3sdk.SubscriptionEditTimingFromEnum(v3sdk.SubscriptionEditTimingEnumImmediate))

		subAddon, err := c.Subscriptions.CreateAddon(t.Context(), subscriptionID, v3sdk.CreateSubscriptionAddonRequest{
			Addon:    v3sdk.AddonReference{ID: addon.ID},
			Quantity: 1,
			Timing:   timing,
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, subAddon)

		assert.NotEmpty(t, subAddon.ID)
		assert.Equal(t, addon.ID, subAddon.Addon.ID)
		assert.EqualValues(t, 1, subAddon.Quantity)
		// Regression guard for the nil-slice → JSON null bug: rate_cards must be a non-nil array
		// and every entry's affected_subscription_item_ids must be a non-nil array too.
		assert.NotNil(t, subAddon.RateCards, "rate_cards must not be null")
		for i, rc := range subAddon.RateCards {
			assert.NotNil(t, rc.AffectedSubscriptionItemIds, "rate_cards[%d].affected_subscription_item_ids must not be null", i)
		}
		// Timeline must be a non-nil array with at least one segment for an active addon.
		require.NotNil(t, subAddon.Timeline)
		require.NotEmpty(t, subAddon.Timeline)
		assert.EqualValues(t, 1, subAddon.Timeline[0].Quantity)

		subAddonID = subAddon.ID
	})

	t.Run("Should return 409 when attaching the same addon twice", func(t *testing.T) {
		require.NotEmpty(t, subAddonID, "first attach must have succeeded")

		timing := lo.Must(v3sdk.SubscriptionEditTimingFromEnum(v3sdk.SubscriptionEditTimingEnumImmediate))

		_, err := c.Subscriptions.CreateAddon(t.Context(), subscriptionID, v3sdk.CreateSubscriptionAddonRequest{
			Addon:    v3sdk.AddonReference{ID: addon.ID},
			Quantity: 1,
			Timing:   timing,
		})
		requireProblem(t, err, http.StatusConflict)
	})

	t.Run("Should reject invalid quantity 0", func(t *testing.T) {
		timing := lo.Must(v3sdk.SubscriptionEditTimingFromEnum(v3sdk.SubscriptionEditTimingEnumImmediate))

		_, err := c.Subscriptions.CreateAddon(t.Context(), subscriptionID, v3sdk.CreateSubscriptionAddonRequest{
			Addon:    v3sdk.AddonReference{ID: addon.ID},
			Quantity: 0,
			Timing:   timing,
		})
		// TypeSpec @minValue(1) rejects this at schema-validation; workflow validation
		// would also reject it. Either is fine — assert 4xx.
		apiErr, ok := v3sdk.AsAPIError(err)
		require.True(t, ok, "expected APIError, got %T: %v", err, err)
		assert.GreaterOrEqual(t, apiErr.StatusCode, http.StatusBadRequest, "expected 4xx for quantity=0, got %d", apiErr.StatusCode)
		assert.Less(t, apiErr.StatusCode, http.StatusInternalServerError, "expected 4xx not 5xx for quantity=0")
	})

	t.Run("Should get the attached addon and surface rate_cards / timeline arrays", func(t *testing.T) {
		require.NotEmpty(t, subAddonID)

		subAddon, err := c.Subscriptions.GetAddon(t.Context(), subscriptionID, subAddonID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, subAddon)

		assert.Equal(t, subAddonID, subAddon.ID)
		assert.NotNil(t, subAddon.RateCards, "GET: rate_cards must not be null")
		assert.NotNil(t, subAddon.Timeline, "GET: timeline must not be null")
		for i, rc := range subAddon.RateCards {
			assert.NotNil(t, rc.AffectedSubscriptionItemIds, "GET: rate_cards[%d].affected_subscription_item_ids must not be null", i)
		}
	})

	t.Run("Should list subscription addons and find the attached addon", func(t *testing.T) {
		require.NotEmpty(t, subAddonID)

		page, err := c.Subscriptions.ListAddons(t.Context(), subscriptionID, v3sdk.SubscriptionAddonListParams{
			Page: &v3sdk.PageParams{Size: lo.ToPtr(100)},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, page)

		found := false
		for _, sa := range page.Data {
			if sa.ID == subAddonID {
				found = true
				assert.NotNil(t, sa.RateCards, "LIST: rate_cards must not be null")
				assert.NotNil(t, sa.Timeline, "LIST: timeline must not be null")
				break
			}
		}
		assert.True(t, found, "attached subscription addon not found in list")
	})
}

// TestV3SubscriptionAddonNextBillingCycle attaches an addon with timing=next_billing_cycle
// and verifies the create endpoint returns 201 (not the pre-fix 404). The new instance's
// active_from is in the future, so the no-current-instance fallback in toAPISubscriptionAddon
// must return quantity 0 instead of erroring out.
func TestV3SubscriptionAddonNextBillingCycle(t *testing.T) {
	c := newV3Client(t)

	// --- Fixture: customer ---

	customerKey := uniqueKey("sub_addon_nbc_customer")
	customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
		Key:          customerKey,
		Name:         "Next Billing Cycle Test Customer",
		Currency:     lo.ToPtr("USD"),
		PrimaryEmail: lo.ToPtr("test-" + customerKey + "@test.com"),
		UsageAttribution: &v3sdk.CustomerUsageAttribution{
			SubjectKeys: []string{customerKey},
		},
	})
	c.requireStatus(http.StatusCreated, err)

	// --- Fixture: draft plan + published addon, attach, then publish plan ---

	plan, err := c.Plans.Create(t.Context(), validPlanRequest("sub_addon_nbc_plan"))
	c.requireStatus(http.StatusCreated, err)
	require.NotEmpty(t, plan.Phases)

	addon, err := c.Addons.Create(t.Context(), validAddonRequest("sub_addon_nbc"))
	c.requireStatus(http.StatusCreated, err)

	_, err = c.Addons.Publish(t.Context(), addon.ID)
	c.requireStatus(http.StatusOK, err)

	_, err = c.PlanAddons.Create(t.Context(), plan.ID, validPlanAddonRequest(plan.Phases[0].Key, addon.ID))
	c.requireStatus(http.StatusCreated, err)

	_, err = c.Plans.Publish(t.Context(), plan.ID)
	c.requireStatus(http.StatusOK, err)

	// Anchor the subscription at a past second so the next billing cycle is reliably
	// in the future at the moment of the addon attach.
	anchor := time.Now().Add(-time.Second)
	subBody := v3sdk.SubscriptionCreate{
		BillingAnchor: &anchor,
		Customer:      v3sdk.SubscriptionChangeCustomer{ID: &customer.ID},
		Plan:          v3sdk.SubscriptionChangePlan{ID: &plan.ID},
	}

	sub, err := c.Subscriptions.Create(t.Context(), subBody)
	c.requireStatus(http.StatusCreated, err)

	t.Run("Should accept next_billing_cycle timing and return 201 with quantity 0", func(t *testing.T) {
		timing := lo.Must(v3sdk.SubscriptionEditTimingFromEnum(v3sdk.SubscriptionEditTimingEnumNextBillingCycle))

		subAddon, err := c.Subscriptions.CreateAddon(t.Context(), sub.ID, v3sdk.CreateSubscriptionAddonRequest{
			Addon:    v3sdk.AddonReference{ID: addon.ID},
			Quantity: 1,
			Timing:   timing,
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, subAddon)

		// Future-active addon: current quantity must be 0 (not an error). The timeline
		// segment carries the requested quantity at its future activation point.
		assert.EqualValues(t, 0, subAddon.Quantity, "current quantity must be 0 for future-active addon")
		require.NotEmpty(t, subAddon.Timeline)
		assert.EqualValues(t, 1, subAddon.Timeline[0].Quantity)
		assert.True(t, subAddon.Timeline[0].ActiveFrom.After(time.Now()), "next_billing_cycle timing must produce a future active_from")
	})
}
