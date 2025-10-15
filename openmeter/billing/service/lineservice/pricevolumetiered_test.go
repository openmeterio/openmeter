package lineservice

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

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
				Mode:  productcatalog.VolumeTieredPrice,
				Tiers: testTiers,
				Commitments: productcatalog.Commitments{
					MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(150)),
				},
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
					ChildUniqueReferenceID: MinSpendChildUniqueReferenceID,
					Period:                 &ubpTestFullPeriod,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Category:               billing.FlatFeeCategoryCommitment,
				},
			},
		})
	})

	t.Run("tiered volume, last price, usage over, min spend", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.VolumeTieredPrice,
				Tiers: testTiers,
				Commitments: productcatalog.Commitments{
					MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
				},
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
				Mode:  productcatalog.VolumeTieredPrice,
				Tiers: testTiers,
				Commitments: productcatalog.Commitments{
					MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(150)),
				},
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
					Period:                 &ubpTestFullPeriod,
					ChildUniqueReferenceID: MinSpendChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Category:               billing.FlatFeeCategoryCommitment,
				},
			},
		})
	})

	t.Run("tiered volume, last price, usage less equals min spend", func(t *testing.T) {
		runUBPTest(t, ubpCalculationTestCase{
			price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.VolumeTieredPrice,
				Tiers: testTiers,
				Commitments: productcatalog.Commitments{
					MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
				},
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
				Mode:  productcatalog.VolumeTieredPrice,
				Tiers: testTiers,
				Commitments: productcatalog.Commitments{
					MaximumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
				},
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
				Mode:  productcatalog.VolumeTieredPrice,
				Tiers: testTiers,
				Commitments: productcatalog.Commitments{
					MaximumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(125)),
				},
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
					AmountDiscounts: []billing.AmountLineDiscountManaged{
						{
							AmountLineDiscount: billing.AmountLineDiscount{
								Amount: alpacadecimal.NewFromFloat(25),
								LineDiscountBase: billing.LineDiscountBase{
									Description:            lo.ToPtr("Maximum spend discount for charges over 125"),
									ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
									Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
								},
							},
						},
					},
				},
			},
		})
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

	pricer := volumeTieredPricer{}

	res, err := pricer.findTierForQuantity(testIn, alpacadecimal.NewFromFloat(3))
	require.NoError(t, err)
	require.Equal(t, findTierForQuantityResult{
		Tier:  &testIn.Tiers[0],
		Index: 0,
	}, res)

	res, err = pricer.findTierForQuantity(testIn, alpacadecimal.NewFromFloat(5))
	require.NoError(t, err)
	require.Equal(t, findTierForQuantityResult{
		Tier:  &testIn.Tiers[0],
		Index: 0,
	}, res)

	res, err = pricer.findTierForQuantity(testIn, alpacadecimal.NewFromFloat(6))
	require.NoError(t, err)
	require.Equal(t, findTierForQuantityResult{
		Tier:  &testIn.Tiers[1],
		Index: 1,
	}, res)

	res, err = pricer.findTierForQuantity(testIn, alpacadecimal.NewFromFloat(100))
	require.NoError(t, err)
	require.Equal(t, findTierForQuantityResult{
		Tier:  &testIn.Tiers[3],
		Index: 3,
	}, res)
}
