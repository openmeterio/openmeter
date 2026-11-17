package invoicecalc

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

// TaxConfigEqualDetectsTaxCode is the regression guard for the fix: billing.TaxConfig.Equal
// now includes the resolved TaxCode entity. Two configs that are identical except for TaxCode
// (one nil, one stamped) must compare as NOT equal, so the adapter diff guard re-upserts the line
// and the snapshot is persisted to the tax_config JSONB column.
func TestTaxConfigEqualDetectsTaxCode(t *testing.T) {
	tc1 := taxcode.TaxCode{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "tc-1"},
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: "txcd_10000000"},
		},
	}

	// Persisted state: TaxCodeID set, but the TaxCode entity snapshot is absent.
	persistedState := &billing.TaxConfig{
		TaxConfig: productcatalog.TaxConfig{
			Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
			Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
			TaxCodeID: lo.ToPtr("tc-1"),
		},
		TaxCode: nil,
	}

	// In-memory state after SnapshotTaxConfigIntoLines stamps the entity. Identical except TaxCode.
	expectedState := &billing.TaxConfig{
		TaxConfig: productcatalog.TaxConfig{
			Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
			Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
			TaxCodeID: lo.ToPtr("tc-1"),
		},
		TaxCode: &tc1,
	}

	assert.False(t, persistedState.Equal(expectedState),
		"Equal must detect the stamped TaxCode so the line is re-upserted and the snapshot persisted")

	// Same-ID sub-case: two configs whose TaxCode shares the same ID but differs in other fields
	// (e.g. AppMappings) must compare as equal — equality is ID-only.
	tc2 := taxcode.TaxCode{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "tc-1"},
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: "txcd_20000000"},
		},
	}

	leftConfig := &billing.TaxConfig{
		TaxConfig: productcatalog.TaxConfig{
			Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
			Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
			TaxCodeID: lo.ToPtr("tc-1"),
		},
		TaxCode: &tc1,
	}
	rightConfig := &billing.TaxConfig{
		TaxConfig: productcatalog.TaxConfig{
			Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
			Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
			TaxCodeID: lo.ToPtr("tc-1"),
		},
		TaxCode: &tc2,
	}

	assert.True(t, leftConfig.Equal(rightConfig),
		"Equal uses ID-only comparison: same TaxCode.ID means equal regardless of other TaxCode fields")
}

func newSnapshotTestInvoice(status billing.StandardInvoiceStatus, lines ...*billing.StandardLine) billing.StandardInvoice {
	return billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Status: status,
		},
		Lines: billing.NewStandardInvoiceLines(lines),
	}
}

func newSnapshotTestLine(tc *productcatalog.TaxConfig) *billing.StandardLine {
	return &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			TaxConfig: billing.FromProductCatalog(tc),
		},
	}
}

// SnapshotTaxConfigIntoLines merges the default into each line and stamps the Tax Code the line's
// ID points at: the default's by ID, others by Stripe code, never one with a different ID.
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
	// Same Stripe mapping as tc1, but deleted.
	deletedDefault := taxcode.TaxCode{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "tc-deleted"},
		ManagedModel: models.ManagedModel{DeletedAt: lo.ToPtr(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))},
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: "txcd_10000000"},
		},
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
			invoice: newSnapshotTestInvoice(
				billing.StandardInvoiceStatusGathering,
				newSnapshotTestLine(&productcatalog.TaxConfig{
					Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
				}),
			),
			deps:      StandardInvoiceCalculatorDependencies{TaxCodes: TaxCodes{"txcd_10000000": tc1}},
			wantTC:    &billing.TaxConfig{TaxConfig: productcatalog.TaxConfig{Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"}}},
			wantNoErr: true,
		},
		{
			name: "stamps TaxCodeID and TaxCode entity when code is in deps",
			invoice: newSnapshotTestInvoice(
				billing.StandardInvoiceStatusDraftCollecting,
				newSnapshotTestLine(&productcatalog.TaxConfig{
					Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
				}),
			),
			deps: StandardInvoiceCalculatorDependencies{TaxCodes: TaxCodes{"txcd_10000000": tc1}},
			wantTC: &billing.TaxConfig{
				TaxConfig: productcatalog.TaxConfig{
					Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
					TaxCodeID: lo.ToPtr("tc-1"),
				},
				TaxCode: &tc1,
			},
			wantNoErr: true,
		},
		{
			// The line points at X, but its Stripe code maps to Y (e.g. X was deleted and Y is the
			// live code with the same mapping): Y must not be stamped onto a line referencing X.
			name: "line whose Stripe code maps to another Tax Code gets no entity",
			invoice: newSnapshotTestInvoice(
				billing.StandardInvoiceStatusDraftCollecting,
				newSnapshotTestLine(&productcatalog.TaxConfig{
					Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
					TaxCodeID: lo.ToPtr("tc-x"),
				}),
			),
			deps: StandardInvoiceCalculatorDependencies{TaxCodes: TaxCodes{"txcd_10000000": tc1}},
			wantTC: &billing.TaxConfig{
				TaxConfig: productcatalog.TaxConfig{
					Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
					TaxCodeID: lo.ToPtr("tc-x"),
				},
			},
			wantNoErr: true,
		},
		{
			name: "stale entity not matching the line's TaxCodeID is cleared",
			invoice: newSnapshotTestInvoice(
				billing.StandardInvoiceStatusDraftCollecting,
				&billing.StandardLine{StandardLineBase: billing.StandardLineBase{TaxConfig: &billing.TaxConfig{
					TaxConfig: productcatalog.TaxConfig{
						Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_20000000"},
						TaxCodeID: lo.ToPtr("tc-x"),
					},
					TaxCode: &tc1,
				}}},
			),
			deps: StandardInvoiceCalculatorDependencies{},
			wantTC: &billing.TaxConfig{
				TaxConfig: productcatalog.TaxConfig{
					Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_20000000"},
					TaxCodeID: lo.ToPtr("tc-x"),
				},
			},
			wantNoErr: true,
		},
		{
			// The default's Tax Code is never in TaxCodes; a deleted one would map to another code.
			name: "line inheriting a deleted default gets the default's Tax Code by ID",
			invoice: newSnapshotTestInvoice(
				billing.StandardInvoiceStatusDraftCollecting,
				newSnapshotTestLine(nil),
			),
			deps: StandardInvoiceCalculatorDependencies{
				TaxCodes: TaxCodes{"txcd_10000000": tc1},
				DefaultTaxConfig: billing.FromProductCatalog(&productcatalog.TaxConfig{
					Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
					TaxCodeID: lo.ToPtr(deletedDefault.ID),
				}),
				DefaultTaxCode: &deletedDefault,
			},
			wantTC: &billing.TaxConfig{
				TaxConfig: productcatalog.TaxConfig{
					Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
					TaxCodeID: lo.ToPtr(deletedDefault.ID),
				},
				TaxCode: &deletedDefault,
			},
			wantNoErr: true,
		},
		{
			name: "line with its own TaxCodeID equal to the default's gets the default's Tax Code",
			invoice: newSnapshotTestInvoice(
				billing.StandardInvoiceStatusDraftCollecting,
				newSnapshotTestLine(&productcatalog.TaxConfig{
					Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
					Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
					TaxCodeID: lo.ToPtr(deletedDefault.ID),
				}),
			),
			deps: StandardInvoiceCalculatorDependencies{
				TaxCodes:       TaxCodes{"txcd_10000000": tc1},
				DefaultTaxCode: &deletedDefault,
			},
			wantTC: &billing.TaxConfig{
				TaxConfig: productcatalog.TaxConfig{
					Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
					Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
					TaxCodeID: lo.ToPtr(deletedDefault.ID),
				},
				TaxCode: &deletedDefault,
			},
			wantNoErr: true,
		},
		{
			name: "gracefully skips when stripe code is absent from deps",
			invoice: newSnapshotTestInvoice(
				billing.StandardInvoiceStatusDraftCollecting,
				newSnapshotTestLine(&productcatalog.TaxConfig{
					Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
				}),
			),
			deps: StandardInvoiceCalculatorDependencies{TaxCodes: TaxCodes{}},
			wantTC: &billing.TaxConfig{
				TaxConfig: productcatalog.TaxConfig{
					Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
				},
			},
			wantNoErr: true,
		},
		{
			name: "DefaultTaxConfig is merged into line with nil TaxConfig",
			invoice: newSnapshotTestInvoice(
				billing.StandardInvoiceStatusDraftCollecting,
				newSnapshotTestLine(nil),
			),
			deps: StandardInvoiceCalculatorDependencies{
				TaxCodes: TaxCodes{"txcd_10000000": tc1},
				DefaultTaxConfig: billing.FromProductCatalog(&productcatalog.TaxConfig{
					Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior),
					Stripe:   &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
				}),
			},
			wantTC: &billing.TaxConfig{
				TaxConfig: productcatalog.TaxConfig{
					Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
					Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
					TaxCodeID: lo.ToPtr("tc-1"),
				},
				TaxCode: &tc1,
			},
			wantNoErr: true,
		},
		{
			name: "line stripe code takes precedence over DefaultTaxConfig stripe code",
			invoice: newSnapshotTestInvoice(
				billing.StandardInvoiceStatusDraftCollecting,
				newSnapshotTestLine(&productcatalog.TaxConfig{
					Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_20000000"},
				}),
			),
			deps: StandardInvoiceCalculatorDependencies{
				TaxCodes: TaxCodes{
					"txcd_10000000": tc1,
					"txcd_20000000": tc2,
				},
				DefaultTaxConfig: billing.FromProductCatalog(&productcatalog.TaxConfig{
					Stripe: &productcatalog.StripeTaxConfig{Code: "txcd_10000000"},
				}),
			},
			wantTC: &billing.TaxConfig{
				TaxConfig: productcatalog.TaxConfig{
					Stripe:    &productcatalog.StripeTaxConfig{Code: "txcd_20000000"},
					TaxCodeID: lo.ToPtr("tc-2"),
				},
				TaxCode: &tc2,
			},
			wantNoErr: true,
		},
		{
			name: "line with behavior-only TaxConfig skips entity stamping",
			invoice: newSnapshotTestInvoice(
				billing.StandardInvoiceStatusDraftCollecting,
				newSnapshotTestLine(&productcatalog.TaxConfig{
					Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
				}),
			),
			deps: StandardInvoiceCalculatorDependencies{TaxCodes: TaxCodes{"txcd_10000000": tc1}},
			wantTC: &billing.TaxConfig{
				TaxConfig: productcatalog.TaxConfig{
					Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
				},
			},
			wantNoErr: true,
		},
		{
			// An inherited ID the batch could not resolve (outside the namespace or gone) keeps
			// the ID on the line without an entity instead of failing calculation.
			name: "line TaxCodeID missing from deps.TaxCodesByID keeps the ID with no entity",
			invoice: newSnapshotTestInvoice(
				billing.StandardInvoiceStatusDraftCollecting,
				newSnapshotTestLine(&productcatalog.TaxConfig{
					TaxCodeID: lo.ToPtr("tc-missing"),
				}),
			),
			deps: StandardInvoiceCalculatorDependencies{
				TaxCodesByID: TaxCodesByID{"tc-1": &tc1},
			},
			wantTC: &billing.TaxConfig{
				TaxConfig: productcatalog.TaxConfig{
					TaxCodeID: lo.ToPtr("tc-missing"),
				},
			},
			wantNoErr: true,
		},
		{
			name: "nil TaxConfig line with no DefaultTaxConfig stays nil",
			invoice: newSnapshotTestInvoice(
				billing.StandardInvoiceStatusDraftCollecting,
				newSnapshotTestLine(nil),
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
