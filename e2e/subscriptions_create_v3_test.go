package e2e

import (
	"net/http"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

// TestV3SubscriptionCreateCustomPlan exercises POST /subscriptions and
// POST /subscriptions/{id}/change with an inline (custom) plan definition instead
// of a published-plan reference. It uses a FIAT currency, so the custom-currency
// feature gate is intentionally not exercised here.
func TestV3SubscriptionCreateCustomPlan(t *testing.T) {
	c := newV3Client(t)

	newCustomer := func(t *testing.T, prefix string) *v3sdk.Customer {
		t.Helper()
		key := uniqueKey(prefix)
		cust, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:          key,
			Name:         "Custom Subscription Test Customer",
			Currency:     lo.ToPtr("USD"),
			PrimaryEmail: lo.ToPtr("test-" + key + "@test.com"),
			UsageAttribution: &v3sdk.CustomerUsageAttribution{
				SubjectKeys: []string{key},
			},
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, cust)
		return cust
	}

	customPlan := func(namePrefix string) *v3sdk.SubscriptionCustomPlan {
		return &v3sdk.SubscriptionCustomPlan{
			Name:           "Inline Plan " + namePrefix,
			Currency:       v3sdk.BillingCurrencyCode("USD"),
			BillingCadence: "P1M",
			Phases:         []v3sdk.PlanPhaseInput{validPlanPhase(namePrefix, true /* isLast */)},
		}
	}

	t.Run("Should create a subscription from an inline plan and return 201", func(t *testing.T) {
		customer := newCustomer(t, "sub_custom_create")

		sub, err := c.Subscriptions.Create(t.Context(), v3sdk.SubscriptionCreate{
			Customer:   v3sdk.SubscriptionChangeCustomer{ID: &customer.ID},
			CustomPlan: customPlan("inline_create"),
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, sub)
		assert.Equal(t, v3sdk.SubscriptionStatusActive, sub.Status)
		// custom (inline) subscriptions are not linked to a persisted plan
		assert.Nil(t, sub.Plan, "inline-plan subscription should not carry a plan reference")
		require.NotEmpty(t, sub.Phases, "expected phases on the created subscription")
	})

	t.Run("Should reject when neither plan nor custom_plan is provided with 400", func(t *testing.T) {
		customer := newCustomer(t, "sub_custom_neither")

		_, err := c.Subscriptions.Create(t.Context(), v3sdk.SubscriptionCreate{
			Customer: v3sdk.SubscriptionChangeCustomer{ID: &customer.ID},
		})
		requireProblem(t, err, http.StatusBadRequest)
	})

	t.Run("Should change a plan-based subscription to an inline plan and return 200", func(t *testing.T) {
		customer := newCustomer(t, "sub_custom_change")

		// Start from a published plan, then change to an inline plan.
		plan, err := c.Plans.Create(t.Context(), validPlanRequest("sub_custom_change_plan"))
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, plan)
		_, err = c.Plans.Publish(t.Context(), plan.ID)
		c.requireStatus(http.StatusOK, err)

		sub, err := c.Subscriptions.Create(t.Context(), v3sdk.SubscriptionCreate{
			Customer: v3sdk.SubscriptionChangeCustomer{ID: &customer.ID},
			Plan:     &v3sdk.SubscriptionChangePlan{ID: &plan.ID},
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, sub)

		timing := lo.Must(v3sdk.SubscriptionEditTimingFromEnum(v3sdk.SubscriptionEditTimingEnumImmediate))
		resp, err := c.Subscriptions.Change(t.Context(), sub.ID, v3sdk.SubscriptionChange{
			Customer:   v3sdk.SubscriptionChangeCustomer{ID: &customer.ID},
			CustomPlan: customPlan("inline_change"),
			Timing:     timing,
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, resp)
		assert.Nil(t, resp.Next.Plan, "changed-to-inline subscription should not carry a plan reference")
	})
}

// TestV3SubscriptionStartingPhase exercises the `starting_phase` field on
// POST /subscriptions and POST /subscriptions/{id}/change. Starting in a later
// phase zeroes the length of every phase before it, so the subscription begins
// directly in the requested phase.
func TestV3SubscriptionStartingPhase(t *testing.T) {
	c := newV3Client(t)

	newCustomer := func(t *testing.T, prefix string) *v3sdk.Customer {
		t.Helper()
		key := uniqueKey(prefix)
		cust, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:          key,
			Name:         "Starting Phase Test Customer",
			Currency:     lo.ToPtr("USD"),
			PrimaryEmail: lo.ToPtr("test-" + key + "@test.com"),
			UsageAttribution: &v3sdk.CustomerUsageAttribution{
				SubjectKeys: []string{key},
			},
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, cust)
		return cust
	}

	// twoPhasePlan builds and publishes a plan with a bounded first phase and a
	// final second phase, returning the published plan and both phase keys.
	twoPhasePlan := func(t *testing.T, prefix string) (*v3sdk.Plan, string, string) {
		t.Helper()
		phase1Key := uniqueKey(prefix + "_p1")
		phase2Key := uniqueKey(prefix + "_p2")

		req := validPlanRequest(prefix)
		req.Phases = []v3sdk.PlanPhaseInput{
			{
				Key:       phase1Key,
				Name:      "Phase 1 " + prefix,
				Duration:  lo.ToPtr("P1M"),
				RateCards: []v3sdk.RateCardInput{validFlatRateCard(prefix + "_fee1")},
			},
			{
				Key:       phase2Key,
				Name:      "Phase 2 " + prefix,
				RateCards: []v3sdk.RateCardInput{validFlatRateCard(prefix + "_fee2")},
			},
		}

		plan, err := c.Plans.Create(t.Context(), req)
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, plan)
		_, err = c.Plans.Publish(t.Context(), plan.ID)
		c.requireStatus(http.StatusOK, err)

		return plan, phase1Key, phase2Key
	}

	// findPhase returns the subscription phase with the given key.
	findPhase := func(t *testing.T, sub *v3sdk.BillingSubscription, key string) v3sdk.SubscriptionPhase {
		t.Helper()
		for _, p := range sub.Phases {
			if p.Key == key {
				return p
			}
		}
		t.Fatalf("phase %q not found on subscription", key)
		return v3sdk.SubscriptionPhase{}
	}

	t.Run("Should start the subscription in the requested phase on create", func(t *testing.T) {
		customer := newCustomer(t, "sub_starting_phase_create")
		plan, phase1Key, phase2Key := twoPhasePlan(t, "starting_phase_create")

		sub, err := c.Subscriptions.Create(t.Context(), v3sdk.SubscriptionCreate{
			Customer:      v3sdk.SubscriptionChangeCustomer{ID: &customer.ID},
			Plan:          &v3sdk.SubscriptionChangePlan{ID: &plan.ID},
			StartingPhase: &phase2Key,
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, sub)

		// The first phase is zeroed, so the starting phase begins at the
		// subscription's active_from.
		phase1 := findPhase(t, sub, phase1Key)
		require.NotNil(t, phase1.ActiveTo, "zeroed phase should be bounded")
		assert.Equal(t, phase1.ActiveFrom, *phase1.ActiveTo, "phase before the starting phase should have zero length")

		phase2 := findPhase(t, sub, phase2Key)
		assert.Equal(t, sub.ActiveFrom, phase2.ActiveFrom, "starting phase should begin at subscription start")
	})

	t.Run("Should reject an unknown starting phase on create with 400", func(t *testing.T) {
		customer := newCustomer(t, "sub_starting_phase_unknown")
		plan, _, _ := twoPhasePlan(t, "starting_phase_unknown")

		_, err := c.Subscriptions.Create(t.Context(), v3sdk.SubscriptionCreate{
			Customer:      v3sdk.SubscriptionChangeCustomer{ID: &customer.ID},
			Plan:          &v3sdk.SubscriptionChangePlan{ID: &plan.ID},
			StartingPhase: lo.ToPtr("does-not-exist"),
		})
		requireProblem(t, err, http.StatusBadRequest)
	})

	t.Run("Should start the subscription in the requested phase on change", func(t *testing.T) {
		customer := newCustomer(t, "sub_starting_phase_change")
		plan, _, _ := twoPhasePlan(t, "starting_phase_change_from")

		sub, err := c.Subscriptions.Create(t.Context(), v3sdk.SubscriptionCreate{
			Customer: v3sdk.SubscriptionChangeCustomer{ID: &customer.ID},
			Plan:     &v3sdk.SubscriptionChangePlan{ID: &plan.ID},
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, sub)

		targetPlan, _, targetPhase2Key := twoPhasePlan(t, "starting_phase_change_to")
		timing := lo.Must(v3sdk.SubscriptionEditTimingFromEnum(v3sdk.SubscriptionEditTimingEnumImmediate))

		resp, err := c.Subscriptions.Change(t.Context(), sub.ID, v3sdk.SubscriptionChange{
			Customer:      v3sdk.SubscriptionChangeCustomer{ID: &customer.ID},
			Plan:          &v3sdk.SubscriptionChangePlan{ID: &targetPlan.ID},
			StartingPhase: &targetPhase2Key,
			Timing:        timing,
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, resp)

		phase2 := findPhase(t, &resp.Next, targetPhase2Key)
		assert.Equal(t, resp.Next.ActiveFrom, phase2.ActiveFrom, "starting phase should begin at the changed subscription's start")
	})
}
