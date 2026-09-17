package transactions

import (
	"testing"

	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"
)

func TestRoutePairingKeyEquality(t *testing.T) {
	usd := "FIAT:USD"
	taxA := "tax_A"
	taxB := "tax_B"

	t.Run("same fields are equal", func(t *testing.T) {
		k1 := routePairingKey{
			currency: usd,
			taxCode:  mo.PointerToOption(&taxA),
		}
		k2 := routePairingKey{
			currency: usd,
			taxCode:  mo.PointerToOption(lo.ToPtr(taxA)),
		}
		assert.True(t, k1 == k2)
	})

	t.Run("different taxCode are not equal", func(t *testing.T) {
		k1 := routePairingKey{
			currency: usd,
			taxCode:  mo.PointerToOption(&taxA),
		}
		k2 := routePairingKey{
			currency: usd,
			taxCode:  mo.PointerToOption(&taxB),
		}
		assert.NotEqual(t, k1, k2)
	})

	t.Run("nil taxCode differs from non-nil", func(t *testing.T) {
		k1 := routePairingKey{
			currency: usd,
			taxCode:  mo.None[string](),
		}
		k2 := routePairingKey{
			currency: usd,
			taxCode:  mo.PointerToOption(&taxA),
		}
		assert.NotEqual(t, k1, k2)
	})

	t.Run("nil taxCode keys are equal", func(t *testing.T) {
		k1 := routePairingKey{
			currency: usd,
			taxCode:  mo.None[string](),
		}
		k2 := routePairingKey{
			currency: usd,
			taxCode:  mo.None[string](),
		}
		assert.True(t, k1 == k2)
	})

	t.Run("absent provenance differs from present sentinel values", func(t *testing.T) {
		absent := routePairingKey{currency: usd}
		for _, value := range []string{"", "null"} {
			assert.False(t, absent == routePairingKey{
				currency:       usd,
				sourceChargeID: mo.Some(value),
			})
			assert.False(t, absent == routePairingKey{
				currency:      usd,
				spendChargeID: mo.Some(value),
			})
			assert.False(t, absent == routePairingKey{
				currency:           usd,
				collectionOriginID: mo.Some(value),
			})
			assert.False(t, absent == routePairingKey{
				currency: usd,
				taxCode:  mo.Some(value),
			})
		}
	})
}

func TestRoutePairingKeyString(t *testing.T) {
	usd := "FIAT:USD"
	tax := "tax_A"

	t.Run("includes taxCode field", func(t *testing.T) {
		k := routePairingKey{
			currency: usd,
			taxCode:  mo.PointerToOption(&tax),
		}
		s := k.String()
		assert.Contains(t, s, "tax_code=tax_A")
		assert.Contains(t, s, "currency=FIAT:USD")
		assert.Contains(t, s, "cost_basis=<unset>")
	})

	t.Run("absent taxCode renders as unset", func(t *testing.T) {
		k := routePairingKey{
			currency: usd,
			taxCode:  mo.None[string](),
		}
		assert.Contains(t, k.String(), "tax_code=<unset>")
	})
}
