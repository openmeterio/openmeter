package billinginvoices

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingevents "github.com/openmeterio/openmeter/openmeter/billing/notificationevents"
	"github.com/openmeterio/openmeter/pkg/models"
)

// The invoice Event payloads reuse the v3 invoice model, so what the v3 API
// returns for an invoice must be a valid payload. The schemas are sealed: a
// field the API adds that the contract lacks fails here.
func TestInvoiceEventPayloadsAcceptV3Invoice(t *testing.T) {
	period := mergeTestPeriod()
	createdAt := period.To

	line := standardLineForMergeTest(t, "01J2KNCJ9V01Z8XAG7R6F5E4BH", period)
	line.UsageBased.FeatureKey = "api_requests"

	invoice := billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Namespace:   "ns",
			ID:          "01J2KNAG7SY9X6V8E5P4D3C2ZF",
			Number:      "INV-0001",
			Description: lo.ToPtr("March usage"),
			Metadata:    models.Metadata{"team": "billing"},
			Currency:    "USD",
			Status:      billing.StandardInvoiceStatusIssued,
			Period:      &period,
			CreatedAt:   createdAt,
			UpdatedAt:   createdAt.Add(time.Hour),
			IssuedAt:    lo.ToPtr(createdAt.Add(time.Hour)),
			DueAt:       lo.ToPtr(createdAt.AddDate(0, 0, 30)),
			Customer: billing.InvoiceCustomer{
				CustomerID: "01J2KNBH8TZ0Y7W9F6Q5E4D3AG",
				Key:        lo.ToPtr("acme"),
				Name:       "Acme Inc.",
			},
			Workflow: billing.InvoiceWorkflow{
				AppReferences: billing.ProfileAppReferences{
					Tax:       app.AppID{Namespace: "ns", ID: "01J2KNDK0W12A9YBH8S7G6F5CJ"},
					Invoicing: app.AppID{Namespace: "ns", ID: "01J2KNDK0W12A9YBH8S7G6F5CJ"},
					Payment:   app.AppID{Namespace: "ns", ID: "01J2KNDK0W12A9YBH8S7G6F5CJ"},
				},
				SourceBillingProfileID: "01J2KNEM1X23BACCJ9T8H7G6DK",
			},
		},
		Lines: billing.NewStandardInvoiceLines([]*billing.StandardLine{
			line,
		}),
	}

	apiInvoice, err := toAPIStandardInvoice(invoice)
	require.NoError(t, err)

	raw, err := json.Marshal(map[string]any{"invoice": apiInvoice})
	require.NoError(t, err)

	for eventType, tc := range map[string]struct {
		schema []byte
		// typed returns a pointer to the generated payload struct of the type.
		typed func() any
	}{
		billingevents.BillingInvoiceCreatedV1EventType.String(): {
			schema: billingevents.BillingInvoiceCreatedV1PayloadJSONSchema,
			typed:  func() any { return &billingevents.BillingInvoiceCreatedV1Payload{} },
		},
		billingevents.BillingInvoiceUpdatedV1EventType.String(): {
			schema: billingevents.BillingInvoiceUpdatedV1PayloadJSONSchema,
			typed:  func() any { return &billingevents.BillingInvoiceUpdatedV1Payload{} },
		},
	} {
		t.Run(eventType, func(t *testing.T) {
			doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(tc.schema))
			require.NoError(t, err)

			c := jsonschema.NewCompiler()
			require.NoError(t, c.AddResource("schema.json", doc))

			schema, err := c.Compile("schema.json")
			require.NoError(t, err)

			payload, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
			require.NoError(t, err)

			require.NoError(t, schema.Validate(payload))

			// The generated payload struct round-trips the API JSON unchanged.
			typed := tc.typed()
			require.NoError(t, json.Unmarshal(raw, typed))

			again, err := json.Marshal(typed)
			require.NoError(t, err)
			require.JSONEq(t, string(raw), string(again))
		})
	}
}
