package httpdriver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/models"
)

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
