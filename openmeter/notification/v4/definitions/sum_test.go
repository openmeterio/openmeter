package definitions

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	v4 "github.com/openmeterio/openmeter/openmeter/notification/v4"
)

func TestFingerprint(t *testing.T) {
	const (
		payloadSchema = `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object"}`
		ruleSchema    = `{"type":"object","properties":{"a":{"enum":["x","y"]}}}`
	)

	def := func(payload string, ruleConfig *string) v4.EventTypeDefinition {
		d := v4.EventTypeDefinition{
			EventType:         v4.EventType{Name: "billing.invoice.created", Version: 1},
			PayloadJSONSchema: json.RawMessage(payload),
		}
		if ruleConfig != nil {
			d.Rule = &v4.RuleConfigDefinition{JSONSchema: json.RawMessage(*ruleConfig)}
		}

		return d
	}

	fingerprint := func(d v4.EventTypeDefinition) string {
		t.Helper()

		f, err := Fingerprint(d)
		require.NoError(t, err)

		return f
	}

	base := fingerprint(def(payloadSchema, lo.ToPtr(ruleSchema)))
	require.Regexp(t, `^sha256:[0-9a-f]{64}$`, base)

	// Schemas are hashed in canonical form, so formatting and key order do not
	// change the fingerprint.
	require.Equal(t, base, fingerprint(def(
		`{"type": "object", "$schema": "https://json-schema.org/draft/2020-12/schema"}`,
		lo.ToPtr(`{"properties": {"a": {"enum": ["x", "y"]}}, "type": "object"}`),
	)))

	// A replacement is set on released definitions, so it is left out.
	deprecated := def(payloadSchema, lo.ToPtr(ruleSchema))
	deprecated.DeprecatedBy = &v4.EventType{Name: "billing.invoice.created", Version: 2}
	require.Equal(t, base, fingerprint(deprecated))

	otherType := def(payloadSchema, lo.ToPtr(ruleSchema))
	otherType.EventType.Version = 2

	changed := []struct {
		name string
		def  v4.EventTypeDefinition
	}{
		{"event type", otherType},
		{"payload", def(`{"type":"object","required":["a"]}`, lo.ToPtr(ruleSchema))},
		{"no rule", def(payloadSchema, nil)},
		{"rule config", def(payloadSchema, lo.ToPtr(`{"type":"object","properties":{"a":{"enum":["x","z"]}}}`))},
	}

	seen := map[string]string{base: "base"}
	for _, tt := range changed {
		t.Run(tt.name, func(t *testing.T) {
			f := fingerprint(tt.def)
			require.NotContains(t, seen, f, "same fingerprint as %s", seen[f])
			seen[f] = tt.name
		})
	}

	_, err := Fingerprint(def(`{`, nil))
	require.ErrorContains(t, err, "payload JSON schema")

	_, err = Fingerprint(def(payloadSchema, lo.ToPtr(`{`)))
	require.ErrorContains(t, err, "rule config JSON schema")
}

func TestGenerate(t *testing.T) {
	invoiceV1 := v4.EventType{Name: "billing.invoice.created", Version: 1}
	invoiceV2 := v4.EventType{Name: "billing.invoice.created", Version: 2}

	defs := []v4.EventTypeDefinition{
		{EventType: invoiceV2, PayloadJSONSchema: json.RawMessage(`{"type":"object"}`)},
		{EventType: invoiceV1, PayloadJSONSchema: json.RawMessage(`{"type":"object"}`), DeprecatedBy: &invoiceV2},
	}

	fingerprint := func(def v4.EventTypeDefinition) string {
		t.Helper()

		f, err := Fingerprint(def)
		require.NoError(t, err)

		return f
	}

	var out bytes.Buffer
	require.NoError(t, generate(&out, defs))

	// Lines are sorted by Event type, and only the deprecated definition gets
	// the replacement column.
	require.Equal(t,
		"v1.billing.invoice.created "+fingerprint(defs[1])+" deprecated_by=v2.billing.invoice.created\n"+
			"v2.billing.invoice.created "+fingerprint(defs[0])+"\n",
		out.String(),
	)
	require.Equal(t, invoiceV2, defs[0].EventType, "the caller's slice is not reordered")

	invalid := []v4.EventTypeDefinition{{EventType: invoiceV1, PayloadJSONSchema: json.RawMessage(`{`)}}
	require.ErrorContains(t, generate(&bytes.Buffer{}, invalid), "v1.billing.invoice.created")
}

func TestVerify(t *testing.T) {
	invoiceV1 := v4.EventType{Name: "billing.invoice.created", Version: 1}
	invoiceV2 := v4.EventType{Name: "billing.invoice.created", Version: 2}
	invoiceV3 := v4.EventType{Name: "billing.invoice.created", Version: 3}

	def := func(payload string, deprecatedBy *v4.EventType) []v4.EventTypeDefinition {
		return []v4.EventTypeDefinition{
			{EventType: invoiceV1, PayloadJSONSchema: json.RawMessage(payload), DeprecatedBy: deprecatedBy},
			{EventType: invoiceV2, PayloadJSONSchema: json.RawMessage(`{"type":"object"}`)},
			{EventType: invoiceV3, PayloadJSONSchema: json.RawMessage(`{"type":"object"}`)},
		}
	}

	var committed bytes.Buffer
	require.NoError(t, generate(&committed, def(`{"type":"object"}`, &invoiceV2)))

	require.NoError(t, verify(committed.Bytes(), def(`{"type":"object"}`, &invoiceV2)))

	changed := []struct {
		name string
		defs []v4.EventTypeDefinition
	}{
		{"payload changed", def(`{"type":"object","required":["a"]}`, &invoiceV2)},
		{"deprecation removed", def(`{"type":"object"}`, nil)},
		{"replacement changed", def(`{"type":"object"}`, &invoiceV3)},
		{"definition removed", def(`{"type":"object"}`, &invoiceV2)[:2]},
	}

	for _, tt := range changed {
		t.Run(tt.name, func(t *testing.T) {
			require.ErrorContains(t, verify(committed.Bytes(), tt.defs), "does not match")
		})
	}

	require.Error(t, verify(committed.Bytes(), def(`{`, &invoiceV2)))
}
