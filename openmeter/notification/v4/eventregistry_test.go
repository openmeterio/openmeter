package v4_test

import (
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	v4 "github.com/openmeterio/openmeter/openmeter/notification/v4"
	"github.com/openmeterio/openmeter/pkg/models"
)

const testSchema = `{"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object"}`

func testDefinition(name string, version int) v4.EventTypeDefinition {
	return v4.EventTypeDefinition{
		EventType:         v4.EventType{Name: name, Version: version},
		PayloadJSONSchema: json.RawMessage(testSchema),
	}
}

func TestNewEventRegistry(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	v1 := testDefinition("billing.invoice.created", 1)
	v2 := testDefinition("billing.invoice.created", 2)
	v3 := testDefinition("billing.invoice.created", 3)
	v10 := testDefinition("billing.invoice.created", 10)

	t.Run("lists definitions by name, then numeric version", func(t *testing.T) {
		v1 := testDefinition("billing.invoice.created", 1)
		v1.DeprecatedBy = &v2.EventType
		v1.Rule = &v4.RuleConfigDefinition{JSONSchema: json.RawMessage(testSchema)}

		system := testDefinition("app.stripe.disconnected", 1)

		registry, err := v4.NewEventRegistry(v4.EventRegistryConfig{
			Logger:      logger,
			Definitions: []v4.EventTypeDefinition{v10, v2, v1, system},
		})
		require.NoError(t, err)

		require.Equal(t, []v4.EventTypeDefinition{system, v1, v2, v10}, registry.EventTypes())

		def, err := registry.EventType(v1.EventType)
		require.NoError(t, err)
		require.Equal(t, v1, def)
	})

	t.Run("returns copies", func(t *testing.T) {
		registry, err := v4.NewEventRegistry(v4.EventRegistryConfig{Logger: logger, Definitions: []v4.EventTypeDefinition{v1}})
		require.NoError(t, err)

		def, err := registry.EventType(v1.EventType)
		require.NoError(t, err)

		def.PayloadJSONSchema[0] = '['

		def, err = registry.EventType(v1.EventType)
		require.NoError(t, err)
		require.Equal(t, testSchema, string(def.PayloadJSONSchema))
	})

	t.Run("unregistered event type is not found", func(t *testing.T) {
		registry, err := v4.NewEventRegistry(v4.EventRegistryConfig{Logger: logger})
		require.NoError(t, err)

		_, err = registry.EventType(v1.EventType)
		require.True(t, models.IsGenericNotFoundError(err), err)
	})

	deprecatedBy := func(def v4.EventTypeDefinition, by v4.EventType) v4.EventTypeDefinition {
		def.DeprecatedBy = &by

		return def
	}

	invalid := []struct {
		name   string
		config v4.EventRegistryConfig
	}{
		{
			name:   "missing logger",
			config: v4.EventRegistryConfig{Definitions: []v4.EventTypeDefinition{v1}},
		},
		{
			name:   "invalid definition",
			config: v4.EventRegistryConfig{Logger: logger, Definitions: []v4.EventTypeDefinition{{EventType: v1.EventType}}},
		},
		{
			name:   "duplicate event type",
			config: v4.EventRegistryConfig{Logger: logger, Definitions: []v4.EventTypeDefinition{v1, v1}},
		},
		{
			name:   "deprecated by unregistered event type",
			config: v4.EventRegistryConfig{Logger: logger, Definitions: []v4.EventTypeDefinition{deprecatedBy(v1, v2.EventType)}},
		},
		{
			name: "deprecation cycle",
			config: v4.EventRegistryConfig{Logger: logger, Definitions: []v4.EventTypeDefinition{
				deprecatedBy(v1, v2.EventType),
				deprecatedBy(v2, v3.EventType),
				deprecatedBy(v3, v2.EventType),
			}},
		},
		{
			name: "invalid payload schema",
			config: v4.EventRegistryConfig{Logger: logger, Definitions: []v4.EventTypeDefinition{{
				EventType:         v1.EventType,
				PayloadJSONSchema: json.RawMessage(`{"type": 5}`),
			}}},
		},
		{
			name: "invalid rule config schema",
			config: v4.EventRegistryConfig{Logger: logger, Definitions: []v4.EventTypeDefinition{{
				EventType:         v1.EventType,
				PayloadJSONSchema: json.RawMessage(testSchema),
				Rule:              &v4.RuleConfigDefinition{JSONSchema: json.RawMessage(`{"minimum": "1"}`)},
			}}},
		},
		{
			name: "external schema reference",
			config: v4.EventRegistryConfig{Logger: logger, Definitions: []v4.EventTypeDefinition{{
				EventType:         v1.EventType,
				PayloadJSONSchema: json.RawMessage(`{"$ref": "other.json"}`),
			}}},
		},
	}

	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			_, err := v4.NewEventRegistry(tt.config)
			require.True(t, models.IsGenericValidationError(err), err)
		})
	}
}
