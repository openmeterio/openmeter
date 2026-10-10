package e2e

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

func TestV3GetCustomerEntitlementValue(t *testing.T) {
	c := newV3Client(t)
	v1 := initClient(t)

	const limit = 15_000_000

	feature := createMeteredFeature(t, c, "ent_value")

	planKey := uniqueKey("ent_value_plan")
	plan, err := c.Plans.Create(t.Context(), v3sdk.CreatePlanRequest{
		Key:            planKey,
		Name:           "Entitlement Value Plan",
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
		Key:  uniqueKey("ent_value_cust"),
		Name: "Entitlement Value Customer",
		UsageAttribution: &v3sdk.CustomerUsageAttribution{
			SubjectKeys: []string{uniqueKey("ent_value_subj")},
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

	listResp, err := v1.ListEntitlementsWithResponse(t.Context(), &api.ListEntitlementsParams{
		Feature:  &[]string{feature.Key},
		Page:     lo.ToPtr(1),
		PageSize: lo.ToPtr(10),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, listResp.StatusCode())
	require.NotNil(t, listResp.JSON200)

	listed, err := listResp.JSON200.AsEntitlementPaginatedResponse()
	require.NoError(t, err)
	require.Len(t, listed.Items, 1)

	metered, err := listed.Items[0].AsEntitlementMetered()
	require.NoError(t, err)
	entitlementID := metered.Id

	t.Run("Should return the metered entitlement without balance details by default", func(t *testing.T) {
		// given a subscribed customer with a metered entitlement
		// when its value is requested without expansion
		// then access is granted without balance details
		access, err := c.Entitlements.GetCustomerValue(t.Context(), customer.ID, entitlementID, v3sdk.GetCustomerEntitlementValueParams{})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, access)

		assert.Equal(t, feature.Key, access.FeatureKey)
		assert.Equal(t, v3sdk.EntitlementTypeMetered, access.Type)
		assert.True(t, access.HasAccess)
		assert.Nil(t, access.Value)
	})

	t.Run("Should expand the balance details", func(t *testing.T) {
		// given a metered entitlement with its initial grant
		// when the value expansion is requested
		// then the balance and usage details are returned
		access, err := c.Entitlements.GetCustomerValue(t.Context(), customer.ID, entitlementID, v3sdk.GetCustomerEntitlementValueParams{
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

	t.Run("Should return the metered value by feature key", func(t *testing.T) {
		// given a subscribed customer with a metered entitlement
		// when its value is requested by feature key with expansion
		// then the same entitlement type and balance are returned
		access, err := c.Entitlements.GetCustomerValueByFeatureKey(t.Context(), customer.ID, feature.Key, v3sdk.GetCustomerEntitlementValueByFeatureKeyParams{
			Expand: []v3sdk.EntitlementAccessExpand{v3sdk.EntitlementAccessExpandValue},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, access)
		require.NotNil(t, access.Value)

		assert.Equal(t, feature.Key, access.FeatureKey)
		assert.Equal(t, lo.ToPtr(v3sdk.EntitlementTypeMetered), access.Type)
		assert.True(t, access.HasAccess)

		balance, err := strconv.ParseFloat(access.Value.Balance, 64)
		require.NoError(t, err)
		assert.Equal(t, float64(limit), balance)
	})

	t.Run("Should return no access for a feature without an entitlement", func(t *testing.T) {
		// given a feature key with no active entitlement
		// when its value is requested
		// then the response denies access without a type or value
		featureKey := uniqueKey("ent_value_missing")
		access, err := c.Entitlements.GetCustomerValueByFeatureKey(t.Context(), customer.ID, featureKey, v3sdk.GetCustomerEntitlementValueByFeatureKeyParams{
			Expand: []v3sdk.EntitlementAccessExpand{v3sdk.EntitlementAccessExpandValue},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, access)

		assert.Equal(t, featureKey, access.FeatureKey)
		assert.False(t, access.HasAccess)
		assert.Nil(t, access.Type)
		assert.Nil(t, access.Value)
	})

	t.Run("Should return no access by feature key before activation", func(t *testing.T) {
		// given an entitlement that was not active yesterday
		// when its earlier value is requested by feature key
		// then no entitlement is selected at that time
		access, err := c.Entitlements.GetCustomerValueByFeatureKey(t.Context(), customer.ID, feature.Key, v3sdk.GetCustomerEntitlementValueByFeatureKeyParams{
			At: lo.ToPtr(time.Now().Add(-24 * time.Hour)),
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, access)

		assert.Equal(t, feature.Key, access.FeatureKey)
		assert.False(t, access.HasAccess)
		assert.Nil(t, access.Type)
	})

	t.Run("Should return no access when evaluated before the entitlement became active", func(t *testing.T) {
		// given an entitlement created after the requested evaluation time
		// when its earlier value is requested
		// then the customer has no access at that time
		access, err := c.Entitlements.GetCustomerValue(t.Context(), customer.ID, entitlementID, v3sdk.GetCustomerEntitlementValueParams{
			At: lo.ToPtr(time.Now().Add(-24 * time.Hour)),
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, access)

		assert.Equal(t, feature.Key, access.FeatureKey)
		assert.False(t, access.HasAccess)
		assert.Nil(t, access.Value)
	})

	t.Run("Should return 404 for an unknown entitlement", func(t *testing.T) {
		// given an entitlement ID that does not exist
		// when the customer requests its value
		// then the endpoint reports not found
		_, err := c.Entitlements.GetCustomerValue(t.Context(), customer.ID, ulid.Make().String(), v3sdk.GetCustomerEntitlementValueParams{})
		requireProblem(t, err, http.StatusNotFound)
	})

	t.Run("Should return 404 for an unknown customer", func(t *testing.T) {
		// given a customer ID that does not exist
		// when it requests a known entitlement value
		// then the endpoint reports not found
		_, err := c.Entitlements.GetCustomerValue(t.Context(), ulid.Make().String(), entitlementID, v3sdk.GetCustomerEntitlementValueParams{})
		requireProblem(t, err, http.StatusNotFound)
	})

	t.Run("Should return 404 for another customer's entitlement", func(t *testing.T) {
		// given another customer that does not own this entitlement
		// when it requests the entitlement value by ID
		// then the endpoint conceals the other customer's resource
		other, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:  uniqueKey("ent_value_other"),
			Name: "Other Customer",
			UsageAttribution: &v3sdk.CustomerUsageAttribution{
				SubjectKeys: []string{uniqueKey("ent_value_other_subj")},
			},
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, other)

		_, err = c.Entitlements.GetCustomerValue(t.Context(), other.ID, entitlementID, v3sdk.GetCustomerEntitlementValueParams{})
		requireProblem(t, err, http.StatusNotFound)
	})

	t.Run("Should return 409 for a deleted customer", func(t *testing.T) {
		// given a customer that is subsequently deleted
		// when its entitlement value is requested
		// then the endpoint reports the customer's conflicting state
		deleted, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:  uniqueKey("ent_value_deleted"),
			Name: "Deleted Customer",
			UsageAttribution: &v3sdk.CustomerUsageAttribution{
				SubjectKeys: []string{uniqueKey("ent_value_deleted_subj")},
			},
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, deleted)

		err = c.Customers.Delete(t.Context(), deleted.ID)
		c.requireStatus(http.StatusNoContent, err)

		_, err = c.Entitlements.GetCustomerValue(t.Context(), deleted.ID, entitlementID, v3sdk.GetCustomerEntitlementValueParams{})
		requireProblem(t, err, http.StatusConflict)
	})

	t.Run("Should return 404 for a deleted entitlement at any time", func(t *testing.T) {
		// given a standalone boolean entitlement, as subscription-managed ones cannot be deleted
		// when it is deleted after activation
		// then its ID is not found even at a time before deletion
		featureKey := uniqueKey("ent_value_del")
		deletedFeature, err := c.Features.Create(t.Context(), v3sdk.CreateFeatureRequest{
			Key:  featureKey,
			Name: "Deleted Feature " + featureKey,
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, deletedFeature)

		var createBody api.CreateCustomerEntitlementV2JSONRequestBody
		require.NoError(t, createBody.FromEntitlementBooleanCreateInputs(api.EntitlementBooleanCreateInputs{
			Type:       api.EntitlementBooleanCreateInputsTypeBoolean,
			FeatureKey: lo.ToPtr(deletedFeature.Key),
		}))

		createResp, err := v1.CreateCustomerEntitlementV2WithResponse(t.Context(), customer.ID, createBody)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, createResp.StatusCode(), "unexpected response body: %s", createResp.Body)

		created, err := createResp.JSON201.AsEntitlementBooleanV2()
		require.NoError(t, err)

		beforeDeletion := time.Now()
		require.True(t, beforeDeletion.After(created.ActiveFrom), "entitlement must be active before the deletion")

		deleteResp, err := v1.DeleteCustomerEntitlementV2WithResponse(t.Context(), customer.ID, deletedFeature.Key)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, deleteResp.StatusCode(), "unexpected response body: %s", deleteResp.Body)

		_, err = c.Entitlements.GetCustomerValue(t.Context(), customer.ID, created.Id, v3sdk.GetCustomerEntitlementValueParams{
			At: lo.ToPtr(beforeDeletion),
		})
		requireProblem(t, err, http.StatusNotFound)

		_, err = c.Entitlements.GetCustomerValue(t.Context(), customer.ID, created.Id, v3sdk.GetCustomerEntitlementValueParams{})
		requireProblem(t, err, http.StatusNotFound)
	})
}
