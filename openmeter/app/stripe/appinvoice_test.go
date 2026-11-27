package appstripe

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v80"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestWouldStandardInvoiceBeEmpty(t *testing.T) {
	lineWithoutDetails := &billing.StandardLine{}
	lineWithDetails := &billing.StandardLine{
		DetailedLines: billing.DetailedLines{{}},
	}

	tests := []struct {
		name  string
		lines []*billing.StandardLine
		total int64
		want  bool
	}{
		{
			name: "no lines",
			want: true,
		},
		{
			name:  "lines without anything to bill",
			lines: []*billing.StandardLine{lineWithoutDetails},
			want:  true,
		},
		{
			name:  "billable lines that add up to zero",
			lines: []*billing.StandardLine{lineWithDetails},
			want:  true,
		},
		{
			name:  "billable lines with an amount to pay",
			lines: []*billing.StandardLine{lineWithDetails},
			total: 10,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			empty, err := appOperations{}.WouldStandardInvoiceBeEmpty(billing.StandardInvoice{
				Lines:  billing.NewStandardInvoiceLines(tt.lines),
				Totals: totals.Totals{Total: alpacadecimal.NewFromInt(tt.total)},
			})

			require.NoError(t, err)
			require.Equal(t, tt.want, empty)
		})
	}
}

// The Stripe tax code is the parent line's billed code: the stamp, else the legacy stored
// code. An empty code leaves it unset so Stripe applies its default.
func TestLeafLinesCarryTheParentBilledTaxCode(t *testing.T) {
	// given: lines in every stamp state
	stamped := &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			InvoicingAppTaxCode: lo.ToPtr("txcd_stamped"),
			LegacyStripeTaxCode: "txcd_legacy",
		},
		DetailedLines: billing.DetailedLines{{}},
	}
	stampedDefault := &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			InvoicingAppTaxCode: lo.ToPtr(""),
			LegacyStripeTaxCode: "txcd_legacy",
		},
		DetailedLines: billing.DetailedLines{{}},
	}
	legacy := &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			LegacyStripeTaxCode: "txcd_legacy",
		},
		DetailedLines: billing.DetailedLines{{}},
	}
	unstamped := &billing.StandardLine{
		DetailedLines: billing.DetailedLines{{}},
	}

	invoice := billing.StandardInvoice{
		Lines: billing.NewStandardInvoiceLines([]*billing.StandardLine{stamped, stampedDefault, legacy, unstamped}),
	}

	// when: the leaf lines are collected for Stripe
	leafLines := invoice.GetLeafLinesWithResolvedTaxConfig(app.AppTypeStripe)

	// then: each carries its parent's billed code
	require.Len(t, leafLines, 4)
	require.Equal(t, "txcd_stamped", leafLines[0].BilledTaxCode)
	require.Equal(t, "", leafLines[1].BilledTaxCode)
	require.Equal(t, "txcd_legacy", leafLines[2].BilledTaxCode)
	require.Equal(t, "", leafLines[3].BilledTaxCode)
}

func TestApplyTaxSettingsToInvoiceItem(t *testing.T) {
	t.Run("sends the billed code and the behavior", func(t *testing.T) {
		// when: the tax settings of a line with a billed code and a behavior are applied to a Stripe invoice item
		params := applyTaxSettingsToInvoiceItem(&stripe.InvoiceItemParams{}, billing.DetailedLineWithResolvedTaxConfig{
			TaxConfig:     &productcatalog.TaxCodeConfig{Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior)},
			BilledTaxCode: "txcd_billed",
		})

		// then: both are sent to Stripe
		require.Equal(t, "txcd_billed", lo.FromPtr(params.TaxCode))
		require.Equal(t, string(stripe.PriceCurrencyOptionsTaxBehaviorExclusive), lo.FromPtr(params.TaxBehavior))
	})

	t.Run("an empty billed code leaves the stripe tax code unset", func(t *testing.T) {
		// when: the tax settings of a line with a Tax Code ID but an empty billed code are applied to a Stripe invoice item
		params := applyTaxSettingsToInvoiceItem(&stripe.InvoiceItemParams{}, billing.DetailedLineWithResolvedTaxConfig{
			TaxConfig: &productcatalog.TaxCodeConfig{TaxCodeID: "01JTAXCODE00000000000000000"},
		})

		// then: Stripe gets neither a tax code nor a behavior, so it applies its default
		require.Nil(t, params.TaxCode)
		require.Nil(t, params.TaxBehavior)
	})
}
