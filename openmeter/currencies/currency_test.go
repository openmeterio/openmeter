package currencies_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils/currency"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestCurrencyValidate(t *testing.T) {
	t.Run("missing currency", func(t *testing.T) {
		err := (currencies.Currency{}).Validate()

		require.ErrorContains(t, err, "currency is required")
	})

	t.Run("delegates validation", func(t *testing.T) {
		err := (currencies.Currency{Currency: &currencyx.FiatCurrency{}}).Validate()

		require.ErrorContains(t, err, "fiat currency is not initialized")
	})

	t.Run("valid currency", func(t *testing.T) {
		currency := currenciestestutils.NewFiatCurrency(t, "USD")

		require.NoError(t, currency.Validate())
	})
}

func TestCurrencyGetCode(t *testing.T) {
	t.Run("missing currency", func(t *testing.T) {
		require.Empty(t, (currencies.Currency{}).GetCode())
	})

	t.Run("returns currency code", func(t *testing.T) {
		currency := currenciestestutils.NewFiatCurrency(t, "USD")

		require.Equal(t, currencyx.Code("USD"), currency.GetCode())
	})
}
