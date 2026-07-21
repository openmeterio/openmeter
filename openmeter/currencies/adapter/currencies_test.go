package adapter_test

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
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
