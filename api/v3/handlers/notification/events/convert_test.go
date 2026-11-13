package events

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apimodels "github.com/openmeterio/openmeter/api/models"
	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/httpdriver"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestFromAPIEventSortField(t *testing.T) {
	testCases := []struct {
		field   string
		want    notification.OrderBy
		wantErr bool
	}{
		{field: "id", want: notification.OrderByID},
		{field: "type", want: notification.OrderByType},
		{field: "created_at", want: notification.OrderByCreatedAt},
		// updated_at is a valid sort on channels but events have no updated_at column,
		// so it must be rejected rather than silently ordered by id.
		{field: "updated_at", wantErr: true},
		{field: "createdAt", wantErr: true},
		{field: "", wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.field, func(t *testing.T) {
			got, err := FromAPIEventSortField(t.Context(), tc.field)
			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestEventTypeWireEqualsDomain pins that the v3 enum reuses the domain values, which is
// what lets ToAPIBillingNotificationEvent cast between the two types instead of mapping them.
func TestEventTypeWireEqualsDomain(t *testing.T) {
	testCases := []struct {
		wire   api.BillingNotificationEventType
		domain notification.EventType
	}{
		{api.BillingNotificationEventTypeEntitlementsBalanceThreshold, notification.EventTypeBalanceThreshold},
		{api.BillingNotificationEventTypeEntitlementsReset, notification.EventTypeEntitlementReset},
		{api.BillingNotificationEventTypeInvoiceCreated, notification.EventTypeInvoiceCreated},
		{api.BillingNotificationEventTypeInvoiceUpdated, notification.EventTypeInvoiceUpdated},
	}

	for _, tc := range testCases {
		t.Run(string(tc.wire), func(t *testing.T) {
			assert.Equal(t, string(tc.domain), string(tc.wire))
		})
	}
}

func TestDeliveryStateCasing(t *testing.T) {
	testCases := []struct {
		wire   api.BillingNotificationEventDeliveryState
		domain notification.EventDeliveryStatusState
	}{
		{api.BillingNotificationEventDeliveryStateSuccess, notification.EventDeliveryStatusStateSuccess},
		{api.BillingNotificationEventDeliveryStateFailed, notification.EventDeliveryStatusStateFailed},
		{api.BillingNotificationEventDeliveryStateSending, notification.EventDeliveryStatusStateSending},
		{api.BillingNotificationEventDeliveryStatePending, notification.EventDeliveryStatusStatePending},
		{api.BillingNotificationEventDeliveryStateResending, notification.EventDeliveryStatusStateResending},
	}

	for _, tc := range testCases {
		t.Run(string(tc.wire), func(t *testing.T) {
			domain, err := FromAPIBillingNotificationEventDeliveryState(tc.wire)
			require.NoError(t, err)
			assert.Equal(t, tc.domain, domain)

			wire, err := ToAPIBillingNotificationEventDeliveryState(tc.domain)
			require.NoError(t, err)
			assert.Equal(t, tc.wire, wire)
		})
	}

	t.Run("uppercase wire value is rejected", func(t *testing.T) {
		_, err := FromAPIBillingNotificationEventDeliveryState(api.BillingNotificationEventDeliveryState("FAILED"))
		require.Error(t, err)
	})
}

func TestToAPIBillingNotificationBalanceThresholdType(t *testing.T) {
	testCases := []struct {
		in   apimodels.NotificationRuleBalanceThresholdValueType
		want api.BillingNotificationBalanceThresholdType
	}{
		{apimodels.NotificationRuleBalanceThresholdValueTypeBalanceValue, api.BillingNotificationBalanceThresholdTypeBalanceValue},
		{apimodels.NotificationRuleBalanceThresholdValueTypeUsagePercentage, api.BillingNotificationBalanceThresholdTypeUsagePercentage},
		{apimodels.NotificationRuleBalanceThresholdValueTypeUsageValue, api.BillingNotificationBalanceThresholdTypeUsageValue},
		{apimodels.NotificationRuleBalanceThresholdValueTypePercent, api.BillingNotificationBalanceThresholdTypeUsagePercentage},
		{apimodels.NotificationRuleBalanceThresholdValueTypeNumber, api.BillingNotificationBalanceThresholdTypeUsageValue},
	}

	for _, tc := range testCases {
		t.Run(string(tc.in), func(t *testing.T) {
			got, err := ToAPIBillingNotificationBalanceThresholdType(tc.in)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}

	t.Run("unknown value is rejected", func(t *testing.T) {
		_, err := ToAPIBillingNotificationBalanceThresholdType(apimodels.NotificationRuleBalanceThresholdValueType("nonsense"))
		require.Error(t, err)
	})
}

// testdata/legacy_payloads.json holds v1 payload data with every property set and
// every union variant used at least once, generated from the v1 OpenAPI schemas.
//
//go:embed testdata/legacy_payloads.json
var legacyPayloadFixtures []byte

// The v3 API must return the legacy payloads with the v1 content under snake_case
// field names, and the v3 payload models must hold all of it.
func TestToAPIBillingNotificationEventPayloadMatchesV1(t *testing.T) {
	var fixtures map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(legacyPayloadFixtures, &fixtures))

	for name, raw := range fixtures {
		for _, event := range legacyFixtureEvents(t, name, raw) {
			t.Run(name+"/"+string(event.Type), func(t *testing.T) {
				var (
					v1Payload any
					err       error
				)

				switch event.Type {
				case notification.EventTypeBalanceThreshold:
					v1Payload, err = httpdriver.FromEventAsBalanceThresholdPayload(event)
				case notification.EventTypeEntitlementReset:
					v1Payload, err = httpdriver.FromEventAsEntitlementResetPayload(event)
				case notification.EventTypeInvoiceCreated:
					v1Payload, err = httpdriver.FromEventAsInvoiceCreatedPayload(event)
				case notification.EventTypeInvoiceUpdated:
					v1Payload, err = httpdriver.FromEventAsInvoiceUpdatedPayload(event)
				}

				require.NoError(t, err)

				v1JSON, err := json.Marshal(v1Payload)
				require.NoError(t, err)

				got, err := ToAPIBillingNotificationEvent(event)
				require.NoError(t, err)

				assert.Equal(t, event.ID, got.Id)
				assert.Equal(t, string(event.Type), string(got.Type))

				v3JSON, err := json.Marshal(got.Payload)
				require.NoError(t, err)

				// Only field names change: every value of the v1 payload is still there.
				assert.ElementsMatch(t, jsonLeaves(t, v1JSON), jsonLeaves(t, v3JSON))

				// Decoding into the typed v3 models and encoding again must not lose
				// or change anything, so every renamed field exists in the v3 models.
				var typed any
				switch event.Type {
				case notification.EventTypeBalanceThreshold:
					typed, err = got.Payload.AsBillingNotificationEventBalanceThresholdPayload()
				case notification.EventTypeEntitlementReset:
					typed, err = got.Payload.AsBillingNotificationEventResetPayload()
				case notification.EventTypeInvoiceCreated:
					typed, err = got.Payload.AsBillingNotificationEventInvoiceCreatedPayload()
				case notification.EventTypeInvoiceUpdated:
					typed, err = got.Payload.AsBillingNotificationEventInvoiceUpdatedPayload()
				}

				require.NoError(t, err)

				typedJSON, err := json.Marshal(typed)
				require.NoError(t, err)
				assert.JSONEq(t, string(v3JSON), string(typedJSON))
			})
		}
	}
}

func TestToAPIBillingNotificationEventPayloadFieldNames(t *testing.T) {
	event := notification.Event{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		Type:         notification.EventTypeEntitlementReset,
		CreatedAt:    time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC),
		Payload: notification.EventPayload{
			EventPayloadMeta: notification.EventPayloadMeta{Type: notification.EventTypeEntitlementReset},
			EntitlementReset: &notification.EntitlementResetPayload{
				Entitlement: apimodels.EntitlementMetered{Id: "01ARZ3NDEKTSV4RRFFQ69G5FAX", IsSoftLimit: lo.ToPtr(true)},
				Feature: apimodels.Feature{
					Id:                  "01ARZ3NDEKTSV4RRFFQ69G5FAY",
					Key:                 "gpt4_tokens",
					Metadata:            &apimodels.Metadata{"someKey": "value"},
					MeterGroupByFilters: &map[string]string{"modelName": "gpt-4"},
				},
				Subject: apimodels.Subject{Key: "customer-1", DisplayName: lo.ToPtr("Customer 1")},
				Value:   apimodels.EntitlementValue{HasAccess: true, Balance: lo.ToPtr(100.0)},
			},
		},
	}

	got, err := ToAPIBillingNotificationEvent(event)
	require.NoError(t, err)

	body, err := json.Marshal(got.Payload)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))

	data := payload["data"].(map[string]any)
	entitlement := data["entitlement"].(map[string]any)
	feature := data["feature"].(map[string]any)
	subject := data["subject"].(map[string]any)
	value := data["value"].(map[string]any)

	assert.Equal(t, true, entitlement["is_soft_limit"])
	assert.Equal(t, "Customer 1", subject["display_name"])
	assert.Equal(t, true, value["has_access"])
	// Map keys are user data and keep their case.
	assert.Equal(t, map[string]any{"someKey": "value"}, feature["metadata"])
	assert.Equal(t, map[string]any{"modelName": "gpt-4"}, feature["meter_group_by_filters"])
}

func TestToSnakeCase(t *testing.T) {
	for in, want := range map[string]string{
		"id":                        "id",
		"createdAt":                 "created_at",
		"maskedAPIKey":              "masked_api_key",
		"intervalISO":               "interval_iso",
		"line1":                     "line1",
		"totalAvailableGrantAmount": "total_available_grant_amount",
		"$eq":                       "$eq",
	} {
		assert.Equal(t, want, toSnakeCase(in), in)
	}
}

// jsonLeaves lists every scalar value of a JSON document, ignoring where it sits.
func jsonLeaves(t *testing.T, body []byte) []string {
	t.Helper()

	var value any
	require.NoError(t, json.Unmarshal(body, &value))

	var leaves []string
	var walk func(v any)
	walk = func(v any) {
		switch v := v.(type) {
		case map[string]any:
			for _, item := range v {
				walk(item)
			}
		case []any:
			for _, item := range v {
				walk(item)
			}
		default:
			leaves = append(leaves, fmt.Sprintf("%v", v))
		}
	}
	walk(value)

	return leaves
}

func legacyFixtureEvents(t *testing.T, name string, raw json.RawMessage) []notification.Event {
	t.Helper()

	event := notification.Event{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		CreatedAt:    time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC),
	}

	withType := func(eventType notification.EventType, payload notification.EventPayload) notification.Event {
		e := event
		e.Type = eventType
		payload.EventPayloadMeta = notification.EventPayloadMeta{Type: eventType}
		e.Payload = payload

		return e
	}

	switch {
	case strings.HasPrefix(name, "invoice_"):
		var invoice apimodels.Invoice
		require.NoError(t, json.Unmarshal(raw, &invoice))

		payload := notification.EventPayload{Invoice: &notification.InvoicePayload{Invoice: invoice}}

		return []notification.Event{
			withType(notification.EventTypeInvoiceCreated, payload),
			withType(notification.EventTypeInvoiceUpdated, payload),
		}
	case strings.HasPrefix(name, "balance_threshold_"):
		var data apimodels.NotificationEventBalanceThresholdPayloadData
		require.NoError(t, json.Unmarshal(raw, &data))

		return []notification.Event{
			withType(notification.EventTypeBalanceThreshold, notification.EventPayload{
				BalanceThreshold: &notification.BalanceThresholdPayload{
					EntitlementValuePayloadBase: notification.EntitlementValuePayloadBase{
						Entitlement: data.Entitlement,
						Feature:     data.Feature,
						Subject:     data.Subject,
						Value:       data.Value,
						Customer:    lo.FromPtr(data.Customer),
					},
					Threshold: data.Threshold,
				},
			}),
		}
	case strings.HasPrefix(name, "entitlement_reset_"):
		var data apimodels.NotificationEventEntitlementValuePayloadBase
		require.NoError(t, json.Unmarshal(raw, &data))

		return []notification.Event{
			withType(notification.EventTypeEntitlementReset, notification.EventPayload{
				EntitlementReset: &notification.EntitlementResetPayload{
					Entitlement: data.Entitlement,
					Feature:     data.Feature,
					Subject:     data.Subject,
					Value:       data.Value,
					Customer:    lo.FromPtr(data.Customer),
				},
			}),
		}
	default:
		require.Failf(t, "unknown fixture", "%s", name)

		return nil
	}
}

func TestToAPIBillingNotificationEvent_MissingPayloadIsRejected(t *testing.T) {
	event := notification.Event{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		Type:         notification.EventTypeBalanceThreshold,
		Rule: notification.Rule{
			NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAW"},
			Type:         notification.EventTypeBalanceThreshold,
		},
	}

	_, err := ToAPIBillingNotificationEvent(event)
	require.Error(t, err)
}

// TestToAPIBillingNotificationEventDeliveryStatuses covers the two behaviors that differ from v1: the channel is
// reported by id only (no lookup into the rule's channel list), and attempts come back
// newest first.
func TestToAPIBillingNotificationEventDeliveryStatuses(t *testing.T) {
	older := time.Date(2024, 5, 1, 10, 0, 0, 0, time.UTC)
	newer := time.Date(2024, 5, 1, 11, 0, 0, 0, time.UTC)
	nextAttempt := time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC)

	got, err := ToAPIBillingNotificationEventDeliveryStatuses([]notification.EventDeliveryStatus{
		{
			ChannelID:   "01ARZ3NDEKTSV4RRFFQ69G5FAV",
			State:       notification.EventDeliveryStatusStateFailed,
			Reason:      "connection refused",
			UpdatedAt:   newer,
			NextAttempt: lo.ToPtr(nextAttempt),
			Attempts: []notification.EventDeliveryAttempt{
				{
					State:     notification.EventDeliveryStatusStateFailed,
					Timestamp: older,
					Response: notification.EventDeliveryAttemptResponse{
						Body:     "first",
						Duration: 1500 * time.Millisecond,
					},
				},
				{
					State:     notification.EventDeliveryStatusStateFailed,
					Timestamp: newer,
					Response: notification.EventDeliveryAttemptResponse{
						StatusCode: lo.ToPtr(500),
						Body:       "second",
						URL:        lo.ToPtr("https://example.com/hook"),
						Duration:   2 * time.Second,
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, got, 1)

	status := got[0]
	assert.Equal(t, "01ARZ3NDEKTSV4RRFFQ69G5FAV", status.ChannelId)
	assert.Equal(t, api.BillingNotificationEventDeliveryStateFailed, status.State)
	assert.Equal(t, "connection refused", status.Reason)
	assert.Equal(t, newer, status.UpdatedAt)
	assert.Equal(t, nextAttempt, lo.FromPtr(status.NextAttempt))

	require.Len(t, status.Attempts, 2)
	assert.Equal(t, newer, status.Attempts[0].Timestamp, "attempts must be newest first")
	assert.Equal(t, int32(500), lo.FromPtr(status.Attempts[0].Response.StatusCode))
	assert.Equal(t, int64(2000), status.Attempts[0].Response.DurationMs)
	assert.Equal(t, "https://example.com/hook", lo.FromPtr(status.Attempts[0].Response.Url))

	assert.Equal(t, older, status.Attempts[1].Timestamp)
	assert.Nil(t, status.Attempts[1].Response.StatusCode, "a missing status code must stay absent, not become 0")
	assert.Equal(t, int64(1500), status.Attempts[1].Response.DurationMs)
}
