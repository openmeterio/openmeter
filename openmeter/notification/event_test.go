package notification_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestCreateEventInputPayloadTypeValidation(t *testing.T) {
	types := []notification.EventType{
		notification.EventTypeBalanceThreshold,
		notification.EventTypeEntitlementReset,
		notification.EventTypeInvoiceCreated,
		notification.EventTypeInvoiceUpdated,
	}

	for _, eventType := range types {
		for _, payloadType := range types {
			t.Run(string(eventType)+"/"+string(payloadType), func(t *testing.T) {
				input := notification.CreateEventInput{
					Type: eventType,
					Payload: notification.EventPayload{
						EventPayloadMeta: notification.EventPayloadMeta{Type: payloadType},
					},
				}

				err := input.Validate()
				if eventType == payloadType {
					require.NoError(t, err)

					return
				}

				require.ErrorContains(t, err, "invalid payload type")

				require.True(t, models.IsGenericValidationError(err))
			})
		}
	}
}
