package stdinvoicesanity_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	stdinvoicesanity "github.com/openmeterio/openmeter/openmeter/billing/stdinvoice/sanity"
)

func TestValidateExpandedLines(t *testing.T) {
	t.Parallel()

	invoice := billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{ID: "invoice-1", Namespace: "test", Currency: "USD"},
	}
	err := stdinvoicesanity.Validate(invoice)
	require.Error(t, err)
	issues, issueErr := billing.ToValidationIssues(err)
	require.NoError(t, issueErr)
	require.Len(t, issues, 1)
	require.Equal(t, stdinvoicesanity.ErrLinesNotExpanded.Code, issues[0].Code)
	require.Equal(t, "/lines", issues[0].Path)

	invoice.Lines = billing.NewStandardInvoiceLines(nil)
	require.NoError(t, stdinvoicesanity.Validate(invoice))
}

func TestValidateCollectsIndependentFailures(t *testing.T) {
	t.Parallel()

	// Given a malformed parent alongside two inconsistent detailed lines.
	invoice := invoiceWithDiscountTotal("20")
	line := invoice.Lines.OrEmpty()[0]
	line.DetailedLines = append(line.DetailedLines, line.DetailedLines[0].Clone())
	line.DetailedLines[1].ID = "detail-2"
	line.DetailedLines[1].ChildUniqueReferenceID = "detail-2"
	setTestTotalsFromDetails(&invoice)
	invoice.Lines = billing.NewStandardInvoiceLines([]*billing.StandardLine{nil, line})

	// When we validate the invoice.
	err := stdinvoicesanity.Validate(invoice)

	// Then the nil parent does not hide the errors in the other lines.
	require.Error(t, err)
	issues, issueErr := billing.ToValidationIssues(err)
	require.NoError(t, issueErr)
	require.Len(t, issues, 3)
	require.Equal(t, stdinvoicesanity.ErrNilLine.Code, issues[0].Code)
	require.Equal(t, "/lines/0", issues[0].Path)
	require.Equal(t, stdinvoicesanity.ErrDiscountTotalMismatch.Code, issues[1].Code)
	require.Equal(t, "/lines/1/detailedLines/0/totals/discountsTotal", issues[1].Path)
	require.Equal(t, stdinvoicesanity.ErrDiscountTotalMismatch.Code, issues[2].Code)
	require.Equal(t, "/lines/1/detailedLines/1/totals/discountsTotal", issues[2].Path)
	require.Equal(t, "0", issues[1].Attributes["expected"])
	require.Equal(t, "20", issues[1].Attributes["actual"])
}
