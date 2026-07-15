package productcatalog

import (
	"context"
	"fmt"
	"testing"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestValidatePlanRateCardCurrencies(t *testing.T) {
	custom := currency.Code("CREDITS")

	tests := []struct {
		name         string
		planCurrency currency.Code
		override     *currency.Code
		expected     error
	}{
		{
			name:         "fiat plan with inherited currency",
			planCurrency: currency.USD,
		},
		{
			name:         "fiat plan with custom override",
			planCurrency: currency.USD,
			override:     &custom,
		},
		{
			name:         "fiat plan with redundant override",
			planCurrency: currency.USD,
			override:     currencyPtr(currency.USD),
			expected:     ErrRateCardCurrencyOverrideRedundant,
		},
		{
			name:         "fiat plan with second fiat",
			planCurrency: currency.USD,
			override:     currencyPtr(currency.EUR),
			expected:     ErrPlanMultipleFiatCurrencies,
		},
		{
			name:         "custom plan with inherited currency",
			planCurrency: custom,
		},
		{
			name:         "custom plan with override",
			planCurrency: custom,
			override:     currencyPtr("TOKENS"),
			expected:     ErrRateCardCurrencyOverrideNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := Plan{
				PlanMeta: PlanMeta{Currency: tt.planCurrency},
				Phases: []Phase{{
					PhaseMeta: PhaseMeta{Key: "default"},
					RateCards: RateCards{&FlatFeeRateCard{
						RateCardMeta: RateCardMeta{Key: "base", Currency: tt.override},
					}},
				}},
			}

			err := ValidatePlanRateCardCurrencies()(plan)
			if tt.expected == nil {
				require.NoError(t, err)
				return
			}

			require.ErrorIs(t, err, tt.expected)
		})
	}
}

func TestValidatePlanWithCurrencies(t *testing.T) {
	custom := currency.Code("CREDITS")
	customCurrency := ResolvedCurrency{
		ID:   "currency-id",
		Code: custom,
		Type: currencyx.CurrencyTypeCustom,
	}

	tests := []struct {
		name     string
		resolver *testCurrencyResolver
		expected error
	}{
		{
			name: "matching cost basis",
			resolver: &testCurrencyResolver{
				currencies: map[currency.Code]ResolvedCurrency{custom: customCurrency},
				costBases:  map[string]bool{"currency-id|USD": true},
			},
		},
		{
			name:     "unknown custom currency",
			resolver: &testCurrencyResolver{},
			expected: ErrCurrencyNotFound,
		},
		{
			name: "missing cost basis",
			resolver: &testCurrencyResolver{
				currencies: map[currency.Code]ResolvedCurrency{custom: customCurrency},
			},
			expected: ErrCurrencyCostBasisNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given:
			// - a fiat plan with one custom-currency rate card
			// when:
			// - managed currency and cost-basis references are validated
			// then:
			// - only a known custom currency with a USD cost-basis pair is valid
			plan := Plan{
				PlanMeta: PlanMeta{Currency: currency.USD},
				Phases: []Phase{{
					PhaseMeta: PhaseMeta{Key: "default"},
					RateCards: RateCards{&FlatFeeRateCard{
						RateCardMeta: RateCardMeta{Key: "base", Currency: &custom},
					}},
				}},
			}

			err := ValidatePlanWithCurrencies(t.Context(), "namespace", tt.resolver)(plan)
			if tt.expected == nil {
				require.NoError(t, err)
				return
			}

			require.ErrorIs(t, err, tt.expected)
		})
	}
}

func TestRateCardCurrencyRequiresPrice(t *testing.T) {
	custom := currency.Code("CREDITS")

	t.Run("currency without price is invalid", func(t *testing.T) {
		err := (RateCardMeta{Currency: &custom}).Validate()
		require.ErrorIs(t, err, ErrRateCardCurrencyRequiresPrice)
	})

	t.Run("currency with price is valid", func(t *testing.T) {
		err := (RateCardMeta{
			Currency: &custom,
			Price: NewPriceFrom(FlatPrice{
				Amount:      decimal.NewFromInt(1),
				PaymentTerm: InAdvancePaymentTerm,
			}),
		}).Validate()
		require.NoError(t, err)
	})
}

func currencyPtr(code currency.Code) *currency.Code {
	return &code
}

type testCurrencyResolver struct {
	currencies map[currency.Code]ResolvedCurrency
	costBases  map[string]bool
}

func (r *testCurrencyResolver) Resolve(_ context.Context, _ string, code currency.Code) (ResolvedCurrency, error) {
	if currencyx.Code(code).IsFiat() {
		return ResolvedCurrency{Code: code, Type: currencyx.CurrencyTypeFiat}, nil
	}

	resolved, ok := r.currencies[code]
	if !ok {
		return ResolvedCurrency{}, models.NewGenericNotFoundError(fmt.Errorf("currency %q", code))
	}

	return resolved, nil
}

func (r *testCurrencyResolver) HasCostBasis(_ context.Context, _ string, customCurrency ResolvedCurrency, fiatCurrency currency.Code) (bool, error) {
	return r.costBases[customCurrency.ID+"|"+fiatCurrency.String()], nil
}

var _ CurrencyResolver = (*testCurrencyResolver)(nil)
