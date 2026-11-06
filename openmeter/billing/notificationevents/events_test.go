package notificationevents

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/require"
)

func TestInvoiceContracts(t *testing.T) {
	createdAt := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)

	full := BillingInvoiceSnapshot{
		Id:          "01J2KNAG7SY9X6V8E5P4D3C2ZF",
		Number:      "INV-0001",
		Type:        "standard",
		Status:      "issued",
		CustomerId:  "01J2KNBH8TZ0Y7W9F6Q5E4D3AG",
		CustomerKey: lo.ToPtr("acme"),
		Currency:    "USD",
		Total:       "12.50",
		ServicePeriod: &BillingInvoiceSnapshotPeriod{
			From: createdAt.AddDate(0, -1, 0),
			To:   createdAt,
		},
		IssuedAt:  lo.ToPtr(createdAt.Add(time.Hour)),
		DueAt:     lo.ToPtr(createdAt.AddDate(0, 0, 30)),
		CreatedAt: createdAt,
		UpdatedAt: createdAt.Add(time.Hour),
	}

	// Optional fields are omitted, and the status is one the schema does not
	// list: statuses are open, so new ones need no new Event type version.
	minimal := BillingInvoiceSnapshot{
		Id:         "01J2KNAG7SY9X6V8E5P4D3C2ZF",
		Number:     "INV-0002",
		Type:       "standard",
		Status:     "some_future_status",
		CustomerId: "01J2KNBH8TZ0Y7W9F6Q5E4D3AG",
		Currency:   "USD",
		Total:      "0",
		CreatedAt:  createdAt,
		UpdatedAt:  createdAt,
	}

	contracts := []struct {
		eventType string
		schema    []byte
		payload   func(BillingInvoiceSnapshot) any
	}{
		{
			eventType: BillingInvoiceCreatedV1EventType.String(),
			schema:    BillingInvoiceCreatedV1PayloadJSONSchema,
			payload:   func(i BillingInvoiceSnapshot) any { return BillingInvoiceCreatedV1Payload{Invoice: i} },
		},
		{
			eventType: BillingInvoiceUpdatedV1EventType.String(),
			schema:    BillingInvoiceUpdatedV1PayloadJSONSchema,
			payload:   func(i BillingInvoiceSnapshot) any { return BillingInvoiceUpdatedV1Payload{Invoice: i} },
		},
	}

	require.Equal(t, "v1.billing.invoice.created", contracts[0].eventType)
	require.Equal(t, "v1.billing.invoice.updated", contracts[1].eventType)

	for _, contract := range contracts {
		t.Run(contract.eventType, func(t *testing.T) {
			// given the schema generated from the same TypeSpec model as the struct
			doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(contract.schema))
			require.NoError(t, err)

			c := jsonschema.NewCompiler()
			require.NoError(t, c.AddResource("schema.json", doc))

			schema, err := c.Compile("schema.json")
			require.NoError(t, err)

			for name, invoice := range map[string]BillingInvoiceSnapshot{"full": full, "minimal": minimal} {
				// when business logic marshals the generated struct
				raw, err := json.Marshal(contract.payload(invoice))
				require.NoError(t, err)

				payload, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
				require.NoError(t, err)

				// then the schema accepts it
				require.NoError(t, schema.Validate(payload), name)
			}
		})
	}
}
