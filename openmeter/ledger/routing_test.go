package ledger

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

func TestBuildRoutingKeyV1(t *testing.T) {
	priority := 7

	key, err := BuildRoutingKeyV1(Route{
		Currency:       "USD",
		CreditPriority: &priority,
	})
	require.NoError(t, err)
	require.Equal(t, RoutingKeyVersionV1, key.Version())
	require.Equal(t, "currency:USD|tax_code:null|features:null|credit_priority:7", key.Value())
}

func TestBuildRoutingKeyV1_Nulls(t *testing.T) {
	key, err := BuildRoutingKeyV1(Route{
		Currency: "USD",
	})
	require.NoError(t, err)
	require.Equal(t, "currency:USD|tax_code:null|features:null|credit_priority:null", key.Value())
}

func TestBuildRoutingKeyV1_SameLiterals_SameKey(t *testing.T) {
	priority := 100
	input := Route{
		Currency:       "USD",
		CreditPriority: &priority,
	}

	key1, err := BuildRoutingKeyV1(input)
	require.NoError(t, err)
	key2, err := BuildRoutingKeyV1(input)
	require.NoError(t, err)
	require.Equal(t, key1.Value(), key2.Value())
}

func TestBuildRoutingKeyV1_DifferentCurrency_DifferentKey(t *testing.T) {
	key1, err := BuildRoutingKeyV1(Route{Currency: "USD"})
	require.NoError(t, err)
	key2, err := BuildRoutingKeyV1(Route{Currency: "EUR"})
	require.NoError(t, err)
	require.NotEqual(t, key1.Value(), key2.Value())
}

func TestBuildRoutingKeyV1_WithTaxCodeAndFeatures(t *testing.T) {
	key, err := BuildRoutingKeyV1(Route{
		Currency: "USD",
		TaxCode:  lo.ToPtr("VAT20"),
		Features: []string{"feat-b", "feat-a"},
	})
	require.NoError(t, err)
	// Features are sorted canonically
	require.Equal(t, "currency:USD|tax_code:VAT20|features:feat-a,feat-b|credit_priority:null", key.Value())
}

func TestBuildRoutingKeyV1_EmptyFeatures(t *testing.T) {
	key, err := BuildRoutingKeyV1(Route{
		Currency: "USD",
		Features: []string{},
	})
	require.NoError(t, err)
	require.Equal(t, "currency:USD|tax_code:null|features:null|credit_priority:null", key.Value())
}

func TestBuildRoutingKeyV1_DifferentPriority_DifferentKey(t *testing.T) {
	key1, err := BuildRoutingKeyV1(Route{Currency: "USD", CreditPriority: lo.ToPtr(1)})
	require.NoError(t, err)
	key2, err := BuildRoutingKeyV1(Route{Currency: "USD", CreditPriority: lo.ToPtr(2)})
	require.NoError(t, err)
	require.NotEqual(t, key1.Value(), key2.Value())
}
