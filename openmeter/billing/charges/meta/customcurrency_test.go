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
	"github.com/openmeterio/openmeter/pkg/models"
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
			NamespacedID: models.NamespacedID{
				Namespace: "namespace",
				ID:        "custom-currency-id",
			},
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

func TestCalculateFiatAmount(t *testing.T) {
	fiatCurrency, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)

	t.Run("converts and rounds to fiat precision", func(t *testing.T) {
		amount, err := CalculateFiatAmount(
			alpacadecimal.NewFromFloat(1.23),
			alpacadecimal.NewFromFloat(1.5),
			fiatCurrency,
		)
		require.NoError(t, err)
		require.Equal(t, float64(1.85), amount.InexactFloat64())
	})

	t.Run("rejects negative amount", func(t *testing.T) {
		_, err := CalculateFiatAmount(
			alpacadecimal.NewFromInt(-1),
			alpacadecimal.NewFromInt(1),
			fiatCurrency,
		)
		require.ErrorContains(t, err, "amount cannot be negative")
	})

	t.Run("rejects non-positive cost basis", func(t *testing.T) {
		_, err := CalculateFiatAmount(
			alpacadecimal.NewFromInt(1),
			alpacadecimal.Zero,
			fiatCurrency,
		)
		require.ErrorContains(t, err, "resolved cost basis must be positive")

		_, err = CalculateFiatAmount(
			alpacadecimal.NewFromInt(1),
			alpacadecimal.NewFromInt(-1),
			fiatCurrency,
		)
		require.ErrorContains(t, err, "resolved cost basis must be positive")
	})

	t.Run("rejects missing fiat currency", func(t *testing.T) {
		_, err := CalculateFiatAmount(
			alpacadecimal.NewFromInt(1),
			alpacadecimal.NewFromInt(1),
			nil,
		)
		require.ErrorContains(t, err, "fiat currency is required")
	})
}
