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
	"github.com/openmeterio/openmeter/pkg/models"
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
	price                productcatalog.Price
	discounts            billing.Discounts
	lineMode             testLineMode
	usage                featureUsageResponse
	expect               newDetailedLinesInput
	previousBilledAmount alpacadecimal.Decimal
}

type ubpLineCalculator interface {
	calculateDetailedLines() (newDetailedLinesInput, error)
}

func runUBPTest(t *testing.T, tc ubpCalculationTestCase) {
	t.Helper()

	usdCurrencyCalc, err := currencyx.Code(currency.USD).Calculator()
	require.NoError(t, err)

	line := &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				ID:   "fake-line",
				Name: "feature",
			}),
			Currency:          "USD",
			RateCardDiscounts: tc.discounts,
		},
		UsageBased: &billing.UsageBasedLine{
			Price: lo.ToPtr(tc.price),
		},
	}

	fakeParentGroup := billing.SplitLineGroup{
		NamespacedID: models.NamespacedID{
			Namespace: "fake-namespace",
			ID:        "fake-parent-group",
		},
		SplitLineGroupMutableFields: billing.SplitLineGroupMutableFields{
			ServicePeriod: ubpTestFullPeriod,
		},
	}

	fakeHierarchy := billing.SplitLineHierarchy{
		Group: fakeParentGroup,
		Lines: []billing.LineWithInvoiceHeader{
			{
				Line: &billing.StandardLine{
					StandardLineBase: billing.StandardLineBase{
						// Period is unset, so this fake line is always in scope for NetAmount calculations
						Totals: billing.Totals{
							Amount: tc.previousBilledAmount,
						},
					},
				},
			},
		},
	}

	switch tc.lineMode {
	case singlePerPeriodLineMode:
		line.Period = ubpTestFullPeriod
	case midPeriodSplitLineMode:
		line.Period = billing.Period{
			Start: ubpTestFullPeriod.Start.Add(time.Hour * 12),
			End:   ubpTestFullPeriod.End.Add(-time.Hour),
		}
		line.SplitLineGroupID = &fakeParentGroup.ID
		line.SplitLineHierarchy = &fakeHierarchy

	case lastInPeriodSplitLineMode:
		line.Period = billing.Period{
			Start: ubpTestFullPeriod.Start.Add(time.Hour * 12),
			End:   ubpTestFullPeriod.End,
		}

		line.SplitLineGroupID = &fakeParentGroup.ID
		line.SplitLineHierarchy = &fakeHierarchy
	}

	// Let's set the usage on the line
	line.UsageBased.Quantity = &tc.usage.LinePeriodQty
	line.UsageBased.MeteredQuantity = &tc.usage.LinePeriodQty
	line.UsageBased.PreLinePeriodQuantity = &tc.usage.PreLinePeriodQty
	line.UsageBased.MeteredPreLinePeriodQuantity = &tc.usage.PreLinePeriodQty

	lineBase := lineBase{
		line:     line,
		currency: usdCurrencyCalc,
	}

	var lineImpl ubpLineCalculator
	switch line.UsageBased.Price.Type() {
	case productcatalog.FlatPriceType:
		lineImpl = &ubpFlatFeeLine{
			lineBase: lineBase,
		}
	default:
		lineImpl = &usageBasedLine{
			lineBase: lineBase,
		}
	}

	res, err := lineImpl.calculateDetailedLines()
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
			AmountDiscounts: []billing.AmountLineDiscountManaged{
				{
					AmountLineDiscount: billing.AmountLineDiscount{
						Amount: alpacadecimal.NewFromFloat(0.01),
						LineDiscountBase: billing.LineDiscountBase{
							Description:            lo.ToPtr("Maximum spend discount for charges over 10000"),
							ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
							Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
						},
					},
				},
			},
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
			AmountDiscounts: []billing.AmountLineDiscountManaged{
				{
					AmountLineDiscount: billing.AmountLineDiscount{
						Amount: alpacadecimal.NewFromFloat(600),
						LineDiscountBase: billing.LineDiscountBase{
							Description:            lo.ToPtr("Maximum spend discount for charges over 10000"),
							ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
							Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
						},
					},
				},
			},
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
			AmountDiscounts: []billing.AmountLineDiscountManaged{
				{
					AmountLineDiscount: billing.AmountLineDiscount{
						Amount: alpacadecimal.NewFromFloat(1000),
						LineDiscountBase: billing.LineDiscountBase{
							Description:            lo.ToPtr("Maximum spend discount for charges over 10000"),
							ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
							Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
						},
					},
				},
			},
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
			AmountDiscounts: []billing.AmountLineDiscountManaged{
				{
					AmountLineDiscount: billing.AmountLineDiscount{
						Amount: alpacadecimal.NewFromFloat(1000),
						LineDiscountBase: billing.LineDiscountBase{
							Description:            lo.ToPtr("Maximum spend discount for charges over 10000"),
							ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
							Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
						},
					},
				},
			},
		}, lineWithDiscount)
	})
}
