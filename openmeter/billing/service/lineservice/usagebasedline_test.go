package lineservice

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
)

type testLineMode string

const (
	singlePerPeriodLineMode   testLineMode = "single_per_period"
	midPeriodSplitLineMode    testLineMode = "mid_period_split"
	lastInPeriodSplitLineMode testLineMode = "last_in_period_split"
)

var ubpTestFullPeriod = billingentity.Period{
	Start: lo.Must(time.Parse(time.RFC3339, "2021-01-01T00:00:00Z")),
	End:   lo.Must(time.Parse(time.RFC3339, "2021-01-02T00:00:00Z")),
}

type ubpCalculationTestCase struct {
	price    plan.Price
	lineMode testLineMode
	usage    featureUsageResponse
	expect   newDetailedLinesInput
}

func runUBPTest(t *testing.T, tc ubpCalculationTestCase) {
	t.Helper()
	l := usageBasedLine{
		lineBase: lineBase{
			line: billingentity.Line{
				LineBase: billingentity.LineBase{
					ID:     "fake-line",
					Type:   billingentity.InvoiceLineTypeUsageBased,
					Status: billingentity.InvoiceLineStatusValid,
					Name:   "feature",
				},
				UsageBased: billingentity.UsageBasedLine{
					Price: tc.price,
				},
			},
		},
	}

	fakeParentLine := billingentity.Line{
		LineBase: billingentity.LineBase{
			ID:     "fake-parent-line",
			Period: ubpTestFullPeriod,
			Status: billingentity.InvoiceLineStatusSplit,
		},
	}

	switch tc.lineMode {
	case singlePerPeriodLineMode:
		l.line.Period = ubpTestFullPeriod
	case midPeriodSplitLineMode:
		l.line.Period = billingentity.Period{
			Start: ubpTestFullPeriod.Start.Add(time.Hour * 12),
			End:   ubpTestFullPeriod.End.Add(-time.Hour),
		}
		l.line.ParentLine = &fakeParentLine
		l.line.ParentLineID = &fakeParentLine.ID

	case lastInPeriodSplitLineMode:
		l.line.Period = billingentity.Period{
			Start: ubpTestFullPeriod.Start.Add(time.Hour * 12),
			End:   ubpTestFullPeriod.End,
		}

		l.line.ParentLine = &fakeParentLine
		l.line.ParentLineID = &fakeParentLine.ID
	}

	res, err := l.calculateDetailedLines(&tc.usage)
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

func TestFlatLineCalculation(t *testing.T) {
	// Flat price tests
	t.Run("flat price no usage", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: plan.InAdvancePaymentTerm,
			}),
			lineMode: singlePerPeriodLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature",
					Amount:                 alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: FlatPriceChildUniqueReferenceID,
					PaymentTerm:            plan.InAdvancePaymentTerm,
				},
			},
		})
	})

	t.Run("flat price, in advance, usage present", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: plan.InAdvancePaymentTerm,
			}),
			lineMode: singlePerPeriodLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature",
					Amount:                 alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: FlatPriceChildUniqueReferenceID,
					PaymentTerm:            plan.InAdvancePaymentTerm,
				},
			},
		})
	})

	t.Run("flat price, in advance, usage present, mid period", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: plan.InAdvancePaymentTerm,
			}),
			lineMode: midPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			expect: newDetailedLinesInput{},
		})
	})

	t.Run("flat price, in arrears, usage present, single period line", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: plan.InArrearsPaymentTerm,
			}),
			lineMode: singlePerPeriodLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature",
					Amount:                 alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: FlatPriceChildUniqueReferenceID,
					PaymentTerm:            plan.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("flat price, in arrears, usage present, mid period line", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: plan.InArrearsPaymentTerm,
			}),
			lineMode: midPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			expect: newDetailedLinesInput{}, // It will be billed in the last period
		})
	})

	t.Run("flat price, in arrears, usage present, last period line", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: plan.InArrearsPaymentTerm,
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature",
					Amount:                 alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: FlatPriceChildUniqueReferenceID,
					PaymentTerm:            plan.InArrearsPaymentTerm,
				},
			},
		})
	})
}

func TestUnitPriceCalculation(t *testing.T) {
	t.Run("unit price, no usage", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			}),
			lineMode: singlePerPeriodLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			expect: newDetailedLinesInput{},
		})
	})

	// When there is no usage, we are still honoring the minimum spend
	t.Run("unit price, no usage, min spend set", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.UnitPrice{
				Amount:        alpacadecimal.NewFromFloat(10),
				MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
			}),
			lineMode: singlePerPeriodLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: minimum spend",
					Amount:                 alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: UnitPriceMinSpendChildUniqueReferenceID,
					Period:                 &ubpTestFullPeriod,
					PaymentTerm:            plan.InArrearsPaymentTerm,
				},
			},
		})
	})

	// Min spend is always billed in arrears => we are not billing it in advance
	t.Run("no usage, not the last line in period, min spend set", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.UnitPrice{
				Amount:        alpacadecimal.NewFromFloat(10),
				MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
			}),
			lineMode: midPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			expect: newDetailedLinesInput{},
		})
	})

	// Min spend is always billed in arrears => we are billing it for the last line
	t.Run("no usage, last line in period, min spend set", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.UnitPrice{
				Amount:        alpacadecimal.NewFromFloat(10),
				MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: minimum spend",
					Amount:                 alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: UnitPriceMinSpendChildUniqueReferenceID,
					Period:                 &ubpTestFullPeriod,
					PaymentTerm:            plan.InArrearsPaymentTerm,
				},
			},
		})
	})

	// Usage is billed regardless of line position
	t.Run("usage present", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(100),
			}),
			lineMode: singlePerPeriodLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: usage in period",
					Amount:                 alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(10),
					ChildUniqueReferenceID: UnitPriceUsageChildUniqueReferenceID,
					PaymentTerm:            plan.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("usage present, mid line", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(100),
			}),
			lineMode: midPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: usage in period",
					Amount:                 alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(10),
					ChildUniqueReferenceID: UnitPriceUsageChildUniqueReferenceID,
					PaymentTerm:            plan.InArrearsPaymentTerm,
				},
			},
		})
	})

	// Max spend is always honored
	t.Run("usage present, max spend set, but not hit", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.UnitPrice{
				Amount:        alpacadecimal.NewFromFloat(10),
				MaximumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
			}),
			lineMode: midPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: usage in period",
					Amount:                 alpacadecimal.NewFromFloat(10),
					Quantity:               alpacadecimal.NewFromFloat(10),
					ChildUniqueReferenceID: UnitPriceUsageChildUniqueReferenceID,
					PaymentTerm:            plan.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("usage present, max spend set, but not hit", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.UnitPrice{
				Amount:        alpacadecimal.NewFromFloat(10),
				MaximumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
			}),
			lineMode: midPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty:    alpacadecimal.NewFromFloat(5),
				PreLinePeriodQty: alpacadecimal.NewFromFloat(7),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: usage in period",
					Amount:                 alpacadecimal.NewFromFloat(10),
					Quantity:               alpacadecimal.NewFromFloat(5),
					ChildUniqueReferenceID: UnitPriceUsageChildUniqueReferenceID,
					PaymentTerm:            plan.InArrearsPaymentTerm,
					Discounts: []billingentity.LineDiscount{
						{
							Description: lo.ToPtr("Maximum spend discount for charges over 100"),
							Amount:      alpacadecimal.NewFromFloat(20),
							Type:        lo.ToPtr(billingentity.MaximumSpendLineDiscountType),
							Source:      billingentity.CalculatedLineDiscountSource,
						},
					},
				},
			},
		})
	})
}

func TestTieredGraduatedCalculation(t *testing.T) {
	testTiers := []plan.PriceTier{
		{
			UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(5)),
			FlatPrice: &plan.PriceTierFlatPrice{
				// 20/unit
				Amount: alpacadecimal.NewFromFloat(100),
			},
		},
		{
			UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(10)),
			FlatPrice: &plan.PriceTierFlatPrice{
				// 10/unit
				Amount: alpacadecimal.NewFromFloat(150),
			},
		},
		{
			UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(15)),
			UnitPrice: &plan.PriceTierUnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			},
		},
		{
			UnitPrice: &plan.PriceTierUnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			},
		},
	}

	t.Run("tiered graduated, mid price", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.TieredPrice{
				Mode:  plan.GraduatedTieredPrice,
				Tiers: testTiers,
			}),
			lineMode: midPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			expect: newDetailedLinesInput{},
		})
	})

	t.Run("tiered graduated, last price, no usage", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.TieredPrice{
				Mode:  plan.GraduatedTieredPrice,
				Tiers: testTiers,
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			expect: newDetailedLinesInput{},
		})
	})

	t.Run("tiered graduated, last price, usage present, tier1 mid", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.TieredPrice{
				Mode:  plan.GraduatedTieredPrice,
				Tiers: testTiers,
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(3),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: flat price for tier 1",
					Amount:                 alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: GraduatedFlatPriceChildUniqueReferenceID,
					PaymentTerm:            plan.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("tiered graduated, last price, usage present, tier1 top", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.TieredPrice{
				Mode:  plan.GraduatedTieredPrice,
				Tiers: testTiers,
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(5),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: flat price for tier 1",
					Amount:                 alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: GraduatedFlatPriceChildUniqueReferenceID,
					PaymentTerm:            plan.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("tiered graduated, last price, usage present, tier4", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.TieredPrice{
				Mode:  plan.GraduatedTieredPrice,
				Tiers: testTiers,
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(100),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: unit price for tier 4",
					Amount:                 alpacadecimal.NewFromFloat(5),
					Quantity:               alpacadecimal.NewFromFloat(100),
					ChildUniqueReferenceID: GraduatedUnitPriceChildUniqueReferenceID,
					PaymentTerm:            plan.InArrearsPaymentTerm,
				},
			},
		})
	})

	// Minimum spend

	t.Run("tiered graduated, last price, no usage, min spend", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.TieredPrice{
				Mode:          plan.GraduatedTieredPrice,
				Tiers:         testTiers,
				MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: minimum spend",
					Amount:                 alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: GraduatedMinSpendChildUniqueReferenceID,
					PaymentTerm:            plan.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("tiered graduated, last price, usage over, min spend", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.TieredPrice{
				Mode:          plan.GraduatedTieredPrice,
				Tiers:         testTiers,
				MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(100),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: unit price for tier 4",
					Amount:                 alpacadecimal.NewFromFloat(5),
					Quantity:               alpacadecimal.NewFromFloat(100),
					ChildUniqueReferenceID: GraduatedUnitPriceChildUniqueReferenceID,
					PaymentTerm:            plan.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("tiered graduated, last price, usage less than min spend", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.TieredPrice{
				Mode:          plan.GraduatedTieredPrice,
				Tiers:         testTiers,
				MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(150)),
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(5),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: flat price for tier 1",
					Amount:                 alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: GraduatedFlatPriceChildUniqueReferenceID,
					PaymentTerm:            plan.InArrearsPaymentTerm,
				},
				{
					Name:                   "feature: minimum spend",
					Amount:                 alpacadecimal.NewFromFloat(50),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: GraduatedMinSpendChildUniqueReferenceID,
					PaymentTerm:            plan.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("tiered graduated, last price, usage less equals min spend", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.TieredPrice{
				Mode:          plan.GraduatedTieredPrice,
				Tiers:         testTiers,
				MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(5),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: flat price for tier 1",
					Amount:                 alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: GraduatedFlatPriceChildUniqueReferenceID,
					PaymentTerm:            plan.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("tiered graduated, no usage, min spend should be returned", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.TieredPrice{
				Mode:          plan.GraduatedTieredPrice,
				Tiers:         testTiers,
				MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: minimum spend",
					Amount:                 alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: GraduatedMinSpendChildUniqueReferenceID,
					PaymentTerm:            plan.InArrearsPaymentTerm,
				},
			},
		})
	})

	// Maximum spend
	t.Run("tiered graduated, first price, usage eq max spend", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.TieredPrice{
				Mode:          plan.GraduatedTieredPrice,
				Tiers:         testTiers,
				MaximumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(5),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: flat price for tier 1",
					Amount:                 alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: GraduatedFlatPriceChildUniqueReferenceID,
					PaymentTerm:            plan.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("tiered graduated, first price, usage above max spend, max spend is not at tier boundary ", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.TieredPrice{
				Mode:          plan.GraduatedTieredPrice,
				Tiers:         testTiers,
				MaximumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(125)),
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(7),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: flat price for tier 2",
					Amount:                 alpacadecimal.NewFromFloat(150),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: GraduatedFlatPriceChildUniqueReferenceID,
					PaymentTerm:            plan.InArrearsPaymentTerm,
					Discounts: []billingentity.LineDiscount{
						{
							Description: lo.ToPtr("Maximum spend discount for charges over 125"),
							Amount:      alpacadecimal.NewFromFloat(25),
							Type:        lo.ToPtr(billingentity.MaximumSpendLineDiscountType),
							Source:      billingentity.CalculatedLineDiscountSource,
						},
					},
				},
			},
		})
	})
}

func TestTieredVolumeCalculation(t *testing.T) {
	testTiers := []plan.PriceTier{
		{
			UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(5)),
			FlatPrice: &plan.PriceTierFlatPrice{
				// 20/unit
				Amount: alpacadecimal.NewFromFloat(100),
			},
		},
		{
			UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(10)),
			FlatPrice: &plan.PriceTierFlatPrice{
				// 10/unit
				Amount: alpacadecimal.NewFromFloat(50),
			},
		},
		{
			UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(15)),
			UnitPrice: &plan.PriceTierUnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			},
		},
		{
			UnitPrice: &plan.PriceTierUnitPrice{
				Amount: alpacadecimal.NewFromFloat(1),
			},
		},
	}

	t.Run("tiered volume, mid price, flat only => no lines are output", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.TieredPrice{
				Mode:  plan.VolumeTieredPrice,
				Tiers: testTiers,
			}),
			lineMode: midPeriodSplitLineMode,
			usage: featureUsageResponse{
				PreLinePeriodQty: alpacadecimal.NewFromFloat(7),
				LinePeriodQty:    alpacadecimal.NewFromFloat(1),
			},
			expect: newDetailedLinesInput{},
		})
	})

	t.Run("tiered volume, last price, no usage", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.TieredPrice{
				Mode:  plan.VolumeTieredPrice,
				Tiers: testTiers,
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			expect: newDetailedLinesInput{},
		})
	})

	t.Run("tiered volume, single period multiple tier usage", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.TieredPrice{
				Mode:  plan.VolumeTieredPrice,
				Tiers: testTiers,
			}),
			lineMode: singlePerPeriodLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(22),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: flat price for tier 1",
					Amount:                 alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: "volume-tiered-1-flat-price",
					PaymentTerm:            plan.InArrearsPaymentTerm,
				},
				{
					Name:                   "feature: flat price for tier 2",
					Amount:                 alpacadecimal.NewFromFloat(50),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: "volume-tiered-2-flat-price",
					PaymentTerm:            plan.InArrearsPaymentTerm,
				},
				{
					Name:                   "feature: usage price for tier 3",
					Amount:                 alpacadecimal.NewFromFloat(5),
					Quantity:               alpacadecimal.NewFromFloat(5),
					ChildUniqueReferenceID: "volume-tiered-3-price-usage",
					PaymentTerm:            plan.InArrearsPaymentTerm,
				},
				{
					Name:                   "feature: usage price for tier 4",
					Amount:                 alpacadecimal.NewFromFloat(1),
					Quantity:               alpacadecimal.NewFromFloat(7),
					ChildUniqueReferenceID: "volume-tiered-4-price-usage",
					PaymentTerm:            plan.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("tiered volume, mid period, multiple tier usage", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.TieredPrice{
				Mode:  plan.VolumeTieredPrice,
				Tiers: testTiers,
			}),
			lineMode: singlePerPeriodLineMode,
			usage: featureUsageResponse{
				PreLinePeriodQty: alpacadecimal.NewFromFloat(12),
				LinePeriodQty:    alpacadecimal.NewFromFloat(10), // total usage is at 22
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: usage price for tier 3",
					Amount:                 alpacadecimal.NewFromFloat(5),
					Quantity:               alpacadecimal.NewFromFloat(3),
					ChildUniqueReferenceID: "volume-tiered-3-price-usage",
					PaymentTerm:            plan.InArrearsPaymentTerm,
				},
				{
					Name:                   "feature: usage price for tier 4",
					Amount:                 alpacadecimal.NewFromFloat(1),
					Quantity:               alpacadecimal.NewFromFloat(7),
					ChildUniqueReferenceID: "volume-tiered-4-price-usage",
					PaymentTerm:            plan.InArrearsPaymentTerm,
				},
			},
		})
	})

	// Minimum spend

	t.Run("tiered volume, last line, no usage, minimum price set", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.TieredPrice{
				Mode:          plan.VolumeTieredPrice,
				Tiers:         testTiers,
				MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(1000)),
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				PreLinePeriodQty: alpacadecimal.NewFromFloat(0),
				LinePeriodQty:    alpacadecimal.NewFromFloat(0),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: minimum spend",
					Amount:                 alpacadecimal.NewFromFloat(1000),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: VolumeMinSpendChildUniqueReferenceID,
					PaymentTerm:            plan.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("tiered volume, last line, no usage, minimum price set", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.TieredPrice{
				Mode:          plan.VolumeTieredPrice,
				Tiers:         testTiers,
				MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(1000)),
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				PreLinePeriodQty: alpacadecimal.NewFromFloat(2),
				LinePeriodQty:    alpacadecimal.NewFromFloat(0),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: minimum spend",
					Amount:                 alpacadecimal.NewFromFloat(900),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: VolumeMinSpendChildUniqueReferenceID,
					PaymentTerm:            plan.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("tiered volume, mid line, no usage, minimum price set", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.TieredPrice{
				Mode:          plan.VolumeTieredPrice,
				Tiers:         testTiers,
				MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(1000)),
			}),
			lineMode: midPeriodSplitLineMode,
			usage: featureUsageResponse{
				PreLinePeriodQty: alpacadecimal.NewFromFloat(2),
				LinePeriodQty:    alpacadecimal.NewFromFloat(0),
			},
			expect: newDetailedLinesInput{},
		})
	})

	// Maximum spend
	t.Run("tiered volume, mid period, multiple tier usage, maximum spend set mid tier 2/3", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: plan.NewPriceFrom(plan.TieredPrice{
				Mode:          plan.VolumeTieredPrice,
				Tiers:         testTiers,
				MaximumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(170)),
			}),
			lineMode: singlePerPeriodLineMode,
			usage: featureUsageResponse{
				PreLinePeriodQty: alpacadecimal.NewFromFloat(12),
				LinePeriodQty:    alpacadecimal.NewFromFloat(10), // total usage is at 22
			},

			// Total previous usage due to the PreLinePeriodQty:
			// tier 1: $100 flat
			// tier 2: $50 flat
			// tier 3: 2*$5 = $10 usage
			// total: $160

			expect: newDetailedLinesInput{
				{
					Name:                   "feature: usage price for tier 3",
					Amount:                 alpacadecimal.NewFromFloat(5),
					Quantity:               alpacadecimal.NewFromFloat(3),
					ChildUniqueReferenceID: "volume-tiered-3-price-usage",
					PaymentTerm:            plan.InArrearsPaymentTerm,
					Discounts: []billingentity.LineDiscount{
						{
							Description: lo.ToPtr("Maximum spend discount for charges over 170"),
							Amount:      alpacadecimal.NewFromFloat(5),
							Type:        lo.ToPtr(billingentity.MaximumSpendLineDiscountType),
							Source:      billingentity.CalculatedLineDiscountSource,
						},
					},
				},
				{
					Name:                   "feature: usage price for tier 4",
					Amount:                 alpacadecimal.NewFromFloat(1),
					Quantity:               alpacadecimal.NewFromFloat(7),
					ChildUniqueReferenceID: "volume-tiered-4-price-usage",
					PaymentTerm:            plan.InArrearsPaymentTerm,
					Discounts: []billingentity.LineDiscount{
						{
							Description: lo.ToPtr("Maximum spend discount for charges over 170"),
							Amount:      alpacadecimal.NewFromFloat(7),
							Type:        lo.ToPtr(billingentity.MaximumSpendLineDiscountType),
							Source:      billingentity.CalculatedLineDiscountSource,
						},
					},
				},
			},
		})
	})
}

func TestAddDiscountForOverage(t *testing.T) {
	l := newDetailedLineInput{
		Amount:   alpacadecimal.NewFromFloat(100),
		Quantity: alpacadecimal.NewFromFloat(10),
	}

	t.Run("no overage", func(t *testing.T) {
		lineWithDiscount := l.AddDiscountForOverage(addDiscountInput{
			MaxSpend:               alpacadecimal.NewFromFloat(10000),
			BilledAmountBeforeLine: alpacadecimal.NewFromFloat(9000),
			// Total $10000 => No max spend is reached
		})

		require.Equal(t, l, lineWithDiscount)
	})

	t.Run("overage and some valid charges", func(t *testing.T) {
		lineWithDiscount := l.AddDiscountForOverage(addDiscountInput{
			MaxSpend:               alpacadecimal.NewFromFloat(10000),
			BilledAmountBeforeLine: alpacadecimal.NewFromFloat(9600),
			// Total $10000 => $500 discount
		})

		require.Equal(t, newDetailedLineInput{
			Amount:   alpacadecimal.NewFromFloat(100),
			Quantity: alpacadecimal.NewFromFloat(10),
			Discounts: []billingentity.LineDiscount{
				{
					Description: lo.ToPtr("Maximum spend discount for charges over 10000"),
					Amount:      alpacadecimal.NewFromFloat(600),
					Type:        lo.ToPtr(billingentity.MaximumSpendLineDiscountType),
					Source:      billingentity.CalculatedLineDiscountSource,
				},
			},
		}, lineWithDiscount)
	})

	t.Run("overage 100% discount", func(t *testing.T) {
		lineWithDiscount := l.AddDiscountForOverage(addDiscountInput{
			MaxSpend:               alpacadecimal.NewFromFloat(10000),
			BilledAmountBeforeLine: alpacadecimal.NewFromFloat(10000),
			// Total $10000 => $1000 discount
		})

		require.Equal(t, newDetailedLineInput{
			Amount:   alpacadecimal.NewFromFloat(100),
			Quantity: alpacadecimal.NewFromFloat(10),
			Discounts: []billingentity.LineDiscount{
				{
					Description: lo.ToPtr("Maximum spend discount for charges over 10000"),
					Amount:      alpacadecimal.NewFromFloat(1000),
					Type:        lo.ToPtr(billingentity.MaximumSpendLineDiscountType),
					Source:      billingentity.CalculatedLineDiscountSource,
				},
			},
		}, lineWithDiscount)
	})

	t.Run("overage and 100% discount when hugely over the max spend", func(t *testing.T) {
		lineWithDiscount := l.AddDiscountForOverage(addDiscountInput{
			MaxSpend:               alpacadecimal.NewFromFloat(10000),
			BilledAmountBeforeLine: alpacadecimal.NewFromFloat(20000),
			// Total $10000 => $1000 discount
		})

		require.Equal(t, newDetailedLineInput{
			Amount:   alpacadecimal.NewFromFloat(100),
			Quantity: alpacadecimal.NewFromFloat(10),
			Discounts: []billingentity.LineDiscount{
				{
					Description: lo.ToPtr("Maximum spend discount for charges over 10000"),
					Amount:      alpacadecimal.NewFromFloat(1000),
					Type:        lo.ToPtr(billingentity.MaximumSpendLineDiscountType),
					Source:      billingentity.CalculatedLineDiscountSource,
				},
			},
		}, lineWithDiscount)
	})
}

func TestFindTierForQuantity(t *testing.T) {
	testIn := plan.TieredPrice{
		Tiers: []plan.PriceTier{
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(5)),
				FlatPrice: &plan.PriceTierFlatPrice{
					// 20/unit
					Amount: alpacadecimal.NewFromFloat(100),
				},
			},
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(10)),
				FlatPrice: &plan.PriceTierFlatPrice{
					// 10/unit
					Amount: alpacadecimal.NewFromFloat(150),
				},
			},
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(15)),
				UnitPrice: &plan.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromFloat(10),
				},
			},
			{
				UnitPrice: &plan.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromFloat(5),
				},
			},
		},
	}

	tier, index := findTierForQuantity(testIn, alpacadecimal.NewFromFloat(3))
	require.Equal(t, 0, index)
	require.Equal(t, testIn.Tiers[0], *tier)

	tier, index = findTierForQuantity(testIn, alpacadecimal.NewFromFloat(5))
	require.Equal(t, 0, index)
	require.Equal(t, testIn.Tiers[0], *tier)

	tier, index = findTierForQuantity(testIn, alpacadecimal.NewFromFloat(6))
	require.Equal(t, 1, index)
	require.Equal(t, testIn.Tiers[1], *tier)

	tier, index = findTierForQuantity(testIn, alpacadecimal.NewFromFloat(100))
	require.Equal(t, 3, index)
	require.Equal(t, testIn.Tiers[3], *tier)
}

func getTotalAmountForVolumeTieredPrice(t *testing.T, qty alpacadecimal.Decimal, price plan.TieredPrice) alpacadecimal.Decimal {
	t.Helper()

	total := alpacadecimal.Zero
	err := tieredPriceCalculator(tieredPriceCalculatorInput{
		TieredPrice: price,
		ToQty:       qty,

		FinalizerFn: func(t alpacadecimal.Decimal) error {
			total = t
			return nil
		},
		IntrospectRangesFn: introspectTieredPriceRangesFn(t),
	})

	require.NoError(t, err)

	return total
}

func introspectTieredPriceRangesFn(t *testing.T) func([]tierRange) {
	return func(qtyRanges []tierRange) {
		for _, qtyRange := range qtyRanges {
			t.Logf("From: %s, To: %s, AtBoundary: %t, Tier[idx=%d]: %+v", qtyRange.FromQty.String(), qtyRange.ToQty.String(), qtyRange.AtTierBoundary, qtyRange.TierIndex, qtyRange.Tier)
		}
	}
}

type mockableTieredPriceCalculator struct {
	mock.Mock
}

func (m *mockableTieredPriceCalculator) TierCallbackFn(i tierCallbackInput) error {
	args := m.Called(i)
	return args.Error(0)
}

func (m *mockableTieredPriceCalculator) FinalizerFn(t alpacadecimal.Decimal) error {
	args := m.Called(t)
	return args.Error(0)
}

func TestTieredPriceCalculator(t *testing.T) {
	testIn := plan.TieredPrice{
		Mode: plan.VolumeTieredPrice,
		Tiers: []plan.PriceTier{
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(5)),
				FlatPrice: &plan.PriceTierFlatPrice{
					// 20/unit
					Amount: alpacadecimal.NewFromFloat(100),
				},
			},
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(10)),
				FlatPrice: &plan.PriceTierFlatPrice{
					// 10/unit
					Amount: alpacadecimal.NewFromFloat(50),
				},
			},
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(15)),
				UnitPrice: &plan.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromFloat(10),
				},
			},
			{
				UnitPrice: &plan.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromFloat(5),
				},
			},
		},
	}

	t.Run("totals, no usage", func(t *testing.T) {
		totalAmount := getTotalAmountForVolumeTieredPrice(t, alpacadecimal.NewFromFloat(0), testIn)
		require.Equal(t, alpacadecimal.NewFromFloat(0), totalAmount)
	})

	t.Run("totals, usage in tier 1", func(t *testing.T) {
		totalAmount := getTotalAmountForVolumeTieredPrice(t, alpacadecimal.NewFromFloat(3), testIn)
		require.Equal(t, alpacadecimal.NewFromFloat(100), totalAmount)

		totalAmount = getTotalAmountForVolumeTieredPrice(t, alpacadecimal.NewFromFloat(5), testIn)
		require.Equal(t, alpacadecimal.NewFromFloat(100), totalAmount)
	})

	t.Run("totals, usage in tier 2", func(t *testing.T) {
		totalAmount := getTotalAmountForVolumeTieredPrice(t, alpacadecimal.NewFromFloat(7), testIn)
		require.Equal(t, alpacadecimal.NewFromFloat(100+50), totalAmount)
	})

	t.Run("totals, usage in tier 3", func(t *testing.T) {
		totalAmount := getTotalAmountForVolumeTieredPrice(t, alpacadecimal.NewFromFloat(12), testIn)
		require.Equal(t, alpacadecimal.NewFromFloat(170 /* = 100+50+2*10 */), totalAmount)
	})

	t.Run("totals, usage in tier 4", func(t *testing.T) {
		totalAmount := getTotalAmountForVolumeTieredPrice(t, alpacadecimal.NewFromFloat(22), testIn)
		require.Equal(t, alpacadecimal.NewFromFloat(235 /* = 100+50+10*5+5*7 */), totalAmount)
	})

	t.Run("tier callback, mid tier invocation", func(t *testing.T) {
		callback := mockableTieredPriceCalculator{}

		callback.On("TierCallbackFn", tierCallbackInput{
			Tier:      testIn.Tiers[0],
			TierIndex: 0,

			AtTierBoundary: false,
			Quantity:       alpacadecimal.NewFromFloat(2),
			// The flat price has been already billed for
			PreviousTotalAmount: alpacadecimal.NewFromFloat(100),
		}).Return(nil).Once()

		callback.On("TierCallbackFn", tierCallbackInput{
			Tier:      testIn.Tiers[1],
			TierIndex: 1,

			AtTierBoundary:      true,
			Quantity:            alpacadecimal.NewFromFloat(2),
			PreviousTotalAmount: alpacadecimal.NewFromFloat(100),
		}).Return(nil).Once()

		callback.On("FinalizerFn", alpacadecimal.NewFromFloat(150)).Return(nil).Once()

		require.NoError(t, tieredPriceCalculator(
			tieredPriceCalculatorInput{
				TieredPrice:        testIn,
				FromQty:            alpacadecimal.NewFromFloat(3), // exclusive
				ToQty:              alpacadecimal.NewFromFloat(7), // inclusive
				TierCallbackFn:     callback.TierCallbackFn,
				FinalizerFn:        callback.FinalizerFn,
				IntrospectRangesFn: introspectTieredPriceRangesFn(t),
			},
		),
		)

		callback.AssertExpectations(t)
	})

	t.Run("tier callback, open ended invocation", func(t *testing.T) {
		callback := mockableTieredPriceCalculator{}

		callback.On("TierCallbackFn", tierCallbackInput{
			Tier:      testIn.Tiers[2],
			TierIndex: 2,

			AtTierBoundary: false,
			Quantity:       alpacadecimal.NewFromFloat(3),
			PreviousTotalAmount: alpacadecimal.Sum(
				testIn.Tiers[0].FlatPrice.Amount,
				testIn.Tiers[1].FlatPrice.Amount,
				testIn.Tiers[2].UnitPrice.Amount.Mul(alpacadecimal.NewFromFloat(2)),
			),
		}).Return(nil).Once()

		callback.On("TierCallbackFn", tierCallbackInput{
			Tier:      testIn.Tiers[3],
			TierIndex: 3,

			AtTierBoundary: true,
			Quantity:       alpacadecimal.NewFromFloat(5),
			PreviousTotalAmount: alpacadecimal.Sum(
				testIn.Tiers[0].FlatPrice.Amount,
				testIn.Tiers[1].FlatPrice.Amount,
				testIn.Tiers[2].UnitPrice.Amount.Mul(alpacadecimal.NewFromFloat(5)),
			),
		}).Return(nil).Once()

		callback.On("FinalizerFn",
			alpacadecimal.Sum(
				testIn.Tiers[0].FlatPrice.Amount,
				testIn.Tiers[1].FlatPrice.Amount,
				testIn.Tiers[2].UnitPrice.Amount.Mul(alpacadecimal.NewFromFloat(5)),
				testIn.Tiers[3].UnitPrice.Amount.Mul(alpacadecimal.NewFromFloat(5)),
			)).Return(nil).Once()

		require.NoError(t, tieredPriceCalculator(
			tieredPriceCalculatorInput{
				TieredPrice:        testIn,
				FromQty:            alpacadecimal.NewFromFloat(12), // exclusive
				ToQty:              alpacadecimal.NewFromFloat(20), // inclusive
				TierCallbackFn:     callback.TierCallbackFn,
				FinalizerFn:        callback.FinalizerFn,
				IntrospectRangesFn: introspectTieredPriceRangesFn(t),
			},
		),
		)

		callback.AssertExpectations(t)
	})

	t.Run("tier callback, callback on boundary", func(t *testing.T) {
		callback := mockableTieredPriceCalculator{}

		callback.On("TierCallbackFn", tierCallbackInput{
			Tier:      testIn.Tiers[1],
			TierIndex: 1,

			AtTierBoundary:      true,
			Quantity:            alpacadecimal.NewFromFloat(5),
			PreviousTotalAmount: testIn.Tiers[0].FlatPrice.Amount,
		}).Return(nil).Once()

		callback.On("FinalizerFn",
			alpacadecimal.Sum(
				testIn.Tiers[0].FlatPrice.Amount,
				testIn.Tiers[1].FlatPrice.Amount,
			)).Return(nil).Once()

		require.NoError(t, tieredPriceCalculator(
			tieredPriceCalculatorInput{
				TieredPrice:        testIn,
				FromQty:            alpacadecimal.NewFromFloat(5),  // exclusive
				ToQty:              alpacadecimal.NewFromFloat(10), // inclusive
				TierCallbackFn:     callback.TierCallbackFn,
				FinalizerFn:        callback.FinalizerFn,
				IntrospectRangesFn: introspectTieredPriceRangesFn(t),
			},
		),
		)

		callback.AssertExpectations(t)
	})

	t.Run("tier callback, from/to in same tier", func(t *testing.T) {
		callback := mockableTieredPriceCalculator{}

		callback.On("TierCallbackFn", tierCallbackInput{
			Tier:      testIn.Tiers[1],
			TierIndex: 1,

			AtTierBoundary: false,
			Quantity:       alpacadecimal.NewFromFloat(1),
			PreviousTotalAmount: alpacadecimal.Sum(
				testIn.Tiers[0].FlatPrice.Amount,
				testIn.Tiers[1].FlatPrice.Amount,
			),
		}).Return(nil).Once()

		callback.On("FinalizerFn",
			alpacadecimal.Sum(
				testIn.Tiers[0].FlatPrice.Amount,
				testIn.Tiers[1].FlatPrice.Amount,
			)).Return(nil).Once()

		require.NoError(t, tieredPriceCalculator(
			tieredPriceCalculatorInput{
				TieredPrice:        testIn,
				FromQty:            alpacadecimal.NewFromFloat(6), // exclusive
				ToQty:              alpacadecimal.NewFromFloat(7), // inclusive
				TierCallbackFn:     callback.TierCallbackFn,
				FinalizerFn:        callback.FinalizerFn,
				IntrospectRangesFn: introspectTieredPriceRangesFn(t),
			},
		),
		)

		callback.AssertExpectations(t)
	})

	t.Run("tier callback, from == to, only finalizer is called ", func(t *testing.T) {
		callback := mockableTieredPriceCalculator{}

		callback.On("FinalizerFn", alpacadecimal.Sum(
			testIn.Tiers[0].FlatPrice.Amount,
			testIn.Tiers[1].FlatPrice.Amount,
		)).Return(nil).Once()

		require.NoError(t, tieredPriceCalculator(
			tieredPriceCalculatorInput{
				TieredPrice:        testIn,
				FromQty:            alpacadecimal.NewFromFloat(6), // exclusive
				ToQty:              alpacadecimal.NewFromFloat(6), // inclusive
				TierCallbackFn:     callback.TierCallbackFn,
				FinalizerFn:        callback.FinalizerFn,
				IntrospectRangesFn: introspectTieredPriceRangesFn(t),
			},
		),
		)

		// Nothing should be called
		callback.AssertExpectations(t)
	})
}
