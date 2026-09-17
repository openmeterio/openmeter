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
		f := createMeteredFeature(t, c, "ent_metered_grants")
		effectiveAt := time.Now().UTC().Truncate(time.Minute)

		req := lo.Must(v3sdk.CreateEntitlementRequestFromCreateEntitlementMeteredRequest(v3sdk.CreateEntitlementMeteredRequest{
			Feature:     v3sdk.FeatureReference{ID: f.ID},
			UsagePeriod: v3sdk.EntitlementRecurringPeriodInput{Interval: "P1M"},
			IsSoftLimit: lo.ToPtr(true),
			Labels:      lo.ToPtr(map[string]string{"team": "billing"}),
			Grants: &[]v3sdk.EntitlementGrantCreateRequest{{
				Amount:       "100",
				EffectiveAt:  effectiveAt,
				ExpiresAfter: lo.ToPtr("P1M"),
			}},
		}))

		created, err := c.Customers.Entitlements.Create(t.Context(), cust.ID, req)
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, created)

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

		// A feature can only have one active entitlement per customer.
		_, err = c.Customers.Entitlements.Create(t.Context(), cust.ID, req)
		requireProblem(t, err, http.StatusConflict)
	})

	t.Run("metered with issue after reset", func(t *testing.T) {
		f := createMeteredFeature(t, c, "ent_metered_issue")

		req := lo.Must(v3sdk.CreateEntitlementRequestFromCreateEntitlementMeteredRequest(v3sdk.CreateEntitlementMeteredRequest{
			Feature:     v3sdk.FeatureReference{ID: f.ID},
			UsagePeriod: v3sdk.EntitlementRecurringPeriodInput{Interval: "P1W"},
			Issue: &v3sdk.EntitlementIssueAfterReset{
				Amount:   "50",
				Priority: lo.ToPtr(uint8(2)),
			},
			MeasureUsageFrom: lo.ToPtr(lo.Must(v3sdk.EntitlementMeasureUsageFromFromPreset(v3sdk.EntitlementMeasureUsageFromPresetCurrentPeriodStart))),
		}))

		created, err := c.Customers.Entitlements.Create(t.Context(), cust.ID, req)
		c.requireStatus(http.StatusCreated, err)

		metered, err := created.AsEntitlementMetered()
		require.NoError(t, err)
		require.NotNil(t, metered.Issue)
		require.Equal(t, "50", metered.Issue.Amount)
		require.Equal(t, uint8(2), lo.FromPtr(metered.Issue.Priority))
		require.True(t, metered.MeasureUsageFrom.Equal(metered.CurrentUsagePeriod.From))
	})

	t.Run("metered rejects issue combined with grants", func(t *testing.T) {
		f := createMeteredFeature(t, c, "ent_metered_conflict")

		req := lo.Must(v3sdk.CreateEntitlementRequestFromCreateEntitlementMeteredRequest(v3sdk.CreateEntitlementMeteredRequest{
			Feature:     v3sdk.FeatureReference{ID: f.ID},
			UsagePeriod: v3sdk.EntitlementRecurringPeriodInput{Interval: "P1M"},
			Issue:       &v3sdk.EntitlementIssueAfterReset{Amount: "50"},
			Grants: &[]v3sdk.EntitlementGrantCreateRequest{{
				Amount:      "100",
				EffectiveAt: time.Now().UTC().Truncate(time.Minute),
			}},
		}))

		_, err := c.Customers.Entitlements.Create(t.Context(), cust.ID, req)
		requireProblem(t, err, http.StatusBadRequest)
	})

	t.Run("static", func(t *testing.T) {
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

		created, err := c.Customers.Entitlements.Create(t.Context(), cust.ID, req)
		c.requireStatus(http.StatusCreated, err)

		static, err := created.AsEntitlementStatic()
		require.NoError(t, err)
		require.Equal(t, f.ID, static.Feature.ID)
		require.Equal(t, map[string]any{"integrations": []any{"github"}}, static.Config)
		require.Nil(t, static.UsagePeriod)
	})

	t.Run("boolean", func(t *testing.T) {
		featureKey := uniqueKey("ent_boolean")
		f, err := c.Features.Create(t.Context(), v3sdk.CreateFeatureRequest{
			Key:  featureKey,
			Name: "Boolean Feature " + featureKey,
		})
		c.requireStatus(http.StatusCreated, err)

		req := lo.Must(v3sdk.CreateEntitlementRequestFromCreateEntitlementBooleanRequest(v3sdk.CreateEntitlementBooleanRequest{
			Feature: v3sdk.FeatureReference{ID: f.ID},
		}))

		created, err := c.Customers.Entitlements.Create(t.Context(), cust.ID, req)
		c.requireStatus(http.StatusCreated, err)

		boolean, err := created.AsEntitlementBoolean()
		require.NoError(t, err)
		require.Equal(t, f.ID, boolean.Feature.ID)
		require.Equal(t, cust.ID, boolean.Customer.ID)
	})

	t.Run("deleted customer", func(t *testing.T) {
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

		_, err = c.Customers.Entitlements.Create(t.Context(), deleted.ID, req)
		requireProblem(t, err, http.StatusPreconditionFailed)
	})

	t.Run("unknown customer", func(t *testing.T) {
		req := lo.Must(v3sdk.CreateEntitlementRequestFromCreateEntitlementBooleanRequest(v3sdk.CreateEntitlementBooleanRequest{
			Feature: v3sdk.FeatureReference{ID: "01K4WAQ0J99ZZ0MD75HXR112H9"},
		}))

		_, err := c.Customers.Entitlements.Create(t.Context(), "01K4WAQ0J99ZZ0MD75HXR112H8", req)
		requireProblem(t, err, http.StatusNotFound)
	})
}
