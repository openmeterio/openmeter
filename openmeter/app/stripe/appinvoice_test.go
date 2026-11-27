package appstripe

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v80"
	"github.com/stripe/stripe-go/v80/form"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
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

type invoiceItemKind string

const (
	invoiceItemKindLine     invoiceItemKind = "line"
	invoiceItemKindDiscount invoiceItemKind = "discount"
	invoiceItemKindCredit   invoiceItemKind = "credit"
)

// addAndUpdateParams returns the Stripe add and update params the sync builds for one kind of invoice item.
func addAndUpdateParams(t *testing.T, kind invoiceItemKind, billedTaxCode string) (*stripe.InvoiceItemParams, *stripe.InvoiceItemParams) {
	t.Helper()

	calculator, err := NewStripeCalculator(currencyx.FiatCode("USD"))
	require.NoError(t, err)

	line := billing.DetailedLineWithResolvedTaxConfig{
		DetailedLine:  billing.DetailedLine{},
		BilledTaxCode: billedTaxCode,
	}
	stripeLine := &stripe.InvoiceLineItem{ID: "il_1"}

	switch kind {
	case invoiceItemKindDiscount:
		discount := billing.AmountLineDiscountManaged{}

		return getDiscountStripeAddInvoiceItemParams(calculator, line, discount, "cus_1"),
			getDiscountStripeUpdateInvoiceItemParams(calculator, line, discount, stripeLine).InvoiceItemParams
	case invoiceItemKindCredit:
		credit := billing.CreditApplied{CreditRealizationID: "cr_1"}

		return getCreditStripeAddInvoiceItemParams(calculator, line, credit, "cus_1"),
			getCreditStripeUpdateInvoiceItemParams(calculator, line, credit, stripeLine).InvoiceItemParams
	default:
		return getStripeAddInvoiceItemParams(line, calculator, "cus_1"),
			getStripeUpdateInvoiceItemParams(calculator, line, stripeLine).InvoiceItemParams
	}
}

// Stripe keeps the old tax_code of an invoice item when an update leaves the field out, so an
// update has to send an empty value to drop a code the item was synced with before.
func TestUpdateInvoiceItemParamsClearAnEmptyTaxCode(t *testing.T) {
	for _, kind := range []invoiceItemKind{invoiceItemKindLine, invoiceItemKindDiscount, invoiceItemKindCredit} {
		t.Run(string(kind)+" with an empty billed code clears the code on update only", func(t *testing.T) {
			// given: an item whose billed tax code is empty
			// when: the add and update params are built
			add, update := addAndUpdateParams(t, kind, "")

			// then: the add leaves the code out, the update sets it to an empty value
			require.Nil(t, add.TaxCode)
			require.NotNil(t, update.TaxCode)
			require.Equal(t, "", *update.TaxCode)

			// then: the update goes out as an empty tax_code form field
			values := &form.Values{}
			form.AppendTo(values, update)
			require.Equal(t, []string{""}, values.Get("tax_code"))

			// then: the add does not send tax_code at all
			addValues := &form.Values{}
			form.AppendTo(addValues, add)
			require.Empty(t, addValues.Get("tax_code"))
		})

		t.Run(string(kind)+" with a billed code sends it on add and update", func(t *testing.T) {
			// given: an item with a billed tax code
			// when: the add and update params are built
			add, update := addAndUpdateParams(t, kind, "txcd_10000000")

			// then: both carry the code
			require.Equal(t, "txcd_10000000", lo.FromPtr(add.TaxCode))
			require.Equal(t, "txcd_10000000", lo.FromPtr(update.TaxCode))

			// then: the update sends it on the wire
			values := &form.Values{}
			form.AppendTo(values, update)
			require.Equal(t, []string{"txcd_10000000"}, values.Get("tax_code"))
		})
	}
}
