package meta

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestConvertCustomCurrencyOverageToFiat(t *testing.T) {
	customCurrency, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode("TOKENS").
		WithName("Tokens").
		WithPrecision(2).
		Build()
	require.NoError(t, err)

	fiatCurrency, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)

	costBasisIntent := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: fiatCurrency,
		Rate:         alpacadecimal.NewFromFloat(1.5),
	})
	resolvedCostBasis := costbasis.State{
		CostBasis:  alpacadecimal.NewFromFloat(1.5),
		ResolvedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	validInput := ConvertCustomCurrencyOverageToFiatInput{
		Currency: currencies.Currency{
			Currency: customCurrency,
		},
		CostBasisIntent:   &costBasisIntent,
		ResolvedCostBasis: &resolvedCostBasis,
		Totals: totals.Totals{
			Total: alpacadecimal.NewFromFloat(1.23),
		},
	}

	t.Run("converts the post-allocation total and rounds to fiat precision", func(t *testing.T) {
		result, err := ConvertCustomCurrencyOverageToFiat(validInput)
		require.NoError(t, err)
		require.Equal(t, currencyx.Code("USD"), result.Currency.Details().Code)
		require.Equal(t, float64(1.85), result.Amount.InexactFloat64())
	})

	t.Run("rejects a non-custom source currency", func(t *testing.T) {
		input := validInput
		input.Currency = currencies.Currency{
			Currency: fiatCurrency,
		}

		_, err := ConvertCustomCurrencyOverageToFiat(input)
		require.ErrorContains(t, err, "currency must be custom")
	})

	t.Run("rejects a negative overage", func(t *testing.T) {
		input := validInput
		input.Totals.Total = alpacadecimal.NewFromInt(-1)

		_, err := ConvertCustomCurrencyOverageToFiat(input)
		require.ErrorContains(t, err, "total is negative")
	})

	t.Run("rejects an overage not rounded to source precision", func(t *testing.T) {
		input := validInput
		input.Totals.Total = alpacadecimal.NewFromFloat(1.234)

		_, err := ConvertCustomCurrencyOverageToFiat(input)
		require.ErrorContains(t, err, "must be rounded to custom currency precision")
	})

	t.Run("requires the cost basis intent", func(t *testing.T) {
		input := validInput
		input.CostBasisIntent = nil

		_, err := ConvertCustomCurrencyOverageToFiat(input)
		require.ErrorContains(t, err, "cost basis intent is required")
	})

	t.Run("requires the resolved cost basis", func(t *testing.T) {
		input := validInput
		input.ResolvedCostBasis = nil

		_, err := ConvertCustomCurrencyOverageToFiat(input)
		require.ErrorContains(t, err, "resolved cost basis is required")
	})
}
