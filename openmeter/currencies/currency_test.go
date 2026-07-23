package currencies_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	currencytestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils/currency"
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

func TestCurrencyReference(t *testing.T) {
	t.Run("custom authoring reference can be unresolved", func(t *testing.T) {
		reference := currencies.NewCurrencyReference("CREDITS")

		require.NoError(t, reference.Validate())

		_, ok := reference.CustomCurrency()
		require.False(t, ok, "currency reference is resolved")
		require.False(t, reference.IsResolved())
	})

	t.Run("resolved custom reference retains managed identity", func(t *testing.T) {
		custom := currenciestestutils.NewManagedCurrency(t, "ns", "currency-1", "CREDITS")

		reference, err := currencies.NewCurrencyReference("CREDITS").WithCurrency(&custom)
		require.NoError(t, err)
		require.True(t, reference.IsResolved())
		require.NotNil(t, reference.CustomCurrencyID)
		require.Equal(t, custom.ID, *reference.CustomCurrencyID)

		resolved, ok := reference.CustomCurrency()
		require.True(t, ok)
		require.Equal(t, custom.ID, resolved.ID)
	})

	t.Run("custom reference requires cost basis expansion", func(t *testing.T) {
		custom := currenciestestutils.NewManagedCurrency(t, "ns", "currency-1", "CREDITS")
		custom.CostBasis = nil

		reference := custom.Reference()
		require.False(t, reference.IsResolved())

		custom.CostBasis = &[]currencies.CostBasis{}
		reference = custom.Reference()
		require.True(t, reference.IsResolved())
	})

	t.Run("equality ignores runtime resolution", func(t *testing.T) {
		custom := currenciestestutils.NewManagedCurrency(t, "ns", "currency-1", "CREDITS")
		resolved := custom.Reference()
		customID := custom.ID
		persisted := currencies.CurrencyReference{
			Code:             custom.GetCode(),
			CustomCurrencyID: &customID,
		}

		require.True(t, resolved.Equal(persisted))
	})

	t.Run("resolution rejects a mismatched managed identity", func(t *testing.T) {
		custom := currenciestestutils.NewManagedCurrency(t, "ns", "currency-2", "CREDITS")
		expectedID := "currency-1"
		reference := currencies.CurrencyReference{
			Code:             "CREDITS",
			CustomCurrencyID: &expectedID,
		}

		_, err := reference.WithCurrency(&custom)
		require.ErrorContains(t, err, "id mismatch between reference and currency")
	})

	t.Run("fiat reference rejects custom identity", func(t *testing.T) {
		customID := "currency-1"
		reference := currencies.CurrencyReference{
			Code:             "USD",
			CustomCurrencyID: &customID,
		}

		require.ErrorContains(t, reference.Validate(), "fiat currency cannot have a custom currency id")
	})
}

func TestCurrencyReferenceClone(t *testing.T) {
	t.Run("unresolved reference", func(t *testing.T) {
		reference := currencies.NewCurrencyReference("CREDITS")

		clone := reference.Clone()

		require.Equal(t, reference, clone)
		require.Nil(t, clone.CustomCurrencyID)
		require.False(t, clone.IsResolved())
	})

	t.Run("resolved reference", func(t *testing.T) {
		custom := currenciestestutils.NewManagedCurrency(t, "ns", "currency-1", "CREDITS")
		custom.CostBasis = &[]currencies.CostBasis{{CurrencyID: custom.ID}}
		reference := custom.Reference()

		clone := reference.Clone()

		require.True(t, reference.Equal(clone))
		require.NotSame(t, reference.CustomCurrencyID, clone.CustomCurrencyID)

		resolved, ok := reference.CustomCurrency()
		require.True(t, ok)
		clonedResolved, ok := clone.CustomCurrency()
		require.True(t, ok)
		require.NotSame(t, resolved, clonedResolved)
		require.NotSame(t, resolved.CostBasis, clonedResolved.CostBasis)

		*clone.CustomCurrencyID = "currency-2"
		clonedResolved.ID = "currency-2"
		(*clonedResolved.CostBasis)[0].CurrencyID = "currency-2"

		require.Equal(t, "currency-1", *reference.CustomCurrencyID)
		require.Equal(t, "currency-1", resolved.ID)
		require.Equal(t, "currency-1", (*resolved.CostBasis)[0].CurrencyID)
	})
}
