package stdinvoicesanity_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	stdinvoicesanity "github.com/openmeterio/openmeter/openmeter/billing/stdinvoice/sanity"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestValidateTotalArithmetic(t *testing.T) {
	t.Parallel()
	for _, level := range []string{"invoice", "parent", "detail"} {
		t.Run(level, func(t *testing.T) {
			t.Parallel()
			invoice := invoiceWithDiscountTotal("0")
			detail := &invoice.Lines.OrEmpty()[0].DetailedLines[0]
			detail.Totals.TaxesInclusiveTotal = alpacadecimal.NewFromInt(10)
			detail.Totals.TaxesExclusiveTotal = alpacadecimal.NewFromInt(5)
			detail.Totals.TaxesTotal = alpacadecimal.NewFromInt(15)
			setTestTotalsFromDetails(&invoice)
			require.NoError(t, stdinvoicesanity.Validate(invoice))
			require.Equal(t, float64(105), invoice.Totals.Total.InexactFloat64())

			path := "/totals/total"
			switch level {
			case "invoice":
				invoice.Totals.Total = alpacadecimal.NewFromInt(115)
			case "parent":
				invoice.Lines.OrEmpty()[0].Totals.Total = alpacadecimal.NewFromInt(115)
				path = "/lines/0/totals/total"
			case "detail":
				detail.Totals.Total = alpacadecimal.NewFromInt(115)
				path = "/lines/0/detailedLines/0/totals/total"
			}

			err := stdinvoicesanity.Validate(invoice)
			require.ErrorIs(t, err, stdinvoicesanity.ErrTotalArithmeticMismatch)
			issues := requireSanityIssues(t, err)
			issue, ok := lo.Find(issues, func(issue billing.ValidationIssue) bool {
				return issue.Code == stdinvoicesanity.ErrTotalArithmeticMismatch.Code && issue.Path == path
			})
			require.True(t, ok)
			require.Equal(t, "105", issue.Attributes["expected"])
			require.Equal(t, "115", issue.Attributes["actual"])
		})
	}
}

func TestValidateAggregationComponents(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		field    string
		change   func(*totals.Totals)
		expected string
	}{
		{field: "amount", change: func(value *totals.Totals) { value.Amount = alpacadecimal.NewFromInt(1) }, expected: "100"},
		{field: "chargesTotal", change: func(value *totals.Totals) { value.ChargesTotal = alpacadecimal.NewFromInt(1) }, expected: "0"},
		{field: "discountsTotal", change: func(value *totals.Totals) { value.DiscountsTotal = alpacadecimal.NewFromInt(1) }, expected: "0"},
		{field: "creditsTotal", change: func(value *totals.Totals) { value.CreditsTotal = alpacadecimal.NewFromInt(1) }, expected: "0"},
		{field: "taxesInclusiveTotal", change: func(value *totals.Totals) { value.TaxesInclusiveTotal = alpacadecimal.NewFromInt(1) }, expected: "0"},
		{field: "taxesExclusiveTotal", change: func(value *totals.Totals) { value.TaxesExclusiveTotal = alpacadecimal.NewFromInt(1) }, expected: "0"},
		{field: "taxesTotal", change: func(value *totals.Totals) { value.TaxesTotal = alpacadecimal.NewFromInt(1) }, expected: "0"},
		{field: "total", change: func(value *totals.Totals) { value.Total = alpacadecimal.NewFromInt(1) }, expected: "100"},
	} {
		t.Run(tc.field, func(t *testing.T) {
			t.Parallel()
			for _, level := range []string{"parent", "invoice"} {
				t.Run(level, func(t *testing.T) {
					t.Parallel()
					invoice := invoiceWithDiscountTotal("0")
					code := stdinvoicesanity.ErrInvoiceTotalsMismatch.Code
					path := "/totals/" + tc.field
					if level == "parent" {
						tc.change(&invoice.Lines.OrEmpty()[0].Totals)
						code = stdinvoicesanity.ErrLineTotalsMismatch.Code
						path = "/lines/0/totals/" + tc.field
					} else {
						tc.change(&invoice.Totals)
					}

					issues := requireSanityIssues(t, stdinvoicesanity.Validate(invoice))
					issue, ok := lo.Find(issues, func(issue billing.ValidationIssue) bool {
						return issue.Code == code && issue.Path == path
					})
					require.True(t, ok)
					require.Equal(t, tc.expected, issue.Attributes["expected"])
					require.Equal(t, "1", issue.Attributes["actual"])
				})
			}
		})
	}
}

func TestValidateAggregationDoesNotHideOffsettingErrors(t *testing.T) {
	t.Parallel()
	invoice := invoiceWithDiscountTotal("0")
	invoice.Totals.Amount = alpacadecimal.NewFromInt(110)
	invoice.Totals.DiscountsTotal = alpacadecimal.NewFromInt(10)
	issues := requireSanityIssues(t, stdinvoicesanity.Validate(invoice))
	require.Len(t, issues, 2)
	for _, issue := range issues {
		require.Equal(t, stdinvoicesanity.ErrInvoiceTotalsMismatch.Code, issue.Code)
	}

	require.Equal(t, "/totals/amount", issues[0].Path)
	require.Equal(t, "/totals/discountsTotal", issues[1].Path)
}

func TestValidateEmptyAndIncompleteAggregates(t *testing.T) {
	t.Parallel()
	invoice := invoiceWithDiscountTotal("0")
	invoice.Lines.OrEmpty()[0].DetailedLines = nil
	require.ErrorIs(t, stdinvoicesanity.Validate(invoice), stdinvoicesanity.ErrLineTotalsMismatch)
	invoice.Lines.OrEmpty()[0].Totals = totals.Totals{}
	invoice.Totals = totals.Totals{}
	require.NoError(t, stdinvoicesanity.Validate(invoice))

	invoice = invoiceWithDiscountTotal("0")
	invoice.Lines = billing.NewStandardInvoiceLines(nil)
	require.ErrorIs(t, stdinvoicesanity.Validate(invoice), stdinvoicesanity.ErrInvoiceTotalsMismatch)

	invoice.Lines = billing.StandardInvoiceLines{}
	issues, issueErr := billing.ToValidationIssues(stdinvoicesanity.Validate(invoice))
	require.NoError(t, issueErr)
	require.Len(t, issues, 1)
	require.Equal(t, stdinvoicesanity.ErrLinesNotExpanded.Code, issues[0].Code)
	invoice.Lines = billing.NewStandardInvoiceLines([]*billing.StandardLine{nil})
	issues, issueErr = billing.ToValidationIssues(stdinvoicesanity.Validate(invoice))
	require.NoError(t, issueErr)
	require.Len(t, issues, 1)
	require.Equal(t, stdinvoicesanity.ErrNilLine.Code, issues[0].Code)
}

func TestValidateMixedCalculatedDetailsAndDeletedFacts(t *testing.T) {
	t.Parallel()
	invoice := invoiceWithDiscountTotal("0")
	line := invoice.Lines.OrEmpty()[0]
	regular := &line.DetailedLines[0]
	regular.AmountDiscounts = billing.AmountLineDiscountsManaged{{ID: "discount-1", AmountLineDiscount: billing.AmountLineDiscount{
		Amount:         alpacadecimal.NewFromInt(10),
		RoundingAmount: alpacadecimal.RequireFromString("-0.01"),
	}}}
	regular.Totals.DiscountsTotal = alpacadecimal.RequireFromString("9.99")
	regular.CreditsApplied = billing.CreditsApplied{{Amount: alpacadecimal.NewFromInt(5), CreditRealizationID: "shared"}}
	regular.Totals.CreditsTotal = alpacadecimal.NewFromInt(5)

	commitment := regular.Clone()
	commitment.ID = "commitment"
	commitment.ChildUniqueReferenceID = "commitment"
	commitment.Category = stddetailedline.CategoryCommitment
	commitment.PerUnitAmount = alpacadecimal.NewFromInt(20)
	commitment.AmountDiscounts = billing.AmountLineDiscountsManaged{}
	commitment.Totals = totals.Totals{ChargesTotal: alpacadecimal.NewFromInt(20), CreditsTotal: alpacadecimal.NewFromInt(5)}
	correction := commitment.Clone()
	correction.ID = "correction"
	correction.ChildUniqueReferenceID = "correction"
	correction.Quantity = alpacadecimal.NewFromInt(-1)
	correction.CreditsApplied = nil
	correction.Totals = totals.Totals{ChargesTotal: alpacadecimal.NewFromInt(-20)}
	line.DetailedLines = append(line.DetailedLines, commitment, correction)
	line.CreditsApplied = billing.CreditsApplied{{Amount: alpacadecimal.NewFromInt(10), CreditRealizationID: "shared"}}
	otherLine, err := line.Clone()
	require.NoError(t, err)
	otherLine.ID = "line-2"
	for idx := range otherLine.DetailedLines {
		otherLine.DetailedLines[idx].ID += "-other"
		for j := range otherLine.DetailedLines[idx].AmountDiscounts {
			otherLine.DetailedLines[idx].AmountDiscounts[j].ID += "-other"
		}
	}

	invoice.Lines = billing.NewStandardInvoiceLines([]*billing.StandardLine{line, otherLine})
	setTestTotalsFromDetails(&invoice)
	require.NoError(t, stdinvoicesanity.Validate(invoice))
	require.Equal(t, 160.02, invoice.Totals.Total.InexactFloat64())

	// Give deleted lines wrong amounts and breakdowns to prove they are skipped.
	deleted := billing.DetailedLine{AmountDiscounts: billing.AmountLineDiscountsManaged{}}
	deleted.DeletedAt = lo.ToPtr(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	deleted.Totals = totals.Totals{Amount: alpacadecimal.NewFromInt(123), CreditsTotal: alpacadecimal.NewFromInt(45)}
	line.DetailedLines = append(line.DetailedLines, deleted)
	deletedParent, err := line.Clone()
	require.NoError(t, err)
	deletedParent.DeletedAt = deleted.DeletedAt
	deletedParent.Totals = deleted.Totals
	invoice.Lines.Append(deletedParent)
	before, err := invoice.Clone()
	require.NoError(t, err)
	require.NoError(t, stdinvoicesanity.Validate(invoice))
	require.Equal(t, before, invoice)
}

// setTestTotalsFromDetails updates the totals after a test changes a detail,
// so unrelated total mismatches do not hide what the test is checking.
func setTestTotalsFromDetails(invoice *billing.StandardInvoice) {
	invoice.Totals = totals.Totals{}
	for _, line := range invoice.Lines.OrEmpty() {
		if line == nil || line.DeletedAt != nil {
			continue
		}

		line.Totals = totals.Totals{}
		for idx := range line.DetailedLines {
			detail := &line.DetailedLines[idx]
			if detail.DeletedAt != nil {
				continue
			}

			detail.Totals.Total = detail.Totals.CalculateTotal()
			line.Totals = line.Totals.Add(detail.Totals)
		}

		invoice.Totals = invoice.Totals.Add(line.Totals)
	}
}

func requireSanityIssues(t *testing.T, err error) billing.ValidationIssues {
	t.Helper()
	require.Error(t, err)
	require.True(t, models.IsGenericValidationError(err))
	issues, issueErr := billing.ToValidationIssues(err)
	require.NoError(t, issueErr)
	require.NotEmpty(t, issues)
	for _, issue := range issues {
		require.Equal(t, stdinvoicesanity.ValidationComponent, issue.Component)
		require.Equal(t, billing.ValidationIssueSeverityCritical, issue.Severity)
		require.NotEmpty(t, issue.Code)
		require.NotEmpty(t, issue.Message)
		require.NotEmpty(t, issue.Path)
		require.Equal(t, "invoice-1", issue.Attributes["invoice_id"])
	}

	return issues
}
