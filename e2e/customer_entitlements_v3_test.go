package e2e

import (
	"net/http"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

func TestV3DeleteCustomerEntitlement(t *testing.T) {
	c := newV3Client(t)
	v1 := initClient(t)

	customerKey := uniqueKey("ent_delete_customer")
	cust, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
		Key:  customerKey,
		Name: "Entitlement Customer " + customerKey,
		UsageAttribution: &v3sdk.CustomerUsageAttribution{
			SubjectKeys: []string{customerKey},
		},
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, cust)

	featureKey := uniqueKey("ent_delete_boolean")
	f, err := c.Features.Create(t.Context(), v3sdk.CreateFeatureRequest{
		Key:  featureKey,
		Name: "Boolean Feature " + featureKey,
	})
	c.requireStatus(http.StatusCreated, err)

	// The v3 create endpoint is not available yet, so the entitlement is seeded
	// through the legacy v2 customer entitlement endpoint.
	createBooleanEntitlement := func(t *testing.T, customerID string) string {
		t.Helper()

		var body api.CreateCustomerEntitlementV2JSONRequestBody
		require.NoError(t, body.FromEntitlementBooleanCreateInputs(api.EntitlementBooleanCreateInputs{
			Type:      api.EntitlementBooleanCreateInputsTypeBoolean,
			FeatureId: lo.ToPtr(f.ID),
		}))

		res, err := v1.CreateCustomerEntitlementV2WithResponse(t.Context(), customerID, body)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, res.StatusCode(), "Invalid status code [response_body=%s]", string(res.Body))

		created, err := res.JSON201.AsEntitlementBooleanV2()
		require.NoError(t, err)

		return created.Id
	}

	entitlementID := createBooleanEntitlement(t, cust.ID)

	t.Run("delete entitlement of another customer", func(t *testing.T) {
		otherKey := uniqueKey("ent_delete_other_customer")
		other, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:  otherKey,
			Name: "Other Customer " + otherKey,
			UsageAttribution: &v3sdk.CustomerUsageAttribution{
				SubjectKeys: []string{otherKey},
			},
		})
		c.requireStatus(http.StatusCreated, err)

		otherEntitlementID := createBooleanEntitlement(t, other.ID)

		err = c.Customers.Entitlements.Delete(t.Context(), cust.ID, otherEntitlementID)
		requireProblem(t, err, http.StatusNotFound)

		// The other customer's entitlement is left untouched.
		res, err := v1.GetCustomerEntitlementV2WithResponse(t.Context(), other.ID, otherEntitlementID)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, res.StatusCode(), "Invalid status code [response_body=%s]", string(res.Body))
	})

	t.Run("delete unknown entitlement", func(t *testing.T) {
		err := c.Customers.Entitlements.Delete(t.Context(), cust.ID, "01K4WAQ0J99ZZ0MD75HXR112H9")
		requireProblem(t, err, http.StatusNotFound)
	})

	t.Run("delete", func(t *testing.T) {
		c.requireStatus(http.StatusNoContent, c.Customers.Entitlements.Delete(t.Context(), cust.ID, entitlementID))

		res, err := v1.GetCustomerEntitlementV2WithResponse(t.Context(), cust.ID, entitlementID)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, res.StatusCode(), "Invalid status code [response_body=%s]", string(res.Body))

		// A soft-deleted entitlement is reported as not found on a repeated delete.
		err = c.Customers.Entitlements.Delete(t.Context(), cust.ID, entitlementID)
		requireProblem(t, err, http.StatusNotFound)
	})

	t.Run("unknown customer", func(t *testing.T) {
		err := c.Customers.Entitlements.Delete(t.Context(), "01K4WAQ0J99ZZ0MD75HXR112H8", entitlementID)
		requireProblem(t, err, http.StatusNotFound)
	})

	t.Run("deleted customer", func(t *testing.T) {
		deletedKey := uniqueKey("ent_delete_deleted_customer")
		deleted, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:  deletedKey,
			Name: "Deleted Customer " + deletedKey,
			UsageAttribution: &v3sdk.CustomerUsageAttribution{
				SubjectKeys: []string{deletedKey},
			},
		})
		c.requireStatus(http.StatusCreated, err)

		deletedEntitlementID := createBooleanEntitlement(t, deleted.ID)
		c.requireStatus(http.StatusNoContent, c.Customers.Delete(t.Context(), deleted.ID))

		err = c.Customers.Entitlements.Delete(t.Context(), deleted.ID, deletedEntitlementID)
		requireProblem(t, err, http.StatusPreconditionFailed)
	})
}
