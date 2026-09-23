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
		list, err := c.Customers.Entitlements.List(t.Context(), cust.ID, v3sdk.ListCustomerEntitlementsParams{
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
		list, err := c.Customers.Entitlements.List(t.Context(), cust.ID, v3sdk.ListCustomerEntitlementsParams{
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
		for name, filter := range map[string]v3sdk.ListCustomerEntitlementsFilter{
			"feature id":  {FeatureID: &v3sdk.StringExactFilter{Eq: lo.ToPtr(meteredFeature.ID)}},
			"feature key": {FeatureKey: &v3sdk.StringExactFilter{Oeq: []string{meteredFeature.Key, "unknown_feature"}}},
			"type":        {Type: &v3sdk.StringExactFilter{Eq: lo.ToPtr("metered")}},
		} {
			list, err := c.Customers.Entitlements.List(t.Context(), cust.ID, v3sdk.ListCustomerEntitlementsParams{
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
		list, err := c.Customers.Entitlements.List(t.Context(), cust.ID, v3sdk.ListCustomerEntitlementsParams{
			Filter: &v3sdk.ListCustomerEntitlementsFilter{Type: &v3sdk.StringExactFilter{Neq: lo.ToPtr("metered")}},
		})
		c.requireStatus(http.StatusOK, err)
		require.Len(t, list.Data, 1)

		item, err := list.Data[0].AsEntitlementBoolean()
		require.NoError(t, err)
		require.Equal(t, booleanEnt.ID, item.ID)
	})

	t.Run("list rejects unsupported filters and sort", func(t *testing.T) {
		_, err := c.Customers.Entitlements.List(t.Context(), cust.ID, v3sdk.ListCustomerEntitlementsParams{
			Filter: &v3sdk.ListCustomerEntitlementsFilter{Type: &v3sdk.StringExactFilter{Eq: lo.ToPtr("unknown")}},
		})
		requireProblem(t, err, http.StatusBadRequest)

		_, err = c.Customers.Entitlements.List(t.Context(), cust.ID, v3sdk.ListCustomerEntitlementsParams{
			Sort: &v3sdk.Sort{By: "feature_key"},
		})
		requireProblem(t, err, http.StatusBadRequest)
	})

	t.Run("unknown customer", func(t *testing.T) {
		_, err := c.Customers.Entitlements.Get(t.Context(), "01K4WAQ0J99ZZ0MD75HXR112H8", meteredEnt.ID)
		requireProblem(t, err, http.StatusNotFound)

		_, err = c.Customers.Entitlements.List(t.Context(), "01K4WAQ0J99ZZ0MD75HXR112H8", v3sdk.ListCustomerEntitlementsParams{})
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

		_, err = c.Customers.Entitlements.List(t.Context(), deleted.ID, v3sdk.ListCustomerEntitlementsParams{})
		requireProblem(t, err, http.StatusConflict)
	})
}

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

	createBooleanEntitlement := func(t *testing.T, customerID string) string {
		t.Helper()

		req := lo.Must(v3sdk.CreateEntitlementRequestFromCreateEntitlementBooleanRequest(v3sdk.CreateEntitlementBooleanRequest{
			Feature: v3sdk.FeatureReference{ID: f.ID},
		}))

		created, err := c.Customers.Entitlements.Create(t.Context(), customerID, req)
		c.requireStatus(http.StatusCreated, err)

		boolean, err := created.AsEntitlementBoolean()
		require.NoError(t, err)

		return boolean.ID
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
}

func TestV3OverrideCustomerEntitlement(t *testing.T) {
	c := newV3Client(t)

	customerKey := uniqueKey("ent_override_customer")
	cust, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
		Key:  customerKey,
		Name: "Entitlement Customer " + customerKey,
		UsageAttribution: &v3sdk.CustomerUsageAttribution{
			SubjectKeys: []string{customerKey},
		},
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, cust)

	f := createMeteredFeature(t, c, "ent_override_metered")

	created, err := c.Customers.Entitlements.Create(t.Context(), cust.ID, lo.Must(v3sdk.CreateEntitlementRequestFromCreateEntitlementMeteredRequest(v3sdk.CreateEntitlementMeteredRequest{
		Feature:     v3sdk.FeatureReference{ID: f.ID},
		UsagePeriod: v3sdk.RecurringPeriodInput{Interval: "P1M"},
	})))
	c.requireStatus(http.StatusCreated, err)
	oldEnt, err := created.AsEntitlementMetered()
	require.NoError(t, err)

	upgrade := lo.Must(v3sdk.CreateEntitlementRequestFromCreateEntitlementMeteredRequest(v3sdk.CreateEntitlementMeteredRequest{
		Feature:     v3sdk.FeatureReference{ID: f.ID},
		UsagePeriod: v3sdk.RecurringPeriodInput{Interval: "P1M"},
		Issue:       &v3sdk.EntitlementIssueAfterReset{Amount: "100"},
	}))

	t.Run("different feature", func(t *testing.T) {
		// given a request for another feature
		other := createMeteredFeature(t, c, "ent_override_other")

		req := lo.Must(v3sdk.CreateEntitlementRequestFromCreateEntitlementMeteredRequest(v3sdk.CreateEntitlementMeteredRequest{
			Feature:     v3sdk.FeatureReference{ID: other.ID},
			UsagePeriod: v3sdk.RecurringPeriodInput{Interval: "P1M"},
		}))

		// when the entitlement is overridden with it, then the request is rejected as invalid
		_, err := c.Customers.Entitlements.Override(t.Context(), cust.ID, oldEnt.ID, req)
		requireProblem(t, err, http.StatusBadRequest)
	})

	t.Run("replaces the entitlement", func(t *testing.T) {
		// when the entitlement is overridden with an upgrade for the same feature
		overridden, err := c.Customers.Entitlements.Override(t.Context(), cust.ID, oldEnt.ID, upgrade)
		c.requireStatus(http.StatusCreated, err)

		// then a new entitlement carries the upgrade and the old one has ended
		newEnt, err := overridden.AsEntitlementMetered()
		require.NoError(t, err)
		require.NotEqual(t, oldEnt.ID, newEnt.ID)
		require.Equal(t, f.ID, newEnt.Feature.ID)
		require.Equal(t, cust.ID, newEnt.Customer.ID)
		require.NotNil(t, newEnt.Issue)
		require.Equal(t, "100", newEnt.Issue.Amount)

		got, err := c.Customers.Entitlements.Get(t.Context(), cust.ID, oldEnt.ID)
		c.requireStatus(http.StatusOK, err)
		ended, err := got.AsEntitlementMetered()
		require.NoError(t, err)
		require.NotNil(t, ended.ActiveTo)

		list, err := c.Customers.Entitlements.List(t.Context(), cust.ID, v3sdk.ListCustomerEntitlementsParams{
			Filter: &v3sdk.ListCustomerEntitlementsFilter{FeatureID: &v3sdk.StringExactFilter{Eq: lo.ToPtr(f.ID)}},
		})
		c.requireStatus(http.StatusOK, err)
		require.Len(t, list.Data, 1)

		listed, err := list.Data[0].AsEntitlementMetered()
		require.NoError(t, err)
		require.Equal(t, newEnt.ID, listed.ID)
	})

	t.Run("unknown entitlement", func(t *testing.T) {
		_, err := c.Customers.Entitlements.Override(t.Context(), cust.ID, "01K4WAQ0J99ZZ0MD75HXR112H9", upgrade)
		requireProblem(t, err, http.StatusNotFound)
	})

	t.Run("unknown customer", func(t *testing.T) {
		_, err := c.Customers.Entitlements.Override(t.Context(), "01K4WAQ0J99ZZ0MD75HXR112H8", oldEnt.ID, upgrade)
		requireProblem(t, err, http.StatusNotFound)
	})

	t.Run("deleted customer", func(t *testing.T) {
		// given a customer that has been deleted
		deletedKey := uniqueKey("ent_override_deleted_customer")
		deleted, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:  deletedKey,
			Name: "Deleted Customer " + deletedKey,
		})
		c.requireStatus(http.StatusCreated, err)
		c.requireStatus(http.StatusNoContent, c.Customers.Delete(t.Context(), deleted.ID))

		// when an entitlement is overridden for it, then the request conflicts with the deleted state
		_, err = c.Customers.Entitlements.Override(t.Context(), deleted.ID, oldEnt.ID, upgrade)
		requireProblem(t, err, http.StatusConflict)
	})
}

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
		requireProblem(t, err, http.StatusConflict)
	})
}

func TestV3CreateCustomerEntitlementGrant(t *testing.T) {
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

	// The usage period is anchored to the past so that grants effective at the
	// anchor are inside the current usage period regardless of the wall clock.
	anchor := time.Now().UTC().Truncate(time.Hour)

	createMeteredEntitlement := func(t *testing.T, customerID string, featureID string) string {
		t.Helper()

		created, err := c.Customers.Entitlements.Create(t.Context(), customerID, lo.Must(v3sdk.CreateEntitlementRequestFromCreateEntitlementMeteredRequest(v3sdk.CreateEntitlementMeteredRequest{
			Feature:     v3sdk.FeatureReference{ID: featureID},
			UsagePeriod: v3sdk.RecurringPeriodInput{Interval: "P1M", Anchor: lo.ToPtr(anchor)},
		})))
		c.requireStatus(http.StatusCreated, err)

		metered, err := created.AsEntitlementMetered()
		require.NoError(t, err)

		return metered.ID
	}

	grantRequest := v3sdk.EntitlementGrantCreateRequest{
		Amount:       "100",
		Priority:     lo.ToPtr(uint8(2)),
		EffectiveAt:  anchor,
		ExpiresAfter: lo.ToPtr("P1M"),
		Labels:       lo.ToPtr(map[string]string{"source": "e2e"}),
	}

	cust := createCustomer(t, "ent_create_grant_customer")
	f := createMeteredFeature(t, c, "ent_create_grant")
	entitlementID := createMeteredEntitlement(t, cust.ID, f.ID)

	t.Run("metered entitlement", func(t *testing.T) {
		// when a grant is issued for the metered entitlement
		g, err := c.Customers.Entitlements.Grants.Create(t.Context(), cust.ID, entitlementID, grantRequest)
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, g)

		// then the response reflects the request with the rollover defaults applied
		require.NotEmpty(t, g.ID)
		require.Equal(t, entitlementID, g.EntitlementID)
		require.Equal(t, "100", g.Amount)
		require.Equal(t, uint8(2), g.Priority)
		require.True(t, anchor.Equal(g.EffectiveAt), "effective at %s != %s", g.EffectiveAt, anchor)
		require.Equal(t, "P1M", lo.FromPtr(g.ExpiresAfter))
		require.True(t, anchor.AddDate(0, 1, 0).Equal(lo.FromPtr(g.ExpiresAt)))
		require.Equal(t, "100", g.MaxRolloverAmount)
		require.Equal(t, "0", g.MinRolloverAmount)
		require.Nil(t, g.Recurrence)
		require.Equal(t, "e2e", g.Labels["source"])

		// and the grant is listed for the entitlement
		res, err := c.Customers.Entitlements.Grants.List(t.Context(), cust.ID, entitlementID, v3sdk.EntitlementGrantListParams{})
		c.requireStatus(http.StatusOK, err)
		require.Equal(t, 1, res.Meta.Page.Total)
		require.Equal(t, g.ID, res.Data[0].ID)
	})

	t.Run("effective before the current usage period", func(t *testing.T) {
		req := grantRequest
		req.EffectiveAt = anchor.Add(-time.Hour)

		_, err := c.Customers.Entitlements.Grants.Create(t.Context(), cust.ID, entitlementID, req)
		requireProblem(t, err, http.StatusBadRequest)
	})

	t.Run("boolean entitlement", func(t *testing.T) {
		featureKey := uniqueKey("ent_create_grant_boolean")
		booleanFeature, err := c.Features.Create(t.Context(), v3sdk.CreateFeatureRequest{
			Key:  featureKey,
			Name: "Boolean Feature " + featureKey,
		})
		c.requireStatus(http.StatusCreated, err)

		created, err := c.Customers.Entitlements.Create(t.Context(), cust.ID, lo.Must(v3sdk.CreateEntitlementRequestFromCreateEntitlementBooleanRequest(v3sdk.CreateEntitlementBooleanRequest{
			Feature: v3sdk.FeatureReference{ID: booleanFeature.ID},
		})))
		c.requireStatus(http.StatusCreated, err)
		booleanEnt, err := created.AsEntitlementBoolean()
		require.NoError(t, err)

		_, err = c.Customers.Entitlements.Grants.Create(t.Context(), cust.ID, booleanEnt.ID, grantRequest)
		requireProblem(t, err, http.StatusBadRequest)
	})

	t.Run("entitlement of another customer", func(t *testing.T) {
		other := createCustomer(t, "ent_create_grant_other_customer")
		otherEntitlementID := createMeteredEntitlement(t, other.ID, f.ID)

		_, err := c.Customers.Entitlements.Grants.Create(t.Context(), cust.ID, otherEntitlementID, grantRequest)
		requireProblem(t, err, http.StatusNotFound)
	})

	t.Run("unknown entitlement", func(t *testing.T) {
		_, err := c.Customers.Entitlements.Grants.Create(t.Context(), cust.ID, "01K4WAQ0J99ZZ0MD75HXR112H9", grantRequest)
		requireProblem(t, err, http.StatusNotFound)
	})

	t.Run("deleted customer", func(t *testing.T) {
		deleted := createCustomer(t, "ent_create_grant_deleted_customer")
		deletedEntitlementID := createMeteredEntitlement(t, deleted.ID, f.ID)

		c.requireStatus(http.StatusNoContent, c.Customers.Delete(t.Context(), deleted.ID))

		_, err := c.Customers.Entitlements.Grants.Create(t.Context(), deleted.ID, deletedEntitlementID, grantRequest)
		requireProblem(t, err, http.StatusConflict)
	})
}
