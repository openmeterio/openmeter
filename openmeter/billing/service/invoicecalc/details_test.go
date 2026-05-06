package invoicecalc

import (
	"fmt"
	"testing"

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
