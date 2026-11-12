package httpdriver

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestChannelTypeCompatibility(t *testing.T) {
	tests := []struct {
		name    string
		value   api.NotificationChannelWebhookCreateRequestType
		wantErr bool
	}{
		{name: "legacy uppercase", value: "WEBHOOK"},
		{name: "lowercase is not a V1 value", value: "webhook", wantErr: true},
		{name: "unknown", value: "unknown", wantErr: true},
		{name: "empty", value: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := api.NotificationChannelWebhookCreateRequest{
				Type: tt.value,
				Name: "webhook",
				Url:  "https://example.com/hook",
			}

			created, createErr := AsChannelWebhookCreateRequest(request, "namespace")
			updated, updateErr := AsChannelWebhookUpdateRequest(request, "namespace", "channel")

			if tt.wantErr {
				require.Error(t, createErr)
				require.Error(t, updateErr)

				assert.True(t, models.IsGenericValidationError(createErr))
				assert.True(t, models.IsGenericValidationError(updateErr))

				return
			}

			require.NoError(t, createErr)
			require.NoError(t, updateErr)

			assert.Equal(t, notification.ChannelType("webhook"), created.Type)
			assert.Equal(t, created.Type, created.Config.Type)
			assert.Equal(t, created.Type, updated.Type)
			assert.Equal(t, updated.Type, updated.Config.Type)

			channel, err := FromChannel(notification.Channel{Type: created.Type, Config: created.Config})
			require.NoError(t, err)

			assert.Equal(t, api.NotificationChannelWebhookTypeWEBHOOK, channel.Type)
			assert.Equal(t, api.NotificationChannelTypeWebhook, ToAPINotificationChannelType(created.Type))
		})
	}
}

func TestRuleChannelTypeCompatibility(t *testing.T) {
	tests := []struct {
		name      string
		eventType notification.EventType
	}{
		{name: "balance threshold", eventType: notification.EventTypeBalanceThreshold},
		{name: "entitlement reset", eventType: notification.EventTypeEntitlementReset},
		{name: "invoice created", eventType: notification.EventTypeInvoiceCreated},
		{name: "invoice updated", eventType: notification.EventTypeInvoiceUpdated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, err := FromRule(notification.Rule{
				Type: tt.eventType,
				Config: notification.RuleConfig{
					BalanceThreshold: &notification.BalanceThresholdRuleConfig{},
					EntitlementReset: &notification.EntitlementResetRuleConfig{},
					Invoice:          &notification.InvoiceRuleConfig{},
				},
				Channels: []notification.Channel{{Type: notification.ChannelTypeWebhook}},
			})
			require.NoError(t, err)

			encoded, err := json.Marshal(rule)
			require.NoError(t, err)

			var decoded struct {
				Channels []api.NotificationChannelMeta `json:"channels"`
			}

			require.NoError(t, json.Unmarshal(encoded, &decoded))
			require.Len(t, decoded.Channels, 1)

			assert.Equal(t, api.NotificationChannelTypeWebhook, decoded.Channels[0].Type)
		})
	}
}

func invoiceEvent(invoice *notification.InvoicePayload) notification.Event {
	return notification.Event{
		NamespacedID: models.NamespacedID{ID: "event-id"},
		CreatedAt:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Type:         notification.EventTypeInvoiceCreated,
		Payload: notification.EventPayload{
			EventPayloadMeta: notification.EventPayloadMeta{
				Type:    notification.EventTypeInvoiceCreated,
				Version: notification.EventPayloadVersionCurrent,
			},
			Invoice: invoice,
		},
	}
}

func TestFromEventAsInvoiceCreatedPayload(t *testing.T) {
	t.Run("passes through api.Invoice into Data", func(t *testing.T) {
		invoice := api.Invoice{Id: "inv-1", Currency: "USD"}
		event := invoiceEvent(&notification.InvoicePayload{Invoice: invoice})

		got, err := FromEventAsInvoiceCreatedPayload(event)
		require.NoError(t, err)

		assert.Equal(t, "event-id", got.Id)
		assert.Equal(t, api.NotificationEventInvoiceCreatedPayloadTypeInvoiceCreated, got.Type)
		assert.Equal(t, invoice, got.Data)
	})

	t.Run("returns error when invoice payload is nil", func(t *testing.T) {
		event := invoiceEvent(nil)

		_, err := FromEventAsInvoiceCreatedPayload(event)
		require.Error(t, err)
	})
}

func TestFromEventAsInvoiceUpdatedPayload(t *testing.T) {
	t.Run("passes through api.Invoice into Data", func(t *testing.T) {
		invoice := api.Invoice{Id: "inv-2", Currency: "EUR"}
		event := invoiceEvent(&notification.InvoicePayload{Invoice: invoice})
		event.Type = notification.EventTypeInvoiceUpdated
		event.Payload.Type = notification.EventTypeInvoiceUpdated

		got, err := FromEventAsInvoiceUpdatedPayload(event)
		require.NoError(t, err)

		assert.Equal(t, "event-id", got.Id)
		assert.Equal(t, api.NotificationEventInvoiceUpdatedPayloadTypeInvoiceUpdated, got.Type)
		assert.Equal(t, invoice, got.Data)
	})

	t.Run("returns error when invoice payload is nil", func(t *testing.T) {
		event := invoiceEvent(nil)

		_, err := FromEventAsInvoiceUpdatedPayload(event)
		require.Error(t, err)
	})
}
