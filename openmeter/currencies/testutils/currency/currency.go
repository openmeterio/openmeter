package currency

import (
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

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
