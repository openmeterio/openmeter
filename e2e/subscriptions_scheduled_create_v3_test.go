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

// TestV3SubscriptionScheduledCreate exercises the `timing` field on
// POST /subscriptions. A future timing schedules a not-yet-active subscription;
// an omitted timing creates immediately (unchanged default); a past timing is
// rejected with 400 rather than 500.
func TestV3SubscriptionScheduledCreate(t *testing.T) {
	c := newV3Client(t)

	newCustomer := func(t *testing.T, prefix string) *v3sdk.Customer {
		t.Helper()
		key := uniqueKey(prefix)
		cust, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:          key,
			Name:         "Scheduled Create Test Customer",
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

	t.Run("Should create a scheduled subscription with a future timing", func(t *testing.T) {
		// given: a customer and a published plan
		customer := newCustomer(t, "sub_sched_create")
		plan := publishedPlan(t, "sched_create")

		// when: creating a subscription with a future timing
		futureTiming := lo.Must(v3sdk.SubscriptionCreateTimingFromCustom(time.Now().Add(48 * time.Hour)))
		sub, err := c.Subscriptions.Create(t.Context(), v3sdk.SubscriptionCreate{
			Customer: v3sdk.SubscriptionChangeCustomer{ID: &customer.ID},
			Plan:     &v3sdk.SubscriptionChangePlan{ID: &plan.ID},
			Timing:   &futureTiming,
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, sub)

		// then: the subscription is scheduled to start in the future
		assert.Equal(t, v3sdk.SubscriptionStatusScheduled, sub.Status, "future-timed create should be scheduled")
		assert.True(t, sub.ActiveFrom.After(time.Now()), "scheduled subscription should start in the future")
	})

	t.Run("Should create immediately when timing is omitted", func(t *testing.T) {
		// given: a customer and a published plan
		customer := newCustomer(t, "sub_immediate_create")
		plan := publishedPlan(t, "immediate_create")

		// when: creating a subscription without a timing
		sub, err := c.Subscriptions.Create(t.Context(), v3sdk.SubscriptionCreate{
			Customer: v3sdk.SubscriptionChangeCustomer{ID: &customer.ID},
			Plan:     &v3sdk.SubscriptionChangePlan{ID: &plan.ID},
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, sub)

		// then: the subscription starts immediately (active)
		assert.Equal(t, v3sdk.SubscriptionStatusActive, sub.Status, "create without timing should be immediate")
	})

	t.Run("Should reject a past timing with 400", func(t *testing.T) {
		// given: a customer and a published plan
		customer := newCustomer(t, "sub_past_create")
		plan := publishedPlan(t, "past_create")

		// when: creating a subscription with a timing in the past
		pastTiming := lo.Must(v3sdk.SubscriptionCreateTimingFromCustom(time.Now().Add(-48 * time.Hour)))
		_, err := c.Subscriptions.Create(t.Context(), v3sdk.SubscriptionCreate{
			Customer: v3sdk.SubscriptionChangeCustomer{ID: &customer.ID},
			Plan:     &v3sdk.SubscriptionChangePlan{ID: &plan.ID},
			Timing:   &pastTiming,
		})

		// then: the request is rejected
		requireProblem(t, err, http.StatusBadRequest)
	})
}
