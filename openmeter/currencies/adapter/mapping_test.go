package adapter_test

import (
	"testing"

	"github.com/invopop/gobl/currency"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	currencyadapter "github.com/openmeterio/openmeter/openmeter/currencies/adapter"
	currencytestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestCurrencyReferenceMapping(t *testing.T) {
	t.Run("fiat code round trip", func(t *testing.T) {
		// given:
		// - a fiat value identity
		// when:
		// - it is mapped to and from the shared DB reference
		// then:
		// - only the fiat code is persisted
		fiat, err := currencies.NewFiatCurrency(currencyx.Code(currency.USD))
		require.NoError(t, err)
		reference := fiat.Reference()

		ref, err := currencyadapter.ToDBCurrencyReference(&reference, false)
		require.NoError(t, err)
		require.NotNil(t, ref.FiatCurrencyCode)
		require.Equal(t, currency.USD.String(), *ref.FiatCurrencyCode)
		require.Nil(t, ref.CustomCurrencyID)

		identity, err := currencyadapter.FromDBCurrencyReference(ref, false)
		require.NoError(t, err)
		require.Equal(t, currencyx.Code(currency.USD), identity.Code)
		require.True(t, identity.IsFiat())
		require.True(t, identity.IsResolved())
	})

	t.Run("code-only fiat reference can be persisted", func(t *testing.T) {
		reference := currencies.NewCurrencyReference(currencyx.Code(currency.EUR))

		ref, err := currencyadapter.ToDBCurrencyReference(&reference, false)
		require.NoError(t, err)
		require.NotNil(t, ref.FiatCurrencyCode)
		require.Equal(t, currency.EUR.String(), *ref.FiatCurrencyCode)
		require.Nil(t, ref.CustomCurrencyID)
	})

	t.Run("code-only custom reference cannot be persisted", func(t *testing.T) {
		reference := currencies.NewCurrencyReference("CREDITS")

		_, err := currencyadapter.ToDBCurrencyReference(&reference, false)
		require.ErrorContains(t, err, "has no managed resource identity")
	})

	t.Run("managed custom identity round trip", func(t *testing.T) {
		// given:
		// - a managed custom currency and its eagerly loaded DB row
		// when:
		// - it is mapped to and from the shared DB reference
		// then:
		// - the managed resource ID, not only the reusable code, is retained
		custom := currencytestutils.NewManagedCurrency(t, "ns", "01J00000000000000000000000", "CREDITS")

		reference := custom.Reference()
		ref, err := currencyadapter.ToDBCurrencyReference(&reference, false)
		require.NoError(t, err)
		require.Nil(t, ref.FiatCurrencyCode)
		require.NotNil(t, ref.CustomCurrencyID)
		require.Equal(t, custom.ID, *ref.CustomCurrencyID)

		ref.CustomCurrency = &entdb.CustomCurrency{
			ID:        custom.ID,
			Namespace: custom.Namespace,
			Code:      custom.GetCode(),
			Name:      custom.Details().Name,
			Symbol:    "cr",
		}
		identity, err := currencyadapter.FromDBCurrencyReference(ref, false)
		require.NoError(t, err)
		require.NotNil(t, identity.CustomCurrencyID)
		require.Equal(t, custom.ID, *identity.CustomCurrencyID)
		require.False(t, identity.IsResolved(), "cost-basis edge was not loaded")
		resolved, ok := identity.CustomCurrency()
		require.True(t, ok)
		require.Equal(t, custom.ID, resolved.ID)
	})

	t.Run("empty reference", func(t *testing.T) {
		identity, err := currencyadapter.FromDBCurrencyReference(currencyadapter.CurrencyReference{}, true)
		require.NoError(t, err)
		require.Nil(t, identity)

		_, err = currencyadapter.FromDBCurrencyReference(currencyadapter.CurrencyReference{}, false)
		require.Error(t, err)
	})
}
