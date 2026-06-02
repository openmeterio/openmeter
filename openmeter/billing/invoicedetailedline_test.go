package billing

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	billingtotals "github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestDetailedLineValidateAllowsNegativeQuantity(t *testing.T) {
	line := validDetailedLineForValidation()
	line.Quantity = alpacadecimal.NewFromInt(-1)

	require.NoError(t, line.Validate())
}

func TestDetailedLineValidateRejectsNegativePerUnitAmount(t *testing.T) {
	line := validDetailedLineForValidation()
	line.PerUnitAmount = alpacadecimal.NewFromInt(-1)

	require.ErrorContains(t, line.Validate(), "price should be positive or zero")
}

func TestDetailedLinesWithCreditsAppliedConsumesCreditsAcrossPositiveTotals(t *testing.T) {
	currencyCalculator, err := currencyx.Code("USD").Calculator()
	require.NoError(t, err)

	lines := DetailedLines{
		detailedLineWithTotal("line-1", 10),
		detailedLineWithTotal("line-2", 5),
		detailedLineWithTotal("zero-line", 0),
	}

	mappedLines, err := lines.WithCreditsApplied(CreditsApplied{
		{
			Amount:              alpacadecimal.NewFromInt(12),
			Description:         "test credit",
			CreditRealizationID: "credit-1",
		},
	}, currencyCalculator)
	require.NoError(t, err)
	require.Len(t, mappedLines, 3)

	requireDecimalEqual(t, alpacadecimal.NewFromInt(10), mappedLines[0].CreditsApplied[0].Amount)
	requireDecimalEqual(t, alpacadecimal.NewFromInt(10), mappedLines[0].Totals.CreditsTotal)
	requireDecimalEqual(t, alpacadecimal.Zero, mappedLines[0].Totals.Total)

	requireDecimalEqual(t, alpacadecimal.NewFromInt(2), mappedLines[1].CreditsApplied[0].Amount)
	requireDecimalEqual(t, alpacadecimal.NewFromInt(2), mappedLines[1].Totals.CreditsTotal)
	requireDecimalEqual(t, alpacadecimal.NewFromInt(3), mappedLines[1].Totals.Total)

	require.Empty(t, mappedLines[2].CreditsApplied)
	requireDecimalEqual(t, alpacadecimal.Zero, mappedLines[2].Totals.CreditsTotal)
	requireDecimalEqual(t, alpacadecimal.Zero, mappedLines[2].Totals.Total)

	require.Empty(t, lines[0].CreditsApplied)
	requireDecimalEqual(t, alpacadecimal.NewFromInt(10), lines[0].Totals.Total)
	require.Empty(t, lines[1].CreditsApplied)
	requireDecimalEqual(t, alpacadecimal.NewFromInt(5), lines[1].Totals.Total)
}

func TestDetailedLinesWithCreditsAppliedReturnsErrorForUnusedCredits(t *testing.T) {
	currencyCalculator, err := currencyx.Code("USD").Calculator()
	require.NoError(t, err)

	lines := DetailedLines{
		detailedLineWithTotal("line-1", 5),
	}

	_, err = lines.WithCreditsApplied(CreditsApplied{
		{
			Amount:              alpacadecimal.NewFromInt(6),
			Description:         "test credit",
			CreditRealizationID: "credit-1",
		},
	}, currencyCalculator)
	require.ErrorIs(t, err, ErrInvoiceLineCreditsNotConsumedFully)

	require.Empty(t, lines[0].CreditsApplied)
	requireDecimalEqual(t, alpacadecimal.NewFromInt(5), lines[0].Totals.Total)
}

func TestDetailedLinesWithCreditsAppliedReplacesExistingCredits(t *testing.T) {
	currencyCalculator, err := currencyx.Code("USD").Calculator()
	require.NoError(t, err)

	lines := DetailedLines{
		detailedLineWithAppliedCredit("line-1", 10, 2),
	}

	mappedLines, err := lines.WithCreditsApplied(CreditsApplied{
		{
			Amount:              alpacadecimal.NewFromInt(3),
			Description:         "replacement credit",
			CreditRealizationID: "replacement-credit",
		},
	}, currencyCalculator)
	require.NoError(t, err)
	require.Len(t, mappedLines, 1)
	require.Len(t, mappedLines[0].CreditsApplied, 1)

	require.Equal(t, "replacement-credit", mappedLines[0].CreditsApplied[0].CreditRealizationID)
	requireDecimalEqual(t, alpacadecimal.NewFromInt(3), mappedLines[0].CreditsApplied[0].Amount)
	requireDecimalEqual(t, alpacadecimal.NewFromInt(3), mappedLines[0].Totals.CreditsTotal)
	requireDecimalEqual(t, alpacadecimal.NewFromInt(7), mappedLines[0].Totals.Total)

	require.Len(t, lines[0].CreditsApplied, 1)
	require.Equal(t, "existing-credit", lines[0].CreditsApplied[0].CreditRealizationID)
	requireDecimalEqual(t, alpacadecimal.NewFromInt(2), lines[0].Totals.CreditsTotal)
	requireDecimalEqual(t, alpacadecimal.NewFromInt(8), lines[0].Totals.Total)
}

func TestDetailedLinesWithCreditsAppliedNoCreditsReturnsClone(t *testing.T) {
	currencyCalculator, err := currencyx.Code("USD").Calculator()
	require.NoError(t, err)

	lines := DetailedLines{
		detailedLineWithAppliedCredit("line-1", 5, 2),
	}

	mappedLines, err := lines.WithCreditsApplied(nil, currencyCalculator)
	require.NoError(t, err)

	require.Empty(t, mappedLines[0].CreditsApplied)
	requireDecimalEqual(t, alpacadecimal.Zero, mappedLines[0].Totals.CreditsTotal)
	requireDecimalEqual(t, alpacadecimal.NewFromInt(5), mappedLines[0].Totals.Total)

	mappedLines[0].Totals.Total = alpacadecimal.Zero

	require.Len(t, lines[0].CreditsApplied, 1)
	requireDecimalEqual(t, alpacadecimal.NewFromInt(3), lines[0].Totals.Total)
	requireDecimalEqual(t, alpacadecimal.Zero, mappedLines[0].Totals.Total)
}

func TestDetailedLinesWithReversedCreditsClearsCreditValues(t *testing.T) {
	lines := DetailedLines{
		detailedLineWithAppliedCredit("line-1", 5, 2),
	}

	mappedLines := lines.WithReversedCredits()
	require.Len(t, mappedLines, 1)

	require.Empty(t, mappedLines[0].CreditsApplied)
	requireDecimalEqual(t, alpacadecimal.Zero, mappedLines[0].Totals.CreditsTotal)
	requireDecimalEqual(t, alpacadecimal.NewFromInt(5), mappedLines[0].Totals.Total)

	require.Len(t, lines[0].CreditsApplied, 1)
	requireDecimalEqual(t, alpacadecimal.NewFromInt(2), lines[0].Totals.CreditsTotal)
	requireDecimalEqual(t, alpacadecimal.NewFromInt(3), lines[0].Totals.Total)
}

func validDetailedLineForValidation() DetailedLine {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	return DetailedLine{
		DetailedLineBase: DetailedLineBase{
			InvoiceID: "inv_123",
			Base: stddetailedline.Base{
				Category:               stddetailedline.CategoryRegular,
				ChildUniqueReferenceID: "child_123",
				PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				ServicePeriod: timeutil.ClosedPeriod{
					From: start,
					To:   start.Add(time.Hour),
				},
				Currency:      currencyx.Code("USD"),
				PerUnitAmount: alpacadecimal.NewFromInt(10),
				Quantity:      alpacadecimal.NewFromInt(1),
			},
		},
	}
}

func detailedLineWithTotal(childUniqueReferenceID string, total int64) DetailedLine {
	line := validDetailedLineForValidation()
	line.ChildUniqueReferenceID = childUniqueReferenceID
	line.Totals = billingtotals.Totals{
		Amount: alpacadecimal.NewFromInt(total),
		Total:  alpacadecimal.NewFromInt(total),
	}

	return line
}

func detailedLineWithAppliedCredit(childUniqueReferenceID string, amount int64, creditAmount int64) DetailedLine {
	line := detailedLineWithTotal(childUniqueReferenceID, amount)
	line.CreditsApplied = CreditsApplied{
		{
			Amount:              alpacadecimal.NewFromInt(creditAmount),
			Description:         "existing credit",
			CreditRealizationID: "existing-credit",
		},
	}
	line.Totals.CreditsTotal = alpacadecimal.NewFromInt(creditAmount)
	line.Totals.Total = line.Totals.CalculateTotal()

	return line
}

func requireDecimalEqual(t *testing.T, expected, actual alpacadecimal.Decimal) {
	t.Helper()

	require.Equal(t, expected.InexactFloat64(), actual.InexactFloat64())
}
