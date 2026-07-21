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
		name         string
		currencyType currencyx.CurrencyType
		code         string
		precision    uint32
		amount       float64
		expected     float64
	}{
		// Fiat: USD has subunits=2
		{
			name:         "USD round down",
			currencyType: currencyx.CurrencyTypeFiat,
			code:         "USD",
			precision:    0,
			amount:       1.23456789,
			expected:     1.23,
		},
		{
			name:         "USD round up",
			currencyType: currencyx.CurrencyTypeFiat,
			code:         "USD",
			precision:    0,
			amount:       1.23556789,
			expected:     1.24,
		},
		// Fiat: JPY has subunits=0
		{
			name:         "JPY round down",
			currencyType: currencyx.CurrencyTypeFiat,
			code:         "JPY",
			precision:    0,
			amount:       1.23456789,
			expected:     1.0,
		},
		{
			name:         "JPY round up",
			currencyType: currencyx.CurrencyTypeFiat,
			code:         "JPY",
			precision:    0,
			amount:       1.9556789,
			expected:     2.0,
		},
		// Custom: precision 4
		{
			name:         "custom precision 4",
			currencyType: currencyx.CurrencyTypeCustom,
			code:         "CREDITS",
			precision:    4,
			amount:       1.23456789,
			expected:     1.2346,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := currencyx.NewCurrencyBuilder(c.currencyType).
				WithCode(currencyx.Code(c.code)).
				WithName(c.code)

			if c.currencyType == currencyx.CurrencyTypeCustom {
				b = b.WithPrecision(c.precision)
			}

			currency, err := b.Build()
			require.NoError(t, err)

			amount := alpacadecimal.NewFromFloat(c.amount)
			result := currency.RoundToPrecision(amount).InexactFloat64()

			require.Equal(t, c.expected, result)
		})
	}
}

func TestCurrencyInterface(t *testing.T) {
	t.Run("fiat", func(t *testing.T) {
		fiat, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeFiat).
			WithCode(currencyx.Code("USD")).
			Build()
		require.NoError(t, err)

		require.Equal(t, currencyx.CurrencyTypeFiat, fiat.Type())
		require.Equal(t, currencyx.Code("USD"), fiat.Details().Code)
		require.Equal(t, uint32(2), fiat.Details().Precision)
		require.Equal(t, "United States Dollar", fiat.Details().Name)
		require.Equal(t, "$", fiat.Details().Symbol)

		_, err = fiat.AsFiat()
		require.NoError(t, err)

		_, err = fiat.AsCustom()
		require.Error(t, err)
	})

	t.Run("custom", func(t *testing.T) {
		custom, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
			WithCode(currencyx.Code("CREDITS")).
			WithName("Credits").
			WithPrecision(6).
			Build()
		require.NoError(t, err)

		require.Equal(t, currencyx.CurrencyTypeCustom, custom.Type())
		require.Equal(t, currencyx.Code("CREDITS"), custom.Details().Code)
		require.Equal(t, uint32(6), custom.Details().Precision)
		require.Equal(t, "Credits", custom.Details().Name)

		_, err = custom.AsCustom()
		require.NoError(t, err)

		_, err = custom.AsFiat()
		require.Error(t, err)
	})
}

func TestCurrencyTypeValidation(t *testing.T) {
	require.NoError(t, currencyx.CurrencyTypeFiat.Validate())
	require.NoError(t, currencyx.CurrencyTypeCustom.Validate())

	err := currencyx.CurrencyType("unknown").Validate()
	require.Error(t, err)
	require.True(t, models.IsGenericValidationError(err), "error must be a validation error")
	require.Contains(t, err.Error(), "invalid currency type: unknown")
}

func TestFiatCurrencyRequiresISODefinition(t *testing.T) {
	_, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeFiat).
		WithCode(currencyx.Code("USD")).
		Build()
	require.NoError(t, err)

	_, err = currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeFiat).
		WithCode(currencyx.Code("BTC")).
		Build()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid fiat currency code")
}

func TestCustomCurrencyValidation(t *testing.T) {
	// Conflicts with fiat currency code
	_, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode(currencyx.Code("USD")).
		WithName("Fake Dollar").
		WithPrecision(2).
		Build()
	require.Error(t, err)
	require.True(t, models.IsGenericValidationError(err), "error must be a validation error")
	require.Contains(t, err.Error(), "fiat currency")

	// Code too short
	_, err = currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode(currencyx.Code("CR")).
		WithName("Credits").
		WithPrecision(2).
		Build()
	require.Error(t, err)
	require.Contains(t, err.Error(), "between")

	// Code contains route delimiter
	_, err = currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode(currencyx.Code("CRE|DITS")).
		WithName("Credits").
		WithPrecision(2).
		Build()
	require.Error(t, err)
	require.Contains(t, err.Error(), "route delimiter")

	// Precision exceeds max
	_, err = currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode(currencyx.Code("CREDITS")).
		WithName("Credits").
		WithPrecision(currencyx.CustomCurrencyMaxPrecision + 1).
		Build()
	require.Error(t, err)
	require.Contains(t, err.Error(), "precision")

	// Missing name
	_, err = currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode(currencyx.Code("CREDITS")).
		WithPrecision(2).
		Build()
	require.Error(t, err)
	require.Contains(t, err.Error(), "name is required")
}

func TestBuilderInvalidCurrencyType(t *testing.T) {
	_, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyType("unknown")).
		WithCode(currencyx.Code("XYZ")).
		Build()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid currency type")
}

func TestIsRoundedToPrecision(t *testing.T) {
	fiat, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeFiat).
		WithCode(currencyx.Code("USD")).
		Build()
	require.NoError(t, err)

	require.True(t, fiat.IsRoundedToPrecision(alpacadecimal.RequireFromString("1.22")))
	require.True(t, fiat.IsRoundedToPrecision(alpacadecimal.RequireFromString("1.23")))
	require.False(t, fiat.IsRoundedToPrecision(alpacadecimal.RequireFromString("1.225")))

	custom, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode(currencyx.Code("TOKENS")).
		WithName("Tokens").
		WithPrecision(2).
		Build()
	require.NoError(t, err)

	require.True(t, custom.IsRoundedToPrecision(alpacadecimal.RequireFromString("1.23")))
	require.False(t, custom.IsRoundedToPrecision(alpacadecimal.RequireFromString("1.225")))
}

func TestFiatCurrencyPrecisionFromDefinition(t *testing.T) {
	// USD has 2 subunits
	usd, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeFiat).
		WithCode(currencyx.Code("USD")).
		Build()
	require.NoError(t, err)
	require.Equal(t, uint32(2), usd.Details().Precision)

	// JPY has 0 subunits
	jpy, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeFiat).
		WithCode(currencyx.Code("JPY")).
		Build()
	require.NoError(t, err)
	require.Equal(t, uint32(0), jpy.Details().Precision)
}

func TestFormatAmount(t *testing.T) {
	cases := []struct {
		name         string
		currencyType currencyx.CurrencyType
		code         string
		precision    uint32
		symbol       string
		amount       string
		expected     string
	}{
		// Fiat USD: symbol "$", decimal mark ".", thousands separator ","
		{
			name:         "USD integer",
			currencyType: currencyx.CurrencyTypeFiat,
			code:         "USD",
			precision:    0,
			symbol:       "",
			amount:       "1",
			expected:     "$1",
		},
		{
			name:         "USD thousands separator",
			currencyType: currencyx.CurrencyTypeFiat,
			code:         "USD",
			precision:    0,
			symbol:       "",
			amount:       "1234567",
			expected:     "$1,234,567",
		},
		{
			name:         "USD two decimals",
			currencyType: currencyx.CurrencyTypeFiat,
			code:         "USD",
			precision:    0,
			symbol:       "",
			amount:       "1.50",
			expected:     "$1.50",
		},
		{
			name:         "USD negative",
			currencyType: currencyx.CurrencyTypeFiat,
			code:         "USD",
			precision:    0,
			symbol:       "",
			amount:       "-1.50",
			expected:     "-$1.50",
		},
		// Fiat JPY: symbol "¥", subunits=0
		{
			name:         "JPY integer",
			currencyType: currencyx.CurrencyTypeFiat,
			code:         "JPY",
			precision:    0,
			symbol:       "",
			amount:       "1234",
			expected:     "¥1,234",
		},
		// Fiat EUR: decimal mark ",", thousands separator "."
		{
			name:         "EUR thousands separator",
			currencyType: currencyx.CurrencyTypeFiat,
			code:         "EUR",
			precision:    0,
			symbol:       "",
			amount:       "1234567",
			expected:     "€1.234.567",
		},
		// Custom: configured symbol with default separators
		{
			name:         "custom thousands separator",
			currencyType: currencyx.CurrencyTypeCustom,
			code:         "CREDITS",
			precision:    2,
			symbol:       "©",
			amount:       "1234",
			expected:     "©1,234",
		},
		{
			name:         "custom two decimals",
			currencyType: currencyx.CurrencyTypeCustom,
			code:         "CREDITS",
			precision:    2,
			symbol:       "©",
			amount:       "1.50",
			expected:     "©1.50",
		},
		{
			name:         "USD below precision rounds to zero",
			currencyType: currencyx.CurrencyTypeFiat,
			code:         "USD",
			amount:       "0.000001",
			expected:     "$0.00",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := currencyx.NewCurrencyBuilder(c.currencyType).
				WithCode(currencyx.Code(c.code)).
				WithName(c.code)

			if c.currencyType == currencyx.CurrencyTypeCustom {
				b = b.WithPrecision(c.precision).WithSymbol(c.symbol)
			}

			currency, err := b.Build()
			require.NoError(t, err)

			result := currency.FormatAmount(alpacadecimal.RequireFromString(c.amount))

			require.Equal(t, c.expected, result)
		})
	}
}

func TestUnit(t *testing.T) {
	cases := []struct {
		name         string
		currencyType currencyx.CurrencyType
		code         string
		precision    uint32
		expected     string
	}{
		{
			name:         "USD",
			currencyType: currencyx.CurrencyTypeFiat,
			code:         "USD",
			precision:    0,
			expected:     "0.01",
		},
		{
			name:         "JPY",
			currencyType: currencyx.CurrencyTypeFiat,
			code:         "JPY",
			precision:    0,
			expected:     "1",
		},
		{
			name:         "custom precision 4",
			currencyType: currencyx.CurrencyTypeCustom,
			code:         "CREDITS",
			precision:    4,
			expected:     "0.0001",
		},
		{
			name:         "custom precision 0",
			currencyType: currencyx.CurrencyTypeCustom,
			code:         "TOKENS",
			precision:    0,
			expected:     "1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := currencyx.NewCurrencyBuilder(tc.currencyType).
				WithCode(currencyx.Code(tc.code)).
				WithName(tc.code)

			if tc.currencyType == currencyx.CurrencyTypeCustom {
				b = b.WithPrecision(tc.precision)
			}

			c, err := b.Build()
			require.NoError(t, err)

			require.Equal(t, tc.expected, c.Unit().String())
		})
	}
}

func TestRoundUpAndRoundDown(t *testing.T) {
	usd, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeFiat).
		WithCode(currencyx.Code("USD")).
		Build()
	require.NoError(t, err)

	// RoundUp always rounds away from zero towards the next subunit; RoundDown truncates towards zero.
	require.Equal(t, "1.23", usd.RoundUp(alpacadecimal.RequireFromString("1.221")).String())
	require.Equal(t, "1.22", usd.RoundDown(alpacadecimal.RequireFromString("1.229")).String())

	// Exact values at the precision are unchanged by either direction.
	require.Equal(t, "1.22", usd.RoundUp(alpacadecimal.RequireFromString("1.22")).String())
	require.Equal(t, "1.22", usd.RoundDown(alpacadecimal.RequireFromString("1.22")).String())
}

func TestCurrencyValidateWith(t *testing.T) {
	fiat, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeFiat).
		WithCode(currencyx.Code("USD")).
		Build()
	require.NoError(t, err)

	require.NoError(t, fiat.(*currencyx.FiatCurrency).Validate())
	require.NoError(t, fiat.(*currencyx.FiatCurrency).ValidateWith())

	custom, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode(currencyx.Code("CREDITS")).
		WithName("Credits").
		WithPrecision(2).
		Build()
	require.NoError(t, err)

	require.NoError(t, custom.(*currencyx.CustomCurrency).Validate())
	require.NoError(t, custom.(*currencyx.CustomCurrency).ValidateWith())
}

func TestRoundingBehavior(t *testing.T) {
	cases := []struct {
		name      string
		precision uint32
		amount    string
		expected  string
	}{
		{
			name:      "round down",
			precision: 2,
			amount:    "1.224",
			expected:  "1.22",
		},
		{
			name:      "round up at midpoint",
			precision: 2,
			amount:    "1.225",
			expected:  "1.23",
		},
		{
			name:      "round up",
			precision: 2,
			amount:    "1.226",
			expected:  "1.23",
		},
		{
			name:      "negative round down",
			precision: 2,
			amount:    "-1.224",
			expected:  "-1.22",
		},
		{
			name:      "negative round up at midpoint",
			precision: 2,
			amount:    "-1.225",
			expected:  "-1.23",
		},
		{
			name:      "zero precision round down",
			precision: 0,
			amount:    "2.4",
			expected:  "2",
		},
		{
			name:      "zero precision round up",
			precision: 0,
			amount:    "2.5",
			expected:  "3",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			custom, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
				WithCode(currencyx.Code("CREDITS")).
				WithName("Credits").
				WithPrecision(tc.precision).
				Build()
			require.NoError(t, err)

			require.Equal(
				t,
				tc.expected,
				custom.RoundToPrecision(alpacadecimal.RequireFromString(tc.amount)).String(),
			)
		})
	}
}

func TestFiatCurrencyReceivers(t *testing.T) {
	fiat, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeFiat).
		WithCode(currencyx.Code("USD")).
		Build()
	require.NoError(t, err)

	currency := fiat.(*currencyx.FiatCurrency)
	require.NoError(t, currency.Validate())
	require.NoError(t, (*currency).Validate())

	// Does not work
	// require.NoError(t, returnFiatCurrency(*currency).Validate())
}

func returnFiatCurrency(x currencyx.FiatCurrency) currencyx.FiatCurrency {
	return x
}
