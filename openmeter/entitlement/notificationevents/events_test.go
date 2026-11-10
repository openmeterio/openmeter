package notificationevents

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/require"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
)

func compileSchema(t *testing.T, raw []byte) *jsonschema.Schema {
	t.Helper()

	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	require.NoError(t, err)

	c := jsonschema.NewCompiler()
	require.NoError(t, c.AddResource("schema.json", doc))

	s, err := c.Compile("schema.json")
	require.NoError(t, err)

	return s
}

func validateJSON(t *testing.T, s *jsonschema.Schema, raw string) error {
	t.Helper()

	v, err := jsonschema.UnmarshalJSON(strings.NewReader(raw))
	require.NoError(t, err)

	return s.Validate(v)
}

// withProperty returns the JSON object raw with one more property, to check
// that the sealed schemas reject it.
func withProperty(t *testing.T, raw []byte, key string, value any) string {
	t.Helper()

	var doc map[string]any
	require.NoError(t, json.Unmarshal(raw, &doc))

	doc[key] = value

	out, err := json.Marshal(doc)
	require.NoError(t, err)

	return string(out)
}

// withoutBalance returns the payload raw without the balance details of its
// value. The API omits them unless asked, but a payload has to carry them.
func withoutBalance(t *testing.T, raw []byte) string {
	t.Helper()

	var doc map[string]any
	require.NoError(t, json.Unmarshal(raw, &doc))

	value, ok := doc["value"].(map[string]any)
	require.True(t, ok)
	delete(value, "value")

	out, err := json.Marshal(doc)
	require.NoError(t, err)

	return string(out)
}

// meteredEntitlement holds the v3 resources both entitlement payloads carry.
type meteredEntitlement struct {
	entitlement apiv3.BillingEntitlementMetered
	feature     apiv3.Feature
	customer    apiv3.BillingCustomer
	value       EntitlementMeteredValueResult
}

func newMeteredEntitlement() meteredEntitlement {
	now := time.Date(2026, 8, 6, 10, 0, 0, 0, time.UTC)

	return meteredEntitlement{
		entitlement: apiv3.BillingEntitlementMetered{
			Id:                 "01J2KNBH8TZ0Y7W9F6Q5E4D3AG",
			Type:               apiv3.BillingEntitlementMeteredTypeMetered,
			Feature:            apiv3.FeatureReference{Id: "01J2KNCJ9V01Z8XAG7R6F5E4BH"},
			Customer:           apiv3.CustomerReference{Id: "01J2KNAG7SY9X6V8E5P4D3C2ZF"},
			ActiveFrom:         now,
			CreatedAt:          now,
			UpdatedAt:          now,
			UsagePeriod:        apiv3.RecurringPeriod{Anchor: now, Interval: "P1M"},
			CurrentUsagePeriod: apiv3.ClosedPeriod{From: now, To: now.AddDate(0, 1, 0)},
			MeasureUsageFrom:   now,
			LastReset:          now,
		},
		feature: apiv3.Feature{
			Id:        "01J2KNCJ9V01Z8XAG7R6F5E4BH",
			Key:       "api_requests",
			Name:      "API requests",
			CreatedAt: now,
			UpdatedAt: now,
		},
		customer: apiv3.BillingCustomer{
			Id:        "01J2KNAG7SY9X6V8E5P4D3C2ZF",
			Key:       "acme",
			Name:      "ACME Inc.",
			CreatedAt: now,
			UpdatedAt: now,
		},
		value: EntitlementMeteredValueResult{
			Type:       apiv3.BillingEntitlementTypeMetered,
			FeatureKey: "api_requests",
			HasAccess:  true,
			Value: apiv3.BillingEntitlementAccessValue{
				Balance:                   "18",
				Usage:                     "82",
				Overage:                   "0",
				TotalAvailableGrantAmount: "100",
				GrantBalances:             map[string]apiv3.Numeric{"01J2KND0A1B2C3D4E5F6G7H8J9": "18"},
			},
		},
	}
}

func TestBalanceThresholdTriggeredContract(t *testing.T) {
	require.Equal(t, "v1.entitlement.balance_threshold.triggered", EntitlementBalanceThresholdTriggeredV1EventType.String())
	require.NoError(t, EntitlementBalanceThresholdTriggeredV1EventType.Validate())

	e := newMeteredEntitlement()
	payload := EntitlementBalanceThresholdTriggeredV1Payload{
		Entitlement: e.entitlement,
		Feature:     e.feature,
		Customer:    e.customer,
		Value:       e.value,
		Threshold: EntitlementBalanceThreshold{
			Type:  EntitlementBalanceThresholdTypeBalanceAtOrBelow,
			Value: 20,
		},
	}

	t.Run("payload struct matches payload schema", func(t *testing.T) {
		// given the schema generated from the same TypeSpec model as the struct
		schema := compileSchema(t, EntitlementBalanceThresholdTriggeredV1PayloadJSONSchema)

		// when business logic marshals the generated struct
		raw, err := json.Marshal(payload)
		require.NoError(t, err)

		// then the schema accepts it and the struct round-trips it unchanged
		require.NoError(t, validateJSON(t, schema, string(raw)))

		var typed EntitlementBalanceThresholdTriggeredV1Payload
		require.NoError(t, json.Unmarshal(raw, &typed))

		again, err := json.Marshal(typed)
		require.NoError(t, err)
		require.JSONEq(t, string(raw), string(again))
	})

	t.Run("payload schema rejects unknown properties and enum values", func(t *testing.T) {
		schema := compileSchema(t, EntitlementBalanceThresholdTriggeredV1PayloadJSONSchema)

		raw, err := json.Marshal(payload)
		require.NoError(t, err)

		require.Error(t, validateJSON(t, schema, withProperty(t, raw, "unexpected", true)))
		require.Error(t, validateJSON(t, schema, withProperty(t, raw, "threshold", map[string]any{"type": "bogus", "value": 20})))
		require.Error(t, validateJSON(t, schema, withoutBalance(t, raw)), "the balance details are required")
	})

	t.Run("rule config struct matches rule config schema", func(t *testing.T) {
		schema := compileSchema(t, EntitlementBalanceThresholdTriggeredV1RuleConfigJSONSchema)

		raw, err := json.Marshal(EntitlementBalanceThresholdTriggeredV1RuleConfig{
			Thresholds: []EntitlementBalanceThreshold{
				{Type: EntitlementBalanceThresholdTypeUsagePercentageAtOrAbove, Value: 80},
				{Type: EntitlementBalanceThresholdTypeUsageAtOrAbove, Value: 1000},
			},
		})
		require.NoError(t, err)
		require.NoError(t, validateJSON(t, schema, string(raw)))

		require.Error(t, validateJSON(t, schema, `{"thresholds": []}`), "at least one threshold is required")
		require.Error(t, validateJSON(t, schema, `{"thresholds": [{"type": "usage_at_or_above", "value": 1000}], "features": []}`), "an empty feature list is not allowed")
	})
}

func TestUsageResetContract(t *testing.T) {
	require.Equal(t, "v1.entitlement.usage.reset", EntitlementUsageResetV1EventType.String())
	require.NoError(t, EntitlementUsageResetV1EventType.Validate())

	e := newMeteredEntitlement()

	t.Run("payload struct matches payload schema", func(t *testing.T) {
		schema := compileSchema(t, EntitlementUsageResetV1PayloadJSONSchema)

		raw, err := json.Marshal(EntitlementUsageResetV1Payload{
			Entitlement: e.entitlement,
			Feature:     e.feature,
			Customer:    e.customer,
			Value:       e.value,
		})
		require.NoError(t, err)

		require.NoError(t, validateJSON(t, schema, string(raw)))
		require.Error(t, validateJSON(t, schema, withProperty(t, raw, "unexpected", true)))
		require.Error(t, validateJSON(t, schema, withoutBalance(t, raw)), "the balance details are required")
	})

	t.Run("rule config struct matches rule config schema", func(t *testing.T) {
		schema := compileSchema(t, EntitlementUsageResetV1RuleConfigJSONSchema)

		for _, config := range []EntitlementUsageResetV1RuleConfig{
			{},
			{Features: &[]ResourceKey{"api_requests"}},
		} {
			raw, err := json.Marshal(config)
			require.NoError(t, err)
			require.NoError(t, validateJSON(t, schema, string(raw)))
		}

		require.Error(t, validateJSON(t, schema, `{"unexpected": true}`))
		require.Error(t, validateJSON(t, schema, `{"features": []}`), "an empty feature list is not allowed")
	})
}
