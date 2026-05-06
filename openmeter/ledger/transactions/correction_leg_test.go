package transactions

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestRoutePairingKeyEquality(t *testing.T) {
	usd := currencyx.Code("USD")
	taxA := "tax_A"
	taxB := "tax_B"

	t.Run("same fields are equal", func(t *testing.T) {
		k1 := routePairingKey{currency: usd, taxCode: taxCodeKey(&taxA), costBasis: "null"}
		k2 := routePairingKey{currency: usd, taxCode: taxCodeKey(&taxA), costBasis: "null"}
		assert.Equal(t, k1, k2)
	})

	t.Run("different taxCode are not equal", func(t *testing.T) {
		k1 := routePairingKey{currency: usd, taxCode: taxCodeKey(&taxA), costBasis: "null"}
		k2 := routePairingKey{currency: usd, taxCode: taxCodeKey(&taxB), costBasis: "null"}
		assert.NotEqual(t, k1, k2)
	})

	t.Run("nil taxCode differs from non-nil", func(t *testing.T) {
		k1 := routePairingKey{currency: usd, taxCode: taxCodeKey(nil), costBasis: "null"}
		k2 := routePairingKey{currency: usd, taxCode: taxCodeKey(&taxA), costBasis: "null"}
		assert.NotEqual(t, k1, k2)
	})

	t.Run("nil taxCode keys are equal", func(t *testing.T) {
		k1 := routePairingKey{currency: usd, taxCode: taxCodeKey(nil), costBasis: "null"}
		k2 := routePairingKey{currency: usd, taxCode: taxCodeKey(nil), costBasis: "null"}
		assert.Equal(t, k1, k2)
	})
}

func TestRoutePairingKeyString(t *testing.T) {
	usd := currencyx.Code("USD")
	tax := "tax_A"

	t.Run("includes taxCode field", func(t *testing.T) {
		k := routePairingKey{currency: usd, taxCode: taxCodeKey(&tax), costBasis: "null"}
		s := k.String()
		assert.Contains(t, s, "tax_code=tax_A")
		assert.Contains(t, s, "currency=USD")
		assert.Contains(t, s, "cost_basis=null")
	})

	t.Run("null taxCode renders as null", func(t *testing.T) {
		k := routePairingKey{currency: usd, taxCode: taxCodeKey(nil), costBasis: "null"}
		assert.Contains(t, k.String(), "tax_code=null")
	})
}

func TestTaxCodeKey(t *testing.T) {
	t.Run("nil pointer returns null", func(t *testing.T) {
		assert.Equal(t, "null", taxCodeKey(nil))
	})

	t.Run("non-nil pointer returns value", func(t *testing.T) {
		v := "tax_digital_services"
		assert.Equal(t, "tax_digital_services", taxCodeKey(&v))
	})

	t.Run("empty string returns empty string", func(t *testing.T) {
		v := ""
		assert.Equal(t, "", taxCodeKey(&v))
	})
}
