package currencyx_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestCodeValidate(t *testing.T) {
	testCases := []struct {
		name          string
		code          currencyx.Code
		expectedError string
	}{
		{
			name:          "empty",
			expectedError: "currency code is required",
		},
		{
			name: "valid fiat",
			code: "USD",
		},
		{
			name:          "unknown three-character code",
			code:          "ZZZ",
			expectedError: "invalid fiat currency code",
		},
		{
			name:          "non-fiat three-character code",
			code:          "BTC",
			expectedError: "invalid fiat currency code",
		},
		{
			name: "custom minimum length",
			code: "TOKN",
		},
		{
			name: "custom maximum length",
			code: currencyx.Code(strings.Repeat("A", currencyx.CustomCurrencyCodeMaxLength)),
		},
		{
			name:          "invalid short fiat code",
			code:          "AB",
			expectedError: "invalid fiat currency code",
		},
		{
			name:          "custom code too long",
			code:          currencyx.Code(strings.Repeat("A", currencyx.CustomCurrencyCodeMaxLength+1)),
			expectedError: "between 4 and 24 characters",
		},
		{
			name:          "custom code contains route delimiter",
			code:          "CRE|DITS",
			expectedError: "cannot contain route delimiter",
		},
		{
			name:          "custom code contains surrounding whitespace",
			code:          " CREDITS",
			expectedError: "cannot contain leading or trailing spaces",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.code.Validate()
			if testCase.expectedError == "" {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)
			require.True(t, models.IsGenericValidationError(err))
			require.Contains(t, err.Error(), testCase.expectedError)
		})
	}
}

func TestCodeEqual(t *testing.T) {
	testCases := []struct {
		name     string
		code     currencyx.Code
		other    currencyx.Code
		expected bool
	}{
		{
			name:     "same fiat code",
			code:     "USD",
			other:    "USD",
			expected: true,
		},
		{
			name:     "different fiat code",
			code:     "USD",
			other:    "EUR",
			expected: false,
		},
		{
			name:     "same custom code",
			code:     "CREDITS",
			other:    "CREDITS",
			expected: true,
		},
		{
			name:     "case sensitive",
			code:     "CREDITS",
			other:    "credits",
			expected: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expected, testCase.code.Equal(testCase.other))
		})
	}
}

func TestCodeType(t *testing.T) {
	testCases := []struct {
		name     string
		code     currencyx.Code
		expected currencyx.CurrencyType
	}{
		{
			name:     "one-character fiat classification",
			code:     "C",
			expected: currencyx.CurrencyTypeFiat,
		},
		{
			name:     "two-character fiat classification",
			code:     "CR",
			expected: currencyx.CurrencyTypeFiat,
		},
		{
			name:     "fiat",
			code:     "USD",
			expected: currencyx.CurrencyTypeFiat,
		},
		{
			name:     "custom",
			code:     "CREDITS",
			expected: currencyx.CurrencyTypeCustom,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expected, testCase.code.Type())
			require.Equal(t, testCase.expected == currencyx.CurrencyTypeFiat, testCase.code.IsFiat())
			require.Equal(t, testCase.expected == currencyx.CurrencyTypeCustom, testCase.code.IsCustom())
		})
	}
}
