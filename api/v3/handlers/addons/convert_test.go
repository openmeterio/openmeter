package addons

import (
	"encoding/json"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

func roundTripStaticConfig(t *testing.T, config interface{}) interface{} {
	t.Helper()

	var apiEnt apiv3.BillingRateCardEntitlement
	require.NoError(t, apiEnt.FromBillingRateCardStaticEntitlement(apiv3.BillingRateCardStaticEntitlement{
		Type:   "static",
		Config: config,
	}))

	domain, err := FromAPIBillingRateCardEntitlement(apiEnt, nil)
	require.NoError(t, err)

	out, err := ToAPIBillingRateCardEntitlement(domain)
	require.NoError(t, err)

	static, err := out.AsBillingRateCardStaticEntitlement()
	require.NoError(t, err)

	return static.Config
}

func TestFromToAPIBillingRateCardEntitlement(t *testing.T) {
	t.Run("static — object config round-trips as a JSON object, not a string", func(t *testing.T) {
		config := map[string]interface{}{
			"integrations": []interface{}{"github", "slack"},
			"limit":        float64(10),
		}

		out := roundTripStaticConfig(t, config)

		assert.Equal(t, config, out)

		_, isObject := out.(map[string]interface{})
		assert.True(t, isObject, "config must round-trip as a JSON object")
	})

	t.Run("static — array config round-trips", func(t *testing.T) {
		config := []interface{}{float64(1), float64(2), float64(3)}
		assert.Equal(t, config, roundTripStaticConfig(t, config))
	})

	t.Run("static — scalar string config round-trips", func(t *testing.T) {
		config := "just-a-string"
		assert.Equal(t, config, roundTripStaticConfig(t, config))
	})

	t.Run("static — nil config round-trips as nil", func(t *testing.T) {
		assert.Nil(t, roundTripStaticConfig(t, nil))
	})

	t.Run("static — domain stores the v1 string-token convention", func(t *testing.T) {
		// Subscription materialization (subscriptionspec.go) unwraps the stored
		// config as a JSON string token containing JSON text. Pin that convention
		// so v3-created static entitlements stay materializable.
		var apiEnt apiv3.BillingRateCardEntitlement
		require.NoError(t, apiEnt.FromBillingRateCardStaticEntitlement(apiv3.BillingRateCardStaticEntitlement{
			Type:   "static",
			Config: map[string]interface{}{"integrations": []interface{}{"github"}},
		}))

		domain, err := FromAPIBillingRateCardEntitlement(apiEnt, nil)
		require.NoError(t, err)

		static, err := domain.AsStatic()
		require.NoError(t, err)

		var text string
		require.NoError(t, json.Unmarshal(static.Config, &text),
			"stored config must be a JSON string token wrapping the JSON text")
		assert.JSONEq(t, `{"integrations":["github"]}`, text)
	})

	t.Run("static — legacy non-token stored config is returned as stored, not an error", func(t *testing.T) {
		// Reads must never hard-fail on data that was valid to persist.
		domain := productcatalog.NewEntitlementTemplateFrom(productcatalog.StaticEntitlementTemplate{
			Config: json.RawMessage(`{"a":1}`),
		})

		out, err := ToAPIBillingRateCardEntitlement(domain)
		require.NoError(t, err)

		static, err := out.AsBillingRateCardStaticEntitlement()
		require.NoError(t, err)
		assert.Equal(t, map[string]interface{}{"a": float64(1)}, static.Config)
	})

	t.Run("metered — limit, soft limit, usage period round-trip", func(t *testing.T) {
		usagePeriod := apiv3.ISO8601Duration("P1M")

		var apiEnt apiv3.BillingRateCardEntitlement
		require.NoError(t, apiEnt.FromBillingRateCardMeteredEntitlement(apiv3.BillingRateCardMeteredEntitlement{
			Type:        "metered",
			IsSoftLimit: lo.ToPtr(true),
			UsagePeriod: &usagePeriod,
			Limit:       lo.ToPtr(float64(1000)),
		}))

		domain, err := FromAPIBillingRateCardEntitlement(apiEnt, nil)
		require.NoError(t, err)

		out, err := ToAPIBillingRateCardEntitlement(domain)
		require.NoError(t, err)

		metered, err := out.AsBillingRateCardMeteredEntitlement()
		require.NoError(t, err)

		assert.Equal(t, "metered", string(metered.Type))
		require.NotNil(t, metered.Limit)
		assert.Equal(t, float64(1000), *metered.Limit)
		require.NotNil(t, metered.IsSoftLimit)
		assert.True(t, *metered.IsSoftLimit)
		require.NotNil(t, metered.UsagePeriod)
		assert.Equal(t, "P1M", *metered.UsagePeriod)
	})

	t.Run("metered — usage period defaults to billing cadence when omitted", func(t *testing.T) {
		var apiEnt apiv3.BillingRateCardEntitlement
		require.NoError(t, apiEnt.FromBillingRateCardMeteredEntitlement(apiv3.BillingRateCardMeteredEntitlement{
			Type:  "metered",
			Limit: lo.ToPtr(float64(5)),
		}))

		cadence, err := datetime.ISODurationString("P3M").Parse()
		require.NoError(t, err)

		domain, err := FromAPIBillingRateCardEntitlement(apiEnt, &cadence)
		require.NoError(t, err)

		metered, err := domain.AsMetered()
		require.NoError(t, err)
		assert.Equal(t, "P3M", metered.UsagePeriod.ISOString().String())
	})

	t.Run("metered — missing usage period with no billing cadence is a validation error", func(t *testing.T) {
		var apiEnt apiv3.BillingRateCardEntitlement
		require.NoError(t, apiEnt.FromBillingRateCardMeteredEntitlement(apiv3.BillingRateCardMeteredEntitlement{
			Type: "metered",
		}))

		domain, err := FromAPIBillingRateCardEntitlement(apiEnt, nil)
		require.Error(t, err)
		assert.Nil(t, domain)
	})

	t.Run("metered — malformed usage period is a validation error", func(t *testing.T) {
		bad := apiv3.ISO8601Duration("not-a-duration")
		var apiEnt apiv3.BillingRateCardEntitlement
		require.NoError(t, apiEnt.FromBillingRateCardMeteredEntitlement(apiv3.BillingRateCardMeteredEntitlement{
			Type:        "metered",
			UsagePeriod: &bad,
		}))

		domain, err := FromAPIBillingRateCardEntitlement(apiEnt, nil)
		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid usage period")
		assert.True(t, models.IsGenericValidationError(err), "malformed durations must surface as validation errors")
		assert.Nil(t, domain)
	})

	t.Run("unknown entitlement type is rejected", func(t *testing.T) {
		var apiEnt apiv3.BillingRateCardEntitlement
		require.NoError(t, json.Unmarshal([]byte(`{"type":"unknown"}`), &apiEnt))

		domain, err := FromAPIBillingRateCardEntitlement(apiEnt, nil)
		require.Error(t, err)
		assert.ErrorContains(t, err, "unsupported entitlement type")
		assert.Nil(t, domain)
	})

	t.Run("boolean — round-trips", func(t *testing.T) {
		var apiEnt apiv3.BillingRateCardEntitlement
		require.NoError(t, apiEnt.FromBillingRateCardBooleanEntitlement(apiv3.BillingRateCardBooleanEntitlement{
			Type: "boolean",
		}))

		domain, err := FromAPIBillingRateCardEntitlement(apiEnt, nil)
		require.NoError(t, err)

		out, err := ToAPIBillingRateCardEntitlement(domain)
		require.NoError(t, err)

		boolean, err := out.AsBillingRateCardBooleanEntitlement()
		require.NoError(t, err)
		assert.Equal(t, "boolean", string(boolean.Type))
	})
}
