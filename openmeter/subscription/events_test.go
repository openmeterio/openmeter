package subscription

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestSubscriptionEventInvoiceCurrencyWireCompatibility(t *testing.T) {
	// given:
	// - a subscription.created.v1 payload written before Currency was renamed in Go
	// when:
	// - the current event model replays and rewrites that payload
	// then:
	// - both the entity and spec retain the invoice fiat under the stable v1 key
	const legacyPayload = `{
		"subscription": {"currency": "USD"},
		"spec": {"currency": "USD"}
	}`

	var event CreatedEvent
	require.NoError(t, json.Unmarshal([]byte(legacyPayload), &event))
	require.Equal(t, currencyx.Code("USD"), event.Subscription.InvoiceCurrency)
	require.Equal(t, currencyx.Code("USD"), event.Spec.InvoiceCurrency)

	payload, err := json.Marshal(event)
	require.NoError(t, err)

	var serialized map[string]any
	require.NoError(t, json.Unmarshal(payload, &serialized))

	serializedSubscription := serialized["subscription"].(map[string]any)
	require.Equal(t, "USD", serializedSubscription["currency"])
	require.NotContains(t, serializedSubscription, "invoiceCurrency")

	serializedSpec := serialized["spec"].(map[string]any)
	require.Equal(t, "USD", serializedSpec["currency"])
	require.NotContains(t, serializedSpec, "invoiceCurrency")
}
