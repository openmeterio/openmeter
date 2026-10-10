package invoicecalc

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
)

func TestRecalculateTotalsAggregatesExistingLinesWithoutLineEngines(t *testing.T) {
	deletedAt := time.Now().Add(-time.Hour)

	// given: already-calculated active and deleted invoice lines
	invoice := billing.StandardInvoice{
		Lines: billing.NewStandardInvoiceLines([]*billing.StandardLine{
			{
				StandardLineBase: billing.StandardLineBase{
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromInt(12),
						Total:  alpacadecimal.NewFromInt(12),
					},
				},
			},
			{
				StandardLineBase: billing.StandardLineBase{
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromInt(100),
						Total:  alpacadecimal.NewFromInt(100),
					},
				},
			},
		}),
	}
	invoice.Lines.OrEmpty()[1].DeletedAt = &deletedAt

	// when: only the invoice aggregate is recalculated
	err := RecalculateTotals(&invoice)

	// then: existing active line totals are summed without recalculating either line
	require.NoError(t, err)
	require.Equal(t, 12.0, invoice.Totals.Amount.InexactFloat64())
	require.Equal(t, 12.0, invoice.Totals.Total.InexactFloat64())
}
