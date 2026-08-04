package currencies_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
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
		require.True(t, reference.IsResolved())
		require.False(t, reference.IsCostBasisResolved())

		custom.CostBasis = &[]currencies.CostBasis{}
		reference = custom.Reference()
		require.True(t, reference.IsResolved())
		require.True(t, reference.IsCostBasisResolved())
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

func TestCurrencyReferenceSerialization(t *testing.T) {
	t.Run("fiat", func(t *testing.T) {
		reference := currencies.NewCurrencyReference("USD")

		serialized, err := reference.MarshalText()
		require.NoError(t, err)
		require.Equal(t, "USD", string(serialized))
		prefix, err := reference.MarshalTextPrefix()
		require.NoError(t, err)
		require.Equal(t, "USD", string(prefix))

		parsed, err := currencies.ParseCurrencyReference(serialized)
		require.NoError(t, err)
		require.True(t, reference.Equal(parsed))
		require.True(t, parsed.IsResolved())

		var unmarshaled currencies.CurrencyReference
		require.NoError(t, unmarshaled.UnmarshalText(serialized))
		require.True(t, reference.Equal(unmarshaled))
	})

	t.Run("custom", func(t *testing.T) {
		custom := currenciestestutils.NewCustomCurrency(t, "CRED:ITS", 2)
		reference := custom.Reference()

		serialized, err := reference.MarshalText()
		require.NoError(t, err)
		require.Equal(t, "custom|v1|CRED:ITS|"+custom.ID+"|2", string(serialized))
		prefix, err := reference.MarshalTextPrefix()
		require.NoError(t, err)
		require.Equal(t, "custom|v1|CRED:ITS|"+custom.ID+"|", string(prefix))

		parsed, err := currencies.ParseCurrencyReference(serialized)
		require.NoError(t, err)
		require.True(t, reference.Equal(parsed))
		require.True(t, parsed.IsResolved())

		var unmarshaled currencies.CurrencyReference
		require.NoError(t, unmarshaled.UnmarshalText(serialized))
		require.True(t, reference.Equal(unmarshaled))
		require.True(t, unmarshaled.IsResolved())

		referenceCurrency, ok := reference.CustomCurrency()
		require.True(t, ok)
		unmarshaledCurrency, ok := unmarshaled.CustomCurrency()
		require.True(t, ok)
		require.Equal(t, referenceCurrency.Details().Precision, unmarshaledCurrency.Details().Precision)
		require.False(t, parsed.IsCostBasisResolved())

		resolved, ok := parsed.CustomCurrency()
		require.True(t, ok)
		require.Equal(t, uint32(2), resolved.Details().Precision)
	})

	t.Run("unresolved custom", func(t *testing.T) {
		reference := currencies.NewCurrencyReference("CREDITS")

		_, err := reference.MarshalText()
		require.ErrorContains(t, err, "custom currency reference must be resolved")
		prefix, err := reference.MarshalTextPrefix()
		require.NoError(t, err)
		require.Equal(t, "custom|v1|CREDITS|", string(prefix))
	})

	t.Run("invalid values", func(t *testing.T) {
		for _, value := range []string{
			"CREDITS",
			"custom|v2|CREDITS|currency-1|2",
			"custom|v1|USD|currency-1|2",
			"custom|v1|CREDITS||2",
			"custom|v1|CREDITS|currency-1|invalid",
			"custom|v1|CREDITS|currency-1|13",
		} {
			t.Run(value, func(t *testing.T) {
				_, err := currencies.ParseCurrencyReference([]byte(value))
				require.Error(t, err)
			})
		}
	})

	t.Run("json representation remains an object", func(t *testing.T) {
		custom := currenciestestutils.NewCustomCurrency(t, "CREDITS", 2)

		serialized, err := json.Marshal(custom.Reference())
		require.NoError(t, err)
		require.JSONEq(t, `{"code":"CREDITS","custom_currency_id":"`+custom.ID+`"}`, string(serialized))

		var unmarshaled currencies.CurrencyReference
		require.NoError(t, json.Unmarshal(serialized, &unmarshaled))
		require.True(t, custom.Reference().Equal(unmarshaled))
		require.False(t, unmarshaled.IsResolved())
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
