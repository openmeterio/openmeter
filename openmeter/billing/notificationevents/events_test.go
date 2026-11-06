package notificationevents

import (
	"bytes"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/require"
)

// The payloads reuse the v3 invoice model; the v3 invoice handler tests check
// that real API output is a valid payload.
func TestInvoiceContracts(t *testing.T) {
	require.Equal(t, "v1.billing.invoice.created", BillingInvoiceCreatedV1EventType.String())
	require.Equal(t, "v1.billing.invoice.updated", BillingInvoiceUpdatedV1EventType.String())

	for _, schemaJSON := range [][]byte{
		BillingInvoiceCreatedV1PayloadJSONSchema,
		BillingInvoiceUpdatedV1PayloadJSONSchema,
	} {
		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaJSON))
		require.NoError(t, err)

		c := jsonschema.NewCompiler()
		require.NoError(t, c.AddResource("schema.json", doc))

		_, err = c.Compile("schema.json")
		require.NoError(t, err)
	}
}
