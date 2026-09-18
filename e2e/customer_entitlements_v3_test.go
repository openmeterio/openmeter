package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

func TestV3CustomerEntitlementGrants(t *testing.T) {
	c := newV3Client(t)
	v1 := initClient(t)

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

	// The usage period is anchored to the past so that grants effective at the
	// anchor are inside the current usage period regardless of the wall clock.
	anchor := time.Now().UTC().Truncate(time.Hour)

	// The v3 create endpoint is not available yet, so entitlements and grants are
	// seeded through the legacy v2 customer entitlement endpoints.
	createMeteredEntitlement := func(t *testing.T, customerID string, featureID string) string {
		t.Helper()

		var interval api.RecurringPeriodInterval
		require.NoError(t, interval.FromRecurringPeriodIntervalEnum(api.RecurringPeriodIntervalEnumMONTH))

		var body api.CreateCustomerEntitlementV2JSONRequestBody
		require.NoError(t, body.FromEntitlementMeteredV2CreateInputs(api.EntitlementMeteredV2CreateInputs{
			Type:      api.EntitlementMeteredV2CreateInputsTypeMetered,
			FeatureId: lo.ToPtr(featureID),
			UsagePeriod: api.RecurringPeriodCreateInput{
				Interval: interval,
				Anchor:   lo.ToPtr(anchor),
			},
		}))

		res, err := v1.CreateCustomerEntitlementV2WithResponse(t.Context(), customerID, body)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, res.StatusCode(), "Invalid status code [response_body=%s]", string(res.Body))

		created, err := res.JSON201.AsEntitlementMeteredV2()
		require.NoError(t, err)

		return created.Id
	}

	createGrant := func(t *testing.T, customerID string, entitlementID string, amount float64, effectiveAt time.Time) api.EntitlementGrantV2 {
		t.Helper()

		res, err := v1.CreateCustomerEntitlementGrantV2WithResponse(t.Context(), customerID, entitlementID, api.EntitlementGrantCreateInputV2{
			Amount:      amount,
			Priority:    lo.ToPtr(uint8(2)),
			EffectiveAt: effectiveAt,
			Expiration:  &api.ExpirationPeriod{Count: 1, Duration: api.ExpirationDurationMONTH},
			Metadata:    &api.Metadata{"source": "e2e"},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, res.StatusCode(), "Invalid status code [response_body=%s]", string(res.Body))

		return *res.JSON201
	}

	cust := createCustomer(t, "ent_grants_customer")
	f := createMeteredFeature(t, c, "ent_grants")
	entitlementID := createMeteredEntitlement(t, cust.ID, f.ID)

	first := createGrant(t, cust.ID, entitlementID, 100, anchor)
	second := createGrant(t, cust.ID, entitlementID, 50, anchor.Add(time.Minute))

	t.Run("list", func(t *testing.T) {
		res, err := c.Customers.Entitlements.Grants.List(t.Context(), cust.ID, entitlementID, v3sdk.EntitlementGrantListParams{
			Sort: &v3sdk.Sort{By: "effective_at", Order: v3sdk.SortOrderAsc},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, res)

		require.Equal(t, 2, res.Meta.Page.Total)
		require.Len(t, res.Data, 2)

		g := res.Data[0]
		require.Equal(t, first.Id, g.ID)
		require.Equal(t, entitlementID, g.EntitlementID)
		require.Equal(t, "100", g.Amount)
		require.Equal(t, uint8(2), g.Priority)
		require.True(t, anchor.Equal(g.EffectiveAt), "effective at %s != %s", g.EffectiveAt, anchor)
		require.Equal(t, "P1M", lo.FromPtr(g.ExpiresAfter))
		require.True(t, anchor.AddDate(0, 1, 0).Equal(lo.FromPtr(g.ExpiresAt)))
		require.Equal(t, "100", g.MaxRolloverAmount)
		require.Equal(t, "0", g.MinRolloverAmount)
		require.Nil(t, g.Recurrence)
		require.Nil(t, g.NextRecurrence)
		require.Nil(t, g.VoidedAt)
		require.Nil(t, g.DeletedAt)
		require.Equal(t, "e2e", g.Labels["source"])

		require.Equal(t, second.Id, res.Data[1].ID)
	})

	t.Run("list paginated and sorted descending", func(t *testing.T) {
		res, err := c.Customers.Entitlements.Grants.List(t.Context(), cust.ID, entitlementID, v3sdk.EntitlementGrantListParams{
			Page: &v3sdk.PageParams{Number: lo.ToPtr(2), Size: lo.ToPtr(1)},
			Sort: &v3sdk.Sort{By: "effective_at", Order: v3sdk.SortOrderDesc},
		})
		c.requireStatus(http.StatusOK, err)

		require.Equal(t, 2, res.Meta.Page.Total)
		require.Equal(t, 2, res.Meta.Page.Number)
		require.Equal(t, 1, res.Meta.Page.Size)
		require.Len(t, res.Data, 1)
		require.Equal(t, first.Id, res.Data[0].ID)
	})

	t.Run("list including deleted", func(t *testing.T) {
		res, err := c.Customers.Entitlements.Grants.List(t.Context(), cust.ID, entitlementID, v3sdk.EntitlementGrantListParams{
			IncludeDeleted: lo.ToPtr(true),
		})
		c.requireStatus(http.StatusOK, err)
		require.Equal(t, 2, res.Meta.Page.Total)
	})

	t.Run("list with unsupported sort field", func(t *testing.T) {
		_, err := c.Customers.Entitlements.Grants.List(t.Context(), cust.ID, entitlementID, v3sdk.EntitlementGrantListParams{
			Sort: &v3sdk.Sort{By: "owner_id"},
		})
		requireProblem(t, err, http.StatusBadRequest)
	})

	t.Run("list of an entitlement without grants", func(t *testing.T) {
		featureKey := uniqueKey("ent_grants_boolean")
		booleanFeature, err := c.Features.Create(t.Context(), v3sdk.CreateFeatureRequest{
			Key:  featureKey,
			Name: "Boolean Feature " + featureKey,
		})
		c.requireStatus(http.StatusCreated, err)

		var body api.CreateCustomerEntitlementV2JSONRequestBody
		require.NoError(t, body.FromEntitlementBooleanCreateInputs(api.EntitlementBooleanCreateInputs{
			Type:      api.EntitlementBooleanCreateInputsTypeBoolean,
			FeatureId: lo.ToPtr(booleanFeature.ID),
		}))

		created, err := v1.CreateCustomerEntitlementV2WithResponse(t.Context(), cust.ID, body)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, created.StatusCode(), "Invalid status code [response_body=%s]", string(created.Body))

		booleanEnt, err := created.JSON201.AsEntitlementBooleanV2()
		require.NoError(t, err)

		res, err := c.Customers.Entitlements.Grants.List(t.Context(), cust.ID, booleanEnt.Id, v3sdk.EntitlementGrantListParams{})
		c.requireStatus(http.StatusOK, err)
		require.Equal(t, 0, res.Meta.Page.Total)
		require.Empty(t, res.Data)
	})

	t.Run("entitlement of another customer", func(t *testing.T) {
		other := createCustomer(t, "ent_grants_other_customer")
		otherEntitlementID := createMeteredEntitlement(t, other.ID, f.ID)
		createGrant(t, other.ID, otherEntitlementID, 10, anchor)

		_, err := c.Customers.Entitlements.Grants.List(t.Context(), cust.ID, otherEntitlementID, v3sdk.EntitlementGrantListParams{})
		requireProblem(t, err, http.StatusNotFound)
	})

	t.Run("unknown entitlement", func(t *testing.T) {
		_, err := c.Customers.Entitlements.Grants.List(t.Context(), cust.ID, "01K4WAQ0J99ZZ0MD75HXR112H9", v3sdk.EntitlementGrantListParams{})
		requireProblem(t, err, http.StatusNotFound)
	})

	t.Run("unknown customer", func(t *testing.T) {
		_, err := c.Customers.Entitlements.Grants.List(t.Context(), "01K4WAQ0J99ZZ0MD75HXR112H8", entitlementID, v3sdk.EntitlementGrantListParams{})
		requireProblem(t, err, http.StatusNotFound)
	})

	t.Run("deleted customer", func(t *testing.T) {
		deleted := createCustomer(t, "ent_grants_deleted_customer")
		deletedEntitlementID := createMeteredEntitlement(t, deleted.ID, f.ID)
		createGrant(t, deleted.ID, deletedEntitlementID, 10, anchor)

		c.requireStatus(http.StatusNoContent, c.Customers.Delete(t.Context(), deleted.ID))

		_, err := c.Customers.Entitlements.Grants.List(t.Context(), deleted.ID, deletedEntitlementID, v3sdk.EntitlementGrantListParams{})
		requireProblem(t, err, http.StatusPreconditionFailed)
	})
}
