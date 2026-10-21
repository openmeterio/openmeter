package stdinvoicesanity_test

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	stdinvoicesanity "github.com/openmeterio/openmeter/openmeter/billing/stdinvoice/sanity"
)

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
