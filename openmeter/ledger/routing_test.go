package ledger

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildRoutingKeyV1(t *testing.T) {
	tax := "01TAX"
	feature := "01FEAT"
	priority := "01PRIO"

	key, err := BuildRoutingKeyV1(SubAccountRouteInput{
		CurrencyDimensionID:       "01CUR",
		TaxCodeDimensionID:        &tax,
		FeaturesDimensionID:       &feature,
		CreditPriorityDimensionID: &priority,
	})
	require.NoError(t, err)
	require.Equal(t, RoutingKeyVersionV1, key.Version())
	require.Equal(t, "currency:01CUR|tax_code:01TAX|features:01FEAT|credit_priority:01PRIO", key.Value())
}

func TestBuildRoutingKeyV1_Nulls(t *testing.T) {
	key, err := BuildRoutingKeyV1(SubAccountRouteInput{
		CurrencyDimensionID: "01CUR",
	})
	require.NoError(t, err)
	require.Equal(t, "currency:01CUR|tax_code:null|features:null|credit_priority:null", key.Value())
}
