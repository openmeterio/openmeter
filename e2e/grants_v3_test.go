package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

func TestV3Grants(t *testing.T) {
	c := newV3Client(t)

	key := uniqueKey("grants_customer")
	cust, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
		Key:              key,
		Name:             "Grants Customer " + key,
		UsageAttribution: &v3sdk.CustomerUsageAttribution{SubjectKeys: []string{key}},
	})
	c.requireStatus(http.StatusCreated, err)

	f := createMeteredFeature(t, c, "grants")

	created, err := c.Customers.Entitlements.Create(t.Context(), cust.ID, lo.Must(v3sdk.CreateEntitlementRequestFromCreateEntitlementMeteredRequest(v3sdk.CreateEntitlementMeteredRequest{
		Feature:     v3sdk.FeatureReference{ID: f.ID},
		UsagePeriod: v3sdk.RecurringPeriodInput{Interval: "P1M"},
	})))
	c.requireStatus(http.StatusCreated, err)

	ent, err := created.AsEntitlementMetered()
	require.NoError(t, err)

	// The first usage period starts at the entitlement's creation minute, so the
	// effective dates must be taken after the entitlement exists.
	effectiveAt := time.Now().UTC().Truncate(time.Minute)

	grantIDs := make([]string, 0, 3)
	for i := range 3 {
		g, err := c.Customers.Entitlements.Grants.Create(t.Context(), cust.ID, ent.ID, v3sdk.EntitlementGrantCreateRequest{
			Amount:      "100",
			EffectiveAt: effectiveAt.Add(time.Duration(i) * time.Minute),
		})
		c.requireStatus(http.StatusCreated, err)

		grantIDs = append(grantIDs, g.ID)
	}

	customerFilter := &v3sdk.ListGrantsFilter{CustomerID: &v3sdk.StringExactFilter{Eq: lo.ToPtr(cust.ID)}}

	t.Run("list filtered by customer and sorted descending", func(t *testing.T) {
		res, err := c.Grants.List(t.Context(), v3sdk.ListGrantsParams{
			Filter: customerFilter,
			Sort:   &v3sdk.Sort{By: "effective_at", Order: v3sdk.SortOrderDesc},
		})
		c.requireStatus(http.StatusOK, err)

		require.Equal(t, lo.Reverse(append([]string{}, grantIDs...)), lo.Map(res.Data, func(g v3sdk.EntitlementGrant, _ int) string { return g.ID }))
		require.Equal(t, ent.ID, res.Data[0].EntitlementID)
		require.Equal(t, "100", res.Data[0].Amount)
		require.Equal(t, 3, res.Meta.Page.Total)
	})

	t.Run("list a page", func(t *testing.T) {
		res, err := c.Grants.List(t.Context(), v3sdk.ListGrantsParams{
			Page:   &v3sdk.PageParams{Number: lo.ToPtr(2), Size: lo.ToPtr(1)},
			Filter: customerFilter,
		})
		c.requireStatus(http.StatusOK, err)

		require.Equal(t, []string{grantIDs[1]}, lo.Map(res.Data, func(g v3sdk.EntitlementGrant, _ int) string { return g.ID }))
		require.Equal(t, 3, res.Meta.Page.Total)
	})

	t.Run("list filtered by feature key", func(t *testing.T) {
		res, err := c.Grants.List(t.Context(), v3sdk.ListGrantsParams{
			Filter: &v3sdk.ListGrantsFilter{Feature: &v3sdk.StringExactFilter{Eq: lo.ToPtr(f.Key)}},
		})
		c.requireStatus(http.StatusOK, err)
		require.Len(t, res.Data, 3)
	})

	t.Run("list with unsupported sort field", func(t *testing.T) {
		_, err := c.Grants.List(t.Context(), v3sdk.ListGrantsParams{
			Sort: &v3sdk.Sort{By: "owner_id"},
		})
		requireProblem(t, err, http.StatusBadRequest)
	})

	t.Run("void", func(t *testing.T) {
		voidedAt := time.Now().UTC().Truncate(time.Minute)

		// when the last grant is voided at a given time
		err := c.Grants.Void(t.Context(), grantIDs[2], v3sdk.VoidGrantParams{VoidedAt: &voidedAt})
		c.requireStatus(http.StatusNoContent, err)

		// then it is still listed with the void time
		res, err := c.Grants.List(t.Context(), v3sdk.ListGrantsParams{Filter: customerFilter})
		c.requireStatus(http.StatusOK, err)
		require.Len(t, res.Data, 3)

		voided, ok := lo.Find(res.Data, func(g v3sdk.EntitlementGrant) bool { return g.ID == grantIDs[2] })
		require.True(t, ok)
		require.True(t, voidedAt.Equal(lo.FromPtr(voided.VoidedAt)), "voided at %v != %v", voided.VoidedAt, voidedAt)

		// and it cannot be voided again
		err = c.Grants.Void(t.Context(), grantIDs[2], v3sdk.VoidGrantParams{})
		requireProblem(t, err, http.StatusBadRequest)
	})

	t.Run("void a missing grant", func(t *testing.T) {
		err := c.Grants.Void(t.Context(), "01K4WAQ0J99ZZ0MD75HXR112H9", v3sdk.VoidGrantParams{})
		requireProblem(t, err, http.StatusNotFound)
	})
}
