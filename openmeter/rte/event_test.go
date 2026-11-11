package rte_test

import (
	"encoding/json"
	"testing"

	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

func TestCalculationRequestPreservesCreditScope(t *testing.T) {
	for name, filter := range map[string]mo.Option[creditpurchase.FeatureFilters]{
		"all":                  customerbalance.AllFeatureFilter(),
		"unrestricted":         customerbalance.NewUnrestrictedFeatureFilter(),
		"empty feature filter": customerbalance.NewFeatureFilter(nil),
		"feature":              customerbalance.NewFeatureFilter([]string{"feature"}),
	} {
		t.Run(name, func(t *testing.T) {
			// Given every supported feature scope and an exact custom currency identity.
			key := creditbalance.Key{
				CustomerID:    customer.CustomerID{Namespace: "rte", ID: "customer"},
				Currency:      currencies.CurrencyReference{Code: "POINTS", CustomCurrencyID: lo.ToPtr("currency")},
				FeatureFilter: filter,
			}
			event, err := rte.NewCalculationRequestedEvent(key)
			require.NoError(t, err)

			// When the request crosses the actual CloudEvent wire boundary.
			codec := marshaler.New(nil)
			message, err := codec.Marshal(event)
			require.NoError(t, err)
			var decoded rte.CalculationRequestedEvent
			require.NoError(t, codec.Unmarshal(message, &decoded))
			decodedKey, err := decoded.DecodeKey()
			require.NoError(t, err)
			actual := decodedKey.(creditbalance.Key)

			// Then routing and calculation keep the original scope and currency.
			require.Equal(t, key.CustomerID, actual.CustomerID)
			require.Equal(t, key.Currency, actual.Currency)
			require.Equal(t, key.FeatureScope(), actual.FeatureScope())
			require.Equal(t, event.EventMetadata(), decoded.EventMetadata())
		})
	}
}

func TestCalculationRequestRejectsInvalidIdentity(t *testing.T) {
	for _, event := range []rte.CalculationTarget{
		{Kind: "unknown", Key: json.RawMessage(`{}`)},
		{Kind: rte.CalculationKindCreditBalance, Key: json.RawMessage(`null`)},
		{Kind: rte.CalculationKindMeteredEntitlement, Key: json.RawMessage(`{"customerId":{"namespace":"rte","id":"customer"}}`)},
	} {
		require.Error(t, event.Validate())
	}
}
