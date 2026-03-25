package ledger

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestBuildRoutingKeyV1(t *testing.T) {
	priority := 7
	costBasis := mustDecimal(t, "0.7")

	key, err := BuildRoutingKeyV1(Route{
		Currency:       currencyx.Code("USD"),
		CostBasis:      &costBasis,
		CreditPriority: &priority,
	})
	require.NoError(t, err)
	require.Equal(t, RoutingKeyVersionV1, key.Version())
	require.Equal(t, "currency:USD|tax_code:null|features:null|cost_basis:0.7|credit_priority:7|transaction_authorization_status:null", key.Value())
}

func TestBuildRoutingKeyV1_Nulls(t *testing.T) {
	key, err := BuildRoutingKeyV1(Route{
		Currency: currencyx.Code("USD"),
	})
	require.NoError(t, err)
	require.Equal(t, "currency:USD|tax_code:null|features:null|cost_basis:null|credit_priority:null|transaction_authorization_status:null", key.Value())
}

func TestBuildRoutingKeyV1_SameLiterals_SameKey(t *testing.T) {
	priority := 100
	input := Route{
		Currency:       currencyx.Code("USD"),
		CreditPriority: &priority,
	}

	key1, err := BuildRoutingKeyV1(input)
	require.NoError(t, err)
	key2, err := BuildRoutingKeyV1(input)
	require.NoError(t, err)
	require.Equal(t, key1.Value(), key2.Value())
}

func TestBuildRoutingKeyV1_DifferentCurrency_DifferentKey(t *testing.T) {
	key1, err := BuildRoutingKeyV1(Route{Currency: currencyx.Code("USD")})
	require.NoError(t, err)
	key2, err := BuildRoutingKeyV1(Route{Currency: currencyx.Code("EUR")})
	require.NoError(t, err)
	require.NotEqual(t, key1.Value(), key2.Value())
}

func TestBuildRoutingKeyV1_WithTaxCodeAndFeatures(t *testing.T) {
	key, err := BuildRoutingKeyV1(Route{
		Currency: currencyx.Code("USD"),
		TaxCode:  lo.ToPtr("VAT20"),
		Features: []string{"feat-b", "feat-a"},
	})
	require.NoError(t, err)
	// Features are sorted canonically
	require.Equal(t, "currency:USD|tax_code:VAT20|features:feat-a,feat-b|cost_basis:null|credit_priority:null|transaction_authorization_status:null", key.Value())
}

func TestBuildRoutingKeyV1_EmptyFeatures(t *testing.T) {
	key, err := BuildRoutingKeyV1(Route{
		Currency: currencyx.Code("USD"),
		Features: []string{},
	})
	require.NoError(t, err)
	require.Equal(t, "currency:USD|tax_code:null|features:null|cost_basis:null|credit_priority:null|transaction_authorization_status:null", key.Value())
}

func TestBuildRoutingKeyV1_DifferentPriority_DifferentKey(t *testing.T) {
	key1, err := BuildRoutingKeyV1(Route{Currency: currencyx.Code("USD"), CreditPriority: lo.ToPtr(1)})
	require.NoError(t, err)
	key2, err := BuildRoutingKeyV1(Route{Currency: currencyx.Code("USD"), CreditPriority: lo.ToPtr(2)})
	require.NoError(t, err)
	require.NotEqual(t, key1.Value(), key2.Value())
}

func TestBuildRoutingKeyV1_CanonicalizesCostBasis(t *testing.T) {
	key1, err := BuildRoutingKeyV1(Route{Currency: currencyx.Code("USD"), CostBasis: lo.ToPtr(mustDecimal(t, "0.70"))})
	require.NoError(t, err)

	key2, err := BuildRoutingKeyV1(Route{Currency: currencyx.Code("USD"), CostBasis: lo.ToPtr(mustDecimal(t, "0.7"))})
	require.NoError(t, err)

	require.Equal(t, key1.Value(), key2.Value())
	require.Equal(t, "currency:USD|tax_code:null|features:null|cost_basis:0.7|credit_priority:null|transaction_authorization_status:null", key1.Value())
}

func TestBuildRoutingKeyV1_DifferentAuthorizationStatus_DifferentKey(t *testing.T) {
	status := TransactionAuthorizationStatusAuthorized

	key1, err := BuildRoutingKeyV1(Route{Currency: currencyx.Code("USD")})
	require.NoError(t, err)

	key2, err := BuildRoutingKeyV1(Route{
		Currency:                       currencyx.Code("USD"),
		TransactionAuthorizationStatus: &status,
	})
	require.NoError(t, err)

	require.NotEqual(t, key1.Value(), key2.Value())
	require.Equal(t, "currency:USD|tax_code:null|features:null|cost_basis:null|credit_priority:null|transaction_authorization_status:authorized", key2.Value())
}

func mustDecimal(t *testing.T, raw string) alpacadecimal.Decimal {
	t.Helper()

	value, err := alpacadecimal.NewFromString(raw)
	require.NoError(t, err)

	return value
}
