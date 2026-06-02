package service_test

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/rating/service/testutil"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestGenerateDetailedLinesMinimumCommitmentOptions(t *testing.T) {
	t.Run("minimum commitment applied by default", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
				Commitments: productcatalog.Commitments{
					MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
				},
			}),
			LineMode: testutil.SinglePerPeriodLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			Expect: rating.DetailedLines{
				{
					Name:                   "feature: minimum spend",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: rating.MinSpendChildUniqueReferenceID,
					Period:                 lo.ToPtr(testutil.TestFullPeriod),
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Category:               stddetailedline.CategoryCommitment,
					Totals: totals.Totals{
						ChargesTotal: alpacadecimal.NewFromFloat(100),
						Total:        alpacadecimal.NewFromFloat(100),
					},
				},
			},
		})
	})

	t.Run("minimum commitment ignored when requested", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
				Commitments: productcatalog.Commitments{
					MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
				},
			}),
			LineMode: testutil.SinglePerPeriodLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(0),
			},
			Options: []rating.GenerateDetailedLinesOption{
				rating.WithMinimumCommitmentIgnored(),
			},
			Expect: rating.DetailedLines{},
		})
	})

	t.Run("maximum spend still applies when minimum commitment is ignored", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
				Commitments: productcatalog.Commitments{
					MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
					MaximumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
				},
			}),
			LineMode: testutil.MidPeriodSplitLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty:    alpacadecimal.NewFromFloat(5),
				PreLinePeriodQty: alpacadecimal.NewFromFloat(7),
			},
			PreviousBilledAmount: alpacadecimal.NewFromFloat(70),
			Options: []rating.GenerateDetailedLinesOption{
				rating.WithMinimumCommitmentIgnored(),
			},
			Expect: rating.DetailedLines{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(10),
					Quantity:               alpacadecimal.NewFromFloat(5),
					ChildUniqueReferenceID: rating.UnitPriceUsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					AmountDiscounts: []billing.AmountLineDiscountManaged{
						{
							AmountLineDiscount: billing.AmountLineDiscount{
								Amount: alpacadecimal.NewFromFloat(20),
								LineDiscountBase: billing.LineDiscountBase{
									Description:            lo.ToPtr("Maximum spend discount for charges over 100"),
									ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
									Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
								},
							},
						},
					},
					Totals: totals.Totals{
						Amount:         alpacadecimal.NewFromFloat(50),
						DiscountsTotal: alpacadecimal.NewFromFloat(20),
						Total:          alpacadecimal.NewFromFloat(30),
					},
				},
			},
		})
	})
}

func TestGenerateDetailedLinesCreditsMutatorOptions(t *testing.T) {
	t.Run("credits mutator applied by default", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			}),
			LineMode: testutil.SinglePerPeriodLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(5),
			},
			CreditsApplied: billing.CreditsApplied{
				{
					Amount: alpacadecimal.NewFromFloat(20),
				},
			},
			Expect: rating.DetailedLines{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(10),
					Quantity:               alpacadecimal.NewFromFloat(5),
					ChildUniqueReferenceID: rating.UnitPriceUsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					CreditsApplied: billing.CreditsApplied{
						{
							Amount: alpacadecimal.NewFromFloat(20),
						},
					},
					Totals: totals.Totals{
						Amount:       alpacadecimal.NewFromFloat(50),
						CreditsTotal: alpacadecimal.NewFromFloat(20),
						Total:        alpacadecimal.NewFromFloat(30),
					},
				},
			},
		})
	})

	t.Run("credits mutator disabled when requested", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			}),
			LineMode: testutil.SinglePerPeriodLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(5),
			},
			CreditsApplied: billing.CreditsApplied{
				{
					Amount: alpacadecimal.NewFromFloat(20),
				},
			},
			Options: []rating.GenerateDetailedLinesOption{
				rating.WithCreditsMutatorDisabled(),
			},
			Expect: rating.DetailedLines{
				{
					Name:                   "feature: usage in period",
					PerUnitAmount:          alpacadecimal.NewFromFloat(10),
					Quantity:               alpacadecimal.NewFromFloat(5),
					ChildUniqueReferenceID: rating.UnitPriceUsageChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(50),
						Total:  alpacadecimal.NewFromFloat(50),
					},
				},
			},
		})
	})
}
