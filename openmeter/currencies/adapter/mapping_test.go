package adapter_test

import (
	"testing"

	"github.com/invopop/gobl/currency"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	currencyadapter "github.com/openmeterio/openmeter/openmeter/currencies/adapter"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestCurrencyReferenceMapping(t *testing.T) {
	t.Run("fiat code round trip", func(t *testing.T) {
		// given:
		// - a fiat value identity
		// when:
		// - it is mapped to and from the shared DB reference
		// then:
		// - only the fiat code is persisted
		ref, err := currencyadapter.ToDBCurrencyReference(currencyx.Code(currency.USD), false)
		require.NoError(t, err)
		require.NotNil(t, ref.FiatCurrencyCode)
		require.Equal(t, currency.USD.String(), *ref.FiatCurrencyCode)
		require.Nil(t, ref.CustomCurrencyID)

		identity, err := currencyadapter.FromDBCurrencyReference(ref, false)
		require.NoError(t, err)
		require.Equal(t, currencyx.Code(currency.USD), identity.GetCode())
		require.True(t, identity.IsFiat())
	})

	t.Run("managed custom identity round trip", func(t *testing.T) {
		// given:
		// - a managed custom currency and its eagerly loaded DB row
		// when:
		// - it is mapped to and from the shared DB reference
		// then:
		// - the managed resource ID, not only the reusable code, is retained
		custom := currencies.Currency{
			NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01J00000000000000000000000"},
			Code:         "CREDITS",
			Name:         "Credits",
		}

		ref, err := currencyadapter.ToDBCurrencyReference(custom, false)
		require.NoError(t, err)
		require.Nil(t, ref.FiatCurrencyCode)
		require.NotNil(t, ref.CustomCurrencyID)
		require.Equal(t, custom.ID, *ref.CustomCurrencyID)

		ref.CustomCurrency = &entdb.CustomCurrency{
			ID:        custom.ID,
			Namespace: custom.Namespace,
			Code:      custom.Code,
			Name:      custom.Name,
			Symbol:    "cr",
		}
		identity, err := currencyadapter.FromDBCurrencyReference(ref, false)
		require.NoError(t, err)
		managed, ok := identity.(currencyx.ManagedCurrency)
		require.True(t, ok)
		require.Equal(t, custom.ID, managed.GetID())
	})

	t.Run("empty reference", func(t *testing.T) {
		identity, err := currencyadapter.FromDBCurrencyReference(currencyadapter.CurrencyReference{}, true)
		require.NoError(t, err)
		require.Nil(t, identity)

		_, err = currencyadapter.FromDBCurrencyReference(currencyadapter.CurrencyReference{}, false)
		require.Error(t, err)
	})
}
