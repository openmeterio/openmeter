package e2e

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

// TestV3GetCustomerEntitlementAccess covers GET
// /customers/{customerId}/entitlement-access/features/{featureKey}. The
// subscriber is provisioned through the v3-only plan path so the metered
// entitlement materializes exactly as a v3 consumer would create it.
func TestV3GetCustomerEntitlementAccess(t *testing.T) {
	c := newV3Client(t)

	const limit = 15_000_000

	feature := createMeteredFeature(t, c, "ent_access")

	planKey := uniqueKey("ent_access_plan")
	plan, err := c.Plans.Create(t.Context(), v3sdk.CreatePlanRequest{
		Key:            planKey,
		Name:           "Entitlement Access Plan",
		Currency:       "USD",
		BillingCadence: "P1M",
		Phases: []v3sdk.PlanPhaseInput{{
			Key:       "subscription",
			Name:      "Subscription",
			RateCards: []v3sdk.RateCardInput{meteredEntitlementRateCard(feature, limit)},
		}},
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, plan)

	_, err = c.Plans.Publish(t.Context(), plan.ID)
	c.requireStatus(http.StatusOK, err)

	customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
		Key:  uniqueKey("ent_access_cust"),
		Name: "Entitlement Access Customer",
		UsageAttribution: &v3sdk.CustomerUsageAttribution{
			SubjectKeys: []string{uniqueKey("ent_access_subj")},
		},
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, customer)

	var subBody v3sdk.SubscriptionCreate
	subBody.Customer.Key = lo.ToPtr(customer.Key)
	subBody.Plan = &v3sdk.SubscriptionChangePlan{
		Key:     lo.ToPtr(planKey),
		Version: lo.ToPtr(int64(1)),
	}

	sub, err := c.Subscriptions.Create(t.Context(), subBody)
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, sub)

	t.Run("Should return the metered entitlement without balance details by default", func(t *testing.T) {
		access, err := c.Entitlements.GetCustomerAccess(t.Context(), customer.ID, feature.Key, v3sdk.GetCustomerEntitlementAccessParams{})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, access)

		assert.Equal(t, feature.Key, access.FeatureKey)
		assert.Equal(t, v3sdk.EntitlementTypeMetered, access.Type)
		assert.True(t, access.HasAccess)
		assert.Nil(t, access.Value)
	})

	t.Run("Should expand the balance details", func(t *testing.T) {
		access, err := c.Entitlements.GetCustomerAccess(t.Context(), customer.ID, feature.Key, v3sdk.GetCustomerEntitlementAccessParams{
			Expand: []v3sdk.EntitlementAccessExpand{v3sdk.EntitlementAccessExpandValue},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, access)
		require.NotNil(t, access.Value)

		balance, err := strconv.ParseFloat(access.Value.Balance, 64)
		require.NoError(t, err)
		assert.Equal(t, float64(limit), balance)

		assert.Equal(t, v3sdk.Numeric("0"), access.Value.Usage)
		assert.Equal(t, v3sdk.Numeric("0"), access.Value.Overage)
		assert.Len(t, access.Value.GrantBalances, 1)
	})

	t.Run("Should return no access for a feature without an entitlement", func(t *testing.T) {
		featureKey := uniqueKey("ent_access_missing")

		access, err := c.Entitlements.GetCustomerAccess(t.Context(), customer.ID, featureKey, v3sdk.GetCustomerEntitlementAccessParams{})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, access)

		assert.Equal(t, featureKey, access.FeatureKey)
		assert.False(t, access.HasAccess)
		assert.Nil(t, access.Value)
	})

	t.Run("Should return 404 for an unknown customer", func(t *testing.T) {
		_, err := c.Entitlements.GetCustomerAccess(t.Context(), ulid.Make().String(), feature.Key, v3sdk.GetCustomerEntitlementAccessParams{})
		requireProblem(t, err, http.StatusNotFound)
	})

	t.Run("Should return 412 for a deleted customer", func(t *testing.T) {
		deleted, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:  uniqueKey("ent_access_deleted"),
			Name: "Deleted Customer",
			UsageAttribution: &v3sdk.CustomerUsageAttribution{
				SubjectKeys: []string{uniqueKey("ent_access_deleted_subj")},
			},
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, deleted)

		err = c.Customers.Delete(t.Context(), deleted.ID)
		c.requireStatus(http.StatusNoContent, err)

		_, err = c.Entitlements.GetCustomerAccess(t.Context(), deleted.ID, feature.Key, v3sdk.GetCustomerEntitlementAccessParams{})
		requireProblem(t, err, http.StatusPreconditionFailed)
	})
}
