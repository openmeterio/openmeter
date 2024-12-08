package lineservice

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
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
				UsageBased: billing.UsageBasedLine{
					Price: tc.price,
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
			price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			lineMode: singlePerPeriodLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: FlatPriceChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InAdvancePaymentTerm,
				},
			},
		})
	})

	t.Run("flat price, in advance, usage present", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			lineMode: singlePerPeriodLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: FlatPriceChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InAdvancePaymentTerm,
				},
			},
		})
	})

	t.Run("flat price, in advance, usage present, mid period", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
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
			price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
			lineMode: singlePerPeriodLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: FlatPriceChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("flat price, in arrears, usage present, mid period line", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
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
			price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(100),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: FlatPriceChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})
}

func TestUnitPriceCalculation(t *testing.T) {
	t.Run("unit price, no usage", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
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
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
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
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: UnitPriceMinSpendChildUniqueReferenceID,
					Period:                 &ubpTestFullPeriod,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Category:               billing.FlatFeeCategoryCommitment,
				},
			},
		})
	})

	// Min spend is always billed in arrears => we are not billing it in advance
	t.Run("no usage, not the last line in period, min spend set", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
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
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
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
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: UnitPriceMinSpendChildUniqueReferenceID,
					Period:                 &ubpTestFullPeriod,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Category:               billing.FlatFeeCategoryCommitment,
				},
			},
		})
	})

	// Usage is billed regardless of line position
	t.Run("usage present", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(100),
			}),
			lineMode: singlePerPeriodLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(10),
					ChildUniqueReferenceID: UnitPriceUsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("usage present, mid line", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(100),
			}),
			lineMode: midPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(10),
					ChildUniqueReferenceID: UnitPriceUsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	// Max spend is always honored
	t.Run("usage present, max spend set, but not hit", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
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
					PerUnitAmount:          alpacadecimal.NewFromFloat(10),
					Quantity:               alpacadecimal.NewFromFloat(10),
					ChildUniqueReferenceID: UnitPriceUsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("usage present, max spend set and hit", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
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
					PerUnitAmount:          alpacadecimal.NewFromFloat(10),
					Quantity:               alpacadecimal.NewFromFloat(5),
					ChildUniqueReferenceID: UnitPriceUsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Discounts: []billing.LineDiscount{
						{
							Description:            lo.ToPtr("Maximum spend discount for charges over 100"),
							Amount:                 alpacadecimal.NewFromFloat(20),
							ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
						},
					},
				},
			},
		})
	})
}

func TestTieredVolumeCalculation(t *testing.T) {
	testTiers := []productcatalog.PriceTier{
		{
			UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(5)),
			FlatPrice: &productcatalog.PriceTierFlatPrice{
				// 20/unit
				Amount: alpacadecimal.NewFromFloat(100),
			},
		},
		{
			UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(10)),
			FlatPrice: &productcatalog.PriceTierFlatPrice{
				// 10/unit
				Amount: alpacadecimal.NewFromFloat(150),
			},
		},
		{
			UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(15)),
			UnitPrice: &productcatalog.PriceTierUnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			},
		},
		{
			UnitPrice: &productcatalog.PriceTierUnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			},
		},
	}

	t.Run("tiered volume, mid price", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.VolumeTieredPrice,
				Tiers: testTiers,
			}),
			lineMode: midPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			expect: newDetailedLinesInput{},
		})
	})

	t.Run("tiered volume, last price, no usage", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.VolumeTieredPrice,
				Tiers: testTiers,
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: flat price for tier 1",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: VolumeFlatPriceChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("tiered volume, ubp first tier, no usage", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode: productcatalog.VolumeTieredPrice,
				Tiers: []productcatalog.PriceTier{
					{
						UnitPrice: &productcatalog.PriceTierUnitPrice{
							Amount: alpacadecimal.NewFromFloat(5),
						},
					},
				},
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			expect: newDetailedLinesInput{},
		})
	})

	t.Run("tiered volume, last price, usage present, tier1 mid", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.VolumeTieredPrice,
				Tiers: testTiers,
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(3),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: flat price for tier 1",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: VolumeFlatPriceChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("tiered volume, last price, usage present, tier1 top", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.VolumeTieredPrice,
				Tiers: testTiers,
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(5),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: flat price for tier 1",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: VolumeFlatPriceChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("tiered volume, last price, usage present, tier3 almost full", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.VolumeTieredPrice,
				Tiers: testTiers,
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(14),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: unit price for tier 3",
					PerUnitAmount:          alpacadecimal.NewFromFloat(10),
					Quantity:               alpacadecimal.NewFromFloat(14),
					ChildUniqueReferenceID: VolumeUnitPriceChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("tiered volume, last price, usage present, tier3 full", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.VolumeTieredPrice,
				Tiers: testTiers,
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(15),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: unit price for tier 3",
					PerUnitAmount:          alpacadecimal.NewFromFloat(10),
					Quantity:               alpacadecimal.NewFromFloat(15),
					ChildUniqueReferenceID: VolumeUnitPriceChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("tiered volume, last price, usage present, tier3 just passed", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.VolumeTieredPrice,
				Tiers: testTiers,
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(16),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: unit price for tier 4",
					PerUnitAmount:          alpacadecimal.NewFromFloat(5),
					Quantity:               alpacadecimal.NewFromFloat(16),
					ChildUniqueReferenceID: VolumeUnitPriceChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("tiered volume, last price, usage present, tier4", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.VolumeTieredPrice,
				Tiers: testTiers,
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(100),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: unit price for tier 4",
					PerUnitAmount:          alpacadecimal.NewFromFloat(5),
					Quantity:               alpacadecimal.NewFromFloat(100),
					ChildUniqueReferenceID: VolumeUnitPriceChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	// Minimum spend

	t.Run("tiered volume, last price, no usage, min spend", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:          productcatalog.VolumeTieredPrice,
				Tiers:         testTiers,
				MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(150)),
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: flat price for tier 1",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: VolumeFlatPriceChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
				{
					Name:                   "feature: minimum spend",
					PerUnitAmount:          alpacadecimal.NewFromFloat(50),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: VolumeMinSpendChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Category:               billing.FlatFeeCategoryCommitment,
				},
			},
		})
	})

	t.Run("tiered volume, last price, usage over, min spend", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:          productcatalog.VolumeTieredPrice,
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
					PerUnitAmount:          alpacadecimal.NewFromFloat(5),
					Quantity:               alpacadecimal.NewFromFloat(100),
					ChildUniqueReferenceID: VolumeUnitPriceChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("tiered volume, last price, usage less than min spend", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:          productcatalog.VolumeTieredPrice,
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
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: VolumeFlatPriceChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
				{
					Name:                   "feature: minimum spend",
					PerUnitAmount:          alpacadecimal.NewFromFloat(50),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: VolumeMinSpendChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Category:               billing.FlatFeeCategoryCommitment,
				},
			},
		})
	})

	t.Run("tiered volume, last price, usage less equals min spend", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:          productcatalog.VolumeTieredPrice,
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
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: VolumeFlatPriceChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	// Maximum spend
	t.Run("tiered volume, first price, usage eq max spend", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:          productcatalog.VolumeTieredPrice,
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
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: VolumeFlatPriceChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("tiered volume, first price, usage above max spend, max spend is not at tier boundary ", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:          productcatalog.VolumeTieredPrice,
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
					PerUnitAmount:          alpacadecimal.NewFromFloat(150),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: VolumeFlatPriceChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Discounts: []billing.LineDiscount{
						{
							Description:            lo.ToPtr("Maximum spend discount for charges over 125"),
							Amount:                 alpacadecimal.NewFromFloat(25),
							ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
						},
					},
				},
			},
		})
	})
}

func TestTieredGraduatedCalculation(t *testing.T) {
	testTiers := []productcatalog.PriceTier{
		{
			UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(5)),
			FlatPrice: &productcatalog.PriceTierFlatPrice{
				// 20/unit
				Amount: alpacadecimal.NewFromFloat(100),
			},
		},
		{
			UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(10)),
			FlatPrice: &productcatalog.PriceTierFlatPrice{
				// 10/unit
				Amount: alpacadecimal.NewFromFloat(50),
			},
		},
		{
			UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(15)),
			UnitPrice: &productcatalog.PriceTierUnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			},
		},
		{
			UnitPrice: &productcatalog.PriceTierUnitPrice{
				Amount: alpacadecimal.NewFromFloat(1),
			},
		},
	}

	t.Run("tiered graduated, mid price, flat only => no lines are output", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.GraduatedTieredPrice,
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

	t.Run("tiered graduated, last price, no usage", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.GraduatedTieredPrice,
				Tiers: testTiers,
			}),
			lineMode: lastInPeriodSplitLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			expect: newDetailedLinesInput{},
		})
	})

	t.Run("tiered graduated, single period multiple tier usage", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.GraduatedTieredPrice,
				Tiers: testTiers,
			}),
			lineMode: singlePerPeriodLineMode,
			usage: featureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(22),
			},
			expect: newDetailedLinesInput{
				{
					Name:                   "feature: flat price for tier 1",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: "graduated-tiered-1-flat-price",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
				{
					Name:                   "feature: flat price for tier 2",
					PerUnitAmount:          alpacadecimal.NewFromFloat(50),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: "graduated-tiered-2-flat-price",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
				{
					Name:                   "feature: usage price for tier 3",
					PerUnitAmount:          alpacadecimal.NewFromFloat(5),
					Quantity:               alpacadecimal.NewFromFloat(5),
					ChildUniqueReferenceID: "graduated-tiered-3-price-usage",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
				{
					Name:                   "feature: usage price for tier 4",
					PerUnitAmount:          alpacadecimal.NewFromFloat(1),
					Quantity:               alpacadecimal.NewFromFloat(7),
					ChildUniqueReferenceID: "graduated-tiered-4-price-usage",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("tiered graduated, mid period, multiple tier usage", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.GraduatedTieredPrice,
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
					PerUnitAmount:          alpacadecimal.NewFromFloat(5),
					Quantity:               alpacadecimal.NewFromFloat(3),
					ChildUniqueReferenceID: "graduated-tiered-3-price-usage",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
				{
					Name:                   "feature: usage price for tier 4",
					PerUnitAmount:          alpacadecimal.NewFromFloat(1),
					Quantity:               alpacadecimal.NewFromFloat(7),
					ChildUniqueReferenceID: "graduated-tiered-4-price-usage",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	// Minimum spend

	t.Run("tiered graduated, last line, no usage, minimum price set", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:          productcatalog.GraduatedTieredPrice,
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
					PerUnitAmount:          alpacadecimal.NewFromFloat(1000),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: GraduatedMinSpendChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Category:               billing.FlatFeeCategoryCommitment,
				},
			},
		})
	})

	t.Run("tiered graduated, last line, no usage, minimum price set", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:          productcatalog.GraduatedTieredPrice,
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
					PerUnitAmount:          alpacadecimal.NewFromFloat(900),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: GraduatedMinSpendChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Category:               billing.FlatFeeCategoryCommitment,
				},
			},
		})
	})

	t.Run("tiered graduated, mid line, no usage, minimum price set", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:          productcatalog.GraduatedTieredPrice,
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
	t.Run("tiered graduated, mid period, multiple tier usage, maximum spend set mid tier 2/3", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:          productcatalog.GraduatedTieredPrice,
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
					PerUnitAmount:          alpacadecimal.NewFromFloat(5),
					Quantity:               alpacadecimal.NewFromFloat(3),
					ChildUniqueReferenceID: "graduated-tiered-3-price-usage",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Discounts: []billing.LineDiscount{
						{
							Description:            lo.ToPtr("Maximum spend discount for charges over 170"),
							Amount:                 alpacadecimal.NewFromFloat(5),
							ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
						},
					},
				},
				{
					Name:                   "feature: usage price for tier 4",
					PerUnitAmount:          alpacadecimal.NewFromFloat(1),
					Quantity:               alpacadecimal.NewFromFloat(7),
					ChildUniqueReferenceID: "graduated-tiered-4-price-usage",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Discounts: []billing.LineDiscount{
						{
							Description:            lo.ToPtr("Maximum spend discount for charges over 170"),
							Amount:                 alpacadecimal.NewFromFloat(7),
							ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
						},
					},
				},
			},
		})
	})
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
			Discounts: []billing.LineDiscount{
				{
					Description:            lo.ToPtr("Maximum spend discount for charges over 10000"),
					Amount:                 alpacadecimal.NewFromFloat(0.01),
					ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
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
			Discounts: []billing.LineDiscount{
				{
					Description:            lo.ToPtr("Maximum spend discount for charges over 10000"),
					Amount:                 alpacadecimal.NewFromFloat(600),
					ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
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
			Discounts: []billing.LineDiscount{
				{
					Description:            lo.ToPtr("Maximum spend discount for charges over 10000"),
					Amount:                 alpacadecimal.NewFromFloat(1000),
					ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
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
			Discounts: []billing.LineDiscount{
				{
					Description:            lo.ToPtr("Maximum spend discount for charges over 10000"),
					Amount:                 alpacadecimal.NewFromFloat(1000),
					ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
				},
			},
		}, lineWithDiscount)
	})
}

func TestFindTierForQuantity(t *testing.T) {
	testIn := productcatalog.TieredPrice{
		Tiers: []productcatalog.PriceTier{
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(5)),
				FlatPrice: &productcatalog.PriceTierFlatPrice{
					// 20/unit
					Amount: alpacadecimal.NewFromFloat(100),
				},
			},
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(10)),
				FlatPrice: &productcatalog.PriceTierFlatPrice{
					// 10/unit
					Amount: alpacadecimal.NewFromFloat(150),
				},
			},
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(15)),
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromFloat(10),
				},
			},
			{
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromFloat(5),
				},
			},
		},
	}

	res, err := findTierForQuantity(testIn, alpacadecimal.NewFromFloat(3))
	require.NoError(t, err)
	require.Equal(t, findTierForQuantityResult{
		Tier:  &testIn.Tiers[0],
		Index: 0,
	}, res)

	res, err = findTierForQuantity(testIn, alpacadecimal.NewFromFloat(5))
	require.NoError(t, err)
	require.Equal(t, findTierForQuantityResult{
		Tier:  &testIn.Tiers[0],
		Index: 0,
	}, res)

	res, err = findTierForQuantity(testIn, alpacadecimal.NewFromFloat(6))
	require.NoError(t, err)
	require.Equal(t, findTierForQuantityResult{
		Tier:  &testIn.Tiers[1],
		Index: 1,
	}, res)

	res, err = findTierForQuantity(testIn, alpacadecimal.NewFromFloat(100))
	require.NoError(t, err)
	require.Equal(t, findTierForQuantityResult{
		Tier:  &testIn.Tiers[3],
		Index: 3,
	}, res)
}

func getTotalAmountForGraduatedTieredPrice(t *testing.T, qty alpacadecimal.Decimal, price productcatalog.TieredPrice) alpacadecimal.Decimal {
	t.Helper()

	total := alpacadecimal.Zero
	err := tieredPriceCalculator(tieredPriceCalculatorInput{
		TieredPrice: price,
		ToQty:       qty,
		Currency:    lo.Must(currencyx.Code(currency.USD).Calculator()),

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
	currency := lo.Must(currencyx.Code(currency.USD).Calculator())

	testIn := productcatalog.TieredPrice{
		Mode: productcatalog.GraduatedTieredPrice,
		Tiers: []productcatalog.PriceTier{
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(5)),
				FlatPrice: &productcatalog.PriceTierFlatPrice{
					// 20/unit
					Amount: alpacadecimal.NewFromFloat(100),
				},
			},
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(10)),
				FlatPrice: &productcatalog.PriceTierFlatPrice{
					// 10/unit
					Amount: alpacadecimal.NewFromFloat(50),
				},
			},
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(15)),
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromFloat(10),
				},
			},
			{
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromFloat(5),
				},
			},
		},
	}

	t.Run("totals, no usage", func(t *testing.T) {
		totalAmount := getTotalAmountForGraduatedTieredPrice(t, alpacadecimal.NewFromFloat(0), testIn)
		require.Equal(t, alpacadecimal.NewFromFloat(0), totalAmount)
	})

	t.Run("totals, usage in tier 1", func(t *testing.T) {
		totalAmount := getTotalAmountForGraduatedTieredPrice(t, alpacadecimal.NewFromFloat(3), testIn)
		require.Equal(t, alpacadecimal.NewFromFloat(100), totalAmount)

		totalAmount = getTotalAmountForGraduatedTieredPrice(t, alpacadecimal.NewFromFloat(5), testIn)
		require.Equal(t, alpacadecimal.NewFromFloat(100), totalAmount)
	})

	t.Run("totals, usage in tier 2", func(t *testing.T) {
		totalAmount := getTotalAmountForGraduatedTieredPrice(t, alpacadecimal.NewFromFloat(7), testIn)
		require.Equal(t, alpacadecimal.NewFromFloat(100+50), totalAmount)
	})

	t.Run("totals, usage in tier 3", func(t *testing.T) {
		totalAmount := getTotalAmountForGraduatedTieredPrice(t, alpacadecimal.NewFromFloat(12), testIn)
		require.Equal(t, alpacadecimal.NewFromFloat(170 /* = 100+50+2*10 */), totalAmount)
	})

	t.Run("totals, usage in tier 4", func(t *testing.T) {
		totalAmount := getTotalAmountForGraduatedTieredPrice(t, alpacadecimal.NewFromFloat(22), testIn)
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
				TieredPrice: testIn,
				FromQty:     alpacadecimal.NewFromFloat(3), // exclusive
				ToQty:       alpacadecimal.NewFromFloat(7), // inclusive
				Currency:    currency,

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
				TieredPrice: testIn,
				FromQty:     alpacadecimal.NewFromFloat(12), // exclusive
				ToQty:       alpacadecimal.NewFromFloat(20), // inclusive
				Currency:    currency,

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
				TieredPrice: testIn,
				FromQty:     alpacadecimal.NewFromFloat(5),  // exclusive
				ToQty:       alpacadecimal.NewFromFloat(10), // inclusive
				Currency:    currency,

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
				TieredPrice: testIn,
				FromQty:     alpacadecimal.NewFromFloat(6), // exclusive
				ToQty:       alpacadecimal.NewFromFloat(7), // inclusive
				Currency:    currency,

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
				TieredPrice: testIn,
				FromQty:     alpacadecimal.NewFromFloat(6), // exclusive
				ToQty:       alpacadecimal.NewFromFloat(6), // inclusive
				Currency:    currency,

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
