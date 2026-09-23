package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

func TestV3CreateCustomerEntitlement(t *testing.T) {
	c := newV3Client(t)

	customerKey := uniqueKey("ent_customer")
	cust, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
		Key:  customerKey,
		Name: "Entitlement Customer " + customerKey,
		UsageAttribution: &v3sdk.CustomerUsageAttribution{
			SubjectKeys: []string{customerKey},
		},
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, cust)

	t.Run("metered with grants", func(t *testing.T) {
		// given a metered feature and a create request carrying one grant
		f := createMeteredFeature(t, c, "ent_metered_grants")
		effectiveAt := time.Now().UTC().Truncate(time.Minute)

		req := lo.Must(v3sdk.CreateEntitlementRequestFromCreateEntitlementMeteredRequest(v3sdk.CreateEntitlementMeteredRequest{
			Feature:     v3sdk.FeatureReference{ID: f.ID},
			UsagePeriod: v3sdk.RecurringPeriodInput{Interval: "P1M"},
			IsSoftLimit: lo.ToPtr(true),
			Labels:      lo.ToPtr(map[string]string{"team": "billing"}),
			Grants: &[]v3sdk.EntitlementGrantCreateRequest{{
				Amount:       "100",
				EffectiveAt:  effectiveAt,
				ExpiresAfter: lo.ToPtr("P1M"),
			}},
		}))

		// when the entitlement is created
		created, err := c.Customers.Entitlements.Create(t.Context(), cust.ID, req)
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, created)

		// then the response reflects the metered request and a second create for the same feature conflicts
		metered, err := created.AsEntitlementMetered()
		require.NoError(t, err)
		require.NotEmpty(t, metered.ID)
		require.Equal(t, f.ID, metered.Feature.ID)
		require.Equal(t, cust.ID, metered.Customer.ID)
		require.Equal(t, "P1M", metered.UsagePeriod.Interval)
		require.False(t, metered.UsagePeriod.Anchor.IsZero())
		require.False(t, metered.CurrentUsagePeriod.From.IsZero())
		require.True(t, lo.FromPtr(metered.IsSoftLimit))
		require.Nil(t, metered.Issue)
		require.Equal(t, "billing", metered.Labels["team"])

		_, err = c.Customers.Entitlements.Create(t.Context(), cust.ID, req)
		requireProblem(t, err, http.StatusConflict)
	})

	t.Run("metered with issue after reset", func(t *testing.T) {
		// given a metered feature and a request issuing credits at every reset
		f := createMeteredFeature(t, c, "ent_metered_issue")

		req := lo.Must(v3sdk.CreateEntitlementRequestFromCreateEntitlementMeteredRequest(v3sdk.CreateEntitlementMeteredRequest{
			Feature:     v3sdk.FeatureReference{ID: f.ID},
			UsagePeriod: v3sdk.RecurringPeriodInput{Interval: "P1W"},
			Issue: &v3sdk.EntitlementIssueAfterReset{
				Amount:   "50",
				Priority: lo.ToPtr(uint8(2)),
			},
			MeasureUsageFrom: lo.ToPtr(lo.Must(v3sdk.EntitlementMeasureUsageFromFromPreset(v3sdk.EntitlementMeasureUsageFromPresetCurrentPeriodStart))),
		}))

		// when the entitlement is created
		created, err := c.Customers.Entitlements.Create(t.Context(), cust.ID, req)
		c.requireStatus(http.StatusCreated, err)

		// then the issue settings and measure-usage-from preset are applied
		metered, err := created.AsEntitlementMetered()
		require.NoError(t, err)
		require.NotNil(t, metered.Issue)
		require.Equal(t, "50", metered.Issue.Amount)
		require.Equal(t, uint8(2), lo.FromPtr(metered.Issue.Priority))
		require.True(t, metered.MeasureUsageFrom.Equal(metered.CurrentUsagePeriod.From))
	})

	t.Run("metered rejects issue combined with grants", func(t *testing.T) {
		// given a request that sets both issue and grants
		f := createMeteredFeature(t, c, "ent_metered_conflict")

		req := lo.Must(v3sdk.CreateEntitlementRequestFromCreateEntitlementMeteredRequest(v3sdk.CreateEntitlementMeteredRequest{
			Feature:     v3sdk.FeatureReference{ID: f.ID},
			UsagePeriod: v3sdk.RecurringPeriodInput{Interval: "P1M"},
			Issue:       &v3sdk.EntitlementIssueAfterReset{Amount: "50"},
			Grants: &[]v3sdk.EntitlementGrantCreateRequest{{
				Amount:      "100",
				EffectiveAt: time.Now().UTC().Truncate(time.Minute),
			}},
		}))

		// when the entitlement is created, then the request is rejected as invalid
		_, err := c.Customers.Entitlements.Create(t.Context(), cust.ID, req)
		requireProblem(t, err, http.StatusBadRequest)
	})

	t.Run("static", func(t *testing.T) {
		// given a feature without a meter and a static config
		featureKey := uniqueKey("ent_static")
		f, err := c.Features.Create(t.Context(), v3sdk.CreateFeatureRequest{
			Key:  featureKey,
			Name: "Static Feature " + featureKey,
		})
		c.requireStatus(http.StatusCreated, err)

		req := lo.Must(v3sdk.CreateEntitlementRequestFromCreateEntitlementStaticRequest(v3sdk.CreateEntitlementStaticRequest{
			Feature: v3sdk.FeatureReference{ID: f.ID},
			Config:  map[string]any{"integrations": []any{"github"}},
		}))

		// when the entitlement is created
		created, err := c.Customers.Entitlements.Create(t.Context(), cust.ID, req)
		c.requireStatus(http.StatusCreated, err)

		// then the config round-trips and no usage period is set
		static, err := created.AsEntitlementStatic()
		require.NoError(t, err)
		require.Equal(t, f.ID, static.Feature.ID)
		require.Equal(t, map[string]any{"integrations": []any{"github"}}, static.Config)
		require.Nil(t, static.UsagePeriod)
	})

	t.Run("boolean", func(t *testing.T) {
		// given a feature without a meter
		featureKey := uniqueKey("ent_boolean")
		f, err := c.Features.Create(t.Context(), v3sdk.CreateFeatureRequest{
			Key:  featureKey,
			Name: "Boolean Feature " + featureKey,
		})
		c.requireStatus(http.StatusCreated, err)

		req := lo.Must(v3sdk.CreateEntitlementRequestFromCreateEntitlementBooleanRequest(v3sdk.CreateEntitlementBooleanRequest{
			Feature: v3sdk.FeatureReference{ID: f.ID},
		}))

		// when the entitlement is created
		created, err := c.Customers.Entitlements.Create(t.Context(), cust.ID, req)
		c.requireStatus(http.StatusCreated, err)

		// then it is bound to the feature and customer
		boolean, err := created.AsEntitlementBoolean()
		require.NoError(t, err)
		require.Equal(t, f.ID, boolean.Feature.ID)
		require.Equal(t, cust.ID, boolean.Customer.ID)
	})

	t.Run("deleted customer", func(t *testing.T) {
		// given a customer that has been deleted
		deletedKey := uniqueKey("ent_deleted_customer")
		deleted, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:  deletedKey,
			Name: "Deleted Customer " + deletedKey,
		})
		c.requireStatus(http.StatusCreated, err)
		c.requireStatus(http.StatusNoContent, c.Customers.Delete(t.Context(), deleted.ID))

		req := lo.Must(v3sdk.CreateEntitlementRequestFromCreateEntitlementBooleanRequest(v3sdk.CreateEntitlementBooleanRequest{
			Feature: v3sdk.FeatureReference{ID: "01K4WAQ0J99ZZ0MD75HXR112H9"},
		}))

		// when an entitlement is created for it, then the request conflicts with the deleted state
		_, err = c.Customers.Entitlements.Create(t.Context(), deleted.ID, req)
		requireProblem(t, err, http.StatusConflict)
	})

	t.Run("unknown customer", func(t *testing.T) {
		// given a customer ID that does not exist
		req := lo.Must(v3sdk.CreateEntitlementRequestFromCreateEntitlementBooleanRequest(v3sdk.CreateEntitlementBooleanRequest{
			Feature: v3sdk.FeatureReference{ID: "01K4WAQ0J99ZZ0MD75HXR112H9"},
		}))

		// when an entitlement is created for it, then the customer is not found
		_, err := c.Customers.Entitlements.Create(t.Context(), "01K4WAQ0J99ZZ0MD75HXR112H8", req)
		requireProblem(t, err, http.StatusNotFound)
	})
}

func TestV3GetAndListCustomerEntitlements(t *testing.T) {
	c := newV3Client(t)

	customerKey := uniqueKey("ent_list_customer")
	cust, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
		Key:  customerKey,
		Name: "Entitlement Customer " + customerKey,
		UsageAttribution: &v3sdk.CustomerUsageAttribution{
			SubjectKeys: []string{customerKey},
		},
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, cust)

	meteredFeature := createMeteredFeature(t, c, "ent_list_metered")
	metered, err := c.Customers.Entitlements.Create(t.Context(), cust.ID, lo.Must(v3sdk.CreateEntitlementRequestFromCreateEntitlementMeteredRequest(v3sdk.CreateEntitlementMeteredRequest{
		Feature:     v3sdk.FeatureReference{ID: meteredFeature.ID},
		UsagePeriod: v3sdk.RecurringPeriodInput{Interval: "P1M"},
	})))
	c.requireStatus(http.StatusCreated, err)
	meteredEnt, err := metered.AsEntitlementMetered()
	require.NoError(t, err)

	booleanFeatureKey := uniqueKey("ent_list_boolean")
	booleanFeature, err := c.Features.Create(t.Context(), v3sdk.CreateFeatureRequest{
		Key:  booleanFeatureKey,
		Name: "Boolean Feature " + booleanFeatureKey,
	})
	c.requireStatus(http.StatusCreated, err)

	boolean, err := c.Customers.Entitlements.Create(t.Context(), cust.ID, lo.Must(v3sdk.CreateEntitlementRequestFromCreateEntitlementBooleanRequest(v3sdk.CreateEntitlementBooleanRequest{
		Feature: v3sdk.FeatureReference{ID: booleanFeature.ID},
	})))
	c.requireStatus(http.StatusCreated, err)
	booleanEnt, err := boolean.AsEntitlementBoolean()
	require.NoError(t, err)

	t.Run("get by id", func(t *testing.T) {
		got, err := c.Customers.Entitlements.Get(t.Context(), cust.ID, meteredEnt.ID)
		c.requireStatus(http.StatusOK, err)

		gotMetered, err := got.AsEntitlementMetered()
		require.NoError(t, err)
		require.Equal(t, meteredEnt.ID, gotMetered.ID)
		require.Equal(t, meteredFeature.ID, gotMetered.Feature.ID)
		require.Equal(t, cust.ID, gotMetered.Customer.ID)
	})

	t.Run("get unknown entitlement", func(t *testing.T) {
		_, err := c.Customers.Entitlements.Get(t.Context(), cust.ID, "01K4WAQ0J99ZZ0MD75HXR112H9")
		requireProblem(t, err, http.StatusNotFound)
	})

	t.Run("get entitlement of another customer", func(t *testing.T) {
		otherKey := uniqueKey("ent_list_other_customer")
		other, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:  otherKey,
			Name: "Other Customer " + otherKey,
		})
		c.requireStatus(http.StatusCreated, err)

		_, err = c.Customers.Entitlements.Get(t.Context(), other.ID, meteredEnt.ID)
		requireProblem(t, err, http.StatusNotFound)
	})

	t.Run("list", func(t *testing.T) {
		list, err := c.Customers.Entitlements.List(t.Context(), cust.ID, v3sdk.EntitlementListParams{
			Sort: &v3sdk.Sort{By: "created_at", Order: v3sdk.SortOrderDesc},
		})
		c.requireStatus(http.StatusOK, err)
		require.EqualValues(t, 2, list.Meta.Page.Total)
		require.Len(t, list.Data, 2)

		first, err := list.Data[0].AsEntitlementBoolean()
		require.NoError(t, err)
		require.Equal(t, booleanEnt.ID, first.ID)

		second, err := list.Data[1].AsEntitlementMetered()
		require.NoError(t, err)
		require.Equal(t, meteredEnt.ID, second.ID)
	})

	t.Run("list paginated", func(t *testing.T) {
		list, err := c.Customers.Entitlements.List(t.Context(), cust.ID, v3sdk.EntitlementListParams{
			Page: &v3sdk.PageParams{Number: lo.ToPtr(2), Size: lo.ToPtr(1)},
		})
		c.requireStatus(http.StatusOK, err)
		require.EqualValues(t, 2, list.Meta.Page.Total)
		require.Len(t, list.Data, 1)

		item, err := list.Data[0].AsEntitlementBoolean()
		require.NoError(t, err)
		require.Equal(t, booleanEnt.ID, item.ID)
	})

	t.Run("list filtered", func(t *testing.T) {
		for name, filter := range map[string]v3sdk.EntitlementFilter{
			"feature id":  {FeatureID: &v3sdk.StringExactFilter{Eq: lo.ToPtr(meteredFeature.ID)}},
			"feature key": {FeatureKey: &v3sdk.StringExactFilter{Oeq: []string{meteredFeature.Key, "unknown_feature"}}},
			"type":        {Type: &v3sdk.StringExactFilter{Eq: lo.ToPtr("metered")}},
		} {
			list, err := c.Customers.Entitlements.List(t.Context(), cust.ID, v3sdk.EntitlementListParams{
				Filter: &filter,
			})
			c.requireStatus(http.StatusOK, err)
			require.Len(t, list.Data, 1, name)

			item, err := list.Data[0].AsEntitlementMetered()
			require.NoError(t, err, name)
			require.Equal(t, meteredEnt.ID, item.ID, name)
		}
	})

	t.Run("list excludes by type", func(t *testing.T) {
		list, err := c.Customers.Entitlements.List(t.Context(), cust.ID, v3sdk.EntitlementListParams{
			Filter: &v3sdk.EntitlementFilter{Type: &v3sdk.StringExactFilter{Neq: lo.ToPtr("metered")}},
		})
		c.requireStatus(http.StatusOK, err)
		require.Len(t, list.Data, 1)

		item, err := list.Data[0].AsEntitlementBoolean()
		require.NoError(t, err)
		require.Equal(t, booleanEnt.ID, item.ID)
	})

	t.Run("list rejects unsupported filters and sort", func(t *testing.T) {
		_, err := c.Customers.Entitlements.List(t.Context(), cust.ID, v3sdk.EntitlementListParams{
			Filter: &v3sdk.EntitlementFilter{Type: &v3sdk.StringExactFilter{Eq: lo.ToPtr("unknown")}},
		})
		requireProblem(t, err, http.StatusBadRequest)

		_, err = c.Customers.Entitlements.List(t.Context(), cust.ID, v3sdk.EntitlementListParams{
			Sort: &v3sdk.Sort{By: "feature_key"},
		})
		requireProblem(t, err, http.StatusBadRequest)
	})

	t.Run("unknown customer", func(t *testing.T) {
		_, err := c.Customers.Entitlements.Get(t.Context(), "01K4WAQ0J99ZZ0MD75HXR112H8", meteredEnt.ID)
		requireProblem(t, err, http.StatusNotFound)

		_, err = c.Customers.Entitlements.List(t.Context(), "01K4WAQ0J99ZZ0MD75HXR112H8", v3sdk.EntitlementListParams{})
		requireProblem(t, err, http.StatusNotFound)
	})

	t.Run("deleted customer", func(t *testing.T) {
		deletedKey := uniqueKey("ent_list_deleted_customer")
		deleted, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:  deletedKey,
			Name: "Deleted Customer " + deletedKey,
		})
		c.requireStatus(http.StatusCreated, err)
		c.requireStatus(http.StatusNoContent, c.Customers.Delete(t.Context(), deleted.ID))

		_, err = c.Customers.Entitlements.Get(t.Context(), deleted.ID, meteredEnt.ID)
		requireProblem(t, err, http.StatusConflict)

		_, err = c.Customers.Entitlements.List(t.Context(), deleted.ID, v3sdk.EntitlementListParams{})
		requireProblem(t, err, http.StatusConflict)
	})
}
