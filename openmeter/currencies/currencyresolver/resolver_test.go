package currencyresolver_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/currencies/currencyresolver"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestNew(t *testing.T) {
	_, err := currencyresolver.New(nil)
	require.EqualError(t, err, "currency service is required")
}

func TestCurrencyResolver(t *testing.T) {
	env := currenciestestutils.NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	namespace := currenciestestutils.NewTestNamespace(t)
	otherNamespace := currenciestestutils.NewTestNamespace(t)

	credits, err := env.Service.CreateCurrency(t.Context(), currencies.CreateCurrencyInput{
		Namespace: namespace,
		CurrencyDetails: currencyx.CurrencyDetails{
			Code:               "CREDITS",
			Name:               "Credits",
			Symbol:             "C",
			Precision:          2,
			DecimalMark:        ".",
			ThousandsSeparator: ",",
		},
	})
	require.NoError(t, err)

	points, err := env.Service.CreateCurrency(t.Context(), currencies.CreateCurrencyInput{
		Namespace: namespace,
		CurrencyDetails: currencyx.CurrencyDetails{
			Code:               "POINTS",
			Name:               "Points",
			Symbol:             "P",
			Precision:          4,
			DecimalMark:        ".",
			ThousandsSeparator: ",",
		},
	})
	require.NoError(t, err)

	otherCredits, err := env.Service.CreateCurrency(t.Context(), currencies.CreateCurrencyInput{
		Namespace: otherNamespace,
		CurrencyDetails: currencyx.CurrencyDetails{
			Code:               "CREDITS",
			Name:               "Other Credits",
			Symbol:             "OC",
			Precision:          6,
			DecimalMark:        ".",
			ThousandsSeparator: ",",
		},
	})
	require.NoError(t, err)

	resolver, err := currencyresolver.New(env.Service)
	require.NoError(t, err)

	namespacedResolver := resolver.WithNamespace(namespace)
	assert.Equal(t, namespace, namespacedResolver.Namespace())

	t.Run("ResolveCurrency", func(t *testing.T) {
		tests := []struct {
			name          string
			ref           currencies.CurrencyRef
			expected      *currencies.Currency
			expectedType  currencyx.CurrencyType
			expectedError string
			notFound      bool
		}{
			{
				name:          "empty reference",
				ref:           currencies.CurrencyRef{},
				expectedError: "currency id or code is required",
			},
			{
				name: "custom currency by id",
				ref: currencies.CurrencyRef{
					ID: credits.ID,
				},
				expected:     &credits,
				expectedType: currencyx.CurrencyTypeCustom,
			},
			{
				name: "custom currency by code",
				ref: currencies.CurrencyRef{
					Code: credits.Details().Code,
				},
				expected:     &credits,
				expectedType: currencyx.CurrencyTypeCustom,
			},
			{
				name: "fiat currency by code",
				ref: currencies.CurrencyRef{
					Code: "USD",
				},
				expectedType: currencyx.CurrencyTypeFiat,
			},
			{
				name: "id takes precedence over code",
				ref: currencies.CurrencyRef{
					ID:   credits.ID,
					Code: points.Details().Code,
				},
				expected:     &credits,
				expectedType: currencyx.CurrencyTypeCustom,
			},
			{
				name: "missing id does not fall back to code",
				ref: currencies.CurrencyRef{
					ID:   "missing",
					Code: credits.Details().Code,
				},
				notFound: true,
			},
			{
				name: "missing code",
				ref: currencies.CurrencyRef{
					Code: "UNKNOWN",
				},
				notFound: true,
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				result, err := namespacedResolver.ResolveCurrency(t.Context(), test.ref)
				switch {
				case test.expectedError != "":
					require.EqualError(t, err, test.expectedError)
					assert.Nil(t, result)
				case test.notFound:
					require.Error(t, err)
					assert.True(t, models.IsGenericNotFoundError(err))
					assert.Nil(t, result)
				default:
					require.NoError(t, err)
					require.NotNil(t, result)
					assert.Equal(t, test.expectedType, result.Type())

					if test.ref.ID == "" {
						assert.Equal(t, test.ref.Code, result.Details().Code)
					}

					if test.expected != nil {
						assert.Equal(t, test.expected.ID, result.ID)
						assert.Equal(t, test.expected.Namespace, result.Namespace)
						assert.NotEqual(t, otherCredits.ID, result.ID)
					}
				}
			})
		}
	})

	t.Run("BatchResolveCurrencies", func(t *testing.T) {
		creditsByID := currencies.CurrencyRef{ID: credits.ID}
		creditsByCode := currencies.CurrencyRef{Code: credits.Details().Code}
		pointsByIDWithDifferentCode := currencies.CurrencyRef{
			ID:   points.ID,
			Code: credits.Details().Code,
		}
		usdByCode := currencies.CurrencyRef{Code: "USD"}
		missingByCode := currencies.CurrencyRef{Code: "UNKNOWN"}

		result, err := namespacedResolver.BatchResolveCurrencies(
			t.Context(),
			creditsByID,
			creditsByCode,
			pointsByIDWithDifferentCode,
			usdByCode,
			missingByCode,
		)
		require.NoError(t, err)
		require.Len(t, result, 5)

		require.NotNil(t, result[creditsByID])
		assert.Equal(t, credits.ID, result[creditsByID].ID)

		require.NotNil(t, result[creditsByCode])
		assert.Equal(t, credits.ID, result[creditsByCode].ID)

		require.NotNil(t, result[pointsByIDWithDifferentCode])
		assert.Equal(t, points.ID, result[pointsByIDWithDifferentCode].ID)

		require.NotNil(t, result[usdByCode])
		assert.Equal(t, currencyx.CurrencyTypeFiat, result[usdByCode].Type())
		assert.Equal(t, currencyx.Code("USD"), result[usdByCode].Details().Code)

		assert.Nil(t, result[missingByCode])
	})

	t.Run("BatchResolveCurrenciesEmpty", func(t *testing.T) {
		result, err := namespacedResolver.BatchResolveCurrencies(t.Context())
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("NamespaceIsRequired", func(t *testing.T) {
		result, err := resolver.BatchResolveCurrencies(t.Context(), "", currencies.CurrencyRef{ID: credits.ID})
		require.EqualError(t, err, "namespace is not set")
		assert.Nil(t, result)
	})
}
