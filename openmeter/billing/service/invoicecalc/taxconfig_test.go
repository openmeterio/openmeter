package invoicecalc

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func newSnapshotTestInvoice(status billing.StandardInvoiceStatus, lines ...*billing.StandardLine) billing.StandardInvoice {
	return billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Status: status,
		},
		Lines: billing.NewStandardInvoiceLines(lines),
	}
}

func newSnapshotTestLine(tc *productcatalog.TaxCodeConfig) *billing.StandardLine {
	return &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			TaxConfig: tc,
		},
	}
}

// SnapshotTaxConfigIntoLines merges the invoice default's Tax Code ID and behavior into each line,
// the line's own values winning. It never resolves or attaches Tax Code entities.
func TestSnapshotTaxConfigIntoLines(t *testing.T) {
	tests := []struct {
		name    string
		invoice billing.StandardInvoice
		deps    StandardInvoiceCalculatorDependencies
		wantTC  *productcatalog.TaxCodeConfig
	}{
		{
			name: "gathering invoice is a no-op",
			invoice: newSnapshotTestInvoice(
				billing.StandardInvoiceStatusGathering,
				newSnapshotTestLine(nil),
			),
			deps: StandardInvoiceCalculatorDependencies{
				DefaultTaxConfig: &productcatalog.TaxCodeConfig{TaxCodeID: "tc-default"},
			},
			wantTC: nil,
		},
		{
			name: "default is merged into line with nil TaxConfig",
			invoice: newSnapshotTestInvoice(
				billing.StandardInvoiceStatusDraftCollecting,
				newSnapshotTestLine(nil),
			),
			deps: StandardInvoiceCalculatorDependencies{
				DefaultTaxConfig: &productcatalog.TaxCodeConfig{
					Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
					TaxCodeID: "tc-default",
				},
			},
			wantTC: &productcatalog.TaxCodeConfig{
				Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
				TaxCodeID: "tc-default",
			},
		},
		{
			name: "line tax code ID takes precedence over the default's",
			invoice: newSnapshotTestInvoice(
				billing.StandardInvoiceStatusDraftCollecting,
				newSnapshotTestLine(&productcatalog.TaxCodeConfig{TaxCodeID: "tc-line"}),
			),
			deps: StandardInvoiceCalculatorDependencies{
				DefaultTaxConfig: &productcatalog.TaxCodeConfig{
					Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
					TaxCodeID: "tc-default",
				},
			},
			wantTC: &productcatalog.TaxCodeConfig{
				Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
				TaxCodeID: "tc-line",
			},
		},
		{
			name: "line behavior-only config inherits the default's tax code ID",
			invoice: newSnapshotTestInvoice(
				billing.StandardInvoiceStatusDraftCollecting,
				newSnapshotTestLine(&productcatalog.TaxCodeConfig{Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior)}),
			),
			deps: StandardInvoiceCalculatorDependencies{
				DefaultTaxConfig: &productcatalog.TaxCodeConfig{
					Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
					TaxCodeID: "tc-default",
				},
			},
			wantTC: &productcatalog.TaxCodeConfig{
				Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
				TaxCodeID: "tc-default",
			},
		},
		{
			name: "nil TaxConfig line with no default stays nil",
			invoice: newSnapshotTestInvoice(
				billing.StandardInvoiceStatusDraftCollecting,
				newSnapshotTestLine(nil),
			),
			deps:   StandardInvoiceCalculatorDependencies{},
			wantTC: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, SnapshotTaxConfigIntoLines(&tt.invoice, tt.deps))

			lines := tt.invoice.Lines.OrEmpty()
			require.Len(t, lines, 1)
			assert.Equal(t, tt.wantTC, lines[0].TaxConfig)
		})
	}
}
