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

func TestRecalculateDetailedLinesAndTotalsSkipsEngineWithSnapshotValidationIssue(t *testing.T) {
	for _, tc := range []struct {
		name     string
		severity billing.ValidationIssueSeverity
	}{
		{name: "critical issue from creation", severity: billing.ValidationIssueSeverityCritical},
		{name: "warning issue downgraded for retry", severity: billing.ValidationIssueSeverityWarning},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given: a line engine has incomplete output after quantity snapshotting failed
			engineType := billing.LineEngineTypeInvoice
			invoice := billing.StandardInvoice{
				ValidationIssues: billing.ValidationIssues{
					{
						Severity:  tc.severity,
						Code:      billing.ErrInvoiceLineFeatureHasNoMeters.Code,
						Component: billing.LineEngineValidationComponent(engineType),
					},
				},
				Lines: billing.NewStandardInvoiceLines([]*billing.StandardLine{
					{
						StandardLineBase: billing.StandardLineBase{
							Engine: engineType,
							Totals: totals.Totals{
								Amount: alpacadecimal.NewFromInt(12),
								Total:  alpacadecimal.NewFromInt(12),
							},
						},
					},
				}),
			}

			// when: invoice calculation runs before collection has repaired the line
			err := RecalculateDetailedLinesAndTotals(&invoice, StandardInvoiceCalculatorDependencies{
				LineEngines: staticLineEngineResolver{},
			})

			// then: the incomplete engine output is preserved without invoking its calculator
			require.NoError(t, err)
			require.Equal(t, float64(12), invoice.Totals.Amount.InexactFloat64())
			require.Equal(t, float64(12), invoice.Totals.Total.InexactFloat64())
		})
	}
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
	require.Equal(t, 12.0, invoice.Totals.Amount.InexactFloat64())
	require.Equal(t, 12.0, invoice.Totals.Total.InexactFloat64())
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
