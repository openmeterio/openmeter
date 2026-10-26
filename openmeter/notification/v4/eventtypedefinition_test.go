package v4

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

const testSchema = `{"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object"}`

func TestEventTypeDefinitionValidate(t *testing.T) {
	eventType := EventType{Name: "billing.invoice.created", Version: 1}
	replacement := EventType{Name: "billing.invoice.created", Version: 2}

	valid := EventTypeDefinition{
		EventType:         eventType,
		PayloadJSONSchema: json.RawMessage(testSchema),
		Rule:              &RuleConfigDefinition{JSONSchema: json.RawMessage(testSchema)},
		DeprecatedBy:      &replacement,
	}

	require.NoError(t, valid.Validate())
	require.True(t, valid.Deprecated())

	tests := []struct {
		name   string
		modify func(d *EventTypeDefinition)
	}{
		{"invalid event type", func(d *EventTypeDefinition) { d.EventType.Version = 0 }},
		{"missing payload schema", func(d *EventTypeDefinition) { d.PayloadJSONSchema = nil }},
		{"payload schema is not JSON", func(d *EventTypeDefinition) { d.PayloadJSONSchema = json.RawMessage(`{`) }},
		{"missing rule config schema", func(d *EventTypeDefinition) { d.Rule = &RuleConfigDefinition{} }},
		{"invalid deprecated by", func(d *EventTypeDefinition) { d.DeprecatedBy = &EventType{Name: "invoice", Version: 1} }},
		{"deprecated by itself", func(d *EventTypeDefinition) { d.DeprecatedBy = &eventType }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := valid.clone()
			tt.modify(&def)

			require.Error(t, def.Validate())
		})
	}
}

func TestEventTypeDefinitionClone(t *testing.T) {
	replacement := EventType{Name: "billing.invoice.created", Version: 2}
	def := EventTypeDefinition{
		EventType:         EventType{Name: "billing.invoice.created", Version: 1},
		PayloadJSONSchema: json.RawMessage(testSchema),
		Rule:              &RuleConfigDefinition{JSONSchema: json.RawMessage(testSchema)},
		DeprecatedBy:      &replacement,
	}

	cloned := def.clone()
	require.Equal(t, def, cloned)

	cloned.PayloadJSONSchema[0] = '['
	cloned.Rule.JSONSchema[0] = '['
	cloned.DeprecatedBy.Version = 3

	require.Equal(t, testSchema, string(def.PayloadJSONSchema))
	require.Equal(t, testSchema, string(def.Rule.JSONSchema))
	require.Equal(t, 2, def.DeprecatedBy.Version)
}
