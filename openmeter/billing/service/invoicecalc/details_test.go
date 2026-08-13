package invoicecalc

import (
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	billingtestutils "github.com/openmeterio/openmeter/openmeter/billing/testutils"
)

func TestRecalculateDetailedLinesAndTotalsSkipsEnginesWithoutCalculator(t *testing.T) {
	invoice := billing.StandardInvoice{
		Lines: billing.NewStandardInvoiceLines([]*billing.StandardLine{
			{
				StandardLineBase: billing.StandardLineBase{
					Engine: billing.LineEngineTypeChargeUsageBased,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromInt(12),
						Total:  alpacadecimal.NewFromInt(12),
					},
				},
			},
		}),
	}

	err := RecalculateDetailedLinesAndTotals(&invoice, StandardInvoiceCalculatorDependencies{
		LineEngines: staticLineEngineResolver{
			billing.LineEngineTypeChargeUsageBased: nonCalculatingLineEngine{
				NoopLineEngine: billingtestutils.NoopLineEngine{
					EngineType: billing.LineEngineTypeChargeUsageBased,
				},
			},
		},
	})
	require.NoError(t, err)
	require.True(t, alpacadecimal.NewFromInt(12).Equal(invoice.Totals.Amount))
	require.True(t, alpacadecimal.NewFromInt(12).Equal(invoice.Totals.Total))
}

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
	require.True(t, alpacadecimal.NewFromInt(12).Equal(invoice.Totals.Amount))
	require.True(t, alpacadecimal.NewFromInt(12).Equal(invoice.Totals.Total))
}

type staticLineEngineResolver map[billing.LineEngineType]billing.LineEngine

func (r staticLineEngineResolver) Get(engineType billing.LineEngineType) (billing.LineEngine, error) {
	engine, ok := r[engineType]
	if !ok {
		return nil, fmt.Errorf("engine %s is not registered", engineType)
	}

	return engine, nil
}

type nonCalculatingLineEngine struct {
	billingtestutils.NoopLineEngine
}
