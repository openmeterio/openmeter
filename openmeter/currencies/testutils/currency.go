package testutils

import (
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func NewManagedCurrency(t *testing.T, namespace, id string, code currencyx.Code) currencies.Currency {
	t.Helper()

	currency, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode(code).
		WithName(code.String()).
		Build()
	require.NoError(t, err)

	return currencies.Currency{
		NamespacedID: models.NamespacedID{
			Namespace: namespace,
			ID:        id,
		},
		Currency:  currency,
		CostBasis: &[]currencies.CostBasis{},
	}
}

func NewCreateCurrencyInput(namespace string, code currencyx.Code, name, symbol string) currencies.CreateCurrencyInput {
	return currencies.CreateCurrencyInput{
		Namespace: namespace,
		CurrencyDetails: currencyx.CurrencyDetails{
			Code:               code,
			Name:               name,
			Symbol:             symbol,
			Precision:          2,
			DecimalMark:        ".",
			ThousandsSeparator: ",",
		},
	}
}

func NewFiatCurrency[T ~string](t testing.TB, code T) currencies.Currency {
	t.Helper()

	currency, err := currencies.NewFiatCurrency(currencyx.Code(code))
	require.NoError(t, err)

	return currency
}

// NewCustomCurrency builds a custom currency value for tests that only need
// the resolved calculator (code, precision) and a stable fixture managed ID,
// not a persisted custom_currencies row.
func NewCustomCurrency[T ~string](t testing.TB, code T, precision uint32) currencies.Currency {
	t.Helper()

	built, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode(currencyx.Code(code)).
		WithName(string(code)).
		WithPrecision(precision).
		Build()
	require.NoError(t, err)

	return currencies.Currency{
		NamespacedID: models.NamespacedID{ID: ulid.Make().String()},
		Currency:     built,
	}
}
