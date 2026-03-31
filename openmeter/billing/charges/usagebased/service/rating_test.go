package service

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/stretchr/testify/require"

	modeltotals "github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestRoundGenerateDetailedLinesResultTotals(t *testing.T) {
	t.Parallel()

	calc, err := currencyx.Code(currency.USD).Calculator()
	require.NoError(t, err)

	in := rating.GenerateDetailedLinesResult{
		Totals: modeltotals.Totals{
			Amount:              alpacadecimal.NewFromFloat(10.005),
			ChargesTotal:        alpacadecimal.NewFromFloat(0.335),
			DiscountsTotal:      alpacadecimal.NewFromFloat(1.115),
			TaxesInclusiveTotal: alpacadecimal.NewFromFloat(0.225),
			TaxesExclusiveTotal: alpacadecimal.NewFromFloat(0.445),
			TaxesTotal:          alpacadecimal.NewFromFloat(0.665),
			CreditsTotal:        alpacadecimal.NewFromFloat(2.775),
			Total:               alpacadecimal.NewFromFloat(7.335),
		},
	}

	got := roundGenerateDetailedLinesResultTotals(in, calc)

	require.True(t, calc.IsRoundedToPrecision(got.Totals.Amount))
	require.True(t, calc.IsRoundedToPrecision(got.Totals.ChargesTotal))
	require.True(t, calc.IsRoundedToPrecision(got.Totals.DiscountsTotal))
	require.True(t, calc.IsRoundedToPrecision(got.Totals.TaxesInclusiveTotal))
	require.True(t, calc.IsRoundedToPrecision(got.Totals.TaxesExclusiveTotal))
	require.True(t, calc.IsRoundedToPrecision(got.Totals.TaxesTotal))
	require.True(t, calc.IsRoundedToPrecision(got.Totals.CreditsTotal))
	require.True(t, calc.IsRoundedToPrecision(got.Totals.Total))

	require.Equal(t, "10.01", got.Totals.Amount.String())
	require.Equal(t, "0.34", got.Totals.ChargesTotal.String())
	require.Equal(t, "1.12", got.Totals.DiscountsTotal.String())
	require.Equal(t, "0.23", got.Totals.TaxesInclusiveTotal.String())
	require.Equal(t, "0.45", got.Totals.TaxesExclusiveTotal.String())
	require.Equal(t, "0.67", got.Totals.TaxesTotal.String())
	require.Equal(t, "2.78", got.Totals.CreditsTotal.String())
	require.Equal(t, "7.34", got.Totals.Total.String())
}
