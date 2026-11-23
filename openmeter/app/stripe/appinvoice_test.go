package appstripe

import (
	"context"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
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

// fakeTaxCodeResolver serves BatchResolve from taxCodes and records whether it was called.
type fakeTaxCodeResolver struct {
	productcatalog.TaxCodeResolver

	taxCodes map[string]*taxcode.TaxCode
	calls    int
}

func (r *fakeTaxCodeResolver) BatchResolve(_ context.Context, _ string, ids ...string) (productcatalog.TaxCodesByID, error) {
	r.calls++

	return lo.PickByKeys(r.taxCodes, ids), nil
}

// A leaf line's TaxCodeID must resolve; a miss means corrupt data and fails the sync instead
// of sending an invoice item without a tax code.
func TestResolveLinesTaxCodes(t *testing.T) {
	const namespace = "ns"
	existingID := "01JTAXCODE00000000000000000"

	t.Run("missing tax code fails with not found", func(t *testing.T) {
		resolver := &fakeTaxCodeResolver{taxCodes: map[string]*taxcode.TaxCode{}}

		_, err := appOperations{TaxCodeResolver: resolver}.resolveLinesTaxCodes(t.Context(), namespace, []billing.DetailedLineWithResolvedTaxConfig{
			{
				DetailedLine: billing.DetailedLine{},
				TaxConfig:    billing.FromProductCatalog(&productcatalog.TaxConfig{TaxCodeID: lo.ToPtr(existingID)}),
			},
		})

		require.Error(t, err)
		require.ErrorIs(t, err, taxcode.ErrTaxCodeNotFound)
		require.ErrorContains(t, err, existingID)
	})

	t.Run("resolves every referenced id in one batch", func(t *testing.T) {
		resolver := &fakeTaxCodeResolver{taxCodes: map[string]*taxcode.TaxCode{
			existingID: {NamespacedID: models.NamespacedID{Namespace: namespace, ID: existingID}},
		}}

		taxCodes, err := appOperations{TaxCodeResolver: resolver}.resolveLinesTaxCodes(t.Context(), namespace, []billing.DetailedLineWithResolvedTaxConfig{
			{
				DetailedLine: billing.DetailedLine{},
				TaxConfig:    billing.FromProductCatalog(&productcatalog.TaxConfig{TaxCodeID: lo.ToPtr(existingID)}),
			},
		})

		require.NoError(t, err)
		require.Len(t, taxCodes, 1)
		require.Equal(t, existingID, taxCodes[existingID].ID)
		require.Equal(t, 1, resolver.calls)
	})

	t.Run("lines without a tax code id skip the lookup", func(t *testing.T) {
		resolver := &fakeTaxCodeResolver{}

		taxCodes, err := appOperations{TaxCodeResolver: resolver}.resolveLinesTaxCodes(t.Context(), namespace, []billing.DetailedLineWithResolvedTaxConfig{
			{DetailedLine: billing.DetailedLine{}},
			{
				DetailedLine: billing.DetailedLine{},
				TaxConfig:    billing.FromProductCatalog(&productcatalog.TaxConfig{Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior)}),
			},
		})

		require.NoError(t, err)
		require.Empty(t, taxCodes)
		require.Zero(t, resolver.calls)
	})
}
