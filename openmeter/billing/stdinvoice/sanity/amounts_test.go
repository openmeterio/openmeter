package stdinvoicesanity_test

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	stdinvoicesanity "github.com/openmeterio/openmeter/openmeter/billing/stdinvoice/sanity"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestValidateGrossAmounts(t *testing.T) {
	t.Parallel()
	for _, category := range []stddetailedline.Category{stddetailedline.CategoryRegular, stddetailedline.CategoryCommitment} {
		t.Run(string(category), func(t *testing.T) {
			t.Parallel()
			for _, tc := range []struct {
				name     string
				currency currencyx.FiatCode
				unit     string
				quantity string
				gross    string
				expected string
			}{
				{name: "zero", currency: "USD", unit: "0", quantity: "0", gross: "0"},
				{name: "small unit price large quantity", currency: "USD", unit: "0.00000075", quantity: "1000000", gross: "0.75"},
				{name: "fractional quantity", currency: "USD", unit: "0.01", quantity: "1.5", gross: "0.02"},
				{name: "signed correction", currency: "USD", unit: "0.01", quantity: "-1.5", gross: "-0.02"},
				{name: "zero decimal currency", currency: "JPY", unit: "0.5", quantity: "1", gross: "1"},
				{name: "three decimal currency", currency: "KWD", unit: "0.0005", quantity: "1", gross: "0.001"},
				{name: "wrong amount", currency: "USD", unit: "10", quantity: "2", gross: "19", expected: "20"},
				{name: "no tolerance", currency: "USD", unit: "10", quantity: "2", gross: "20.001", expected: "20"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					t.Parallel()
					invoice := invoiceWithDiscountTotal("0")
					invoice.Currency = tc.currency
					invoice.Lines.OrEmpty()[0].Currency = tc.currency
					detail := &invoice.Lines.OrEmpty()[0].DetailedLines[0]
					detail.Category = category
					detail.PerUnitAmount = alpacadecimal.RequireFromString(tc.unit)
					detail.Quantity = alpacadecimal.RequireFromString(tc.quantity)
					field := "amount"
					detail.Totals.Amount = alpacadecimal.RequireFromString(tc.gross)
					if category == stddetailedline.CategoryCommitment {
						field = "chargesTotal"
						detail.Totals.ChargesTotal = detail.Totals.Amount
						detail.Totals.Amount = alpacadecimal.Zero
					}

					if detail.Quantity.IsNegative() {
						balanced := detail.Clone()
						balanced.ID = "balance"
						balanced.ChildUniqueReferenceID = "balance"
						balanced.Quantity = balanced.Quantity.Neg()
						balanced.Totals.Amount = balanced.Totals.Amount.Neg()
						balanced.Totals.ChargesTotal = balanced.Totals.ChargesTotal.Neg()
						invoice.Lines.OrEmpty()[0].DetailedLines = append(invoice.Lines.OrEmpty()[0].DetailedLines, balanced)
					}

					setTestTotalsFromDetails(&invoice)
					before, err := invoice.Clone()
					require.NoError(t, err)

					err = stdinvoicesanity.Validate(invoice)
					require.Equal(t, before, invoice)
					if tc.expected == "" {
						require.NoError(t, err)
						return
					}

					issues := requireSanityIssues(t, err)
					issues = lo.Filter(issues, func(issue billing.ValidationIssue, _ int) bool {
						return issue.Code == stdinvoicesanity.ErrGrossAmountMismatch.Code
					})
					require.Len(t, issues, 1)
					require.ErrorIs(t, err, stdinvoicesanity.ErrGrossAmountMismatch)
					require.Equal(t, "/lines/0/detailedLines/0/totals/"+field, issues[0].Path)
					require.Equal(t, tc.expected, issues[0].Attributes["expected"])
					require.Equal(t, tc.gross, issues[0].Attributes["actual"])
				})
			}
		})
	}
}

func TestValidateCommitmentPlacement(t *testing.T) {
	t.Parallel()
	for _, category := range []stddetailedline.Category{stddetailedline.CategoryRegular, stddetailedline.CategoryCommitment, "", "unknown"} {
		t.Run(string(category), func(t *testing.T) {
			t.Parallel()
			invoice := invoiceWithDiscountTotal("0")
			detail := &invoice.Lines.OrEmpty()[0].DetailedLines[0]
			detail.Category = category
			field := "chargesTotal"
			detail.Totals.ChargesTotal = alpacadecimal.NewFromInt(1)
			if category == stddetailedline.CategoryCommitment {
				field = "amount"
				detail.Totals.ChargesTotal = alpacadecimal.NewFromInt(100)
				detail.Totals.Amount = alpacadecimal.NewFromInt(-1)
			}

			setTestTotalsFromDetails(&invoice)

			err := stdinvoicesanity.Validate(invoice)
			issues := requireSanityIssues(t, err)
			require.Len(t, issues, 1)
			if category == "" || category == "unknown" {
				require.ErrorIs(t, err, stdinvoicesanity.ErrInvalidDetailCategory)
				require.Equal(t, "/lines/0/detailedLines/0/category", issues[0].Path)

				return
			}

			require.ErrorIs(t, err, stdinvoicesanity.ErrCommitmentPlacementMismatch)
			require.Equal(t, "/lines/0/detailedLines/0/totals/"+field, issues[0].Path)
			require.Equal(t, "0", issues[0].Attributes["expected"])
			require.Equal(t, string(category), issues[0].Attributes["category"])
		})
	}
}
