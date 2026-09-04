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
