package lineservice

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type testLineMode string

const (
	singlePerPeriodLineMode   testLineMode = "single_per_period"
	midPeriodSplitLineMode    testLineMode = "mid_period_split"
	lastInPeriodSplitLineMode testLineMode = "last_in_period_split"
)

var ubpTestFullPeriod = billing.Period{
	Start: lo.Must(time.Parse(time.RFC3339, "2021-01-01T00:00:00Z")),
	End:   lo.Must(time.Parse(time.RFC3339, "2021-01-02T00:00:00Z")),
}

type ubpCalculationTestCase struct {
	price    productcatalog.Price
	lineMode testLineMode
	usage    featureUsageResponse
	expect   newDetailedLinesInput
}

func runUBPTest(t *testing.T, tc ubpCalculationTestCase) {
	t.Helper()

	usdCurrencyCalc, err := currencyx.Code(currency.USD).Calculator()
	require.NoError(t, err)

	l := usageBasedLine{
		lineBase: lineBase{
			line: &billing.Line{
				LineBase: billing.LineBase{
					Currency: "USD",
					ID:       "fake-line",
					Type:     billing.InvoiceLineTypeUsageBased,
					Status:   billing.InvoiceLineStatusValid,
					Name:     "feature",
				},
				UsageBased: &billing.UsageBasedLine{
					Price: lo.ToPtr(tc.price),
				},
			},
			currency: usdCurrencyCalc,
		},
	}

	fakeParentLine := billing.Line{
		LineBase: billing.LineBase{
			ID:     "fake-parent-line",
			Period: ubpTestFullPeriod,
			Status: billing.InvoiceLineStatusSplit,
		},
	}

	switch tc.lineMode {
	case singlePerPeriodLineMode:
		l.line.Period = ubpTestFullPeriod
	case midPeriodSplitLineMode:
		l.line.Period = billing.Period{
			Start: ubpTestFullPeriod.Start.Add(time.Hour * 12),
			End:   ubpTestFullPeriod.End.Add(-time.Hour),
		}
		l.line.ParentLine = &fakeParentLine
		l.line.ParentLineID = &fakeParentLine.ID

	case lastInPeriodSplitLineMode:
		l.line.Period = billing.Period{
			Start: ubpTestFullPeriod.Start.Add(time.Hour * 12),
			End:   ubpTestFullPeriod.End,
		}

		l.line.ParentLine = &fakeParentLine
		l.line.ParentLineID = &fakeParentLine.ID
	}

	// Let's set the usage on the line
	l.line.UsageBased.Quantity = &tc.usage.LinePeriodQty
	l.line.UsageBased.PreLinePeriodQuantity = &tc.usage.PreLinePeriodQty

	res, err := l.calculateDetailedLines()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// let's get around nil slices
	if len(tc.expect) == 0 && len(res) == 0 {
		return
	}

	expectJSON, err := json.Marshal(tc.expect)
	require.NoError(t, err)

	resJSON, err := json.Marshal(res)
	require.NoError(t, err)

	require.JSONEq(t, string(expectJSON), string(resJSON))
}

func TestAddDiscountForOverage(t *testing.T) {
	currency, err := currencyx.Code(currency.USD).Calculator()
	require.NoError(t, err)

	l := newDetailedLineInput{
		PerUnitAmount: alpacadecimal.NewFromFloat(100),
		Quantity:      alpacadecimal.NewFromFloat(10),
	}

	t.Run("no overage", func(t *testing.T) {
		lineWithDiscount := l.AddDiscountForOverage(addDiscountInput{
			MaxSpend:               alpacadecimal.NewFromFloat(10000),
			BilledAmountBeforeLine: alpacadecimal.NewFromFloat(9000),
			// Total $10000 => No max spend is reached
			Currency: currency,
		})

		require.Equal(t, l, lineWithDiscount)
	})

	// currency rounding
	t.Run("no overage", func(t *testing.T) {
		lineWithDiscount := l.AddDiscountForOverage(addDiscountInput{
			MaxSpend:               alpacadecimal.NewFromFloat(10000.001),
			BilledAmountBeforeLine: alpacadecimal.NewFromFloat(9000.001),
			// Total $10000 => No max spend is reached
			Currency: currency,
		})

		require.Equal(t, l, lineWithDiscount)
	})

	t.Run("overage, rounding", func(t *testing.T) {
		lineWithDiscount := l.AddDiscountForOverage(addDiscountInput{
			MaxSpend:               alpacadecimal.NewFromFloat(10000.001),
			BilledAmountBeforeLine: alpacadecimal.NewFromFloat(9000.01123),
			// Total $10000 => No max spend is reached
			Currency: currency,
		})

		require.Equal(t, newDetailedLineInput{
			PerUnitAmount: alpacadecimal.NewFromFloat(100),
			Quantity:      alpacadecimal.NewFromFloat(10),
			Discounts: billing.NewLineDiscounts(
				billing.NewLineDiscountFrom(billing.AmountLineDiscount{
					Amount: alpacadecimal.NewFromFloat(0.01),
					LineDiscountBase: billing.LineDiscountBase{
						Description:            lo.ToPtr("Maximum spend discount for charges over 10000"),
						ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
						Reason:                 billing.LineDiscountReasonMaximumSpend,
					},
				},
				),
			),
		}, lineWithDiscount)
	})

	t.Run("overage and some valid charges", func(t *testing.T) {
		lineWithDiscount := l.AddDiscountForOverage(addDiscountInput{
			MaxSpend:               alpacadecimal.NewFromFloat(10000),
			BilledAmountBeforeLine: alpacadecimal.NewFromFloat(9600),
			// Total $10000 => $500 discount
			Currency: currency,
		})

		require.Equal(t, newDetailedLineInput{
			PerUnitAmount: alpacadecimal.NewFromFloat(100),
			Quantity:      alpacadecimal.NewFromFloat(10),
			Discounts: billing.NewLineDiscounts(
				billing.NewLineDiscountFrom(billing.AmountLineDiscount{
					Amount: alpacadecimal.NewFromFloat(600),
					LineDiscountBase: billing.LineDiscountBase{
						Description:            lo.ToPtr("Maximum spend discount for charges over 10000"),
						ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
						Reason:                 billing.LineDiscountReasonMaximumSpend,
					},
				},
				),
			),
		}, lineWithDiscount)
	})

	t.Run("overage 100% discount", func(t *testing.T) {
		lineWithDiscount := l.AddDiscountForOverage(addDiscountInput{
			MaxSpend:               alpacadecimal.NewFromFloat(10000),
			BilledAmountBeforeLine: alpacadecimal.NewFromFloat(10000),
			// Total $10000 => $1000 discount
			Currency: currency,
		})

		require.Equal(t, newDetailedLineInput{
			PerUnitAmount: alpacadecimal.NewFromFloat(100),
			Quantity:      alpacadecimal.NewFromFloat(10),
			Discounts: billing.NewLineDiscounts(
				billing.NewLineDiscountFrom(billing.AmountLineDiscount{
					Amount: alpacadecimal.NewFromFloat(1000),
					LineDiscountBase: billing.LineDiscountBase{
						Description:            lo.ToPtr("Maximum spend discount for charges over 10000"),
						ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
						Reason:                 billing.LineDiscountReasonMaximumSpend,
					},
				},
				),
			),
		}, lineWithDiscount)
	})

	t.Run("overage and 100% discount when hugely over the max spend", func(t *testing.T) {
		lineWithDiscount := l.AddDiscountForOverage(addDiscountInput{
			MaxSpend:               alpacadecimal.NewFromFloat(10000),
			BilledAmountBeforeLine: alpacadecimal.NewFromFloat(20000),
			// Total $10000 => $1000 discount
			Currency: currency,
		})

		require.Equal(t, newDetailedLineInput{
			PerUnitAmount: alpacadecimal.NewFromFloat(100),
			Quantity:      alpacadecimal.NewFromFloat(10),
			Discounts: billing.NewLineDiscounts(
				billing.NewLineDiscountFrom(billing.AmountLineDiscount{
					Amount: alpacadecimal.NewFromFloat(1000),
					LineDiscountBase: billing.LineDiscountBase{
						Description:            lo.ToPtr("Maximum spend discount for charges over 10000"),
						ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
						Reason:                 billing.LineDiscountReasonMaximumSpend,
					},
				}),
			),
		}, lineWithDiscount)
	})
}
