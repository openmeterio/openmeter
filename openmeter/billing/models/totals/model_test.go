package totals

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestTotalsRoundToPrecision(t *testing.T) {
	t.Parallel()

	calc, err := currencyx.Code(currency.USD).Calculator()
	require.NoError(t, err)

	in := Totals{
		Amount:              alpacadecimal.NewFromFloat(10.005),
		ChargesTotal:        alpacadecimal.NewFromFloat(0.335),
		DiscountsTotal:      alpacadecimal.NewFromFloat(1.115),
		TaxesInclusiveTotal: alpacadecimal.NewFromFloat(0.225),
		TaxesExclusiveTotal: alpacadecimal.NewFromFloat(0.445),
		TaxesTotal:          alpacadecimal.NewFromFloat(0.665),
		CreditsTotal:        alpacadecimal.NewFromFloat(2.775),
		Total:               alpacadecimal.NewFromFloat(7.335),
	}

	got := in.RoundToPrecision(calc)

	require.True(t, calc.IsRoundedToPrecision(got.Amount))
	require.True(t, calc.IsRoundedToPrecision(got.ChargesTotal))
	require.True(t, calc.IsRoundedToPrecision(got.DiscountsTotal))
	require.True(t, calc.IsRoundedToPrecision(got.TaxesInclusiveTotal))
	require.True(t, calc.IsRoundedToPrecision(got.TaxesExclusiveTotal))
	require.True(t, calc.IsRoundedToPrecision(got.TaxesTotal))
	require.True(t, calc.IsRoundedToPrecision(got.CreditsTotal))
	require.True(t, calc.IsRoundedToPrecision(got.Total))

	require.Equal(t, "10.01", got.Amount.String())
	require.Equal(t, "0.34", got.ChargesTotal.String())
	require.Equal(t, "1.12", got.DiscountsTotal.String())
	require.Equal(t, "0.23", got.TaxesInclusiveTotal.String())
	require.Equal(t, "0.45", got.TaxesExclusiveTotal.String())
	require.Equal(t, "0.67", got.TaxesTotal.String())
	require.Equal(t, "2.78", got.CreditsTotal.String())
	require.Equal(t, "7.34", got.Total.String())
}
