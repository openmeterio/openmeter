package adapter

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billinghttp "github.com/openmeterio/openmeter/openmeter/billing/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/notification"
)

// minimalLegacyInvoice builds a billing.EventStandardInvoice with enough fields
// populated for MapEventInvoiceToAPI to succeed without error.
func minimalLegacyInvoice() billing.EventStandardInvoice {
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return billing.EventStandardInvoice{
		Invoice: billing.StandardInvoice{
			StandardInvoiceBase: billing.StandardInvoiceBase{
				ID:        "test-invoice-id",
				Namespace: "test-namespace",
				Type:      billing.InvoiceTypeStandard,
				Status:    billing.StandardInvoiceStatusIssued,
				Currency:  "USD",
				CreatedAt: now,
				UpdatedAt: now,
				Customer: billing.InvoiceCustomer{
					CustomerID: "test-customer-id",
					Name:       "Test Customer",
				},
				Supplier: billing.SupplierContact{
					ID:   "test-supplier-id",
					Name: "Test Supplier",
				},
				Workflow: billing.InvoiceWorkflow{
					Config: billing.WorkflowConfig{
						Collection: billing.CollectionConfig{
							Alignment: billing.AlignmentKindSubscription,
						},
					},
				},
			},
		},
	}
}

func TestEventPayloadFromJSON_V0LegacyInvoice(t *testing.T) {
	event := minimalLegacyInvoice()

	// Build the legacy JSONB shape: no "version" field, invoice stored as billing.EventStandardInvoice
	legacyEnvelope := struct {
		notification.EventPayloadMeta
		Invoice *billing.EventStandardInvoice `json:"invoice,omitempty"`
	}{
		EventPayloadMeta: notification.EventPayloadMeta{
			Type: notification.EventTypeInvoiceCreated,
			// Version intentionally omitted → zero value → v0 path
		},
		Invoice: &event,
	}

	data, err := json.Marshal(legacyEnvelope)
	require.NoError(t, err)

	// Compute expected output using the same transformation as the v0 read path
	expectedInvoice, err := billinghttp.MapEventInvoiceToAPI(event)
	require.NoError(t, err)

	payload, err := eventPayloadFromJSON(data)
	require.NoError(t, err)

	require.NotNil(t, payload.Invoice)
	assert.Equal(t, expectedInvoice, payload.Invoice.Invoice)
}

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

func TestEventPayloadFromJSON_V0MissingInvoice(t *testing.T) {
	// v0 payload with invoice type but no "invoice" key
	data := []byte(`{"type":"invoice.created"}`)

	_, err := eventPayloadFromJSON(data)
	require.ErrorContains(t, err, "missing invoice")
}

func TestEventPayloadFromJSON_UnknownVersion(t *testing.T) {
	data := []byte(`{"type":"invoice.created","version":99}`)

	_, err := eventPayloadFromJSON(data)
	require.ErrorContains(t, err, "unsupported")
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
