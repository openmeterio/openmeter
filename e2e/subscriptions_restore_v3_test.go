package e2e

import (
	"net/http"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

// TestV3SubscriptionRestore exercises POST /subscriptions/{id}/restore, which
// deletes any later-scheduled successor subscription and continues the current
// one indefinitely — the inverse of a future-dated change. Restore is not
// available for namespaces with multi-subscription enabled; the e2e namespace is
// single-subscription, so the happy path applies here.
func TestV3SubscriptionRestore(t *testing.T) {
	c := newV3Client(t)

	newCustomer := func(t *testing.T, prefix string) *v3sdk.Customer {
		t.Helper()
		key := uniqueKey(prefix)
		cust, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:          key,
			Name:         "Restore Test Customer",
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

	t.Run("Should restore after a future-dated change, deleting the scheduled successor", func(t *testing.T) {
		// given: an active subscription that has been changed at a future time,
		// scheduling a not-yet-active successor.
		customer := newCustomer(t, "sub_restore_ok")
		planA := publishedPlan(t, "restore_ok_a")
		planB := publishedPlan(t, "restore_ok_b")

		sub, err := c.Subscriptions.Create(t.Context(), v3sdk.SubscriptionCreate{
			Customer: v3sdk.SubscriptionChangeCustomer{ID: &customer.ID},
			Plan:     &v3sdk.SubscriptionChangePlan{ID: &planA.ID},
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, sub)

		// next_billing_cycle keeps the change aligned with the billing period (an
		// arbitrary custom instant is rejected when canceling an aligned
		// subscription); the successor is scheduled at the upcoming cycle boundary.
		nextCycle := lo.Must(v3sdk.SubscriptionEditTimingFromEnum(v3sdk.SubscriptionEditTimingEnumNextBillingCycle))
		resp, err := c.Subscriptions.Change(t.Context(), sub.ID, v3sdk.SubscriptionChange{
			Customer: v3sdk.SubscriptionChangeCustomer{ID: &customer.ID},
			Plan:     &v3sdk.SubscriptionChangePlan{ID: &planB.ID},
			Timing:   nextCycle,
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, resp)
		successor := resp.Next
		require.Equal(t, v3sdk.SubscriptionStatusScheduled, successor.Status, "successor should be scheduled")

		// when: restoring the current subscription.
		restored, err := c.Subscriptions.Restore(t.Context(), resp.Current.ID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, restored)

		// then: the current subscription continues indefinitely and the scheduled
		// successor is gone.
		assert.Equal(t, sub.ID, restored.ID, "restore should continue the current subscription")
		assert.Equal(t, v3sdk.SubscriptionStatusActive, restored.Status)
		assert.Nil(t, restored.ActiveTo, "restored subscription should have no scheduled end")

		_, err = c.Subscriptions.Get(t.Context(), successor.ID)
		requireProblem(t, err, http.StatusNotFound)
	})

	t.Run("Should return 404 for an unknown subscription id", func(t *testing.T) {
		_, err := c.Subscriptions.Restore(t.Context(), "01JAAAAAAAAAAAAAAAAAAAAAAA")
		requireProblem(t, err, http.StatusNotFound)
	})
}
