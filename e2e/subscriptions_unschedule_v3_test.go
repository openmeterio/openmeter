package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

// TestV3SubscriptionUnschedule exercises POST /subscriptions/{id}/unschedule,
// which deletes a scheduled (not-yet-active) subscription. The scheduled target
// is produced directly via a future-timed create. Deleting a running
// subscription is rejected with 403 (the domain state machine forbids the delete
// transition outside the scheduled state), and an unknown id returns 404.
func TestV3SubscriptionUnschedule(t *testing.T) {
	c := newV3Client(t)

	newCustomer := func(t *testing.T, prefix string) *v3sdk.Customer {
		t.Helper()
		key := uniqueKey(prefix)
		cust, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:          key,
			Name:         "Unschedule Test Customer",
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

	publishedPlan := func(t *testing.T, prefix string) *v3sdk.Plan {
		t.Helper()
		plan, err := c.Plans.Create(t.Context(), validPlanRequest(prefix))
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, plan)
		_, err = c.Plans.Publish(t.Context(), plan.ID)
		c.requireStatus(http.StatusOK, err)
		return plan
	}

	t.Run("Should unschedule a scheduled subscription and return 204, leaving it gone", func(t *testing.T) {
		// given: a scheduled (not-yet-active) subscription created with a future timing.
		customer := newCustomer(t, "sub_unschedule_ok")
		plan := publishedPlan(t, "unschedule_ok")

		future := lo.Must(v3sdk.SubscriptionCreateTimingFromCustom(time.Now().Add(48 * time.Hour)))
		sub, err := c.Subscriptions.Create(t.Context(), v3sdk.SubscriptionCreate{
			Customer: v3sdk.SubscriptionChangeCustomer{ID: &customer.ID},
			Plan:     &v3sdk.SubscriptionChangePlan{ID: &plan.ID},
			Timing:   &future,
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, sub)
		require.Equal(t, v3sdk.SubscriptionStatusScheduled, sub.Status, "subscription should be scheduled")

		// when/then: the scheduled subscription can be unscheduled (204) and is gone.
		c.requireStatus(http.StatusNoContent, c.Subscriptions.Unschedule(t.Context(), sub.ID))

		_, err = c.Subscriptions.Get(t.Context(), sub.ID)
		requireProblem(t, err, http.StatusNotFound)
	})

	t.Run("Should reject unscheduling a running subscription with 403", func(t *testing.T) {
		// given: a running (active) subscription
		customer := newCustomer(t, "sub_unschedule_active")
		plan := publishedPlan(t, "unschedule_active")

		sub, err := c.Subscriptions.Create(t.Context(), v3sdk.SubscriptionCreate{
			Customer: v3sdk.SubscriptionChangeCustomer{ID: &customer.ID},
			Plan:     &v3sdk.SubscriptionChangePlan{ID: &plan.ID},
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, sub)
		require.Equal(t, v3sdk.SubscriptionStatusActive, sub.Status)

		// then: unscheduling a running subscription is forbidden (it is not
		// scheduled) — distinct from cancel.
		requireProblem(t, c.Subscriptions.Unschedule(t.Context(), sub.ID), http.StatusForbidden)
	})

	t.Run("Should return 404 for an unknown subscription id", func(t *testing.T) {
		requireProblem(t, c.Subscriptions.Unschedule(t.Context(), "01JAAAAAAAAAAAAAAAAAAAAAAA"), http.StatusNotFound)
	})
}
