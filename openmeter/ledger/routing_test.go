package ledger

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestBuildRoutingKeyV1(t *testing.T) {
	priority := 7
	costBasis := mustDecimal(t, "0.7")
	taxcode := "GST10"

	key, err := BuildRoutingKeyV1(Route{
		TaxCode:        &taxcode,
		TaxBehavior:    nil,
		Currency:       currencyx.Code("USD"),
		CostBasis:      &costBasis,
		CreditPriority: &priority,
	})
	require.NoError(t, err)
	require.Equal(t, RoutingKeyVersionV1, key.Version())
	require.Equal(t, "currency:USD|tax_code:GST10|tax_behavior:null|features:null|cost_basis:0.7|credit_priority:7|transaction_authorization_status:null", key.Value())
}

func TestBuildRoutingKeyV1_Nulls(t *testing.T) {
	key, err := BuildRoutingKeyV1(Route{
		Currency: currencyx.Code("USD"),
	})
	require.NoError(t, err)
	require.Equal(t, "currency:USD|tax_code:null|tax_behavior:null|features:null|cost_basis:null|credit_priority:null|transaction_authorization_status:null", key.Value())
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
	require.Equal(t, "currency:USD|tax_code:VAT20|tax_behavior:null|features:feat-a,feat-b|cost_basis:null|credit_priority:null|transaction_authorization_status:null", key.Value())
}

func TestBuildRoutingKeyV1_EmptyFeatures(t *testing.T) {
	key, err := BuildRoutingKeyV1(Route{
		Currency: currencyx.Code("USD"),
		Features: []string{},
	})
	require.NoError(t, err)
	require.Equal(t, "currency:USD|tax_code:null|tax_behavior:null|features:null|cost_basis:null|credit_priority:null|transaction_authorization_status:null", key.Value())
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
	require.Equal(t, "currency:USD|tax_code:null|tax_behavior:null|features:null|cost_basis:0.7|credit_priority:null|transaction_authorization_status:null", key1.Value())
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
	require.Equal(t, "currency:USD|tax_code:null|tax_behavior:null|features:null|cost_basis:null|credit_priority:null|transaction_authorization_status:authorized", key2.Value())
}

func TestBuildRoutingKeyV1_DifferentTaxBehavior_DifferentKey(t *testing.T) {
	base := Route{Currency: currencyx.Code("USD"), TaxCode: lo.ToPtr("GST10")}

	keyNil, err := BuildRoutingKeyV1(base)
	require.NoError(t, err)

	keyInclusive := base
	keyInclusive.TaxBehavior = lo.ToPtr(TaxBehaviorInclusive)
	k1, err := BuildRoutingKeyV1(keyInclusive)
	require.NoError(t, err)

	keyExclusive := base
	keyExclusive.TaxBehavior = lo.ToPtr(TaxBehaviorExclusive)
	k2, err := BuildRoutingKeyV1(keyExclusive)
	require.NoError(t, err)

	require.NotEqual(t, keyNil.Value(), k1.Value())
	require.NotEqual(t, keyNil.Value(), k2.Value())
	require.NotEqual(t, k1.Value(), k2.Value())
}

func TestTaxBehaviorValidate(t *testing.T) {
	require.NoError(t, TaxBehaviorInclusive.Validate())
	require.NoError(t, TaxBehaviorExclusive.Validate())
	require.Error(t, TaxBehavior("bogus").Validate())
	require.Error(t, TaxBehavior("").Validate())
}

func TestRouteValidate_InvalidTaxBehavior(t *testing.T) {
	r := Route{
		Currency:    currencyx.Code("USD"),
		TaxBehavior: lo.ToPtr(TaxBehavior("bogus")),
	}
	require.Error(t, r.Validate())
}

func TestRouteFilter_NormalizePreservesTaxCode(t *testing.T) {
	tc := "VAT20"
	f := RouteFilter{
		Currency: currencyx.Code("USD"),
		TaxCode:  mo.Some[*string](&tc),
	}
	norm, err := f.Normalize()
	require.NoError(t, err)
	require.True(t, norm.TaxCode.IsPresent())
	got, _ := norm.TaxCode.Get()
	require.Equal(t, &tc, got)
}

func TestRouteFilter_NormalizePreservesTaxBehavior(t *testing.T) {
	b := TaxBehaviorExclusive
	f := RouteFilter{
		Currency:    currencyx.Code("USD"),
		TaxBehavior: mo.Some[*TaxBehavior](&b),
	}
	norm, err := f.Normalize()
	require.NoError(t, err)
	require.True(t, norm.TaxBehavior.IsPresent())
	got, _ := norm.TaxBehavior.Get()
	require.Equal(t, &b, got)
}

func TestRouteFilter_NormalizeAbsentTaxFieldsStayAbsent(t *testing.T) {
	f := RouteFilter{Currency: currencyx.Code("USD")}
	norm, err := f.Normalize()
	require.NoError(t, err)
	require.True(t, norm.TaxCode.IsAbsent())
	require.True(t, norm.TaxBehavior.IsAbsent())
}

func TestRouteFilter_NormalizeSomeNilTaxCodePreserved(t *testing.T) {
	f := RouteFilter{
		Currency: currencyx.Code("USD"),
		TaxCode:  mo.Some[*string](nil),
	}
	norm, err := f.Normalize()
	require.NoError(t, err)
	require.True(t, norm.TaxCode.IsPresent())
	got, _ := norm.TaxCode.Get()
	require.Nil(t, got)
}

func TestRouteToFilter_TaxFieldsPinned(t *testing.T) {
	tc := "GST10"
	b := TaxBehaviorInclusive
	r := Route{
		Currency:    currencyx.Code("USD"),
		TaxCode:     &tc,
		TaxBehavior: &b,
	}
	f := r.Filter()

	require.True(t, f.TaxCode.IsPresent())
	gotCode, _ := f.TaxCode.Get()
	require.Equal(t, &tc, gotCode)

	require.True(t, f.TaxBehavior.IsPresent())
	gotBehavior, _ := f.TaxBehavior.Get()
	require.Equal(t, &b, gotBehavior)
}

func TestRouteToFilter_NilTaxFieldsPinnedAsPresent(t *testing.T) {
	r := Route{Currency: currencyx.Code("USD")}
	f := r.Filter()

	// nil Route fields become Some(nil) in filter — "filter for null", not "don't care"
	require.True(t, f.TaxCode.IsPresent())
	tc, _ := f.TaxCode.Get()
	require.Nil(t, tc)

	require.True(t, f.TaxBehavior.IsPresent())
	tb, _ := f.TaxBehavior.Get()
	require.Nil(t, tb)
}

func mustDecimal(t *testing.T, raw string) alpacadecimal.Decimal {
	t.Helper()

	value, err := alpacadecimal.NewFromString(raw)
	require.NoError(t, err)

	return value
}

func TestBuildRoutingKeyV1_WithTaxBehaviorAndTaxCode(t *testing.T) {
	key, err := BuildRoutingKeyV1(Route{
		Currency:    currencyx.Code("USD"),
		TaxCode:     lo.ToPtr("GST10"),
		TaxBehavior: lo.ToPtr(TaxBehaviorExclusive),
	})
	require.NoError(t, err)
	require.Equal(t, "currency:USD|tax_code:GST10|tax_behavior:exclusive|features:null|cost_basis:null|credit_priority:null|transaction_authorization_status:null", key.Value())
}
