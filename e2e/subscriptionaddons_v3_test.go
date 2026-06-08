package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
)

// TestV3SubscriptionAddonAttach exercises POST /subscriptions/{id}/addons end to end:
// build a published plan + published addon, create a subscription, attach the addon,
// verify the response shape (rate_cards/timeline arrays, never null), then confirm
// the conflict path returns 409 when the same addon is attached twice.
func TestV3SubscriptionAddonAttach(t *testing.T) {
	c := newV3Client(t)

	// --- Fixture: customer ---

	customerKey := uniqueKey("sub_addon_customer")
	custStatus, customer, custProblem := c.CreateCustomer(apiv3.CreateCustomerRequest{
		Key:          customerKey,
		Name:         "Subscription Addon Test Customer",
		Currency:     lo.ToPtr(apiv3.CurrencyCode("USD")),
		PrimaryEmail: lo.ToPtr("test-" + customerKey + "@test.com"),
		UsageAttribution: &apiv3.BillingCustomerUsageAttribution{
			SubjectKeys: []string{customerKey},
		},
	})
	require.Equal(t, http.StatusCreated, custStatus, "problem: %+v", custProblem)
	require.NotNil(t, customer)

	// --- Fixture: draft plan + published addon, attach addon, then publish plan ---
	// Order matters: addons can only be attached to a plan while it is still in draft,
	// and the addon must be published before attach.

	planBody := validPlanRequest("sub_addon_plan")
	planStatus, plan, planProblem := c.CreatePlan(planBody)
	require.Equal(t, http.StatusCreated, planStatus, "problem: %+v", planProblem)
	require.NotNil(t, plan)
	require.NotEmpty(t, plan.Phases, "plan must have at least one phase to attach an addon")

	addonBody := validAddonRequest("sub_addon")
	addonStatus, addon, addonProblem := c.CreateAddon(addonBody)
	require.Equal(t, http.StatusCreated, addonStatus, "problem: %+v", addonProblem)
	require.NotNil(t, addon)

	pubAddonStatus, _, pubAddonProblem := c.PublishAddon(addon.Id)
	require.Equal(t, http.StatusOK, pubAddonStatus, "problem: %+v", pubAddonProblem)

	attachStatus, _, attachProblem := c.AttachAddon(plan.Id, validPlanAddonRequest(plan.Phases[0].Key, addon.Id))
	require.Equal(t, http.StatusCreated, attachStatus, "problem: %+v", attachProblem)

	pubStatus, _, pubProblem := c.PublishPlan(plan.Id)
	require.Equal(t, http.StatusOK, pubStatus, "problem: %+v", pubProblem)

	// --- Fixture: subscription on the published plan ---

	subBody := apiv3.BillingSubscriptionCreate{}
	subBody.Customer.Id = &customer.Id
	subBody.Plan.Id = &plan.Id

	subStatus, sub, subProblem := c.CreateSubscription(subBody)
	require.Equal(t, http.StatusCreated, subStatus, "problem: %+v", subProblem)
	require.NotNil(t, sub)
	subscriptionID := sub.Id

	// --- Test: attach addon ---

	var subAddonID string

	t.Run("Should attach addon with immediate timing and return 201", func(t *testing.T) {
		var timing apiv3.BillingSubscriptionEditTiming
		require.NoError(t, timing.FromBillingSubscriptionEditTimingEnum(apiv3.BillingSubscriptionEditTimingEnum("immediate")))

		status, subAddon, problem := c.CreateSubscriptionAddon(subscriptionID, apiv3.CreateSubscriptionAddonRequest{
			Addon:    apiv3.AddonReference{Id: addon.Id},
			Quantity: 1,
			Timing:   timing,
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, subAddon)

		assert.NotEmpty(t, subAddon.Id)
		assert.Equal(t, addon.Id, subAddon.Addon.Id)
		assert.Equal(t, 1, subAddon.Quantity)
		// Regression guard for the nil-slice → JSON null bug: rate_cards must be a non-nil array
		// and every entry's affected_subscription_item_ids must be a non-nil array too.
		assert.NotNil(t, subAddon.RateCards, "rate_cards must not be null")
		for i, rc := range subAddon.RateCards {
			assert.NotNil(t, rc.AffectedSubscriptionItemIds, "rate_cards[%d].affected_subscription_item_ids must not be null", i)
		}
		// Timeline must be a non-nil array with at least one segment for an active addon.
		require.NotNil(t, subAddon.Timeline)
		require.NotEmpty(t, subAddon.Timeline)
		assert.Equal(t, 1, subAddon.Timeline[0].Quantity)

		subAddonID = subAddon.Id
	})

	t.Run("Should return 409 when attaching the same addon twice", func(t *testing.T) {
		require.NotEmpty(t, subAddonID, "first attach must have succeeded")

		var timing apiv3.BillingSubscriptionEditTiming
		require.NoError(t, timing.FromBillingSubscriptionEditTimingEnum(apiv3.BillingSubscriptionEditTimingEnum("immediate")))

		status, _, problem := c.CreateSubscriptionAddon(subscriptionID, apiv3.CreateSubscriptionAddonRequest{
			Addon:    apiv3.AddonReference{Id: addon.Id},
			Quantity: 1,
			Timing:   timing,
		})
		require.Equal(t, http.StatusConflict, status, "expected 409, got %d (problem: %+v)", status, problem)
	})

	t.Run("Should reject invalid quantity 0", func(t *testing.T) {
		var timing apiv3.BillingSubscriptionEditTiming
		require.NoError(t, timing.FromBillingSubscriptionEditTimingEnum(apiv3.BillingSubscriptionEditTimingEnum("immediate")))

		status, _, _ := c.CreateSubscriptionAddon(subscriptionID, apiv3.CreateSubscriptionAddonRequest{
			Addon:    apiv3.AddonReference{Id: addon.Id},
			Quantity: 0,
			Timing:   timing,
		})
		// TypeSpec @minValue(1) rejects this at schema-validation; workflow validation
		// would also reject it. Either is fine — assert 4xx.
		assert.GreaterOrEqual(t, status, http.StatusBadRequest, "expected 4xx for quantity=0, got %d", status)
		assert.Less(t, status, http.StatusInternalServerError, "expected 4xx not 5xx for quantity=0")
	})

	t.Run("Should get the attached addon and surface rate_cards / timeline arrays", func(t *testing.T) {
		require.NotEmpty(t, subAddonID)

		status, subAddon, problem := c.GetSubscriptionAddon(subscriptionID, subAddonID)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, subAddon)

		assert.Equal(t, subAddonID, subAddon.Id)
		assert.NotNil(t, subAddon.RateCards, "GET: rate_cards must not be null")
		assert.NotNil(t, subAddon.Timeline, "GET: timeline must not be null")
		for i, rc := range subAddon.RateCards {
			assert.NotNil(t, rc.AffectedSubscriptionItemIds, "GET: rate_cards[%d].affected_subscription_item_ids must not be null", i)
		}
	})

	t.Run("Should list subscription addons and find the attached addon", func(t *testing.T) {
		require.NotEmpty(t, subAddonID)

		status, page, problem := c.ListSubscriptionAddons(subscriptionID, withPageSize(100))
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, page)

		found := false
		for _, sa := range page.Data {
			if sa.Id == subAddonID {
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
	custStatus, customer, custProblem := c.CreateCustomer(apiv3.CreateCustomerRequest{
		Key:          customerKey,
		Name:         "Next Billing Cycle Test Customer",
		Currency:     lo.ToPtr(apiv3.CurrencyCode("USD")),
		PrimaryEmail: lo.ToPtr("test-" + customerKey + "@test.com"),
		UsageAttribution: &apiv3.BillingCustomerUsageAttribution{
			SubjectKeys: []string{customerKey},
		},
	})
	require.Equal(t, http.StatusCreated, custStatus, "problem: %+v", custProblem)

	// --- Fixture: draft plan + published addon, attach, then publish plan ---

	planStatus, plan, planProblem := c.CreatePlan(validPlanRequest("sub_addon_nbc_plan"))
	require.Equal(t, http.StatusCreated, planStatus, "problem: %+v", planProblem)
	require.NotEmpty(t, plan.Phases)

	addonStatus, addon, addonProblem := c.CreateAddon(validAddonRequest("sub_addon_nbc"))
	require.Equal(t, http.StatusCreated, addonStatus, "problem: %+v", addonProblem)

	pubAddonStatus, _, pubAddonProblem := c.PublishAddon(addon.Id)
	require.Equal(t, http.StatusOK, pubAddonStatus, "problem: %+v", pubAddonProblem)

	attachStatus, _, attachProblem := c.AttachAddon(plan.Id, validPlanAddonRequest(plan.Phases[0].Key, addon.Id))
	require.Equal(t, http.StatusCreated, attachStatus, "problem: %+v", attachProblem)

	pubPlanStatus, _, pubPlanProblem := c.PublishPlan(plan.Id)
	require.Equal(t, http.StatusOK, pubPlanStatus, "problem: %+v", pubPlanProblem)

	// Anchor the subscription at a past second so the next billing cycle is reliably
	// in the future at the moment of the addon attach.
	anchor := time.Now().Add(-time.Second)
	subBody := apiv3.BillingSubscriptionCreate{BillingAnchor: &anchor}
	subBody.Customer.Id = &customer.Id
	subBody.Plan.Id = &plan.Id

	subStatus, sub, subProblem := c.CreateSubscription(subBody)
	require.Equal(t, http.StatusCreated, subStatus, "problem: %+v", subProblem)

	t.Run("Should accept next_billing_cycle timing and return 201 with quantity 0", func(t *testing.T) {
		var timing apiv3.BillingSubscriptionEditTiming
		require.NoError(t, timing.FromBillingSubscriptionEditTimingEnum(apiv3.BillingSubscriptionEditTimingEnum("next_billing_cycle")))

		status, subAddon, problem := c.CreateSubscriptionAddon(sub.Id, apiv3.CreateSubscriptionAddonRequest{
			Addon:    apiv3.AddonReference{Id: addon.Id},
			Quantity: 1,
			Timing:   timing,
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, subAddon)

		// Future-active addon: current quantity must be 0 (not an error). The timeline
		// segment carries the requested quantity at its future activation point.
		assert.Equal(t, 0, subAddon.Quantity, "current quantity must be 0 for future-active addon")
		require.NotEmpty(t, subAddon.Timeline)
		assert.Equal(t, 1, subAddon.Timeline[0].Quantity)
		assert.True(t, subAddon.Timeline[0].ActiveFrom.After(time.Now()), "next_billing_cycle timing must produce a future active_from")
	})
}
