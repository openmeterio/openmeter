package invoicecalc

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestSnapshotTaxConfigIntoLines(t *testing.T) {
	tc1 := taxcode.TaxCode{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "tc-1"},
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: "txcd_10000000"},
		},
	}
	tc2 := taxcode.TaxCode{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "tc-2"},
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: "txcd_20000000"},
		},
	}

	newInvoice := func(status billing.StandardInvoiceStatus, defaultTC *productcatalog.TaxConfig, lines ...*billing.StandardLine) billing.StandardInvoice {
		return billing.StandardInvoice{
			StandardInvoiceBase: billing.StandardInvoiceBase{
				Status: status,
				Workflow: billing.InvoiceWorkflow{
					Config: billing.WorkflowConfig{
						Invoicing: billing.InvoicingConfig{
							DefaultTaxConfig: defaultTC,
						},
					},
				},
			},
			Lines: billing.NewStandardInvoiceLines(lines),
		}
	}

	newLine := func(tc *productcatalog.TaxConfig) *billing.StandardLine {
		return &billing.StandardLine{
			StandardLineBase: billing.StandardLineBase{
				TaxConfig: tc,
			},
		}
	}

	tests := []struct {
		name      string
		invoice   billing.StandardInvoice
		deps      CalculatorDependencies
		wantTC    *productcatalog.TaxConfig
		wantNoErr bool
	}{
		{
			name: "gathering invoice is a no-op",
			invoice: newInvoice(
				billing.StandardInvoiceStatusGathering,
				nil,
				newLine(&productcatalog.TaxConfig{
					Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
				}),
			),
			deps:      CalculatorDependencies{TaxCodes: TaxCodes{"txcd_10000000": tc1}},
			wantTC:    &productcatalog.TaxConfig{Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"}},
			wantNoErr: true,
		},
		{
			name: "stamps TaxCodeID and TaxCode entity when code is in deps",
			invoice: newInvoice(
				billing.StandardInvoiceStatusDraftCollecting,
				nil,
				newLine(&productcatalog.TaxConfig{
					Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
				}),
			),
			deps: CalculatorDependencies{TaxCodes: TaxCodes{"txcd_10000000": tc1}},
			wantTC: &productcatalog.TaxConfig{
				Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
				TaxCodeID: lo.ToPtr("tc-1"),
				TaxCode:   &tc1,
			},
			wantNoErr: true,
		},
		{
			name: "preserves existing TaxCodeID but still stamps TaxCode entity",
			invoice: newInvoice(
				billing.StandardInvoiceStatusDraftCollecting,
				nil,
				newLine(&productcatalog.TaxConfig{
					Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
					TaxCodeID: lo.ToPtr("already-set"),
				}),
			),
			deps: CalculatorDependencies{TaxCodes: TaxCodes{"txcd_10000000": tc1}},
			wantTC: &productcatalog.TaxConfig{
				Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
				TaxCodeID: lo.ToPtr("already-set"),
				TaxCode:   &tc1,
			},
			wantNoErr: true,
		},
		{
			name: "gracefully skips when stripe code is absent from deps",
			invoice: newInvoice(
				billing.StandardInvoiceStatusDraftCollecting,
				nil,
				newLine(&productcatalog.TaxConfig{
					Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
				}),
			),
			deps: CalculatorDependencies{TaxCodes: TaxCodes{}},
			wantTC: &productcatalog.TaxConfig{
				Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
			},
			wantNoErr: true,
		},
		{
			name: "DefaultTaxConfig is merged into line with nil TaxConfig",
			invoice: newInvoice(
				billing.StandardInvoiceStatusDraftCollecting,
				&productcatalog.TaxConfig{
					Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior),
					Stripe:   &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
				},
				newLine(nil),
			),
			deps: CalculatorDependencies{TaxCodes: TaxCodes{"txcd_10000000": tc1}},
			wantTC: &productcatalog.TaxConfig{
				Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
				Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
				TaxCodeID: lo.ToPtr("tc-1"),
				TaxCode:   &tc1,
			},
			wantNoErr: true,
		},
		{
			name: "line stripe code takes precedence over DefaultTaxConfig stripe code",
			invoice: newInvoice(
				billing.StandardInvoiceStatusDraftCollecting,
				&productcatalog.TaxConfig{
					Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
				},
				newLine(&productcatalog.TaxConfig{
					Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_20000000"},
				}),
			),
			deps: CalculatorDependencies{TaxCodes: TaxCodes{
				"txcd_10000000": tc1,
				"txcd_20000000": tc2,
			}},
			wantTC: &productcatalog.TaxConfig{
				Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_20000000"},
				TaxCodeID: lo.ToPtr("tc-2"),
				TaxCode:   &tc2,
			},
			wantNoErr: true,
		},
		{
			name: "line with behavior-only TaxConfig skips entity stamping",
			invoice: newInvoice(
				billing.StandardInvoiceStatusDraftCollecting,
				nil,
				newLine(&productcatalog.TaxConfig{
					Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
				}),
			),
			deps: CalculatorDependencies{TaxCodes: TaxCodes{"txcd_10000000": tc1}},
			wantTC: &productcatalog.TaxConfig{
				Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
			},
			wantNoErr: true,
		},
		{
			name: "nil TaxConfig line with no DefaultTaxConfig stays nil",
			invoice: newInvoice(
				billing.StandardInvoiceStatusDraftCollecting,
				nil,
				newLine(nil),
			),
			deps:      CalculatorDependencies{},
			wantTC:    nil,
			wantNoErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SnapshotTaxConfigIntoLines(&tt.invoice, tt.deps)
			if tt.wantNoErr {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				return
			}

			lines := tt.invoice.Lines.OrEmpty()
			require.Len(t, lines, 1)
			assert.Equal(t, tt.wantTC, lines[0].TaxConfig)
		})
	}
}
