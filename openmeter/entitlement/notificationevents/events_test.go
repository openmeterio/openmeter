package notificationevents

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/require"
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

func TestBalanceThresholdTriggeredContract(t *testing.T) {
	require.Equal(t, "v1.entitlement.balance_threshold.triggered", EntitlementBalanceThresholdTriggeredV1EventType.String())
	require.NoError(t, EntitlementBalanceThresholdTriggeredV1EventType.Validate())

	t.Run("payload struct matches payload schema", func(t *testing.T) {
		// given the schema generated from the same TypeSpec model as the struct
		schema := compileSchema(t, EntitlementBalanceThresholdTriggeredV1PayloadJSONSchema)

		// when business logic marshals the generated struct
		raw, err := json.Marshal(EntitlementBalanceThresholdTriggeredV1Payload{
			CustomerId:     "01J2KNAG7SY9X6V8E5P4D3C2ZF",
			EntitlementId:  "01J2KNBH8TZ0Y7W9F6Q5E4D3AG",
			FeatureKey:     "api_requests",
			CurrentBalance: 18,
			Threshold: EntitlementBalanceThreshold{
				Type:  EntitlementBalanceThresholdTypeBalanceAtOrBelow,
				Value: 20,
			},
		})
		require.NoError(t, err)

		// then the schema accepts it
		require.NoError(t, validateJSON(t, schema, string(raw)))
	})

	t.Run("payload schema rejects unknown properties and enum values", func(t *testing.T) {
		schema := compileSchema(t, EntitlementBalanceThresholdTriggeredV1PayloadJSONSchema)

		require.Error(t, validateJSON(t, schema, `{
			"customer_id": "01J2KNAG7SY9X6V8E5P4D3C2ZF",
			"entitlement_id": "01J2KNBH8TZ0Y7W9F6Q5E4D3AG",
			"feature_key": "api_requests",
			"current_balance": 18,
			"threshold": {"type": "balance_at_or_below", "value": 20},
			"unexpected": true
		}`))

		require.Error(t, validateJSON(t, schema, `{
			"customer_id": "01J2KNAG7SY9X6V8E5P4D3C2ZF",
			"entitlement_id": "01J2KNBH8TZ0Y7W9F6Q5E4D3AG",
			"feature_key": "api_requests",
			"current_balance": 18,
			"threshold": {"type": "bogus", "value": 20}
		}`))
	})

	t.Run("rule config struct matches rule config schema", func(t *testing.T) {
		schema := compileSchema(t, EntitlementBalanceThresholdTriggeredV1RuleConfigJSONSchema)

		raw, err := json.Marshal(EntitlementBalanceThresholdTriggeredV1RuleConfig{
			Thresholds: []EntitlementBalanceThreshold{
				{Type: EntitlementBalanceThresholdTypeUsagePercentageAtOrAbove, Value: 80},
			},
		})
		require.NoError(t, err)
		require.NoError(t, validateJSON(t, schema, string(raw)))

		require.Error(t, validateJSON(t, schema, `{"thresholds": []}`), "at least one threshold is required")
	})
}
