package adapter

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/notification"
)

func TestEventPayloadFromJSON_V1Invoice(t *testing.T) {
	original := notification.EventPayload{
		EventPayloadMeta: notification.EventPayloadMeta{
			Type:    notification.EventTypeInvoiceCreated,
			Version: notification.EventPayloadVersionCurrent,
		},
		Invoice: &notification.InvoicePayload{
			Invoice: api.Invoice{
				Id:       "test-invoice-id",
				Currency: "USD",
			},
		},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	payload, err := eventPayloadFromJSON(data)
	require.NoError(t, err)

	require.NotNil(t, payload.Invoice)
	assert.Equal(t, original.Invoice.Invoice, payload.Invoice.Invoice)
}

func TestEventPayloadFromJSON_UnsupportedVersion(t *testing.T) {
	// version 0 (legacy) and any unknown version are both rejected
	for _, data := range [][]byte{
		[]byte(`{"type":"invoice.created"}`),              // missing version → 0
		[]byte(`{"type":"invoice.created","version":99}`), // unknown future version
	} {
		_, err := eventPayloadFromJSON(data)
		require.ErrorContains(t, err, "unsupported")
	}
}

func TestEventPayloadFromJSON_NonInvoiceType(t *testing.T) {
	original := notification.EventPayload{
		EventPayloadMeta: notification.EventPayloadMeta{
			Type: notification.EventTypeBalanceThreshold,
		},
		BalanceThreshold: &notification.BalanceThresholdPayload{},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	payload, err := eventPayloadFromJSON(data)
	require.NoError(t, err)

	assert.NotNil(t, payload.BalanceThreshold)
	assert.Nil(t, payload.Invoice)
}

// TestEventPayloadV1JSONShape verifies that a v1 EventPayload serializes with api.Invoice
// fields directly under the "invoice" key — not nested under an extra "Invoice" wrapper.
// This guards against embedding/tag regressions that would silently corrupt the stored JSONB.
func TestEventPayloadV1JSONShape(t *testing.T) {
	payload := notification.EventPayload{
		EventPayloadMeta: notification.EventPayloadMeta{
			Type:    notification.EventTypeInvoiceCreated,
			Version: notification.EventPayloadVersionCurrent,
		},
		Invoice: &notification.InvoicePayload{
			Invoice: api.Invoice{
				Id:       "shape-test-id",
				Currency: "USD",
			},
		},
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(data, &raw))

	// Top-level "version" must be present and correct
	assert.EqualValues(t, notification.EventPayloadVersionCurrent, raw["version"])

	// "invoice" must be a flat object containing api.Invoice fields directly
	invoiceRaw, ok := raw["invoice"].(map[string]any)
	require.True(t, ok, "expected \"invoice\" to be a JSON object")

	assert.Equal(t, "shape-test-id", invoiceRaw["id"], "api.Invoice.Id should be at invoice.id")
	assert.Equal(t, "USD", invoiceRaw["currency"], "api.Invoice.Currency should be at invoice.currency")

	// Must NOT have an extra nesting level (e.g. invoice.Invoice or invoice.invoice)
	assert.NotContains(t, invoiceRaw, "Invoice", "invoice object must not have an extra \"Invoice\" wrapper")
	assert.NotContains(t, invoiceRaw, "invoice", "invoice object must not have an extra \"invoice\" wrapper")
}
