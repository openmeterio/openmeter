package customersentitlements

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/datetime"
)

func parseCreateBody(t *testing.T, body string) api.CreateEntitlementRequest {
	t.Helper()

	var req api.CreateEntitlementRequest
	require.NoError(t, json.Unmarshal([]byte(body), &req))

	return req
}

func TestFromAPICreateEntitlementRequest(t *testing.T) {
	now := time.Date(2026, 9, 16, 10, 30, 0, 0, time.UTC)
	customerID := customer.CustomerID{Namespace: "ns", ID: "01K4WAQ0J99ZZ0MD75HXR112H8"}

	t.Run("metered with issue and preset measure usage from", func(t *testing.T) {
		body := parseCreateBody(t, `{
			"type": "metered",
			"feature": {"id": "01K4WAQ0J99ZZ0MD75HXR112H9"},
			"usage_period": {"interval": "P1M", "anchor": "2026-09-01T00:00:00Z"},
			"issue": {"amount": "100.5", "priority": 3},
			"measure_usage_from": "current_period_start",
			"is_soft_limit": true,
			"preserve_overage_at_reset": true,
			"labels": {"team": "billing"}
		}`)

		got, err := fromAPICreateEntitlementRequest(customerID, body, now)
		require.NoError(t, err)

		require.Equal(t, customerID, got.CustomerID)
		require.Empty(t, got.Grants)

		ent := got.Entitlement
		require.Equal(t, entitlement.EntitlementTypeMetered, ent.EntitlementType)
		require.Equal(t, "01K4WAQ0J99ZZ0MD75HXR112H9", *ent.FeatureID)
		require.Nil(t, ent.FeatureKey)
		require.Equal(t, 100.5, *ent.IssueAfterReset)
		require.Equal(t, uint8(3), *ent.IssueAfterResetPriority)
		require.True(t, *ent.IsSoftLimit)
		require.True(t, *ent.PreserveOverageAtReset)
		require.Equal(t, map[string]string{"team": "billing"}, ent.Metadata)

		anchor := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
		require.Equal(t, anchor, ent.UsagePeriod.GetValue().Anchor)
		require.Equal(t, anchor, ent.UsagePeriod.GetTime())
		require.Equal(t, "P1M", ent.UsagePeriod.GetValue().Interval.ISOString().String())
		require.Equal(t, anchor, ent.MeasureUsageFrom.Get())
	})

	t.Run("metered with explicit measure usage from and grants", func(t *testing.T) {
		body := parseCreateBody(t, `{
			"type": "metered",
			"feature": {"id": "01K4WAQ0J99ZZ0MD75HXR112H9"},
			"usage_period": {"interval": "P1W"},
			"measure_usage_from": "2026-09-10T00:00:00Z",
			"grants": [{
				"amount": "250",
				"priority": 2,
				"effective_at": "2026-09-15T00:00:00Z",
				"expires_after": "P3M",
				"min_rollover_amount": "10",
				"labels": {"source": "promo"},
				"recurrence": {"interval": "P1M"}
			}]
		}`)

		got, err := fromAPICreateEntitlementRequest(customerID, body, now)
		require.NoError(t, err)

		ent := got.Entitlement
		require.Equal(t, now, ent.UsagePeriod.GetValue().Anchor)
		require.Equal(t, time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC), ent.MeasureUsageFrom.Get())
		require.Nil(t, ent.IssueAfterReset)

		require.Len(t, got.Grants, 1)
		g := got.Grants[0]
		effectiveAt := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
		require.Equal(t, 250.0, g.Amount)
		require.Equal(t, uint8(2), g.Priority)
		require.Equal(t, effectiveAt, g.EffectiveAt)
		require.Equal(t, 250.0, g.ResetMaxRollover)
		require.Equal(t, 10.0, g.ResetMinRollover)
		require.Equal(t, &grant.ExpirationPeriod{Count: 3, Duration: grant.ExpirationPeriodDurationMonth}, g.Expiration)
		require.Equal(t, map[string]string{"source": "promo"}, g.Metadata)
		require.Equal(t, effectiveAt, g.Recurrence.Anchor)
		require.Equal(t, "P1M", g.Recurrence.Interval.ISOString().String())
	})

	t.Run("static keeps the config JSON verbatim", func(t *testing.T) {
		body := parseCreateBody(t, `{
			"type": "static",
			"feature": {"id": "01K4WAQ0J99ZZ0MD75HXR112H9"},
			"config": {"integrations": ["github"], "limit": 1e2}
		}`)

		got, err := fromAPICreateEntitlementRequest(customerID, body, now)
		require.NoError(t, err)

		require.Equal(t, entitlement.EntitlementTypeStatic, got.Entitlement.EntitlementType)
		require.Equal(t, `{"integrations":["github"],"limit":1e2}`, *got.Entitlement.Config)
		require.Nil(t, got.Entitlement.UsagePeriod)
	})

	t.Run("static without config leaves it unset for domain validation", func(t *testing.T) {
		body := parseCreateBody(t, `{"type": "static", "feature": {"id": "01K4WAQ0J99ZZ0MD75HXR112H9"}, "config": null}`)

		got, err := fromAPICreateEntitlementRequest(customerID, body, now)
		require.NoError(t, err)
		require.Nil(t, got.Entitlement.Config)
	})

	t.Run("boolean with usage period", func(t *testing.T) {
		body := parseCreateBody(t, `{
			"type": "boolean",
			"feature": {"id": "01K4WAQ0J99ZZ0MD75HXR112H9"},
			"usage_period": {"interval": "P1Y"}
		}`)

		got, err := fromAPICreateEntitlementRequest(customerID, body, now)
		require.NoError(t, err)

		require.Equal(t, entitlement.EntitlementTypeBoolean, got.Entitlement.EntitlementType)
		require.Equal(t, "P1Y", got.Entitlement.UsagePeriod.GetValue().Interval.ISOString().String())
		require.Equal(t, now, got.Entitlement.UsagePeriod.GetValue().Anchor)
	})

	t.Run("metered accepts the deprecated flat issue fields", func(t *testing.T) {
		body := parseCreateBody(t, `{
			"type": "metered",
			"feature": {"id": "01K4WAQ0J99ZZ0MD75HXR112H9"},
			"usage_period": {"interval": "P1M"},
			"issue_after_reset": "40",
			"issue_after_reset_priority": 5
		}`)

		got, err := fromAPICreateEntitlementRequest(customerID, body, now)
		require.NoError(t, err)
		require.Equal(t, 40.0, *got.Entitlement.IssueAfterReset)
		require.Equal(t, uint8(5), *got.Entitlement.IssueAfterResetPriority)
	})

	t.Run("metered prefers issue over the deprecated flat fields", func(t *testing.T) {
		body := parseCreateBody(t, `{
			"type": "metered",
			"feature": {"id": "01K4WAQ0J99ZZ0MD75HXR112H9"},
			"usage_period": {"interval": "P1M"},
			"issue": {"amount": "100", "priority": 2},
			"issue_after_reset": "40",
			"issue_after_reset_priority": 5
		}`)

		got, err := fromAPICreateEntitlementRequest(customerID, body, now)
		require.NoError(t, err)
		require.Equal(t, 100.0, *got.Entitlement.IssueAfterReset)
		require.Equal(t, uint8(2), *got.Entitlement.IssueAfterResetPriority)
	})

	t.Run("metered with now preset measures from creation time", func(t *testing.T) {
		body := parseCreateBody(t, `{
			"type": "metered",
			"feature": {"id": "01K4WAQ0J99ZZ0MD75HXR112H9"},
			"usage_period": {"interval": "P1M", "anchor": "2026-09-01T00:00:00Z"},
			"measure_usage_from": "now"
		}`)

		got, err := fromAPICreateEntitlementRequest(customerID, body, now)
		require.NoError(t, err)
		require.Equal(t, now, got.Entitlement.MeasureUsageFrom.Get())
	})

	t.Run("rejects unknown type", func(t *testing.T) {
		body := parseCreateBody(t, `{"type": "unlimited", "feature": {"id": "01K4WAQ0J99ZZ0MD75HXR112H9"}}`)

		_, err := fromAPICreateEntitlementRequest(customerID, body, now)
		require.Error(t, err)
	})

	t.Run("rejects invalid usage period interval", func(t *testing.T) {
		body := parseCreateBody(t, `{"type": "boolean", "feature": {"id": "01K4WAQ0J99ZZ0MD75HXR112H9"}, "usage_period": {"interval": "monthly"}}`)

		_, err := fromAPICreateEntitlementRequest(customerID, body, now)
		require.ErrorContains(t, err, "invalid usage period")
	})
}

func TestExpirationPeriodFromISODuration(t *testing.T) {
	accepted := map[string]grant.ExpirationPeriod{
		"P1Y":   {Count: 1, Duration: grant.ExpirationPeriodDurationYear},
		"P3M":   {Count: 3, Duration: grant.ExpirationPeriodDurationMonth},
		"P2W":   {Count: 2, Duration: grant.ExpirationPeriodDurationWeek},
		"P7D":   {Count: 7, Duration: grant.ExpirationPeriodDurationDay},
		"PT12H": {Count: 12, Duration: grant.ExpirationPeriodDurationHour},
	}

	for iso, want := range accepted {
		t.Run(iso, func(t *testing.T) {
			d, err := datetime.ISODurationString(iso).Parse()
			require.NoError(t, err)

			got, err := expirationPeriodFromISODuration(d)
			require.NoError(t, err)
			require.Equal(t, want, got)
		})
	}

	rejected := []string{"P1M1D", "P1.5M", "PT30M", "P0D", "-P1M", "P1DT1H", "P4294967296D"}

	for _, iso := range rejected {
		t.Run(iso, func(t *testing.T) {
			d, err := datetime.ISODurationString(iso).Parse()
			require.NoError(t, err)

			_, err = expirationPeriodFromISODuration(d)
			require.Error(t, err)
		})
	}
}

func TestToAPIEntitlementStaticConfig(t *testing.T) {
	ent := &entitlement.Entitlement{
		GenericProperties: entitlement.GenericProperties{
			ID:              "01K4WAQ0J99ZZ0MD75HXR112H7",
			FeatureID:       "01K4WAQ0J99ZZ0MD75HXR112H9",
			CustomerID:      "01K4WAQ0J99ZZ0MD75HXR112H8",
			EntitlementType: entitlement.EntitlementTypeStatic,
		},
		Config: lo.ToPtr(`{"integrations":["github"]}`),
	}

	out, err := toAPIEntitlement(ent)
	require.NoError(t, err)

	encoded, err := json.Marshal(out)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(encoded, &decoded))

	require.Equal(t, "static", decoded["type"])
	require.Equal(t, map[string]any{"integrations": []any{"github"}}, decoded["config"])
	require.Equal(t, map[string]any{"id": "01K4WAQ0J99ZZ0MD75HXR112H8"}, decoded["customer"])
	require.Equal(t, map[string]any{"id": "01K4WAQ0J99ZZ0MD75HXR112H9"}, decoded["feature"])
}
