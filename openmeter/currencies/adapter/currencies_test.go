package adapter_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func TestListCustomCurrenciesFiltersCurrencyType(t *testing.T) {
	env := currenciestestutils.NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	namespace := currenciestestutils.NewTestNamespace(t)
	created, err := env.Service.CreateCurrency(t.Context(), currencies.CreateCurrencyInput{
		Namespace: namespace,
		CurrencyDetails: currencyx.CurrencyDetails{
			Code:               "TOKENS",
			Name:               "Tokens",
			Symbol:             "T",
			Precision:          2,
			DecimalMark:        ".",
			ThousandsSeparator: ",",
		},
	})
	require.NoError(t, err)

	testCases := []struct {
		name          string
		currencyType  currencies.CurrencyType
		expectedCodes []currencyx.Code
	}{
		{
			name:          "custom",
			currencyType:  currencies.CurrencyTypeCustom,
			expectedCodes: []currencyx.Code{created.Details().Code},
		},
		{
			name:          "fiat",
			currencyType:  currencies.CurrencyTypeFiat,
			expectedCodes: nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// given:
			// - a persisted custom currency and a requested currency type
			// when:
			// - custom currencies are listed directly through the repository
			result, err := env.Repository.ListCustomCurrencies(t.Context(), currencies.ListCurrenciesInput{
				Page:         pagination.NewPage(1, 10),
				Namespace:    namespace,
				CurrencyType: &testCase.currencyType,
			})

			// then:
			// - the adapter returns custom records only for the custom type
			require.NoError(t, err)
			actualCodes := lo.Map(result.Items, func(item currencies.Currency, _ int) currencyx.Code {
				return item.Details().Code
			})
			assert.ElementsMatch(t, testCase.expectedCodes, actualCodes)
			assert.Equal(t, len(testCase.expectedCodes), result.TotalCount)
		})
	}
}

func TestGetCostBasisAt(t *testing.T) {
	env := currenciestestutils.NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	namespace := currenciestestutils.NewTestNamespace(t)
	currency, err := env.Service.CreateCurrency(t.Context(), currencies.CreateCurrencyInput{
		Namespace: namespace,
		CurrencyDetails: currencyx.CurrencyDetails{
			Code:               "TOKENS",
			Name:               "Tokens",
			Symbol:             "T",
			Precision:          2,
			DecimalMark:        ".",
			ThousandsSeparator: ",",
		},
	})
	require.NoError(t, err)

	firstEffectiveFrom := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	secondEffectiveFrom := firstEffectiveFrom.Add(24 * time.Hour)

	// given:
	// - consecutive USD cost bases and a EUR cost basis ending at the same boundary
	firstUSD, err := env.Client.CurrencyCostBasis.Create().
		SetNamespace(namespace).
		SetCurrencyID(currency.ID).
		SetFiatCode("USD").
		SetRate(alpacadecimal.RequireFromString("0.01")).
		SetEffectiveFrom(firstEffectiveFrom).
		SetEffectiveTo(secondEffectiveFrom).
		Save(t.Context())
	require.NoError(t, err)

	secondUSD, err := env.Client.CurrencyCostBasis.Create().
		SetNamespace(namespace).
		SetCurrencyID(currency.ID).
		SetFiatCode("USD").
		SetRate(alpacadecimal.RequireFromString("0.02")).
		SetEffectiveFrom(secondEffectiveFrom).
		Save(t.Context())
	require.NoError(t, err)

	_, err = env.Client.CurrencyCostBasis.Create().
		SetNamespace(namespace).
		SetCurrencyID(currency.ID).
		SetFiatCode("EUR").
		SetRate(alpacadecimal.RequireFromString("0.009")).
		SetEffectiveFrom(firstEffectiveFrom).
		SetEffectiveTo(secondEffectiveFrom).
		Save(t.Context())
	require.NoError(t, err)

	testCases := []struct {
		name       string
		fiatCode   currencyx.Code
		at         time.Time
		expectedID string
		notFound   bool
	}{
		{
			name:       "effective from is inclusive",
			fiatCode:   "USD",
			at:         firstEffectiveFrom,
			expectedID: firstUSD.ID,
		},
		{
			name:       "newer cost basis wins at its effective start",
			fiatCode:   "USD",
			at:         secondEffectiveFrom,
			expectedID: secondUSD.ID,
		},
		{
			name:       "open interval remains effective",
			fiatCode:   "USD",
			at:         secondEffectiveFrom.Add(30 * 24 * time.Hour),
			expectedID: secondUSD.ID,
		},
		{
			name:     "before first interval",
			fiatCode: "USD",
			at:       firstEffectiveFrom.Add(-time.Nanosecond),
			notFound: true,
		},
		{
			name:     "effective to is exclusive",
			fiatCode: "EUR",
			at:       secondEffectiveFrom,
			notFound: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// when:
			// - the cost basis effective at the requested instant is queried
			result, err := env.Repository.GetCostBasisAt(t.Context(), currencies.GetCostBasisAtInput{
				Namespace:  namespace,
				CurrencyID: currency.ID,
				FiatCode:   testCase.fiatCode,
				At:         testCase.at,
			})

			// then:
			// - interval boundaries select the matching row or return a typed not-found error
			if testCase.notFound {
				require.Error(t, err)
				assert.True(t, models.IsGenericNotFoundError(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.expectedID, result.ID)
		})
	}
}
