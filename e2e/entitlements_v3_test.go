package e2e

import (
	"net/http"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

func TestV3GetAndListEntitlements(t *testing.T) {
	c := newV3Client(t)

	createCustomer := func(t *testing.T, prefix string) *v3sdk.Customer {
		t.Helper()

		key := uniqueKey(prefix)
		cust, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:  key,
			Name: "Entitlement Customer " + key,
			UsageAttribution: &v3sdk.CustomerUsageAttribution{
				SubjectKeys: []string{key},
			},
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, cust)

		return cust
	}

	// given two customers holding a boolean entitlement for the same feature
	firstCust := createCustomer(t, "ent_global_first")
	secondCust := createCustomer(t, "ent_global_second")

	featureKey := uniqueKey("ent_global_boolean")
	f, err := c.Features.Create(t.Context(), v3sdk.CreateFeatureRequest{
		Key:  featureKey,
		Name: "Boolean Feature " + featureKey,
	})
	c.requireStatus(http.StatusCreated, err)

	createEntitlement := func(t *testing.T, customerID string) *v3sdk.EntitlementBoolean {
		t.Helper()

		created, err := c.Customers.Entitlements.Create(t.Context(), customerID, lo.Must(v3sdk.CreateEntitlementRequestFromCreateEntitlementBooleanRequest(v3sdk.CreateEntitlementBooleanRequest{
			Feature: v3sdk.FeatureReference{ID: f.ID},
		})))
		c.requireStatus(http.StatusCreated, err)

		ent, err := created.AsEntitlementBoolean()
		require.NoError(t, err)

		return ent
	}

	firstEnt := createEntitlement(t, firstCust.ID)
	secondEnt := createEntitlement(t, secondCust.ID)

	t.Run("get by id", func(t *testing.T) {
		got, err := c.Entitlements.Get(t.Context(), secondEnt.ID)
		c.requireStatus(http.StatusOK, err)

		ent, err := got.AsEntitlementBoolean()
		require.NoError(t, err)
		require.Equal(t, secondEnt.ID, ent.ID)
		require.Equal(t, f.ID, ent.Feature.ID)
		require.Equal(t, secondCust.ID, ent.Customer.ID)
	})

	t.Run("get unknown entitlement", func(t *testing.T) {
		_, err := c.Entitlements.Get(t.Context(), "01K4WAQ0J99ZZ0MD75HXR112H9")
		requireProblem(t, err, http.StatusNotFound)
	})

	t.Run("list across customers", func(t *testing.T) {
		list, err := c.Entitlements.List(t.Context(), v3sdk.ListEntitlementsParams{
			Sort: &v3sdk.Sort{By: "created_at", Order: v3sdk.SortOrderDesc},
			Filter: &v3sdk.ListEntitlementsFilter{
				CustomerID: &v3sdk.StringExactFilter{Oeq: []string{firstCust.ID, secondCust.ID}},
			},
		})
		c.requireStatus(http.StatusOK, err)
		require.EqualValues(t, 2, list.Meta.Page.Total)
		require.Len(t, list.Data, 2)

		ids := lo.Map(list.Data, func(item v3sdk.Entitlement, _ int) string {
			ent, err := item.AsEntitlementBoolean()
			require.NoError(t, err)

			return ent.ID
		})
		require.Equal(t, []string{secondEnt.ID, firstEnt.ID}, ids)
	})

	t.Run("list filtered by customer", func(t *testing.T) {
		list, err := c.Entitlements.List(t.Context(), v3sdk.ListEntitlementsParams{
			Filter: &v3sdk.ListEntitlementsFilter{
				CustomerID: &v3sdk.StringExactFilter{Eq: lo.ToPtr(firstCust.ID)},
				FeatureID:  &v3sdk.StringExactFilter{Eq: lo.ToPtr(f.ID)},
			},
		})
		c.requireStatus(http.StatusOK, err)
		require.Len(t, list.Data, 1)

		ent, err := list.Data[0].AsEntitlementBoolean()
		require.NoError(t, err)
		require.Equal(t, firstEnt.ID, ent.ID)
	})

	t.Run("list paginated", func(t *testing.T) {
		list, err := c.Entitlements.List(t.Context(), v3sdk.ListEntitlementsParams{
			Page: &v3sdk.PageParams{Number: lo.ToPtr(2), Size: lo.ToPtr(1)},
			Filter: &v3sdk.ListEntitlementsFilter{
				FeatureKey: &v3sdk.StringExactFilter{Eq: lo.ToPtr(featureKey)},
			},
		})
		c.requireStatus(http.StatusOK, err)
		require.EqualValues(t, 2, list.Meta.Page.Total)
		require.Len(t, list.Data, 1)

		ent, err := list.Data[0].AsEntitlementBoolean()
		require.NoError(t, err)
		require.Equal(t, secondEnt.ID, ent.ID)
	})

	t.Run("list rejects unsupported filters and sort", func(t *testing.T) {
		_, err := c.Entitlements.List(t.Context(), v3sdk.ListEntitlementsParams{
			Filter: &v3sdk.ListEntitlementsFilter{CustomerID: &v3sdk.StringExactFilter{Eq: lo.ToPtr("not-a-ulid")}},
		})
		requireProblem(t, err, http.StatusBadRequest)

		_, err = c.Entitlements.List(t.Context(), v3sdk.ListEntitlementsParams{
			Sort: &v3sdk.Sort{By: "feature_key"},
		})
		requireProblem(t, err, http.StatusBadRequest)
	})
}
