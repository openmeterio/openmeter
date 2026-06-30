package currencyx

import (
	"testing"

	"github.com/invopop/gobl/currency"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/models"
)

func TestNewCalculatorRequiresISOFiatDefinition(t *testing.T) {
	calculator, err := NewCalculator(Code("USD"))
	require.NoError(t, err)
	require.Equal(t, Code("USD"), calculator.CurrencyCode())

	_, err = NewCalculator(Code("BTC"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "fiat currency definition is required for BTC")
}

func TestCalculatorValidateRequiresISOFiatDefinition(t *testing.T) {
	def := currency.Get(currency.Code("BTC"))
	require.NotNil(t, def)
	require.Empty(t, def.ISONumeric)

	err := Calculator{
		currency:     Code("BTC"),
		def:          def,
		currencyType: CurrencyTypeFiat,
		precision:    int32(def.Subunits),
		roundingMode: RoundingModeHalfAwayFromZero,
	}.Validate()
	require.Error(t, err)
	require.True(t, models.IsGenericValidationError(err), "error must be a validation error")
	require.Contains(t, err.Error(), "fiat currency definition is required for BTC")
}
