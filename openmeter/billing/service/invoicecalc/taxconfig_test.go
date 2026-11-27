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
			// when: the invoice default is snapshotted into the lines
			require.NoError(t, SnapshotTaxConfigIntoLines(&tt.invoice, tt.deps))

			// then: the line carries the expected merged tax config
			lines := tt.invoice.Lines.OrEmpty()
			require.Len(t, lines, 1)
			assert.Equal(t, tt.wantTC, lines[0].TaxConfig)
		})
	}
}

func newStampTestTaxCode(id string, deleted bool, mappings ...taxcode.TaxCodeAppMapping) *taxcode.TaxCode {
	tc := &taxcode.TaxCode{
		NamespacedID: models.NamespacedID{ID: id},
		AppMappings:  mappings,
	}

	if deleted {
		tc.DeletedAt = lo.ToPtr(time.Now())
	}

	return tc
}

// StampInvoicingAppTaxCodes runs right after SnapshotTaxConfigIntoLines and stamps the
// invoicing app's provider code (or "") on every line, overwriting any previous stamp.
func TestStampInvoicingAppTaxCodes(t *testing.T) {
	stripeMapping := taxcode.TaxCodeAppMapping{AppType: app.AppTypeStripe, TaxCode: "txcd_stripe"}
	customMapping := taxcode.TaxCodeAppMapping{AppType: app.AppTypeCustomInvoicing, TaxCode: "custom-code"}

	taxCodes := productcatalog.TaxCodesByID{
		"tc-mapped":   newStampTestTaxCode("tc-mapped", false, stripeMapping, customMapping),
		"tc-unmapped": newStampTestTaxCode("tc-unmapped", false, customMapping),
		"tc-deleted":  newStampTestTaxCode("tc-deleted", true, stripeMapping),
	}

	stampedLine := newSnapshotTestLine(&productcatalog.TaxCodeConfig{TaxCodeID: "tc-unmapped"})
	stampedLine.InvoicingAppTaxCode = lo.ToPtr("txcd_old")

	tests := []struct {
		name    string
		invoice billing.StandardInvoice
		deps    StandardInvoiceCalculatorDependencies
		want    *string
		wantErr string
	}{
		{
			name: "mapping for the invoicing app is stamped",
			invoice: newSnapshotTestInvoice(
				billing.StandardInvoiceStatusDraftCreated,
				newSnapshotTestLine(&productcatalog.TaxCodeConfig{TaxCodeID: "tc-mapped"}),
			),
			deps: StandardInvoiceCalculatorDependencies{TaxCodes: taxCodes, InvoicingAppType: app.AppTypeStripe},
			want: lo.ToPtr("txcd_stripe"),
		},
		{
			name: "custom invoicing app picks the custom invoicing mapping",
			invoice: newSnapshotTestInvoice(
				billing.StandardInvoiceStatusDraftCreated,
				newSnapshotTestLine(&productcatalog.TaxCodeConfig{TaxCodeID: "tc-mapped"}),
			),
			deps: StandardInvoiceCalculatorDependencies{TaxCodes: taxCodes, InvoicingAppType: app.AppTypeCustomInvoicing},
			want: lo.ToPtr("custom-code"),
		},
		{
			name: "tax code without a mapping for the invoicing app is stamped empty",
			invoice: newSnapshotTestInvoice(
				billing.StandardInvoiceStatusDraftCreated,
				newSnapshotTestLine(&productcatalog.TaxCodeConfig{TaxCodeID: "tc-unmapped"}),
			),
			deps: StandardInvoiceCalculatorDependencies{TaxCodes: taxCodes, InvoicingAppType: app.AppTypeStripe},
			want: lo.ToPtr(""),
		},
		{
			name: "line without a tax code is stamped empty",
			invoice: newSnapshotTestInvoice(
				billing.StandardInvoiceStatusDraftCreated,
				newSnapshotTestLine(&productcatalog.TaxCodeConfig{Behavior: lo.ToPtr(productcatalog.InclusiveTaxBehavior)}),
			),
			deps: StandardInvoiceCalculatorDependencies{TaxCodes: taxCodes, InvoicingAppType: app.AppTypeStripe},
			want: lo.ToPtr(""),
		},
		{
			name: "line without a tax config is stamped empty",
			invoice: newSnapshotTestInvoice(
				billing.StandardInvoiceStatusDraftCreated,
				newSnapshotTestLine(nil),
			),
			deps: StandardInvoiceCalculatorDependencies{TaxCodes: taxCodes, InvoicingAppType: app.AppTypeStripe},
			want: lo.ToPtr(""),
		},
		{
			name: "line inheriting a soft-deleted default tax code gets its mapping",
			invoice: newSnapshotTestInvoice(
				billing.StandardInvoiceStatusDraftCreated,
				newSnapshotTestLine(nil),
			),
			deps: StandardInvoiceCalculatorDependencies{
				DefaultTaxConfig: &productcatalog.TaxCodeConfig{TaxCodeID: "tc-deleted"},
				TaxCodes:         taxCodes,
				InvoicingAppType: app.AppTypeStripe,
			},
			want: lo.ToPtr("txcd_stripe"),
		},
		{
			name:    "existing stamp is overwritten",
			invoice: newSnapshotTestInvoice(billing.StandardInvoiceStatusDraftUpdating, stampedLine),
			deps:    StandardInvoiceCalculatorDependencies{TaxCodes: taxCodes, InvoicingAppType: app.AppTypeStripe},
			want:    lo.ToPtr(""),
		},
		{
			name: "gathering invoice is left untouched",
			invoice: newSnapshotTestInvoice(
				billing.StandardInvoiceStatusGathering,
				newSnapshotTestLine(&productcatalog.TaxCodeConfig{TaxCodeID: "tc-mapped"}),
			),
			deps: StandardInvoiceCalculatorDependencies{TaxCodes: taxCodes, InvoicingAppType: app.AppTypeStripe},
			want: nil,
		},
		{
			name: "no invoicing app type leaves the stamp untouched",
			invoice: newSnapshotTestInvoice(
				billing.StandardInvoiceStatusDraftCreated,
				newSnapshotTestLine(&productcatalog.TaxCodeConfig{TaxCodeID: "tc-mapped"}),
			),
			deps: StandardInvoiceCalculatorDependencies{TaxCodes: taxCodes},
			want: nil,
		},
		{
			name: "unknown tax code ID is an error",
			invoice: newSnapshotTestInvoice(
				billing.StandardInvoiceStatusDraftCreated,
				newSnapshotTestLine(&productcatalog.TaxCodeConfig{TaxCodeID: "tc-unknown"}),
			),
			deps:    StandardInvoiceCalculatorDependencies{TaxCodes: taxCodes, InvoicingAppType: app.AppTypeStripe},
			wantErr: "tc-unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: the invoice default is already snapshotted into the lines, as the calculator does first
			require.NoError(t, SnapshotTaxConfigIntoLines(&tt.invoice, tt.deps))

			// when: the invoicing app tax codes are stamped
			err := StampInvoicingAppTaxCodes(&tt.invoice, tt.deps)

			// then: the stamp is as expected, or the call fails for an unknown tax code
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)

			lines := tt.invoice.Lines.OrEmpty()
			require.Len(t, lines, 1)
			assert.Equal(t, tt.want, lines[0].InvoicingAppTaxCode)
		})
	}
}
