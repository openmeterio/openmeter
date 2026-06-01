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

// TestBug2_TaxConfigEqualDetectsTaxCode is the regression guard for the fix: billing.TaxConfig.Equal
// now includes the resolved TaxCode entity. Two configs that are identical except for TaxCode
// (one nil, one stamped) must compare as NOT equal, so the adapter diff guard re-upserts the line
// and the snapshot is persisted to the tax_config JSONB column.
func TestBug2_TaxConfigEqualDetectsTaxCode(t *testing.T) {
	tc1 := taxcode.TaxCode{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "tc-1"},
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: "txcd_10000000"},
		},
	}

	// Persisted state: TaxCodeID set, but the TaxCode entity snapshot is absent.
	persistedState := &billing.TaxConfig{
		Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
		Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
		TaxCodeID: lo.ToPtr("tc-1"),
		TaxCode:   nil,
	}

	// In-memory state after SnapshotTaxConfigIntoLines stamps the entity. Identical except TaxCode.
	expectedState := &billing.TaxConfig{
		Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
		Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
		TaxCodeID: lo.ToPtr("tc-1"),
		TaxCode:   &tc1,
	}

	assert.False(t, persistedState.Equal(expectedState),
		"Equal must detect the stamped TaxCode so the line is re-upserted and the snapshot persisted")

	// Deep-equal sub-case: two configs whose TaxCode has the same ID/namespace/key/name but
	// different AppMappings must compare as NOT equal (shallow ID comparison would miss this).
	tc2 := taxcode.TaxCode{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "tc-1"},
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: "txcd_20000000"},
		},
	}

	leftConfig := &billing.TaxConfig{
		Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
		Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
		TaxCodeID: lo.ToPtr("tc-1"),
		TaxCode:   &tc1,
	}
	rightConfig := &billing.TaxConfig{
		Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
		Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
		TaxCodeID: lo.ToPtr("tc-1"),
		TaxCode:   &tc2,
	}

	assert.False(t, leftConfig.Equal(rightConfig),
		"Equal must detect different AppMappings even when TaxCode.ID is the same")
}

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
				TaxConfig: billing.FromProductCatalog(tc),
			},
		}
	}

	tests := []struct {
		name      string
		invoice   billing.StandardInvoice
		deps      StandardInvoiceCalculatorDependencies
		wantTC    *billing.TaxConfig
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
			deps:      StandardInvoiceCalculatorDependencies{TaxCodes: TaxCodes{"txcd_10000000": tc1}},
			wantTC:    &billing.TaxConfig{Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"}},
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
			deps: StandardInvoiceCalculatorDependencies{TaxCodes: TaxCodes{"txcd_10000000": tc1}},
			wantTC: &billing.TaxConfig{
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
			deps: StandardInvoiceCalculatorDependencies{TaxCodes: TaxCodes{"txcd_10000000": tc1}},
			wantTC: &billing.TaxConfig{
				Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
				TaxCodeID: lo.ToPtr("already-set"),
				TaxCode:   &tc1,
			},
			wantNoErr: true,
		},
		{
			// Reproduces the dedup production scenario:
			// line.TaxCodeID points to the loser UUID (written by a code path that resolved
			// the tax code outside of GetOrCreateByAppMapping), while deps.TaxCodes maps the
			// Stripe code to the winner entity.
			//
			// Expected: TaxCode snapshot IS stamped with winner data (so reads of tax_config.tax_code
			// return the correct entity); TaxCodeID is NOT overwritten (stays as loser) because the
			// non-nil guard prevents it. The dedup migration and a separate backfill are needed to
			// repair the TaxCodeID → winner realignment.
			//
			// If this test fails (TaxCode is nil), the bug is inside SnapshotTaxConfigIntoLines itself.
			// If this test passes, the production missing-snapshot bug is in a layer outside this
			// function (persistence path or a code path that bypasses calculateInvoice entirely).
			name: "dedup scenario: loser TaxCodeID is preserved but winner entity snapshot is stamped",
			invoice: newInvoice(
				billing.StandardInvoiceStatusDraftCollecting,
				nil,
				newLine(&productcatalog.TaxConfig{
					Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
					Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
					TaxCodeID: lo.ToPtr("loser-uuid"),
				}),
			),
			deps: StandardInvoiceCalculatorDependencies{TaxCodes: TaxCodes{"txcd_10000000": tc1}},
			wantTC: &billing.TaxConfig{
				Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
				Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
				TaxCodeID: lo.ToPtr("loser-uuid"), // not overwritten to winner
				TaxCode:   &tc1,                   // winner entity IS stamped
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
			deps: StandardInvoiceCalculatorDependencies{TaxCodes: TaxCodes{}},
			wantTC: &billing.TaxConfig{
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
			deps: StandardInvoiceCalculatorDependencies{TaxCodes: TaxCodes{"txcd_10000000": tc1}},
			wantTC: &billing.TaxConfig{
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
			deps: StandardInvoiceCalculatorDependencies{TaxCodes: TaxCodes{
				"txcd_10000000": tc1,
				"txcd_20000000": tc2,
			}},
			wantTC: &billing.TaxConfig{
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
			deps: StandardInvoiceCalculatorDependencies{TaxCodes: TaxCodes{"txcd_10000000": tc1}},
			wantTC: &billing.TaxConfig{
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
			deps:      StandardInvoiceCalculatorDependencies{},
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
