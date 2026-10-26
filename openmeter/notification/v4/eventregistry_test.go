package v4_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	billingevents "github.com/openmeterio/openmeter/openmeter/billing/notificationevents"
	entitlementevents "github.com/openmeterio/openmeter/openmeter/entitlement/notificationevents"
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

	t.Run("stores copies of definitions", func(t *testing.T) {
		v1 := testDefinition("billing.invoice.created", 1)
		v1.DeprecatedBy = &v2.EventType
		v1.Rule = &v4.RuleConfigDefinition{JSONSchema: json.RawMessage(testSchema)}

		registry, err := v4.NewEventRegistry(v4.EventRegistryConfig{
			Logger:      logger,
			Definitions: []v4.EventTypeDefinition{v2, v1},
		})
		require.NoError(t, err)

		expected := testDefinition("billing.invoice.created", 1)
		expected.DeprecatedBy = &v2.EventType
		expected.Rule = &v4.RuleConfigDefinition{JSONSchema: json.RawMessage(testSchema)}

		// Changing the inputs after construction must not change the registry.
		v1.PayloadJSONSchema[0] = '['
		v1.Rule.JSONSchema[0] = '['

		def, err := registry.EventType(v1.EventType)
		require.NoError(t, err)
		require.Equal(t, expected, def)
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

func TestEventRegistryEventTypes(t *testing.T) {
	// Canonical order: version first, numeric, then name.
	all := []v4.EventTypeDefinition{
		testDefinition("app.stripe.disconnected", 1),
		testDefinition("billing.invoice.created", 1),
		testDefinition("billing.invoice.updated", 1),
		testDefinition("entitlement.balance_threshold.triggered", 1),
		testDefinition("billing.invoice.created", 2),
		testDefinition("billing.invoice.created", 10),
	}

	registry, err := v4.NewEventRegistry(v4.EventRegistryConfig{
		Logger:      slog.New(slog.DiscardHandler),
		Definitions: []v4.EventTypeDefinition{all[5], all[3], all[1], all[4], all[0], all[2]},
	})
	require.NoError(t, err)

	pattern := func(s string) *v4.EventTypePattern {
		p, err := v4.ParseEventTypePattern(s)
		require.NoError(t, err)

		return &p
	}

	require.Equal(t, all, registry.EventTypes(nil))
	require.Equal(t, all[1:3], registry.EventTypes(pattern("v1.billing.*")))
	require.Equal(t, all[4:5], registry.EventTypes(pattern("v2.billing.invoice.created")))
	require.Equal(t, []v4.EventTypeDefinition{}, registry.EventTypes(pattern("v3.*")))

	// Returned definitions are copies.
	listed := registry.EventTypes(nil)
	listed[0].PayloadJSONSchema[0] = '['
	require.Equal(t, all, registry.EventTypes(nil))
}

func TestEventRegistryValidators(t *testing.T) {
	errDomain := errors.New("domain rejected rule config")

	registry, err := v4.NewEventRegistry(v4.EventRegistryConfig{
		Logger: slog.New(slog.DiscardHandler),
		Definitions: []v4.EventTypeDefinition{
			{
				EventType:         billingevents.BillingInvoiceCreatedV1EventType,
				PayloadJSONSchema: billingevents.BillingInvoiceCreatedV1PayloadJSONSchema,
			},
			{
				EventType:         entitlementevents.EntitlementBalanceThresholdTriggeredV1EventType,
				PayloadJSONSchema: entitlementevents.EntitlementBalanceThresholdTriggeredV1PayloadJSONSchema,
				Rule: &v4.RuleConfigDefinition{
					JSONSchema: entitlementevents.EntitlementBalanceThresholdTriggeredV1RuleConfigJSONSchema,
					Validator: func(_ context.Context, namespace string, _ json.RawMessage) error {
						if namespace != "default" {
							return errDomain
						}

						return nil
					},
				},
			},
		},
	})
	require.NoError(t, err)

	t.Run("payload", func(t *testing.T) {
		validate, err := registry.PayloadValidator(billingevents.BillingInvoiceCreatedV1EventType)
		require.NoError(t, err)

		// given a payload marshaled from the generated struct
		payload, err := json.Marshal(billingevents.BillingInvoiceCreatedV1Payload{
			InvoiceId:  "01J2KNAG7SY9X6V8E5P4D3C2ZF",
			CustomerId: "01J2KNBH8TZ0Y7W9F6Q5E4D3AG",
			Status:     "draft",
			Currency:   "USD",
			Total:      "12.50",
		})
		require.NoError(t, err)

		// then the registered schema accepts it
		require.NoError(t, validate(t.Context(), "default", payload))

		// and rejects payloads that break the contract
		for _, invalid := range []string{`{`, `{}`, `[]`} {
			err := validate(t.Context(), "default", json.RawMessage(invalid))
			require.True(t, models.IsGenericValidationError(err), invalid)
		}
	})

	t.Run("rule config", func(t *testing.T) {
		validate, err := registry.RuleConfigValidator(entitlementevents.EntitlementBalanceThresholdTriggeredV1EventType)
		require.NoError(t, err)

		config, err := json.Marshal(entitlementevents.EntitlementBalanceThresholdTriggeredV1RuleConfig{
			Thresholds: []entitlementevents.EntitlementBalanceThreshold{
				{Type: entitlementevents.EntitlementBalanceThresholdTypeUsagePercentageAtOrAbove, Value: 80},
			},
		})
		require.NoError(t, err)

		require.NoError(t, validate(t.Context(), "default", config))

		// when the schema rejects the configuration the domain validator is not reached
		err = validate(t.Context(), "other", json.RawMessage(`{"thresholds": []}`))
		require.True(t, models.IsGenericValidationError(err), err)
		require.NotErrorIs(t, err, errDomain)

		// when the schema accepts it the domain validator decides
		require.ErrorIs(t, validate(t.Context(), "other", config), errDomain)
	})

	t.Run("rule config of system event type", func(t *testing.T) {
		_, err := registry.RuleConfigValidator(billingevents.BillingInvoiceCreatedV1EventType)
		require.True(t, models.IsGenericValidationError(err), err)
	})

	t.Run("unregistered event type", func(t *testing.T) {
		eventType := v4.EventType{Name: "billing.invoice.created", Version: 2}

		_, err := registry.PayloadValidator(eventType)
		require.True(t, models.IsGenericNotFoundError(err), err)

		_, err = registry.RuleConfigValidator(eventType)
		require.True(t, models.IsGenericNotFoundError(err), err)
	})
}
