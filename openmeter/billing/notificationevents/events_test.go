package notificationevents

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/require"
)

func TestInvoiceCreatedContract(t *testing.T) {
	require.Equal(t, "v1.billing.invoice.created", BillingInvoiceCreatedV1EventType.String())
	require.NoError(t, BillingInvoiceCreatedV1EventType.Validate())

	// given the schema generated from the same TypeSpec model as the struct
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(BillingInvoiceCreatedV1PayloadJSONSchema))
	require.NoError(t, err)

	c := jsonschema.NewCompiler()
	require.NoError(t, c.AddResource("schema.json", doc))

	schema, err := c.Compile("schema.json")
	require.NoError(t, err)

	// when business logic marshals the generated struct
	raw, err := json.Marshal(BillingInvoiceCreatedV1Payload{
		InvoiceId:  "01J2KNAG7SY9X6V8E5P4D3C2ZF",
		CustomerId: "01J2KNBH8TZ0Y7W9F6Q5E4D3AG",
		Status:     "draft",
		Currency:   "USD",
		Total:      "12.50",
	})
	require.NoError(t, err)

	payload, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	require.NoError(t, err)

	// then the schema accepts it
	require.NoError(t, schema.Validate(payload))
}
