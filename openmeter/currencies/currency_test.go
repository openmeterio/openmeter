package currencies

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestCurrencyIdentity(t *testing.T) {
	// given:
	// - managed custom currencies whose codes may be reused by another resource
	// when:
	// - their identity and compatibility methods are used
	// then:
	// - custom equality follows the managed ID while fiat equality follows code
	credits := Currency{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "currency-1"},
		Code:         "CREDITS",
	}
	sameIdentity := Currency{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "currency-1"},
		Code:         "RENAMED",
	}
	reusedCode := Currency{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "currency-2"},
		Code:         "CREDITS",
	}

	require.NoError(t, credits.Validate())
	assert.True(t, credits.IsCustom())
	assert.False(t, credits.IsFiat())
	assert.Equal(t, currencyx.Code("CREDITS"), credits.GetCode())
	assert.True(t, credits.Equal(sameIdentity))
	assert.False(t, credits.Equal(reusedCode))
	assert.False(t, credits.Equal(currencyx.Code("CREDITS")))

	usd := currencyx.Code("USD")
	assert.True(t, usd.IsFiat())
	assert.True(t, usd.Equal(currencyx.Code("USD")))
	assert.False(t, usd.Equal(currencyx.Code("EUR")))
}

func TestManagedCustomCurrencyRequiresID(t *testing.T) {
	err := (Currency{Code: "CREDITS"}).Validate()
	require.Error(t, err)
}
