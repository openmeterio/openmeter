package currencyx_test

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestRoundToPrecision(t *testing.T) {
	cases := []struct {
		def      string
		amount   float64
		expected float64
	}{
		// Subunits = 2, smallestDenomination = 1
		{"USD", 1.23456789, 1.23},
		{"USD", 1.23556789, 1.24},

		// Subunits = 0, smallestDenomination = 1
		{"JPY", 1.23456789, 1.0},
		{"JPY", 1.9556789, 2.0},
	}

	for _, c := range cases {
		calculator, err := currencyx.Code(c.def).Calculator()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		amount := alpacadecimal.NewFromFloat(c.amount)
		result := calculator.RoundToPrecision(amount).InexactFloat64()

		require.Equal(t, c.expected, result)
	}
}

func TestCurrencyInterface(t *testing.T) {
	var fiat currencyx.Currency = currencyx.Code("USD")
	require.Equal(t, currencyx.Code("USD"), fiat.CurrencyCode())
	require.Equal(t, currencyx.CurrencyTypeFiat, fiat.CurrencyType())
	require.Equal(t, int32(2), fiat.CurrencyPrecision())
	require.Equal(t, currencyx.RoundingModeHalfAwayFromZero, fiat.CurrencyRoundingMode())

	custom, err := currencyx.NewCustomCurrency(currencyx.Code("CREDITS"), 6)
	require.NoError(t, err)

	var currency currencyx.Currency = custom
	require.Equal(t, currencyx.Code("CREDITS"), currency.CurrencyCode())
	require.Equal(t, currencyx.CurrencyTypeCustom, currency.CurrencyType())
	require.Equal(t, int32(6), currency.CurrencyPrecision())
	require.Equal(t, currencyx.RoundingModeBankers, currency.CurrencyRoundingMode())
}

func TestNewCurrency(t *testing.T) {
	fiat, err := currencyx.NewCurrency(currencyx.Code("JPY"), currencyx.CurrencyTypeFiat, 0)
	require.NoError(t, err)
	require.Equal(t, currencyx.Code("JPY"), fiat.CurrencyCode())
	require.Equal(t, currencyx.CurrencyTypeFiat, fiat.CurrencyType())
	require.Equal(t, int32(0), fiat.CurrencyPrecision())
	require.Equal(t, currencyx.RoundingModeHalfAwayFromZero, fiat.CurrencyRoundingMode())

	custom, err := currencyx.NewCurrency(currencyx.Code("CREDITS"), currencyx.CurrencyTypeCustom, 4)
	require.NoError(t, err)
	require.Equal(t, currencyx.Code("CREDITS"), custom.CurrencyCode())
	require.Equal(t, currencyx.CurrencyTypeCustom, custom.CurrencyType())
	require.Equal(t, int32(4), custom.CurrencyPrecision())
	require.Equal(t, currencyx.RoundingModeBankers, custom.CurrencyRoundingMode())
}

func TestCurrencyTypeValidation(t *testing.T) {
	require.NoError(t, currencyx.CurrencyTypeFiat.Validate())
	require.NoError(t, currencyx.CurrencyTypeCustom.Validate())

	err := currencyx.CurrencyType("unknown").Validate()
	require.Error(t, err)
	require.True(t, models.IsGenericValidationError(err), "error must be a validation error")
	require.Contains(t, err.Error(), "invalid currency type: unknown")
}

func TestRoundingModeValidation(t *testing.T) {
	require.NoError(t, currencyx.RoundingModeHalfAwayFromZero.Validate())
	require.NoError(t, currencyx.RoundingModeBankers.Validate())

	err := currencyx.RoundingMode("unknown").Validate()
	require.Error(t, err)
	require.True(t, models.IsGenericValidationError(err), "error must be a validation error")
	require.Contains(t, err.Error(), "invalid rounding mode: unknown")
}

func TestNewFiatCurrencyRequiresISOFiatDefinition(t *testing.T) {
	fiat, err := currencyx.NewFiatCurrency(currencyx.Code("USD"))
	require.NoError(t, err)
	require.Equal(t, currencyx.Code("USD"), fiat)

	_, err = currencyx.NewFiatCurrency(currencyx.Code("BTC"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "fiat currency definition is required for BTC")
}

func TestCustomCurrencyValidation(t *testing.T) {
	_, err := currencyx.NewCustomCurrency(currencyx.Code("USD"), 2)
	require.Error(t, err)
	require.True(t, models.IsGenericValidationError(err), "error must be a validation error")
	require.Contains(t, err.Error(), "conflicts with fiat currency code")

	_, err = currencyx.NewCustomCurrency(currencyx.Code("CR"), 2)
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least")

	_, err = currencyx.NewCustomCurrency(currencyx.Code("CRE|DITS"), 2)
	require.Error(t, err)
	require.Contains(t, err.Error(), "route delimiter")

	_, err = currencyx.NewCustomCurrency(currencyx.Code("CREDITS"), currencyx.MaxPrecision+1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "precision")

	_, err = currencyx.NewCustomCurrencyWithRounding(currencyx.Code("CREDITS"), 2, currencyx.RoundingMode("unknown"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "rounding mode")
}

func TestCodeValidationBoundaries(t *testing.T) {
	require.NoError(t, currencyx.Code("CREDITS").ValidateFormat())
	require.Error(t, currencyx.Code("CREDITS").Validate())
	require.NoError(t, currencyx.Code("USD").Validate())
	require.True(t, currencyx.Code("USD").IsKnownFiat())
}

func TestCalculatorUsesCurrencyInterface(t *testing.T) {
	fiatCalculator, err := currencyx.NewCalculator(currencyx.Code("USD"))
	require.NoError(t, err)
	require.Equal(t, currencyx.CurrencyTypeFiat, fiatCalculator.CurrencyType())
	require.Equal(t, currencyx.Code("USD"), fiatCalculator.CurrencyCode())
	require.Equal(t, int32(2), fiatCalculator.CurrencyPrecision())
	require.Equal(t, currencyx.RoundingModeHalfAwayFromZero, fiatCalculator.RoundingMode())
	require.NotNil(t, fiatCalculator.Definition())
	require.Equal(t, "0.01", fiatCalculator.Unit().String())
	require.Equal(t, float64(1.24), fiatCalculator.RoundToPrecision(alpacadecimal.NewFromFloat(1.235)).InexactFloat64())

	custom, err := currencyx.NewCustomCurrency(currencyx.Code("CREDITS"), 6)
	require.NoError(t, err)

	customCalculator, err := currencyx.NewCalculator(custom)
	require.NoError(t, err)
	require.Equal(t, currencyx.CurrencyTypeCustom, customCalculator.CurrencyType())
	require.Equal(t, currencyx.Code("CREDITS"), customCalculator.CurrencyCode())
	require.Equal(t, int32(6), customCalculator.CurrencyPrecision())
	require.Equal(t, currencyx.RoundingModeBankers, customCalculator.RoundingMode())
	require.Nil(t, customCalculator.Definition())
	require.Equal(t, float64(1.234568), customCalculator.RoundToPrecision(alpacadecimal.NewFromFloat(1.2345678)).InexactFloat64())
}

func TestCalculatorUsesCurrencyTypeToSelectRounding(t *testing.T) {
	fiat, err := currencyx.NewCalculator(testCurrency{
		code:         currencyx.Code("USD"),
		currencyType: currencyx.CurrencyTypeFiat,
		precision:    12,
		roundingMode: currencyx.RoundingModeBankers,
	})
	require.NoError(t, err)
	require.Equal(t, currencyx.CurrencyTypeFiat, fiat.CurrencyType())
	require.Equal(t, int32(2), fiat.CurrencyPrecision())
	require.Equal(t, currencyx.RoundingModeHalfAwayFromZero, fiat.RoundingMode())
	require.Equal(t, "1.23", fiat.RoundToPrecision(alpacadecimal.RequireFromString("1.225")).String())

	custom, err := currencyx.NewCalculator(testCurrency{
		code:         currencyx.Code("CREDITS"),
		currencyType: currencyx.CurrencyTypeCustom,
		precision:    2,
		roundingMode: currencyx.RoundingModeBankers,
	})
	require.NoError(t, err)
	require.Equal(t, currencyx.CurrencyTypeCustom, custom.CurrencyType())
	require.Equal(t, int32(2), custom.CurrencyPrecision())
	require.Equal(t, currencyx.RoundingModeBankers, custom.RoundingMode())
	require.Equal(t, "1.22", custom.RoundToPrecision(alpacadecimal.RequireFromString("1.225")).String())
}

func TestCustomCurrencyRoundingModes(t *testing.T) {
	bankersCurrency, err := currencyx.NewCustomCurrency(currencyx.Code("CREDITS"), 2)
	require.NoError(t, err)

	bankersCalculator, err := currencyx.NewCalculator(bankersCurrency)
	require.NoError(t, err)
	require.Equal(t, currencyx.RoundingModeBankers, bankersCalculator.RoundingMode())
	require.Equal(t, "1.22", bankersCalculator.RoundToPrecision(alpacadecimal.RequireFromString("1.225")).String())
	require.Equal(t, "1.24", bankersCalculator.RoundToPrecision(alpacadecimal.RequireFromString("1.235")).String())

	halfAwayCurrency, err := currencyx.NewCustomCurrencyWithRounding(
		currencyx.Code("TOKENS"),
		2,
		currencyx.RoundingModeHalfAwayFromZero,
	)
	require.NoError(t, err)

	halfAwayCalculator, err := currencyx.NewCalculator(halfAwayCurrency)
	require.NoError(t, err)
	require.Equal(t, currencyx.RoundingModeHalfAwayFromZero, halfAwayCalculator.RoundingMode())
	require.Equal(t, "1.23", halfAwayCalculator.RoundToPrecision(alpacadecimal.RequireFromString("1.225")).String())
	require.Equal(t, "1.24", halfAwayCalculator.RoundToPrecision(alpacadecimal.RequireFromString("1.235")).String())
}

func TestCustomCurrencyBankersRoundingTiesToEven(t *testing.T) {
	cases := []struct {
		name      string
		precision int32
		amount    string
		expected  string
	}{
		{name: "down to even cent", precision: 2, amount: "1.225", expected: "1.22"},
		{name: "up to even cent", precision: 2, amount: "1.235", expected: "1.24"},
		{name: "negative down to even cent", precision: 2, amount: "-1.225", expected: "-1.22"},
		{name: "negative up to even cent", precision: 2, amount: "-1.235", expected: "-1.24"},
		{name: "zero precision down to even integer", precision: 0, amount: "2.5", expected: "2"},
		{name: "zero precision up to even integer", precision: 0, amount: "3.5", expected: "4"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			custom, err := currencyx.NewCustomCurrency(currencyx.Code("CREDITS"), tc.precision)
			require.NoError(t, err)

			calculator, err := currencyx.NewCalculator(custom)
			require.NoError(t, err)

			require.Equal(
				t,
				tc.expected,
				calculator.RoundToPrecision(alpacadecimal.RequireFromString(tc.amount)).String(),
			)
		})
	}
}

func TestCustomCurrencyDefaultRoundingModeIsBankers(t *testing.T) {
	calculator, err := currencyx.NewCalculator(currencyx.CustomCurrency{
		Code:      currencyx.Code("CREDITS"),
		Precision: 2,
	})
	require.NoError(t, err)

	require.Equal(t, currencyx.RoundingModeBankers, calculator.RoundingMode())
	require.Equal(t, "1.22", calculator.RoundToPrecision(alpacadecimal.RequireFromString("1.225")).String())
}

func TestIsRoundedToPrecisionUsesConfiguredRoundingMode(t *testing.T) {
	bankersCurrency, err := currencyx.NewCustomCurrency(currencyx.Code("CREDITS"), 2)
	require.NoError(t, err)

	bankersCalculator, err := currencyx.NewCalculator(bankersCurrency)
	require.NoError(t, err)
	require.True(t, bankersCalculator.IsRoundedToPrecision(alpacadecimal.RequireFromString("1.22")))
	require.True(t, bankersCalculator.IsRoundedToPrecision(alpacadecimal.RequireFromString("1.23")))
	require.False(t, bankersCalculator.IsRoundedToPrecision(alpacadecimal.RequireFromString("1.225")))

	halfAwayCurrency, err := currencyx.NewCustomCurrencyWithRounding(
		currencyx.Code("TOKENS"),
		2,
		currencyx.RoundingModeHalfAwayFromZero,
	)
	require.NoError(t, err)

	halfAwayCalculator, err := currencyx.NewCalculator(halfAwayCurrency)
	require.NoError(t, err)
	require.True(t, halfAwayCalculator.IsRoundedToPrecision(alpacadecimal.RequireFromString("1.23")))
	require.False(t, halfAwayCalculator.IsRoundedToPrecision(alpacadecimal.RequireFromString("1.225")))
}

type testCurrency struct {
	code         currencyx.Code
	currencyType currencyx.CurrencyType
	precision    int32
	roundingMode currencyx.RoundingMode
}

func (c testCurrency) CurrencyCode() currencyx.Code {
	return c.code
}

func (c testCurrency) CurrencyType() currencyx.CurrencyType {
	return c.currencyType
}

func (c testCurrency) CurrencyPrecision() int32 {
	return c.precision
}

func (c testCurrency) CurrencyRoundingMode() currencyx.RoundingMode {
	return c.roundingMode
}
