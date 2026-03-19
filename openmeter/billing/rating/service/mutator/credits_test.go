package mutator_test

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/rating/service/testutil"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestCreditsMutator(t *testing.T) {
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
			UnitPrice: &productcatalog.PriceTierUnitPrice{
				Amount: alpacadecimal.NewFromFloat(1),
			},
		},
	}

	testCredit1Description := "test credit 1"
	testCredit2Description := "test credit 2"

	t.Run("credits mutator, paid fully in credits", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.GraduatedTieredPrice,
				Tiers: testTiers,
			}),
			LineMode: testutil.SinglePerPeriodLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			CreditsApplied: billing.CreditsApplied{
				{
					Amount:      alpacadecimal.NewFromFloat(150),
					Description: testCredit1Description,
				},
			},

			Expect: rating.DetailedLines{
				{
					Name:                   "feature: flat price for tier 1",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: "graduated-tiered-1-flat-price",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					CreditsApplied: []billing.CreditApplied{
						{
							Amount:      alpacadecimal.NewFromFloat(100),
							Description: testCredit1Description,
						},
					},
					Totals: totals.Totals{
						Amount:       alpacadecimal.NewFromFloat(100),
						CreditsTotal: alpacadecimal.NewFromFloat(100),
						Total:        alpacadecimal.NewFromFloat(0),
					},
				},
				{
					Name:                   "feature: flat price for tier 2",
					PerUnitAmount:          alpacadecimal.NewFromFloat(50),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: "graduated-tiered-2-flat-price",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					CreditsApplied: []billing.CreditApplied{
						{
							Amount:      alpacadecimal.NewFromFloat(50),
							Description: testCredit1Description,
						},
					},
					Totals: totals.Totals{
						Amount:       alpacadecimal.NewFromFloat(50),
						CreditsTotal: alpacadecimal.NewFromFloat(50),
						Total:        alpacadecimal.NewFromFloat(0),
					},
				},
			},
		})
	})

	t.Run("credits mutator, tier 1 + 2 paid fully in credits", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.GraduatedTieredPrice,
				Tiers: testTiers,
			}),
			LineMode: testutil.SinglePerPeriodLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(15),
			},
			CreditsApplied: billing.CreditsApplied{
				{
					Amount:      alpacadecimal.NewFromFloat(150),
					Description: testCredit1Description,
				},
			},

			Expect: rating.DetailedLines{
				{
					Name:                   "feature: flat price for tier 1",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: "graduated-tiered-1-flat-price",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					CreditsApplied: []billing.CreditApplied{
						{
							Amount:      alpacadecimal.NewFromFloat(100),
							Description: testCredit1Description,
						},
					},
					Totals: totals.Totals{
						Amount:       alpacadecimal.NewFromFloat(100),
						CreditsTotal: alpacadecimal.NewFromFloat(100),
						Total:        alpacadecimal.NewFromFloat(0),
					},
				},
				{
					Name:                   "feature: flat price for tier 2",
					PerUnitAmount:          alpacadecimal.NewFromFloat(50),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: "graduated-tiered-2-flat-price",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					CreditsApplied: []billing.CreditApplied{
						{
							Amount:      alpacadecimal.NewFromFloat(50),
							Description: testCredit1Description,
						},
					},
					Totals: totals.Totals{
						Amount:       alpacadecimal.NewFromFloat(50),
						CreditsTotal: alpacadecimal.NewFromFloat(50),
						Total:        alpacadecimal.NewFromFloat(0),
					},
				},
				{
					Name:                   "feature: usage price for tier 3",
					PerUnitAmount:          alpacadecimal.NewFromFloat(1),
					Quantity:               alpacadecimal.NewFromFloat(5),
					ChildUniqueReferenceID: "graduated-tiered-3-price-usage",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromFloat(5),
						Total:  alpacadecimal.NewFromFloat(5),
					},
				},
			},
		})
	})

	t.Run("credits mutator, paid fully from multiple credits", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.GraduatedTieredPrice,
				Tiers: testTiers,
			}),
			LineMode: testutil.SinglePerPeriodLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			CreditsApplied: billing.CreditsApplied{
				{
					Amount:      alpacadecimal.NewFromFloat(75),
					Description: testCredit1Description,
				},
				{
					Amount:      alpacadecimal.NewFromFloat(75),
					Description: testCredit2Description,
				},
			},

			Expect: rating.DetailedLines{
				{
					Name:                   "feature: flat price for tier 1",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: "graduated-tiered-1-flat-price",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					CreditsApplied: []billing.CreditApplied{
						{
							Amount:      alpacadecimal.NewFromFloat(75),
							Description: testCredit1Description,
						},
						{
							Amount:      alpacadecimal.NewFromFloat(25),
							Description: testCredit2Description,
						},
					},
					Totals: totals.Totals{
						Amount:       alpacadecimal.NewFromFloat(100),
						CreditsTotal: alpacadecimal.NewFromFloat(100),
						Total:        alpacadecimal.NewFromFloat(0),
					},
				},
				{
					Name:                   "feature: flat price for tier 2",
					PerUnitAmount:          alpacadecimal.NewFromFloat(50),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: "graduated-tiered-2-flat-price",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					CreditsApplied: []billing.CreditApplied{
						{
							Amount:      alpacadecimal.NewFromFloat(50),
							Description: testCredit2Description,
						},
					},
					Totals: totals.Totals{
						Amount:       alpacadecimal.NewFromFloat(50),
						CreditsTotal: alpacadecimal.NewFromFloat(50),
						Total:        alpacadecimal.NewFromFloat(0),
					},
				},
			},
		})
	})

	t.Run("credits mutator, paid partially in credits", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.GraduatedTieredPrice,
				Tiers: testTiers,
			}),
			LineMode: testutil.SinglePerPeriodLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			CreditsApplied: billing.CreditsApplied{
				{
					Amount:      alpacadecimal.NewFromFloat(125),
					Description: testCredit1Description,
				},
			},

			Expect: rating.DetailedLines{
				{
					Name:                   "feature: flat price for tier 1",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: "graduated-tiered-1-flat-price",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					CreditsApplied: []billing.CreditApplied{
						{
							Amount:      alpacadecimal.NewFromFloat(100),
							Description: testCredit1Description,
						},
					},
					Totals: totals.Totals{
						Amount:       alpacadecimal.NewFromFloat(100),
						CreditsTotal: alpacadecimal.NewFromFloat(100),
						Total:        alpacadecimal.NewFromFloat(0),
					},
				},
				{
					Name:                   "feature: flat price for tier 2",
					PerUnitAmount:          alpacadecimal.NewFromFloat(50),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: "graduated-tiered-2-flat-price",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					CreditsApplied: []billing.CreditApplied{
						{
							Amount:      alpacadecimal.NewFromFloat(25),
							Description: testCredit1Description,
						},
					},
					Totals: totals.Totals{
						Amount:       alpacadecimal.NewFromFloat(50),
						CreditsTotal: alpacadecimal.NewFromFloat(25),
						Total:        alpacadecimal.NewFromFloat(25),
					},
				},
			},
		})
	})

	t.Run("credits mutator, commitment settled by credits", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.GraduatedTieredPrice,
				Tiers: testTiers,
				Commitments: productcatalog.Commitments{
					MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(200)),
				},
			}),
			LineMode: testutil.SinglePerPeriodLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			CreditsApplied: billing.CreditsApplied{
				{
					Amount:      alpacadecimal.NewFromFloat(175),
					Description: testCredit1Description,
				},
			},

			Expect: rating.DetailedLines{
				{
					Name:                   "feature: flat price for tier 1",
					PerUnitAmount:          alpacadecimal.NewFromFloat(100),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: "graduated-tiered-1-flat-price",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					CreditsApplied: []billing.CreditApplied{
						{
							Amount:      alpacadecimal.NewFromFloat(100),
							Description: testCredit1Description,
						},
					},
					Totals: totals.Totals{
						Amount:       alpacadecimal.NewFromFloat(100),
						CreditsTotal: alpacadecimal.NewFromFloat(100),
						Total:        alpacadecimal.NewFromFloat(0),
					},
				},
				{
					Name:                   "feature: flat price for tier 2",
					PerUnitAmount:          alpacadecimal.NewFromFloat(50),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: "graduated-tiered-2-flat-price",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					CreditsApplied: []billing.CreditApplied{
						{
							Amount:      alpacadecimal.NewFromFloat(50),
							Description: testCredit1Description,
						},
					},
					Totals: totals.Totals{
						Amount:       alpacadecimal.NewFromFloat(50),
						CreditsTotal: alpacadecimal.NewFromFloat(50),
						Total:        alpacadecimal.NewFromFloat(0),
					},
				},
				{
					Name:                   "feature: minimum spend",
					PerUnitAmount:          alpacadecimal.NewFromFloat(50),
					Quantity:               alpacadecimal.NewFromFloat(1),
					ChildUniqueReferenceID: rating.MinSpendChildUniqueReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Category:               billing.FlatFeeCategoryCommitment,
					Period:                 lo.ToPtr(testutil.TestFullPeriod.ToClosedPeriod()),
					CreditsApplied: []billing.CreditApplied{
						{
							Amount:      alpacadecimal.NewFromFloat(25),
							Description: testCredit1Description,
						},
					},
					Totals: totals.Totals{
						ChargesTotal: alpacadecimal.NewFromFloat(50),
						CreditsTotal: alpacadecimal.NewFromFloat(25),
						Total:        alpacadecimal.NewFromFloat(25),
					},
				},
			},
		})
	})

	t.Run("credits mutator, errors when credits are not consumed fully", func(t *testing.T) {
		testutil.RunCalculationTestCase(t, testutil.CalculationTestCase{
			Price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
				Mode:  productcatalog.GraduatedTieredPrice,
				Tiers: testTiers,
			}),
			LineMode: testutil.SinglePerPeriodLineMode,
			Usage: testutil.FeatureUsageResponse{
				LinePeriodQty: alpacadecimal.NewFromFloat(10),
			},
			CreditsApplied: billing.CreditsApplied{
				{
					Amount:      alpacadecimal.NewFromFloat(300),
					Description: testCredit1Description,
				},
			},

			ExpectErrorIs: billing.ErrInvoiceLineCreditsNotConsumedFully,
		})
	})
}
