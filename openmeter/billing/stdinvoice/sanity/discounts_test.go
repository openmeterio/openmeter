package stdinvoicesanity_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	stdinvoicesanity "github.com/openmeterio/openmeter/openmeter/billing/stdinvoice/sanity"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestValidateDiscountTotals(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		currency currencyx.FiatCode
		total    string
		amounts  []string
		rounding []string
		mismatch bool
	}{
		{name: "no discounts", currency: "USD", total: "0"},
		{name: "matching entries", currency: "USD", total: "20", amounts: []string{"12", "8"}},
		{name: "missing breakdown", currency: "USD", total: "20", mismatch: true},
		{name: "partial breakdown", currency: "USD", total: "20", amounts: []string{"12"}, mismatch: true},
		{name: "stale cumulative breakdown", currency: "USD", total: "10", amounts: []string{"20"}, mismatch: true},
		{name: "unexpected discount", currency: "USD", total: "0", amounts: []string{"20"}, mismatch: true},
		{name: "positive rounding correction", currency: "USD", total: "20.01", amounts: []string{"10", "10"}, rounding: []string{"0.01", "0"}},
		{name: "negative rounding correction", currency: "USD", total: "19.99", amounts: []string{"10", "10"}, rounding: []string{"-0.01", "0"}},
		{name: "round each entry", currency: "USD", total: "0.02", amounts: []string{"0.005", "0.005"}},
		{name: "no tolerance", currency: "USD", total: "20.01", amounts: []string{"20"}, mismatch: true},
		{name: "unrounded stored total", currency: "USD", total: "20.001", amounts: []string{"20"}, mismatch: true},
		{name: "zero decimal currency", currency: "JPY", total: "2", amounts: []string{"0.5", "0.5"}},
		{name: "three decimal currency", currency: "KWD", total: "0.002", amounts: []string{"0.0005", "0.0005"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given a stored discount total and a separately supplied breakdown.
			invoice := invoiceWithDiscountTotal(tc.total)
			invoice.Currency = tc.currency
			invoice.Lines.OrEmpty()[0].Currency = tc.currency
			detail := &invoice.Lines.OrEmpty()[0].DetailedLines[0]
			for idx, amount := range tc.amounts {
				discount := billing.AmountLineDiscountManaged{
					ID: fmt.Sprintf("discount-%d", idx),
					AmountLineDiscount: billing.AmountLineDiscount{
						Amount: alpacadecimal.RequireFromString(amount),
					},
				}
				if len(tc.rounding) > 0 {
					discount.RoundingAmount = alpacadecimal.RequireFromString(tc.rounding[idx])
				}

				detail.AmountDiscounts = append(detail.AmountDiscounts, discount)
			}

			before, err := invoice.Clone()
			require.NoError(t, err)

			// When the invoice is checked without recalculating it.
			err = stdinvoicesanity.Validate(invoice)

			// Then a mismatch produces a critical issue without changing the invoice.
			require.Equal(t, before, invoice)
			if !tc.mismatch {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			require.True(t, models.IsGenericValidationError(err))
			issues, issueErr := billing.ToValidationIssues(err)
			require.NoError(t, issueErr)
			issues = lo.Filter(issues, func(issue billing.ValidationIssue, _ int) bool {
				return issue.Code == stdinvoicesanity.ErrDiscountTotalMismatch.Code
			})
			require.Len(t, issues, 1)
			require.ErrorIs(t, err, stdinvoicesanity.ErrDiscountTotalMismatch)
			require.Equal(t, stdinvoicesanity.ErrDiscountTotalMismatch.Message, issues[0].Message)
			require.Equal(t, stdinvoicesanity.ErrDiscountTotalMismatch.Code, issues[0].Code)
			require.Equal(t, stdinvoicesanity.ValidationComponent, issues[0].Component)
			require.Equal(t, billing.ValidationIssueSeverityCritical, issues[0].Severity)
			require.Equal(t, "/lines/0/detailedLines/0/totals/discountsTotal", issues[0].Path)
			require.Equal(t, "invoice-1", issues[0].Attributes["invoice_id"])
			require.Equal(t, "line-1", issues[0].Attributes["line_id"])
			require.Equal(t, "detail-1", issues[0].Attributes["detailed_line_id"])
			require.Equal(t, tc.total, issues[0].Attributes["actual"])
			require.NotEmpty(t, issues[0].Attributes["expected"])
		})
	}
}

func TestValidateInvalidCurrency(t *testing.T) {
	t.Parallel()

	invoice := invoiceWithDiscountTotal("0")
	invoice.Currency = "INVALID"
	err := stdinvoicesanity.Validate(invoice)
	require.Error(t, err)
	issues, issueErr := billing.ToValidationIssues(err)
	require.NoError(t, issueErr)
	require.Len(t, issues, 1)
	require.ErrorIs(t, err, stdinvoicesanity.ErrInvalidCurrency)
	require.Equal(t, stdinvoicesanity.ErrInvalidCurrency.Message, issues[0].Message)
	require.Equal(t, "INVALID", issues[0].Attributes["currency"])
	require.NotEmpty(t, issues[0].Attributes["error"])
	require.Equal(t, stdinvoicesanity.ErrInvalidCurrency.Code, issues[0].Code)
	require.Equal(t, "/currency", issues[0].Path)
}

func TestValidateIgnoresDeletedDiscountFacts(t *testing.T) {
	t.Parallel()

	deletedAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	t.Run("deleted parent", func(t *testing.T) {
		t.Parallel()

		invoice := invoiceWithDiscountTotal("20")
		invoice.Lines.OrEmpty()[0].DeletedAt = lo.ToPtr(deletedAt)
		invoice.Period = nil
		invoice.Totals = totals.Totals{}
		require.NoError(t, stdinvoicesanity.Validate(invoice))
	})
	t.Run("deleted detail", func(t *testing.T) {
		t.Parallel()

		invoice := invoiceWithDiscountTotal("20")
		invoice.Lines.OrEmpty()[0].DetailedLines[0].DeletedAt = lo.ToPtr(deletedAt)
		invoice.Lines.OrEmpty()[0].Totals = totals.Totals{}
		invoice.Totals = totals.Totals{}
		require.NoError(t, stdinvoicesanity.Validate(invoice))
	})
	t.Run("deleted discount", func(t *testing.T) {
		t.Parallel()

		invoice := invoiceWithDiscountTotal("0")
		invoice.Lines.OrEmpty()[0].DetailedLines[0].AmountDiscounts = billing.AmountLineDiscountsManaged{{
			ManagedModelWithID: models.ManagedModelWithID{
				ManagedModel: models.ManagedModel{DeletedAt: lo.ToPtr(deletedAt)},
			},
			AmountLineDiscount: billing.AmountLineDiscount{Amount: alpacadecimal.NewFromInt(20)},
		}}
		require.NoError(t, stdinvoicesanity.Validate(invoice))
		invoice.Lines.OrEmpty()[0].DetailedLines[0].Totals.DiscountsTotal = alpacadecimal.NewFromInt(20)
		require.Error(t, stdinvoicesanity.Validate(invoice))
	})
}

// invoiceWithDiscountTotal recreates the missing-discount case: the totals add
// up, but the detail has no discount entries to explain its discount total.
func invoiceWithDiscountTotal(discountTotal string) billing.StandardInvoice {
	detail := billing.DetailedLine{AmountDiscounts: billing.AmountLineDiscountsManaged{}}
	detail.ID = "detail-1"
	detail.Namespace = "test"
	detail.InvoiceID = "invoice-1"
	detail.ChildUniqueReferenceID = "detail-1"
	detail.ServicePeriod = timeutil.ClosedPeriod{From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)}
	detail.Category = stddetailedline.CategoryRegular
	detail.PerUnitAmount = alpacadecimal.NewFromInt(100)
	detail.Quantity = alpacadecimal.NewFromInt(1)
	detail.Totals.Amount = alpacadecimal.NewFromInt(100)
	detail.Totals.DiscountsTotal = alpacadecimal.RequireFromString(discountTotal)
	detail.Totals.Total = detail.Totals.CalculateTotal()
	line := &billing.StandardLine{
		UsageBased:    &billing.UsageBasedLine{Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{Amount: alpacadecimal.NewFromInt(100)})},
		DetailedLines: billing.DetailedLines{detail},
	}
	line.ID = "line-1"
	line.Namespace = "test"
	line.InvoiceID = "invoice-1"
	line.Currency = "USD"
	line.Period = detail.ServicePeriod
	line.Totals = detail.Totals
	invoice := billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{Currency: "USD"},
		Lines:               billing.NewStandardInvoiceLines([]*billing.StandardLine{line}),
		Totals:              line.Totals,
	}
	invoice.ID = "invoice-1"
	invoice.Namespace = "test"
	invoice.Period = lo.ToPtr(line.Period)

	return invoice
}

func TestValidateDiscountSigns(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		amount   string
		discount string
		mismatch bool
	}{
		{name: "positive line with discount", amount: "100", discount: "20"},
		{name: "reversed line with reversed discount", amount: "-100", discount: "-20"},
		{name: "zero discount on reversed line", amount: "-100", discount: "0"},
		{name: "discount correction without usage", amount: "0", discount: "20"},
		{name: "reversed discount correction without usage", amount: "0", discount: "-20"},
		{name: "positive line with reversed discount", amount: "100", discount: "-20", mismatch: true},
		{name: "reversed line with positive discount", amount: "-100", discount: "20", mismatch: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Given a detailed line whose discount total reconciles with its single discount entry,
			// next to a regular detail that keeps the parent total payable.
			invoice := invoiceWithDiscountTotal(tc.discount)
			line := invoice.Lines.OrEmpty()[0]
			line.DetailedLines[0].Totals.Amount = alpacadecimal.RequireFromString(tc.amount)
			line.DetailedLines[0].AmountDiscounts = billing.AmountLineDiscountsManaged{{
				ID: "discount-1",
				AmountLineDiscount: billing.AmountLineDiscount{
					Amount: alpacadecimal.RequireFromString(tc.discount),
				},
			}}
			regular := line.DetailedLines[0].Clone()
			regular.ID = "detail-2"
			regular.ChildUniqueReferenceID = "detail-2"
			regular.AmountDiscounts = billing.AmountLineDiscountsManaged{}
			regular.Totals = totals.Totals{Amount: alpacadecimal.NewFromInt(1000)}
			line.DetailedLines = append(line.DetailedLines, regular)
			setTestTotalsFromDetails(&invoice)

			// When the invoice is checked without recalculating it.
			err := stdinvoicesanity.Validate(invoice)

			// Then only discounts working against the line amount are reported.
			if !tc.mismatch {
				require.NoError(t, err)
				return
			}

			issues := requireSanityIssues(t, err)
			require.Len(t, issues, 1)
			require.ErrorIs(t, err, stdinvoicesanity.ErrDiscountSignMismatch)
			require.Equal(t, stdinvoicesanity.ErrDiscountSignMismatch.Code, issues[0].Code)
			require.Equal(t, "/lines/0/detailedLines/0/amountDiscounts/0/amount", issues[0].Path)
			require.Equal(t, tc.amount, issues[0].Attributes["line_amount"])
			require.Equal(t, tc.discount, issues[0].Attributes["discount_amount"])
		})
	}
}
