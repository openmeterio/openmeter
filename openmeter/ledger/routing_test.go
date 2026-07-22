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
		Currency:       currencyx.Code("USD"),
		CostBasis:      &costBasis,
		CreditPriority: &priority,
	})
	require.NoError(t, err)
	require.Equal(t, RoutingKeyVersionV1, key.Version())
	require.Equal(t, "currency:USD|tax_code:GST10|features:null|cost_basis:0.7|credit_priority:7|transaction_authorization_status:null", key.Value())
}

func TestBuildRoutingKeyV1_Nulls(t *testing.T) {
	key, err := BuildRoutingKeyV1(Route{
		Currency: currencyx.Code("USD"),
	})
	require.NoError(t, err)
	require.Equal(t, "currency:USD|tax_code:null|features:null|cost_basis:null|credit_priority:null|transaction_authorization_status:null", key.Value())
}

func TestBuildRoutingKeyExchangeSourceCurrency(t *testing.T) {
	key, err := BuildRoutingKey(Route{
		Currency:               currencyx.Code("ACME"),
		ExchangeSourceCurrency: lo.ToPtr(currencyx.Code("USD")),
	})
	require.NoError(t, err)
	require.Equal(t, RoutingKeyVersionV3, key.Version())
	require.Contains(t, key.Value(), "currency:ACME|exchange_source_currency:USD|")
}

func TestBuildRoutingKeyEmptyExchangeSourceCurrency(t *testing.T) {
	key, err := BuildRoutingKey(Route{
		Currency:               currencyx.Code("USD"),
		ExchangeSourceCurrency: lo.ToPtr(currencyx.Code("")),
	})
	require.NoError(t, err)
	require.Equal(t, RoutingKeyVersionV1, key.Version())
	require.NotContains(t, key.Value(), "exchange_source_currency:")
}

func TestRouteValidateExchangeSourceCurrency(t *testing.T) {
	tests := []struct {
		name                   string
		currency               currencyx.Code
		exchangeSourceCurrency *currencyx.Code
		wantErr                bool
	}{
		{name: "fiat without source", currency: currencyx.Code("USD")},
		{name: "fiat with empty source", currency: currencyx.Code("USD"), exchangeSourceCurrency: lo.ToPtr(currencyx.Code(""))},
		{name: "custom without source", currency: currencyx.Code("ACME")},
		{name: "custom with fiat source", currency: currencyx.Code("ACME"), exchangeSourceCurrency: lo.ToPtr(currencyx.Code("USD"))},
		{name: "fiat with source", currency: currencyx.Code("USD"), exchangeSourceCurrency: lo.ToPtr(currencyx.Code("EUR")), wantErr: true},
		{name: "custom with custom source", currency: currencyx.Code("ACME"), exchangeSourceCurrency: lo.ToPtr(currencyx.Code("POINTS")), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := (Route{
				Currency:               tt.currency,
				ExchangeSourceCurrency: tt.exchangeSourceCurrency,
			}).Validate()
			if tt.wantErr {
				require.ErrorIs(t, err, ErrCurrencyInvalid)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestRouteFilterExchangeSourceCurrency(t *testing.T) {
	exchangeSourceCurrency := lo.ToPtr(currencyx.Code("USD"))
	route := Route{
		Currency:               currencyx.Code("ACME"),
		ExchangeSourceCurrency: exchangeSourceCurrency,
	}

	require.True(t, route.Matches(RouteFilter{ExchangeSourceCurrency: mo.Some(exchangeSourceCurrency)}))
	require.False(t, route.Matches(RouteFilter{ExchangeSourceCurrency: mo.Some(lo.ToPtr(currencyx.Code("EUR")))}))
	require.False(t, route.Matches(RouteFilter{ExchangeSourceCurrency: mo.Some[*currencyx.Code](nil)}))
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

func TestRouteValidateRejectsInvalidFeatures(t *testing.T) {
	require.Error(t, Route{
		Currency: currencyx.Code("USD"),
		Features: []string{""},
	}.Validate())

	require.Error(t, Route{
		Currency: currencyx.Code("USD"),
		Features: []string{"api-calls", "api-calls"},
	}.Validate())
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

func TestBuildRoutingKeyV2_DifferentTaxBehavior_DifferentKey(t *testing.T) {
	base := Route{Currency: currencyx.Code("USD"), TaxCode: lo.ToPtr("GST10"), TaxBehavior: lo.ToPtr(TaxBehaviorInclusive)}

	k1, err := BuildRoutingKeyV2(base)
	require.NoError(t, err)
	require.Equal(t, RoutingKeyVersionV2, k1.Version())

	exclusive := base
	exclusive.TaxBehavior = lo.ToPtr(TaxBehaviorExclusive)
	k2, err := BuildRoutingKeyV2(exclusive)
	require.NoError(t, err)

	require.NotEqual(t, k1.Value(), k2.Value())
}

func TestTaxBehaviorValidate(t *testing.T) {
	require.NoError(t, TaxBehaviorInclusive.Validate())
	require.NoError(t, TaxBehaviorExclusive.Validate())
	require.Error(t, TaxBehavior("bogus").Validate())
	require.Error(t, TaxBehavior("").Validate())
}

func TestValidateCurrency(t *testing.T) {
	testCases := []struct {
		name    string
		code    currencyx.Code
		wantErr bool
	}{
		{
			name: "fiat currency",
			code: "USD",
		},
		{
			name: "custom currency",
			code: "CREDITS",
		},
		{
			name:    "invalid currency",
			code:    "INVALID|CURRENCY",
			wantErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateCurrency(testCase.code)
			if testCase.wantErr {
				require.ErrorIs(t, err, ErrCurrencyInvalid)

				return
			}

			require.NoError(t, err)
		})
	}
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

	require.True(t, f.Features.IsPresent())
	features, _ := f.Features.Get()
	require.Empty(t, features)
}

func TestRouteMatches(t *testing.T) {
	costBasis := mustDecimal(t, "0.7")
	otherCostBasis := mustDecimal(t, "0.8")
	priority := 3
	otherPriority := 4
	taxCode := "tax-standard"
	otherTaxCode := "tax-reduced"
	taxBehavior := TaxBehaviorInclusive
	otherTaxBehavior := TaxBehaviorExclusive
	authStatus := TransactionAuthorizationStatusOpen
	otherAuthStatus := TransactionAuthorizationStatusAuthorized

	route := Route{
		Currency:                       currencyx.Code("USD"),
		TaxCode:                        &taxCode,
		TaxBehavior:                    &taxBehavior,
		Features:                       []string{"storage", "api-calls"},
		CostBasis:                      &costBasis,
		CreditPriority:                 &priority,
		TransactionAuthorizationStatus: &authStatus,
	}
	unrestrictedRoute := Route{
		Currency:       currencyx.Code("USD"),
		CreditPriority: &priority,
	}

	tests := []struct {
		name   string
		route  Route
		filter RouteFilter
		want   bool
	}{
		{
			name:  "empty filter matches populated route",
			route: route,
			want:  true,
		},
		{
			name:   "currency match",
			route:  route,
			filter: RouteFilter{Currency: currencyx.Code("USD")},
			want:   true,
		},
		{
			name:   "currency mismatch",
			route:  route,
			filter: RouteFilter{Currency: currencyx.Code("EUR")},
			want:   false,
		},
		{
			name:   "tax code absent ignores populated route tax code",
			route:  route,
			filter: RouteFilter{},
			want:   true,
		},
		{
			name:   "tax code match",
			route:  route,
			filter: RouteFilter{TaxCode: mo.Some(&taxCode)},
			want:   true,
		},
		{
			name:   "tax code mismatch",
			route:  route,
			filter: RouteFilter{TaxCode: mo.Some(&otherTaxCode)},
			want:   false,
		},
		{
			name:   "nil tax code filter rejects populated route tax code",
			route:  route,
			filter: RouteFilter{TaxCode: mo.Some[*string](nil)},
			want:   false,
		},
		{
			name:   "nil tax code filter matches nil route tax code",
			route:  unrestrictedRoute,
			filter: RouteFilter{TaxCode: mo.Some[*string](nil)},
			want:   true,
		},
		{
			name:   "populated tax code filter rejects nil route tax code",
			route:  unrestrictedRoute,
			filter: RouteFilter{TaxCode: mo.Some(&taxCode)},
			want:   false,
		},
		{
			name:   "tax behavior absent ignores populated route tax behavior",
			route:  route,
			filter: RouteFilter{},
			want:   true,
		},
		{
			name:   "tax behavior match",
			route:  route,
			filter: RouteFilter{TaxBehavior: mo.Some(&taxBehavior)},
			want:   true,
		},
		{
			name:   "tax behavior mismatch",
			route:  route,
			filter: RouteFilter{TaxBehavior: mo.Some(&otherTaxBehavior)},
			want:   false,
		},
		{
			name:   "nil tax behavior filter rejects populated route tax behavior",
			route:  route,
			filter: RouteFilter{TaxBehavior: mo.Some[*TaxBehavior](nil)},
			want:   false,
		},
		{
			name:   "nil tax behavior filter matches nil route tax behavior",
			route:  unrestrictedRoute,
			filter: RouteFilter{TaxBehavior: mo.Some[*TaxBehavior](nil)},
			want:   true,
		},
		{
			name:   "populated tax behavior filter rejects nil route tax behavior",
			route:  unrestrictedRoute,
			filter: RouteFilter{TaxBehavior: mo.Some(&taxBehavior)},
			want:   false,
		},
		{
			name:   "features absent ignores populated route features",
			route:  route,
			filter: RouteFilter{},
			want:   true,
		},
		{
			name:   "features match regardless of order",
			route:  route,
			filter: RouteFilter{Features: mo.Some([]string{"api-calls", "storage"})},
			want:   true,
		},
		{
			name:   "partial features mismatch",
			route:  route,
			filter: RouteFilter{Features: mo.Some([]string{"api-calls"})},
			want:   false,
		},
		{
			name:   "extra features mismatch",
			route:  route,
			filter: RouteFilter{Features: mo.Some([]string{"api-calls", "storage", "compute"})},
			want:   false,
		},
		{
			name:   "nil features filter rejects populated route features",
			route:  route,
			filter: RouteFilter{Features: mo.Some[[]string](nil)},
			want:   false,
		},
		{
			name:   "nil features filter matches empty route features",
			route:  unrestrictedRoute,
			filter: RouteFilter{Features: mo.Some[[]string](nil)},
			want:   true,
		},
		{
			name:   "populated features filter rejects empty route features",
			route:  unrestrictedRoute,
			filter: RouteFilter{Features: mo.Some([]string{"api-calls"})},
			want:   false,
		},
		{
			name:   "match feature matches populated route containing feature",
			route:  route,
			filter: RouteFilter{MatchFeature: "api-calls"},
			want:   true,
		},
		{
			name:   "match feature matches unrestricted route",
			route:  unrestrictedRoute,
			filter: RouteFilter{MatchFeature: "api-calls"},
			want:   true,
		},
		{
			name:   "match feature rejects populated route without feature",
			route:  route,
			filter: RouteFilter{MatchFeature: "compute"},
			want:   false,
		},
		{
			name:   "cost basis absent ignores populated route cost basis",
			route:  route,
			filter: RouteFilter{},
			want:   true,
		},
		{
			name:   "cost basis match",
			route:  route,
			filter: RouteFilter{CostBasis: mo.Some(&costBasis)},
			want:   true,
		},
		{
			name:   "cost basis mismatch",
			route:  route,
			filter: RouteFilter{CostBasis: mo.Some(&otherCostBasis)},
			want:   false,
		},
		{
			name:   "nil cost basis filter rejects populated route cost basis",
			route:  route,
			filter: RouteFilter{CostBasis: mo.Some[*alpacadecimal.Decimal](nil)},
			want:   false,
		},
		{
			name:   "nil cost basis filter matches nil route cost basis",
			route:  unrestrictedRoute,
			filter: RouteFilter{CostBasis: mo.Some[*alpacadecimal.Decimal](nil)},
			want:   true,
		},
		{
			name:   "populated cost basis filter rejects nil route cost basis",
			route:  unrestrictedRoute,
			filter: RouteFilter{CostBasis: mo.Some(&costBasis)},
			want:   false,
		},
		{
			name:   "credit priority absent ignores populated route credit priority",
			route:  route,
			filter: RouteFilter{},
			want:   true,
		},
		{
			name:   "credit priority match",
			route:  route,
			filter: RouteFilter{CreditPriority: &priority},
			want:   true,
		},
		{
			name:   "credit priority mismatch",
			route:  route,
			filter: RouteFilter{CreditPriority: &otherPriority},
			want:   false,
		},
		{
			name:   "credit priority filter rejects nil route credit priority",
			route:  Route{Currency: currencyx.Code("USD")},
			filter: RouteFilter{CreditPriority: &priority},
			want:   false,
		},
		{
			name:   "authorization status absent ignores populated route authorization status",
			route:  route,
			filter: RouteFilter{},
			want:   true,
		},
		{
			name:   "authorization status match",
			route:  route,
			filter: RouteFilter{TransactionAuthorizationStatus: &authStatus},
			want:   true,
		},
		{
			name:   "authorization status mismatch",
			route:  route,
			filter: RouteFilter{TransactionAuthorizationStatus: &otherAuthStatus},
			want:   false,
		},
		{
			name:   "authorization status filter rejects nil route authorization status",
			route:  unrestrictedRoute,
			filter: RouteFilter{TransactionAuthorizationStatus: &authStatus},
			want:   false,
		},
		{
			name:   "full route filter matches same route",
			route:  route,
			filter: route.Filter(),
			want:   true,
		},
		{
			name:   "full unrestricted route filter matches same unrestricted route",
			route:  unrestrictedRoute,
			filter: unrestrictedRoute.Filter(),
			want:   true,
		},
		{
			name:  "multiple fields match together",
			route: route,
			filter: RouteFilter{
				Currency:                       currencyx.Code("USD"),
				TaxCode:                        mo.Some(&taxCode),
				TaxBehavior:                    mo.Some(&taxBehavior),
				Features:                       mo.Some([]string{"storage", "api-calls"}),
				CostBasis:                      mo.Some(&costBasis),
				CreditPriority:                 &priority,
				TransactionAuthorizationStatus: &authStatus,
			},
			want: true,
		},
		{
			name:  "one mismatch makes multi-field filter fail",
			route: route,
			filter: RouteFilter{
				Currency:                       currencyx.Code("USD"),
				TaxCode:                        mo.Some(&taxCode),
				TaxBehavior:                    mo.Some(&taxBehavior),
				Features:                       mo.Some([]string{"storage", "api-calls"}),
				CostBasis:                      mo.Some(&costBasis),
				CreditPriority:                 &priority,
				TransactionAuthorizationStatus: &otherAuthStatus,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.route.Matches(tt.filter))
		})
	}
}

func TestRouteFilter_NormalizeRejectsExactAndMatchFeatures(t *testing.T) {
	_, err := RouteFilter{
		Features:     mo.Some([]string{"api-calls"}),
		MatchFeature: "api-calls",
	}.Normalize()
	require.Error(t, err)
}

func mustDecimal(t *testing.T, raw string) alpacadecimal.Decimal {
	t.Helper()

	value, err := alpacadecimal.NewFromString(raw)
	require.NoError(t, err)

	return value
}

func TestBuildRoutingKeyV2_WithTaxBehaviorAndTaxCode(t *testing.T) {
	key, err := BuildRoutingKeyV2(Route{
		Currency:    currencyx.Code("USD"),
		TaxCode:     lo.ToPtr("GST10"),
		TaxBehavior: lo.ToPtr(TaxBehaviorExclusive),
	})
	require.NoError(t, err)
	require.Equal(t, RoutingKeyVersionV2, key.Version())
	require.Equal(t, "currency:USD|tax_code:GST10|tax_behavior:exclusive|features:null|cost_basis:null|credit_priority:null|transaction_authorization_status:null", key.Value())
}
