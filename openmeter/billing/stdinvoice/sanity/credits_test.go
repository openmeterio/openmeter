package stdinvoicesanity_test

import (
	"fmt"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	stdinvoicesanity "github.com/openmeterio/openmeter/openmeter/billing/stdinvoice/sanity"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestValidateCreditTotals(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		currency currencyx.FiatCode
		amounts  []string
		total    string
		expected string
	}{
		{name: "no credits", currency: "USD", total: "0"},
		{name: "multiple allocations", currency: "USD", amounts: []string{"12", "8"}, total: "20"},
		{name: "round each credit", currency: "USD", amounts: []string{"0.005", "0.005"}, total: "0.02"},
		{name: "zero decimal currency", currency: "JPY", amounts: []string{"0.5", "0.5"}, total: "2"},
		{name: "three decimal currency", currency: "KWD", amounts: []string{"0.0005", "0.0005"}, total: "0.002"},
		{name: "missing allocation", currency: "USD", total: "20", expected: "0"},
		{name: "partial allocation", currency: "USD", amounts: []string{"12"}, total: "20", expected: "12"},
		{name: "unexpected allocation", currency: "USD", amounts: []string{"20"}, total: "0", expected: "20"},
		{name: "no tolerance", currency: "USD", amounts: []string{"20"}, total: "20.001", expected: "20"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			invoice := invoiceWithDiscountTotal("0")
			invoice.Currency = tc.currency
			invoice.Lines.OrEmpty()[0].Currency = tc.currency
			detail := &invoice.Lines.OrEmpty()[0].DetailedLines[0]
			detail.Totals.CreditsTotal = alpacadecimal.RequireFromString(tc.total)
			for _, amount := range tc.amounts {
				detail.CreditsApplied = append(detail.CreditsApplied, billing.CreditApplied{
					Amount:              alpacadecimal.RequireFromString(amount),
					CreditRealizationID: "shared-realization",
				})
			}

			invoice.Lines.OrEmpty()[0].CreditsApplied = append(billing.CreditsApplied{}, detail.CreditsApplied...)
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
				return issue.Code == stdinvoicesanity.ErrCreditTotalMismatch.Code
			})
			require.Len(t, issues, 1)
			require.ErrorIs(t, err, stdinvoicesanity.ErrCreditTotalMismatch)
			require.Equal(t, "/lines/0/detailedLines/0/totals/creditsTotal", issues[0].Path)
			require.Equal(t, tc.expected, issues[0].Attributes["expected"])
			require.Equal(t, tc.total, issues[0].Attributes["actual"])
			require.Equal(t, "detail-1", issues[0].Attributes["detailed_line_id"])
		})
	}
}

func TestValidateCreditConservation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                  string
		parent, first, second string
		firstID, secondID     string
		fail                  bool
	}{
		{"split realization", "20", "12", "8", "shared", "shared", false},
		{"round entries individually", "0.02", "0.005", "0.005", "shared", "shared", false},
		{"missing detail allocation", "20", "12", "0", "shared", "shared", true},
		{"duplicate allocation", "20", "20", "20", "shared", "shared", true},
		{"wrong realization same total", "20", "12", "8", "shared", "different", true},
		{"detail only", "0", "12", "8", "shared", "shared", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			invoice := invoiceWithDiscountTotal("0")
			line := invoice.Lines.OrEmpty()[0]
			other := line.DetailedLines[0].Clone()
			other.ID = "detail-2"
			other.ChildUniqueReferenceID = "detail-2"
			line.DetailedLines = append(line.DetailedLines, other)
			if tc.parent != "0" {
				line.CreditsApplied = billing.CreditsApplied{{Amount: alpacadecimal.RequireFromString(tc.parent), CreditRealizationID: "shared"}}
			}

			currency, err := invoice.Currency.AsFiatCurrency()
			require.NoError(t, err)
			for idx, entry := range []struct{ amount, id string }{{tc.first, tc.firstID}, {tc.second, tc.secondID}} {
				if entry.amount == "0" {
					continue
				}

				d := &line.DetailedLines[idx]
				d.CreditsApplied = billing.CreditsApplied{{Amount: alpacadecimal.RequireFromString(entry.amount), CreditRealizationID: entry.id}}
				d.Totals.CreditsTotal = d.CreditsApplied.SumAmount(currency)
			}

			setTestTotalsFromDetails(&invoice)
			before, err := invoice.Clone()
			require.NoError(t, err)
			err = stdinvoicesanity.Validate(invoice)
			require.Equal(t, before, invoice)
			if tc.fail {
				require.ErrorIs(t, err, stdinvoicesanity.ErrCreditConservationMismatch)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateCreditEntries(t *testing.T) {
	t.Parallel()
	for _, level := range []string{"parent", "detail"} {
		t.Run(level, func(t *testing.T) {
			t.Parallel()
			for _, amount := range []int64{-1, 0, 1} {
				t.Run(fmt.Sprint(amount), func(t *testing.T) {
					t.Parallel()
					invoice := invoiceWithDiscountTotal("0")
					credits := billing.CreditsApplied{{Amount: alpacadecimal.NewFromInt(amount)}}
					if level == "parent" {
						invoice.Lines.OrEmpty()[0].CreditsApplied = credits
					} else {
						invoice.Lines.OrEmpty()[0].DetailedLines[0].CreditsApplied = credits
					}

					err := stdinvoicesanity.Validate(invoice)
					require.ErrorIs(t, err, stdinvoicesanity.ErrMissingCreditRealizationID)
					if amount <= 0 {
						require.ErrorIs(t, err, stdinvoicesanity.ErrInvalidCreditAmount)
					} else {
						require.NotErrorIs(t, err, stdinvoicesanity.ErrInvalidCreditAmount)
					}
				})
			}
		})
	}
}

func TestValidateCreditAllocationLimits(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                         string
		gross, discount, tax, credit int64
		fail                         bool
	}{
		{"partial", 100, 0, 0, 20, false},
		{"exact", 100, 0, 0, 100, false},
		{"over", 100, 0, 0, 101, true},
		{"discount reduces budget", 100, 20, 0, 81, true},
		{"tax cannot extend budget", 100, 0, 10, 101, true},
		{"zero balance", 0, 0, 0, 1, true},
		{"negative balance", -10, 0, 0, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			invoice := invoiceWithDiscountTotal("0")
			line := invoice.Lines.OrEmpty()[0]
			d := &line.DetailedLines[0]
			d.Totals.Amount = alpacadecimal.NewFromInt(tc.gross)
			d.Totals.DiscountsTotal = alpacadecimal.NewFromInt(tc.discount)
			d.Totals.TaxesExclusiveTotal = alpacadecimal.NewFromInt(tc.tax)
			d.CreditsApplied = billing.CreditsApplied{{Amount: alpacadecimal.NewFromInt(tc.credit), CreditRealizationID: "shared"}}
			// Leave CreditsTotal unchanged to prove the limit check uses the allocations.
			err := stdinvoicesanity.Validate(invoice)
			if tc.fail {
				require.ErrorIs(t, err, stdinvoicesanity.ErrCreditAllocationLimit)
			} else {
				require.NotErrorIs(t, err, stdinvoicesanity.ErrCreditAllocationLimit)
			}
		})
	}
}
