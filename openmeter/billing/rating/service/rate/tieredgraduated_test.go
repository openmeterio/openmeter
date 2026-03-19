package rate_test

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/rating/service/rate"
	"github.com/openmeterio/openmeter/openmeter/billing/rating/service/testutil"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestTieredGraduatedCalculation(t *testing.T) {
	testTiers := []productcatalog.PriceTier{
		{
			UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(5)),
			FlatPrice: &productcatalog.PriceTierFlatPrice{
				// 20/unit
				Amount: alpacadecimal.NewFromFloat(100),
			},
			UnitPrice: &productcatalog.PriceTierUnitPrice{
				Amount: alpacadecimal.NewFromFloat(0),
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
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.GraduatedTieredPrice,
				Tiers: testTiers,
			}),
			LineMode: testutil.MidPeriodSplitLineMode,
			Usage: testutil.FeatureUsageResponse{
				PreLinePeriodQty: alpacadecimal.NewFromFloat(7),
				LinePeriodQty:    alpacadecimal.NewFromFloat(1),
			},
			Expect: rating.DetailedLines{},
		})
	})

	t.Run("tiered graduated, last price, no usage", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.GraduatedTieredPrice,
				Tiers: testTiers,
			}),
			LineMode: testutil.LastInPeriodSplitLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			Expect: rating.DetailedLines{
				{
					Name:                   "feature: usage price for tier 1",
					PerUnitAmount:          alpacadecimal.NewFromFloat(0),
					Quantity:               alpacadecimal.NewFromFloat(0),
					ChildUniqueReferenceID: "graduated-tiered-1-price-usage",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
			},
		})
	})

	t.Run("tiered graduated, single period multiple tier usage", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.GraduatedTieredPrice,
				Tiers: testTiers,
			}),
			LineMode: testutil.SinglePerPeriodLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(22),
			},
			Expect: rating.DetailedLines{
				{
					Name:                   "feature: usage price for tier 1",
					PerUnitAmount:          alpacadecimal.NewFromFloat(0),
					Quantity:               alpacadecimal.NewFromFloat(5),
					ChildUniqueReferenceID: "graduated-tiered-1-price-usage",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
				{
					Name:                   "feature: flat price for tier 1",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: "graduated-tiered-1-flat-price",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(100),
						Total:  alpacadecimal.NewFromFloat(100),
					},
				},
				{
					Name:                   "feature: flat price for tier 2",
					PerUnitAmount:          alpacadecimal.NewFromFloat(50),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: "graduated-tiered-2-flat-price",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(50),
						Total:  alpacadecimal.NewFromFloat(50),
					},
				},
				{
					Name:                   "feature: usage price for tier 3",
					PerUnitAmount:          alpacadecimal.NewFromFloat(5),
					Quantity:               alpacadecimal.NewFromFloat(5),
					ChildUniqueReferenceID: "graduated-tiered-3-price-usage",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(25),
						Total:  alpacadecimal.NewFromFloat(25),
					},
				},
				{
					Name:                   "feature: usage price for tier 4",
					PerUnitAmount:          alpacadecimal.NewFromFloat(1),
					Quantity:               alpacadecimal.NewFromFloat(7),
					ChildUniqueReferenceID: "graduated-tiered-4-price-usage",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(7),
						Total:  alpacadecimal.NewFromFloat(7),
					},
				},
			},
		})
	})

	t.Run("tiered graduated, mid period, multiple tier usage", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.GraduatedTieredPrice,
				Tiers: testTiers,
			}),
			LineMode: testutil.SinglePerPeriodLineMode,
			Usage: testutil.FeatureUsageResponse{
				PreLinePeriodQty: alpacadecimal.NewFromFloat(12),
				LinePeriodQty:    alpacadecimal.NewFromFloat(10), // total usage is at 22
			},
			Expect: rating.DetailedLines{
				{
					Name:                   "feature: usage price for tier 3",
					PerUnitAmount:          alpacadecimal.NewFromFloat(5),
					Quantity:               alpacadecimal.NewFromFloat(3),
					ChildUniqueReferenceID: "graduated-tiered-3-price-usage",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(15),
						Total:  alpacadecimal.NewFromFloat(15),
					},
				},
				{
					Name:                   "feature: usage price for tier 4",
					PerUnitAmount:          alpacadecimal.NewFromFloat(1),
					Quantity:               alpacadecimal.NewFromFloat(7),
					ChildUniqueReferenceID: "graduated-tiered-4-price-usage",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(7),
						Total:  alpacadecimal.NewFromFloat(7),
					},
				},
			},
		})
	})

	// Minimum spend

	t.Run("tiered graduated, last line, no usage, minimum price set", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.GraduatedTieredPrice,
				Tiers: testTiers,
				Commitments: productcatalog.Commitments{
					MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(1000)),
				},
			}),
			LineMode: testutil.LastInPeriodSplitLineMode,
			Usage: testutil.FeatureUsageResponse{
				PreLinePeriodQty: alpacadecimal.NewFromFloat(0),
				LinePeriodQty:    alpacadecimal.NewFromFloat(0),
			},
			PreviousBilledAmount: alpacadecimal.NewFromFloat(100), // Due to flat fee of 100 for tier 1
			Expect: rating.DetailedLines{
				{
					Name:                   "feature: usage price for tier 1",
					PerUnitAmount:          alpacadecimal.NewFromFloat(0),
					Quantity:               alpacadecimal.NewFromFloat(0),
					ChildUniqueReferenceID: "graduated-tiered-1-price-usage",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				},
				{
					Name: "feature: minimum spend",
					// We have a flat fee of 100 for tier 1, and given that it was invoiced as part of the previous split we need to remove
					// that from the charge.
					PerUnitAmount:          alpacadecimal.NewFromFloat(900),
					Quantity:               alpacadecimal.NewFromFloat(1),
					Period:                 lo.ToPtr(testutil.TestFullPeriod.ToClosedPeriod()),
					ChildUniqueReferenceID: rating.MinSpendChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Category:               billing.FlatFeeCategoryCommitment,
					Totals: totals.Totals{
						ChargesTotal: alpacadecimal.NewFromFloat(900),
						Total:        alpacadecimal.NewFromFloat(900),
					},
				},
			},
		})
	})

	t.Run("tiered graduated, last line, previous usage exists, no current usage, minimum price set", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.GraduatedTieredPrice,
				Tiers: testTiers,
				Commitments: productcatalog.Commitments{
					MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(1000)),
				},
			}),
			LineMode: testutil.LastInPeriodSplitLineMode,
			Usage: testutil.FeatureUsageResponse{
				PreLinePeriodQty: alpacadecimal.NewFromFloat(2),
				LinePeriodQty:    alpacadecimal.NewFromFloat(0),
			},
			PreviousBilledAmount: alpacadecimal.NewFromFloat(100), // Due to flat fee of 100 for tier 1
			Expect: rating.DetailedLines{
				{
					Name:                   "feature: minimum spend",
					PerUnitAmount:          alpacadecimal.NewFromFloat(900),
					Quantity:               alpacadecimal.NewFromFloat(1),
					Period:                 lo.ToPtr(testutil.TestFullPeriod.ToClosedPeriod()),
					ChildUniqueReferenceID: rating.MinSpendChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Category:               billing.FlatFeeCategoryCommitment,
					Totals: totals.Totals{
						ChargesTotal: alpacadecimal.NewFromFloat(900),
						Total:        alpacadecimal.NewFromFloat(900),
					},
				},
			},
		})
	})

	t.Run("tiered graduated, mid line, no usage, minimum price set", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.GraduatedTieredPrice,
				Tiers: testTiers,
				Commitments: productcatalog.Commitments{
					MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(1000)),
				},
			}),
			LineMode: testutil.MidPeriodSplitLineMode,
			Usage: testutil.FeatureUsageResponse{
				PreLinePeriodQty: alpacadecimal.NewFromFloat(2),
				LinePeriodQty:    alpacadecimal.NewFromFloat(0),
			},
			Expect: rating.DetailedLines{},
		})
	})

	// Maximum spend
	t.Run("tiered graduated, mid period, multiple tier usage, maximum spend set mid tier 2/3", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.GraduatedTieredPrice,
				Tiers: testTiers,
				Commitments: productcatalog.Commitments{
					MaximumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(170)),
				},
			}),
			LineMode: testutil.MidPeriodSplitLineMode,
			Usage: testutil.FeatureUsageResponse{
				PreLinePeriodQty: alpacadecimal.NewFromFloat(12),
				LinePeriodQty:    alpacadecimal.NewFromFloat(10), // total usage is at 22
			},

			// Total previous usage due to the PreLinePeriodQty:
			// tier 1: $100 flat
			// tier 2: $50 flat
			// tier 3: 2*$5 = $10 usage
			// total: $160
			PreviousBilledAmount: alpacadecimal.NewFromFloat(160),

			Expect: rating.DetailedLines{
				{
					Name:                   "feature: usage price for tier 3",
					PerUnitAmount:          alpacadecimal.NewFromFloat(5),
					Quantity:               alpacadecimal.NewFromFloat(3),
					ChildUniqueReferenceID: "graduated-tiered-3-price-usage",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					AmountDiscounts: []billing.AmountLineDiscountManaged{
						{
							AmountLineDiscount: billing.AmountLineDiscount{
								Amount: alpacadecimal.NewFromFloat(5),
								LineDiscountBase: billing.LineDiscountBase{
									Description:            lo.ToPtr("Maximum spend discount for charges over 170"),
									ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
									Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
								},
							},
						},
					},
					Totals: totals.Totals{
						Amount:         alpacadecimal.NewFromFloat(15),
						DiscountsTotal: alpacadecimal.NewFromFloat(5),
						Total:          alpacadecimal.NewFromFloat(10),
					},
				},
				{
					Name:                   "feature: usage price for tier 4",
					PerUnitAmount:          alpacadecimal.NewFromFloat(1),
					Quantity:               alpacadecimal.NewFromFloat(7),
					ChildUniqueReferenceID: "graduated-tiered-4-price-usage",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					AmountDiscounts: []billing.AmountLineDiscountManaged{
						{
							AmountLineDiscount: billing.AmountLineDiscount{
								Amount: alpacadecimal.NewFromFloat(7),
								LineDiscountBase: billing.LineDiscountBase{
									Description:            lo.ToPtr("Maximum spend discount for charges over 170"),
									ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
									Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
								},
							},
						},
					},
					Totals: totals.Totals{
						Amount:         alpacadecimal.NewFromFloat(7),
						DiscountsTotal: alpacadecimal.NewFromFloat(7),
						Total:          alpacadecimal.NewFromFloat(0),
					},
				},
			},
		})
	})
}

func TestTieredPriceCalculator(t *testing.T) {
	currency := lo.Must(currencyx.Code(currency.USD).Calculator())

	graduatedTiered := rate.GraduatedTiered{}

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
		// If there's no usage in the first tier we need to bill the flat fee regardless.
		totalAmount := getTotalAmountForGraduatedTieredPrice(t, alpacadecimal.NewFromFloat(0), testIn)
		require.Equal(t, alpacadecimal.NewFromFloat(100), totalAmount)
	})

	t.Run("totals, usage in tier 1", func(t *testing.T) {
		totalAmount := getTotalAmountForGraduatedTieredPrice(t, alpacadecimal.NewFromFloat(3), testIn)
		require.Equal(t, alpacadecimal.NewFromFloat(100), totalAmount)

		totalAmount = getTotalAmountForGraduatedTieredPrice(t, alpacadecimal.NewFromFloat(5), testIn)
		require.Equal(t, alpacadecimal.NewFromFloat(100), totalAmount)
	})

	t.Run("totals, usage in tier 2", func(t *testing.T) {
		totalAmount := getTotalAmountForGraduatedTieredPrice(t, alpacadecimal.NewFromFloat(5.001), testIn)
		require.Equal(t, alpacadecimal.NewFromFloat(100+50), totalAmount)

		totalAmount = getTotalAmountForGraduatedTieredPrice(t, alpacadecimal.NewFromFloat(7), testIn)
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

		callback.On("TierCallbackFn", rate.TierCallbackInput{
			Tier:      testIn.Tiers[0],
			TierIndex: 0,

			AtTierBoundary: false,
			Quantity:       alpacadecimal.NewFromFloat(2),
			// The flat price has been already billed for
			PreviousTotalAmount: alpacadecimal.NewFromFloat(100),
		}).Return(nil).Once()

		callback.On("TierCallbackFn", rate.TierCallbackInput{
			Tier:      testIn.Tiers[1],
			TierIndex: 1,

			AtTierBoundary:      true,
			Quantity:            alpacadecimal.NewFromFloat(2),
			PreviousTotalAmount: alpacadecimal.NewFromFloat(100),
		}).Return(nil).Once()

		callback.On("FinalizerFn", alpacadecimal.NewFromFloat(150)).Return(nil).Once()

		require.NoError(t, graduatedTiered.TieredPriceCalculator(
			rate.TieredPriceCalculatorInput{
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

		callback.On("TierCallbackFn", rate.TierCallbackInput{
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

		callback.On("TierCallbackFn", rate.TierCallbackInput{
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

		require.NoError(t, graduatedTiered.TieredPriceCalculator(
			rate.TieredPriceCalculatorInput{
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

		callback.On("TierCallbackFn", rate.TierCallbackInput{
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

		require.NoError(t, graduatedTiered.TieredPriceCalculator(
			rate.TieredPriceCalculatorInput{
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

		callback.On("TierCallbackFn", rate.TierCallbackInput{
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

		require.NoError(t, graduatedTiered.TieredPriceCalculator(
			rate.TieredPriceCalculatorInput{
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

		require.NoError(t, graduatedTiered.TieredPriceCalculator(
			rate.TieredPriceCalculatorInput{
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

func getTotalAmountForGraduatedTieredPrice(t *testing.T, qty alpacadecimal.Decimal, tieredPrice productcatalog.TieredPrice) alpacadecimal.Decimal {
	t.Helper()

	graduatedTiered := rate.GraduatedTiered{}

	total := alpacadecimal.Zero
	err := graduatedTiered.TieredPriceCalculator(rate.TieredPriceCalculatorInput{
		TieredPrice: tieredPrice,
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

func introspectTieredPriceRangesFn(t *testing.T) func([]rate.TierRange) {
	return func(qtyRanges []rate.TierRange) {
		for _, qtyRange := range qtyRanges {
			t.Logf("From: %s, To: %s, AtBoundary: %t, Tier[idx=%d]: %+v", qtyRange.FromQty.String(), qtyRange.ToQty.String(), qtyRange.AtTierBoundary, qtyRange.TierIndex, qtyRange.Tier)
		}
	}
}

type mockableTieredPriceCalculator struct {
	mock.Mock
}

func (m *mockableTieredPriceCalculator) TierCallbackFn(i rate.TierCallbackInput) error {
	args := m.Called(i)
	return args.Error(0)
}

func (m *mockableTieredPriceCalculator) FinalizerFn(t alpacadecimal.Decimal) error {
	args := m.Called(t)
	return args.Error(0)
}
