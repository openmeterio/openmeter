package eventhandler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apimodels "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestEventAsPayload(t *testing.T) {
	invoice := &notification.InvoicePayload{Invoice: apimodels.Invoice{Id: "inv-1"}}
	reset := &notification.EntitlementResetPayload{Subject: apimodels.Subject{Key: "subject-1"}}
	threshold := &notification.BalanceThresholdPayload{
		Threshold: apimodels.NotificationRuleBalanceThresholdValue{Type: apimodels.NotificationRuleBalanceThresholdValueTypeUsageValue, Value: 80},
	}

	tests := []struct {
		eventType notification.EventType
		payload   notification.EventPayload
		dataPath  []string
		wantData  any
		nilError  string
	}{
		{
			eventType: notification.EventTypeBalanceThreshold,
			payload:   notification.EventPayload{BalanceThreshold: threshold},
			dataPath:  []string{"threshold", "value"},
			wantData:  80.0,
			nilError:  "balance threshold is nil",
		},
		{
			eventType: notification.EventTypeEntitlementReset,
			payload:   notification.EventPayload{EntitlementReset: reset},
			dataPath:  []string{"subject", "key"},
			wantData:  "subject-1",
			nilError:  "entitlement reset is nil",
		},
		{
			eventType: notification.EventTypeInvoiceCreated,
			payload:   notification.EventPayload{Invoice: invoice},
			dataPath:  []string{"id"},
			wantData:  "inv-1",
			nilError:  "invoice is nil",
		},
		{
			eventType: notification.EventTypeInvoiceUpdated,
			payload:   notification.EventPayload{Invoice: invoice},
			dataPath:  []string{"id"},
			wantData:  "inv-1",
			nilError:  "invoice is nil",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.eventType), func(t *testing.T) {
			event := notification.Event{
				NamespacedID: models.NamespacedID{ID: "event-1"},
				Type:         tt.eventType,
				CreatedAt:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Payload:      tt.payload,
			}
			payload, err := eventAsPayload(&event)
			require.NoError(t, err)
			require.Equal(t, event.ID, payload["id"])
			require.Equal(t, string(tt.eventType), payload["type"])
			require.Equal(t, "2024-01-01T00:00:00Z", payload["timestamp"])
			value := payload["data"]
			for _, key := range tt.dataPath {
				object, ok := value.(map[string]any)
				require.True(t, ok, "expected object containing %q", key)
				value = object[key]
			}

			require.Equal(t, tt.wantData, value)

			event.Payload = notification.EventPayload{}
			_, err = eventAsPayload(&event)
			require.EqualError(t, err, "failed to cast event payload: "+tt.nilError)
		})
	}
}
