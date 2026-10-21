package stdinvoicesanity_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	stdinvoicesanity "github.com/openmeterio/openmeter/openmeter/billing/stdinvoice/sanity"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestValidatePersistedStructure(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		change func(*billing.StandardInvoice)
		issue  billing.ValidationIssue
		path   string
	}{
		{"invoice ID", func(i *billing.StandardInvoice) { i.ID = "" }, stdinvoicesanity.ErrMissingIdentity, "/id"},
		{"namespace", func(i *billing.StandardInvoice) { i.Namespace = "" }, stdinvoicesanity.ErrMissingIdentity, "/namespace"},
		{"parent ID", func(i *billing.StandardInvoice) { i.Lines.OrEmpty()[0].ID = "" }, stdinvoicesanity.ErrMissingIdentity, "/lines/0/id"},
		{"detail ID", func(i *billing.StandardInvoice) { i.Lines.OrEmpty()[0].DetailedLines[0].ID = "" }, stdinvoicesanity.ErrMissingIdentity, "/lines/0/detailedLines/0/id"},
		{"child reference", func(i *billing.StandardInvoice) { i.Lines.OrEmpty()[0].DetailedLines[0].ChildUniqueReferenceID = "" }, stdinvoicesanity.ErrMissingIdentity, "/lines/0/detailedLines/0/childUniqueReferenceID"},
		{"parent owner", func(i *billing.StandardInvoice) { i.Lines.OrEmpty()[0].InvoiceID = "another" }, stdinvoicesanity.ErrOwnershipMismatch, "/lines/0/invoiceID"},
		{"parent tenant", func(i *billing.StandardInvoice) { i.Lines.OrEmpty()[0].Namespace = "another" }, stdinvoicesanity.ErrOwnershipMismatch, "/lines/0/namespace"},
		{"detail owner", func(i *billing.StandardInvoice) { i.Lines.OrEmpty()[0].DetailedLines[0].InvoiceID = "another" }, stdinvoicesanity.ErrOwnershipMismatch, "/lines/0/detailedLines/0/invoiceID"},
		{"detail tenant", func(i *billing.StandardInvoice) { i.Lines.OrEmpty()[0].DetailedLines[0].Namespace = "another" }, stdinvoicesanity.ErrOwnershipMismatch, "/lines/0/detailedLines/0/namespace"},
		{"currency", func(i *billing.StandardInvoice) { i.Lines.OrEmpty()[0].Currency = "JPY" }, stdinvoicesanity.ErrCurrencyMismatch, "/lines/0/currency"},
		{"usage based", func(i *billing.StandardInvoice) { i.Lines.OrEmpty()[0].UsageBased = nil }, stdinvoicesanity.ErrMissingUsageBased, "/lines/0/usageBased"},
		{"price", func(i *billing.StandardInvoice) { i.Lines.OrEmpty()[0].UsageBased.Price = nil }, stdinvoicesanity.ErrMissingPrice, "/lines/0/usageBased/price"},
		{"parent total", func(i *billing.StandardInvoice) { i.Lines.OrEmpty()[0].Totals.Total = alpacadecimal.NewFromInt(-1) }, stdinvoicesanity.ErrNegativeParentTotal, "/lines/0/totals/total"},
		{"parent period", func(i *billing.StandardInvoice) {
			i.Lines.OrEmpty()[0].Period.From = i.Lines.OrEmpty()[0].Period.To.Add(time.Hour)
		}, stdinvoicesanity.ErrInvalidServicePeriod, "/lines/0/period"},
		{"detail period", func(i *billing.StandardInvoice) {
			d := &i.Lines.OrEmpty()[0].DetailedLines[0]
			d.ServicePeriod.From = d.ServicePeriod.To.Add(time.Hour)
		}, stdinvoicesanity.ErrInvalidServicePeriod, "/lines/0/detailedLines/0/servicePeriod"},
		{"missing invoice period", func(i *billing.StandardInvoice) { i.Period = nil }, stdinvoicesanity.ErrInvoicePeriodMismatch, "/period"},
		{"stale invoice period", func(i *billing.StandardInvoice) { i.Period.To = i.Period.To.Add(time.Hour) }, stdinvoicesanity.ErrInvoicePeriodMismatch, "/period"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Given a valid invoice with one broken field.
			invoice := invoiceWithDiscountTotal("0")
			before, err := invoice.Clone()
			require.NoError(t, err)
			tc.change(&invoice)
			tc.change(&before)
			// When we validate the invoice.
			err = stdinvoicesanity.Validate(invoice)
			require.ErrorIs(t, err, tc.issue)
			issues, issueErr := billing.ToValidationIssues(err)
			require.NoError(t, issueErr)
			// Then the issue names the broken field and the invoice stays unchanged.
			require.Contains(t, lo.Map(issues, func(issue billing.ValidationIssue, _ int) string { return issue.Path }), tc.path)
			require.Equal(t, before, invoice)
		})
	}
}

func TestValidateIdentityScopes(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"parent", "detail", "discount", "child reference"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			invoice := invoiceWithDiscountTotal("0")
			line := invoice.Lines.OrEmpty()[0]
			switch kind {
			case "parent":
				other, err := line.Clone()
				require.NoError(t, err)
				other.DetailedLines = nil
				other.Totals = totals.Totals{}
				invoice.Lines.Append(other)
			case "detail", "child reference":
				other := line.DetailedLines[0].Clone()
				if kind == "detail" {
					other.ChildUniqueReferenceID = "other"
				} else {
					other.ID = "other"
				}

				line.DetailedLines = append(line.DetailedLines, other)
			case "discount":
				line.DetailedLines[0].AmountDiscounts = billing.AmountLineDiscountsManaged{{ID: "same"}, {ID: "same"}}
			}

			setTestTotalsFromDetails(&invoice)
			require.ErrorIs(t, stdinvoicesanity.Validate(invoice), stdinvoicesanity.ErrDuplicateIdentity)
		})
	}

	t.Run("empty discount ID", func(t *testing.T) {
		t.Parallel()
		invoice := invoiceWithDiscountTotal("0")
		invoice.Lines.OrEmpty()[0].DetailedLines[0].AmountDiscounts = billing.AmountLineDiscountsManaged{{}}
		require.ErrorIs(t, stdinvoicesanity.Validate(invoice), stdinvoicesanity.ErrMissingIdentity)
	})
	t.Run("separate entity kinds and parents", func(t *testing.T) {
		t.Parallel()
		invoice := invoiceWithDiscountTotal("0")
		line := invoice.Lines.OrEmpty()[0]
		line.DetailedLines[0].ID = line.ID
		line.DetailedLines[0].AmountDiscounts = billing.AmountLineDiscountsManaged{{ID: line.ID}}
		other, err := line.Clone()
		require.NoError(t, err)
		other.ID = "other"
		other.DetailedLines[0].ID = "other"
		other.DetailedLines[0].AmountDiscounts[0].ID = "other"
		invoice.Lines.Append(other)
		setTestTotalsFromDetails(&invoice)
		require.NoError(t, stdinvoicesanity.Validate(invoice))
	})
}

func TestValidateServicePeriodEnvelope(t *testing.T) {
	t.Parallel()
	invoice := invoiceWithDiscountTotal("0")
	line := invoice.Lines.OrEmpty()[0]
	// Corrections can extend beyond the parent period. The same instant can also use a different time zone.
	line.DetailedLines[0].ServicePeriod.From = line.Period.From.Add(-time.Hour)
	line.DetailedLines[0].ServicePeriod.To = line.Period.To.Add(time.Hour)
	invoice.Period.From = invoice.Period.From.In(time.FixedZone("offset", 3600))
	require.NoError(t, stdinvoicesanity.Validate(invoice))
	other, err := line.Clone()
	require.NoError(t, err)
	other.ID = "other"
	other.DetailedLines = nil
	other.Totals = totals.Totals{}
	other.Period.From = line.Period.From.Add(-time.Hour)
	other.Period.To = line.Period.To.Add(time.Hour)
	invoice.Lines.Append(other)
	invoice.Period = lo.ToPtr(other.Period)
	require.NoError(t, stdinvoicesanity.Validate(invoice))
	other.DeletedAt = lo.ToPtr(time.Now())
	invoice.Period = lo.ToPtr(line.Period)
	require.NoError(t, stdinvoicesanity.Validate(invoice))
	line.Period = timeutil.ClosedPeriod{From: line.Period.From, To: line.Period.From}
	invoice.Period = lo.ToPtr(line.Period)
	require.NoError(t, stdinvoicesanity.Validate(invoice))
	line.DeletedAt = other.DeletedAt
	invoice.Totals = totals.Totals{}
	invoice.Period = nil
	require.NoError(t, stdinvoicesanity.Validate(invoice))
	invoice.Period = lo.ToPtr(line.Period)
	require.ErrorIs(t, stdinvoicesanity.Validate(invoice), stdinvoicesanity.ErrInvoicePeriodMismatch)
}

func TestValidateAllCalculatedPrecision(t *testing.T) {
	t.Parallel()
	for _, level := range []string{"invoice", "parent", "detail"} {
		t.Run(level, func(t *testing.T) {
			t.Parallel()
			invoice := invoiceWithDiscountTotal("0")
			fraction := alpacadecimal.RequireFromString("0.001")
			value := totals.Totals{Amount: fraction, ChargesTotal: fraction, DiscountsTotal: fraction, CreditsTotal: fraction, TaxesInclusiveTotal: fraction, TaxesExclusiveTotal: fraction, TaxesTotal: fraction, Total: fraction}
			switch level {
			case "invoice":
				invoice.Totals = value
			case "parent":
				invoice.Lines.OrEmpty()[0].Totals = value
			case "detail":
				invoice.Lines.OrEmpty()[0].DetailedLines[0].Totals = value
			}

			issues := requireSanityIssues(t, stdinvoicesanity.Validate(invoice))
			precision := lo.Filter(issues, func(issue billing.ValidationIssue, _ int) bool {
				return issue.Code == stdinvoicesanity.ErrTotalsPrecision.Code
			})
			require.Len(t, precision, 8)
		})
	}
}
