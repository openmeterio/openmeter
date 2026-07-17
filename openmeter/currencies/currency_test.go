package currencies_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	currencytestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils/currency"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
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

	t.Run("managed custom currency requires ID", func(t *testing.T) {
		custom := currenciestestutils.NewManagedCurrency(t, "ns", "", "CREDITS")

		require.ErrorContains(t, custom.Validate(), "managed custom currency ID is required")
	})

	t.Run("valid currency", func(t *testing.T) {
		currency := currencytestutils.NewFiatCurrency(t, "USD")

		require.NoError(t, currency.Validate())
	})
}

func TestCurrencyGetCode(t *testing.T) {
	t.Run("missing currency", func(t *testing.T) {
		require.Empty(t, (currencies.Currency{}).GetCode())
	})

	t.Run("returns currency code", func(t *testing.T) {
		currency := currencytestutils.NewFiatCurrency(t, "USD")

		require.Equal(t, currencyx.Code("USD"), currency.GetCode())
	})
}

func TestCurrencyIdentity(t *testing.T) {
	// given:
	// - managed custom currencies whose codes may be reused by another resource
	// when:
	// - their identity and compatibility methods are used
	// then:
	// - custom equality follows the managed ID while fiat equality follows code
	credits := currenciestestutils.NewManagedCurrency(t, "ns", "currency-1", "CREDITS")
	sameIdentity := currenciestestutils.NewManagedCurrency(t, "ns", "currency-1", "RENAMED")
	reusedCode := currenciestestutils.NewManagedCurrency(t, "ns", "currency-2", "CREDITS")

	require.NoError(t, credits.Validate())
	assert.True(t, credits.IsCustom())
	assert.False(t, credits.IsFiat())
	assert.Equal(t, currencyx.Code("CREDITS"), credits.GetCode())
	assert.True(t, credits.Equal(sameIdentity))
	assert.False(t, credits.Equal(reusedCode))
	assert.False(t, credits.Equal(currencyx.Code("CREDITS")))

	usd := currencies.Currency{
		NamespacedID: models.NamespacedID{Namespace: "ns"},
		Currency:     currencytestutils.NewFiatCurrency(t, "USD").Currency,
	}
	assert.True(t, usd.IsFiat())
	assert.True(t, usd.Equal(currencyx.Code("USD")))
	assert.False(t, usd.Equal(currencyx.Code("EUR")))
}
