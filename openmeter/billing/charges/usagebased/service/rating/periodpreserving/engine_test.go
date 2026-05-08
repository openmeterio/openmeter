package periodpreserving

import (
	"strconv"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	ratingtestutils "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating/testutils"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
	billingratingservice "github.com/openmeterio/openmeter/openmeter/billing/rating/service"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type lateEventRatingTestCase struct {
	name      string
	price     productcatalog.Price
	discounts productcatalog.Discounts
	phases    []lateEventRatingPhase
}

type lateEventRatingPhase struct {
	runID                     string
	period                    timeutil.ClosedPeriod
	usagePerPhaseCumulative   []float64
	expectedDetailedLines     []ratingtestutils.ExpectedDetailedLine
	expectedTotals            ratingtestutils.ExpectedTotals
	mutateBookedDetailedLines func(usagebased.DetailedLines) usagebased.DetailedLines
}

func TestLateEventRatingUnitPrice(t *testing.T) {
	t.Parallel()

	periods := lateEventRatingTestPeriods()
	period1RunID := "period-1-run-id"
	period2RunID := "period-2-run-id"
	percentageDiscount50 := productcatalog.PercentageDiscount{
		Percentage: models.NewPercentage(50),
	}
	usageDiscount5 := productcatalog.UsageDiscount{
		Quantity: alpacadecimal.NewFromInt(5),
	}

	runLateEventRatingTestCase(t, lateEventRatingTestCase{
		name: "late usage is billed on the original period",
		price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(10),
		}),
		phases: []lateEventRatingPhase{
			{
				runID:                   period1RunID,
				period:                  periods.period1,
				usagePerPhaseCumulative: []float64{3},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          10,
						Quantity:               3,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 30,
							Total:  30,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 30,
					Total:  30,
				},
			},
			{
				runID:                   period2RunID,
				period:                  periods.period2,
				usagePerPhaseCumulative: []float64{3, 5},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period2),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          10,
						Quantity:               2,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 20,
							Total:  20,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 20,
					Total:  20,
				},
			},
			{
				period:                  periods.period3,
				usagePerPhaseCumulative: []float64{4, 6, 9},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						CorrectsRunID:          lo.ToPtr(period1RunID),
						PerUnitAmount:          10,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 10,
							Total:  10,
						},
					},
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period3),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period3),
						PerUnitAmount:          10,
						Quantity:               3,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 30,
							Total:  30,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 40,
					Total:  40,
				},
			},
		},
	})

	runLateEventRatingTestCase(t, lateEventRatingTestCase{
		name: "late usage corrects maximum spend after limit was reached",
		price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(10),
			Commitments: productcatalog.Commitments{
				MaximumAmount: lo.ToPtr(alpacadecimal.NewFromInt(50)),
			},
		}),
		phases: []lateEventRatingPhase{
			{
				runID:                   period1RunID,
				period:                  periods.period1,
				usagePerPhaseCumulative: []float64{3},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          10,
						Quantity:               3,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 30,
							Total:  30,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 30,
					Total:  30,
				},
			},
			{
				runID:                   period2RunID,
				period:                  periods.period2,
				usagePerPhaseCumulative: []float64{3, 6},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period2),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          10,
						Quantity:               3,
						Totals: ratingtestutils.ExpectedTotals{
							Amount:         30,
							DiscountsTotal: 10,
							Total:          20,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount:         30,
					DiscountsTotal: 10,
					Total:          20,
				},
			},
			{
				period:                  periods.period3,
				usagePerPhaseCumulative: []float64{4, 7, 7},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						CorrectsRunID:          lo.ToPtr(period1RunID),
						PerUnitAmount:          10,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 10,
							Total:  10,
						},
					},
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period2),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						CorrectsRunID:          lo.ToPtr(period2RunID),
						PerUnitAmount:          10,
						Quantity:               0,
						Totals: ratingtestutils.ExpectedTotals{
							DiscountsTotal: 10,
							Total:          -10,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount:         10,
					DiscountsTotal: 10,
				},
			},
		},
	})

	runLateEventRatingTestCase(t, lateEventRatingTestCase{
		name: "late usage and additional current usage correct maximum spend after limit was reached",
		price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(10),
			Commitments: productcatalog.Commitments{
				MaximumAmount: lo.ToPtr(alpacadecimal.NewFromInt(50)),
			},
		}),
		phases: []lateEventRatingPhase{
			{
				runID:                   period1RunID,
				period:                  periods.period1,
				usagePerPhaseCumulative: []float64{3},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          10,
						Quantity:               3,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 30,
							Total:  30,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 30,
					Total:  30,
				},
			},
			{
				runID:                   period2RunID,
				period:                  periods.period2,
				usagePerPhaseCumulative: []float64{3, 6},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period2),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          10,
						Quantity:               3,
						Totals: ratingtestutils.ExpectedTotals{
							Amount:         30,
							DiscountsTotal: 10,
							Total:          20,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount:         30,
					DiscountsTotal: 10,
					Total:          20,
				},
			},
			{
				period:                  periods.period3,
				usagePerPhaseCumulative: []float64{4, 7, 8},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						CorrectsRunID:          lo.ToPtr(period1RunID),
						PerUnitAmount:          10,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 10,
							Total:  10,
						},
					},
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period2),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						CorrectsRunID:          lo.ToPtr(period2RunID),
						PerUnitAmount:          10,
						Quantity:               0,
						Totals: ratingtestutils.ExpectedTotals{
							DiscountsTotal: 10,
							Total:          -10,
						},
					},
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period3),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period3),
						PerUnitAmount:          10,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount:         10,
							DiscountsTotal: 10,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount:         20,
					DiscountsTotal: 20,
				},
			},
		},
	})

	runLateEventRatingTestCase(t, lateEventRatingTestCase{
		name: "late usage keeps percentage discount on the original period",
		price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(10),
		}),
		discounts: productcatalog.Discounts{
			Percentage: lo.ToPtr(percentageDiscount50),
		},
		phases: []lateEventRatingPhase{
			{
				runID:                   period1RunID,
				period:                  periods.period1,
				usagePerPhaseCumulative: []float64{3},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          10,
						Quantity:               3,
						Totals: ratingtestutils.ExpectedTotals{
							Amount:         30,
							DiscountsTotal: 15,
							Total:          15,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount:         30,
					DiscountsTotal: 15,
					Total:          15,
				},
			},
			{
				runID:                   period2RunID,
				period:                  periods.period2,
				usagePerPhaseCumulative: []float64{3, 5},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period2),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          10,
						Quantity:               2,
						Totals: ratingtestutils.ExpectedTotals{
							Amount:         20,
							DiscountsTotal: 10,
							Total:          10,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount:         20,
					DiscountsTotal: 10,
					Total:          10,
				},
			},
			{
				period:                  periods.period3,
				usagePerPhaseCumulative: []float64{4, 6, 6},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						CorrectsRunID:          lo.ToPtr(period1RunID),
						PerUnitAmount:          10,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount:         10,
							DiscountsTotal: 5,
							Total:          5,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount:         10,
					DiscountsTotal: 5,
					Total:          5,
				},
			},
		},
	})

	runLateEventRatingTestCase(t, lateEventRatingTestCase{
		name: "late usage consumes usage discount on the original period",
		price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(10),
		}),
		discounts: productcatalog.Discounts{
			Usage: lo.ToPtr(usageDiscount5),
		},
		phases: []lateEventRatingPhase{
			{
				runID:                   period1RunID,
				period:                  periods.period1,
				usagePerPhaseCumulative: []float64{3},
				expectedDetailedLines:   []ratingtestutils.ExpectedDetailedLine{},
				expectedTotals:          ratingtestutils.ExpectedTotals{},
			},
			{
				runID:                   period2RunID,
				period:                  periods.period2,
				usagePerPhaseCumulative: []float64{3, 7},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period2),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          10,
						Quantity:               2,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 20,
							Total:  20,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 20,
					Total:  20,
				},
			},
			{
				period:                  periods.period3,
				usagePerPhaseCumulative: []float64{6, 8, 8},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						CorrectsRunID:          lo.ToPtr(period1RunID),
						PerUnitAmount:          10,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 10,
							Total:  10,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 10,
					Total:  10,
				},
			},
		},
	})

	runLateEventRatingTestCase(t, lateEventRatingTestCase{
		name: "late usage corrects maximum spend after percentage discount",
		price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(10),
			Commitments: productcatalog.Commitments{
				MaximumAmount: lo.ToPtr(alpacadecimal.NewFromInt(25)),
			},
		}),
		discounts: productcatalog.Discounts{
			Percentage: lo.ToPtr(percentageDiscount50),
		},
		phases: []lateEventRatingPhase{
			{
				runID:                   period1RunID,
				period:                  periods.period1,
				usagePerPhaseCumulative: []float64{3},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          10,
						Quantity:               3,
						Totals: ratingtestutils.ExpectedTotals{
							Amount:         30,
							DiscountsTotal: 15,
							Total:          15,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount:         30,
					DiscountsTotal: 15,
					Total:          15,
				},
			},
			{
				runID:                   period2RunID,
				period:                  periods.period2,
				usagePerPhaseCumulative: []float64{3, 6},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period2),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          10,
						Quantity:               3,
						Totals: ratingtestutils.ExpectedTotals{
							Amount:         30,
							DiscountsTotal: 20,
							Total:          10,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount:         30,
					DiscountsTotal: 20,
					Total:          10,
				},
			},
			{
				period:                  periods.period3,
				usagePerPhaseCumulative: []float64{4, 7, 7},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						CorrectsRunID:          lo.ToPtr(period1RunID),
						PerUnitAmount:          10,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount:         10,
							DiscountsTotal: 5,
							Total:          5,
						},
					},
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period2),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						CorrectsRunID:          lo.ToPtr(period2RunID),
						PerUnitAmount:          10,
						Quantity:               0,
						Totals: ratingtestutils.ExpectedTotals{
							DiscountsTotal: 5,
							Total:          -5,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount:         10,
					DiscountsTotal: 10,
				},
			},
		},
	})

	runLateEventRatingTestCase(t, lateEventRatingTestCase{
		name: "minimum commitment is charged only in the final phase",
		price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(10),
			Commitments: productcatalog.Commitments{
				MinimumAmount: lo.ToPtr(alpacadecimal.NewFromInt(100)),
			},
		}),
		phases: []lateEventRatingPhase{
			{
				runID:                   period1RunID,
				period:                  periods.period1,
				usagePerPhaseCumulative: []float64{2},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          10,
						Quantity:               2,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 20,
							Total:  20,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 20,
					Total:  20,
				},
			},
			{
				runID:                   period2RunID,
				period:                  periods.period2,
				usagePerPhaseCumulative: []float64{2, 5},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period2),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          10,
						Quantity:               3,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 30,
							Total:  30,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 30,
					Total:  30,
				},
			},
			{
				period:                  periods.period3,
				usagePerPhaseCumulative: []float64{3, 6, 6},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						CorrectsRunID:          lo.ToPtr(period1RunID),
						PerUnitAmount:          10,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 10,
							Total:  10,
						},
					},
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.MinSpendChildUniqueReferenceID, periods.period3),
						Category:               stddetailedline.CategoryCommitment,
						ServicePeriod:          lo.ToPtr(periods.period3),
						PerUnitAmount:          40,
						Quantity:               1,
						Totals: ratingtestutils.ExpectedTotals{
							ChargesTotal: 40,
							Total:        40,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount:       10,
					ChargesTotal: 40,
					Total:        50,
				},
			},
		},
	})
}

func TestLateEventRatingCreditsAllocation(t *testing.T) {
	t.Parallel()

	periods := lateEventRatingTestPeriods()
	period1RunID := "period-1-run-id"
	percentageDiscount50 := productcatalog.PercentageDiscount{
		Percentage: models.NewPercentage(50),
	}

	runLateEventRatingTestCase(t, lateEventRatingTestCase{
		name: "prior credits are ignored when subtracting already billed lines",
		price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(10),
		}),
		phases: []lateEventRatingPhase{
			{
				runID:                   period1RunID,
				period:                  periods.period1,
				usagePerPhaseCumulative: []float64{10},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          10,
						Quantity:               10,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 100,
							Total:  100,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 100,
					Total:  100,
				},
				mutateBookedDetailedLines: func(lines usagebased.DetailedLines) usagebased.DetailedLines {
					lines = lines.Clone()
					lines[0].CreditsApplied = billing.CreditsApplied{
						{
							Amount:              alpacadecimal.NewFromInt(40),
							Description:         "test credit allocation",
							CreditRealizationID: "test-credit-realization-id",
						},
					}
					lines[0].Totals.CreditsTotal = alpacadecimal.NewFromInt(40)
					lines[0].Totals.Total = lines[0].Totals.CalculateTotal()

					return lines
				},
			},
			{
				period:                  periods.period2,
				usagePerPhaseCumulative: []float64{10, 10},
				expectedDetailedLines:   []ratingtestutils.ExpectedDetailedLine{},
				expectedTotals:          ratingtestutils.ExpectedTotals{},
			},
			{
				period:                  periods.period3,
				usagePerPhaseCumulative: []float64{12, 12, 12},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						CorrectsRunID:          lo.ToPtr(period1RunID),
						PerUnitAmount:          10,
						Quantity:               2,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 20,
							Total:  20,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 20,
					Total:  20,
				},
			},
		},
	})

	runLateEventRatingTestCase(t, lateEventRatingTestCase{
		name: "prior credits are ignored but prior discounts are preserved",
		price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(10),
		}),
		discounts: productcatalog.Discounts{
			Percentage: lo.ToPtr(percentageDiscount50),
		},
		phases: []lateEventRatingPhase{
			{
				runID:                   period1RunID,
				period:                  periods.period1,
				usagePerPhaseCumulative: []float64{10},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          10,
						Quantity:               10,
						Totals: ratingtestutils.ExpectedTotals{
							Amount:         100,
							DiscountsTotal: 50,
							Total:          50,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount:         100,
					DiscountsTotal: 50,
					Total:          50,
				},
				mutateBookedDetailedLines: func(lines usagebased.DetailedLines) usagebased.DetailedLines {
					lines = lines.Clone()
					lines[0].CreditsApplied = billing.CreditsApplied{
						{
							Amount:              alpacadecimal.NewFromInt(40),
							Description:         "test credit allocation",
							CreditRealizationID: "test-credit-realization-id",
						},
					}
					lines[0].Totals.CreditsTotal = alpacadecimal.NewFromInt(40)
					lines[0].Totals.Total = lines[0].Totals.CalculateTotal()

					return lines
				},
			},
			{
				period:                  periods.period2,
				usagePerPhaseCumulative: []float64{10, 10},
				expectedDetailedLines:   []ratingtestutils.ExpectedDetailedLine{},
				expectedTotals:          ratingtestutils.ExpectedTotals{},
			},
			{
				period:                  periods.period3,
				usagePerPhaseCumulative: []float64{12, 12, 12},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						CorrectsRunID:          lo.ToPtr(period1RunID),
						PerUnitAmount:          10,
						Quantity:               2,
						Totals: ratingtestutils.ExpectedTotals{
							Amount:         20,
							DiscountsTotal: 10,
							Total:          10,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount:         20,
					DiscountsTotal: 10,
					Total:          10,
				},
			},
		},
	})
}

func TestLateEventRatingVolumeTieredPrice(t *testing.T) {
	t.Parallel()

	periods := lateEventRatingTestPeriods()
	period1RunID := "period-1-run-id"
	volumeTieredPrice := *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
		Mode: productcatalog.VolumeTieredPrice,
		Tiers: []productcatalog.PriceTier{
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(15)),
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromInt(10),
				},
			},
			{
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromInt(5),
				},
			},
		},
	})

	runLateEventRatingTestCase(t, lateEventRatingTestCase{
		name:  "volume tier repricing keeps reversal on original period",
		price: volumeTieredPrice,
		phases: []lateEventRatingPhase{
			{
				runID:                   period1RunID,
				period:                  periods.period1,
				usagePerPhaseCumulative: []float64{15},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.VolumeUnitPriceChildUniqueReferenceID, periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          10,
						Quantity:               15,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 150,
							Total:  150,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 150,
					Total:  150,
				},
			},
			{
				period:                  periods.period2,
				usagePerPhaseCumulative: []float64{15, 16},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.VolumeUnitPriceChildUniqueReferenceID, periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						CorrectsRunID:          lo.ToPtr(period1RunID),
						PerUnitAmount:          10,
						Quantity:               -15,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: -150,
							Total:  -150,
						},
					},
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.VolumeUnitPriceChildUniqueReferenceID, periods.period2),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period2),
						PerUnitAmount:          5,
						Quantity:               16,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 80,
							Total:  80,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: -70,
					Total:  -70,
				},
			},
			{
				period:                  periods.period3,
				usagePerPhaseCumulative: []float64{15, 16, 16},
				expectedDetailedLines:   []ratingtestutils.ExpectedDetailedLine{},
				expectedTotals:          ratingtestutils.ExpectedTotals{},
			},
		},
	})

	runLateEventRatingTestCase(t, lateEventRatingTestCase{
		name:  "late usage reprices original volume tier period",
		price: volumeTieredPrice,
		phases: []lateEventRatingPhase{
			{
				runID:                   period1RunID,
				period:                  periods.period1,
				usagePerPhaseCumulative: []float64{15},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.VolumeUnitPriceChildUniqueReferenceID, periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          10,
						Quantity:               15,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 150,
							Total:  150,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 150,
					Total:  150,
				},
			},
			{
				period:                  periods.period2,
				usagePerPhaseCumulative: []float64{15, 15},
				expectedDetailedLines:   []ratingtestutils.ExpectedDetailedLine{},
				expectedTotals:          ratingtestutils.ExpectedTotals{},
			},
			{
				period:                  periods.period3,
				usagePerPhaseCumulative: []float64{16, 16, 16},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: formatCorrectionDetailedLineChildUniqueReferenceID(billingrating.VolumeUnitPriceChildUniqueReferenceID, "phase-1-line-1", periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						CorrectsRunID:          lo.ToPtr(period1RunID),
						PerUnitAmount:          10,
						Quantity:               -15,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: -150,
							Total:  -150,
						},
					},
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.VolumeUnitPriceChildUniqueReferenceID, periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						CorrectsRunID:          lo.ToPtr(period1RunID),
						PerUnitAmount:          5,
						Quantity:               16,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 80,
							Total:  80,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: -70,
					Total:  -70,
				},
			},
			{
				period:                  periods.period4,
				usagePerPhaseCumulative: []float64{16, 16, 16, 16},
				expectedDetailedLines:   []ratingtestutils.ExpectedDetailedLine{},
				expectedTotals:          ratingtestutils.ExpectedTotals{},
			},
		},
	})

	runLateEventRatingTestCase(t, lateEventRatingTestCase{
		name: "late volume repricing keeps max spend correction on original period",
		price: *productcatalog.NewPriceFrom(productcatalog.TieredPrice{
			Mode: productcatalog.VolumeTieredPrice,
			Tiers: []productcatalog.PriceTier{
				{
					UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(15)),
					UnitPrice: &productcatalog.PriceTierUnitPrice{
						Amount: alpacadecimal.NewFromInt(10),
					},
				},
				{
					UnitPrice: &productcatalog.PriceTierUnitPrice{
						Amount: alpacadecimal.NewFromInt(5),
					},
				},
			},
			Commitments: productcatalog.Commitments{
				MaximumAmount: lo.ToPtr(alpacadecimal.NewFromInt(100)),
			},
		}),
		phases: []lateEventRatingPhase{
			{
				runID:                   period1RunID,
				period:                  periods.period1,
				usagePerPhaseCumulative: []float64{15},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.VolumeUnitPriceChildUniqueReferenceID, periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          10,
						Quantity:               15,
						Totals: ratingtestutils.ExpectedTotals{
							Amount:         150,
							DiscountsTotal: 50,
							Total:          100,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount:         150,
					DiscountsTotal: 50,
					Total:          100,
				},
			},
			{
				period:                  periods.period2,
				usagePerPhaseCumulative: []float64{15, 15},
				expectedDetailedLines:   []ratingtestutils.ExpectedDetailedLine{},
				expectedTotals:          ratingtestutils.ExpectedTotals{},
			},
			{
				period:                  periods.period3,
				usagePerPhaseCumulative: []float64{16, 16, 16},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: formatCorrectionDetailedLineChildUniqueReferenceID(billingrating.VolumeUnitPriceChildUniqueReferenceID, "phase-1-line-1", periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						CorrectsRunID:          lo.ToPtr(period1RunID),
						PerUnitAmount:          10,
						Quantity:               -15,
						Totals: ratingtestutils.ExpectedTotals{
							Amount:         -150,
							DiscountsTotal: -50,
							Total:          -100,
						},
					},
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.VolumeUnitPriceChildUniqueReferenceID, periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						CorrectsRunID:          lo.ToPtr(period1RunID),
						PerUnitAmount:          5,
						Quantity:               16,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 80,
							Total:  80,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount:         -70,
					DiscountsTotal: -50,
					Total:          -20,
				},
			},
		},
	})

	runLateEventRatingTestCase(t, lateEventRatingTestCase{
		name:  "late volume repricing consumes usage discount on original period",
		price: volumeTieredPrice,
		discounts: productcatalog.Discounts{
			Usage: lo.ToPtr(productcatalog.UsageDiscount{
				Quantity: alpacadecimal.NewFromInt(5),
			}),
		},
		phases: []lateEventRatingPhase{
			{
				runID:                   period1RunID,
				period:                  periods.period1,
				usagePerPhaseCumulative: []float64{20},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.VolumeUnitPriceChildUniqueReferenceID, periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						PerUnitAmount:          10,
						Quantity:               15,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 150,
							Total:  150,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: 150,
					Total:  150,
				},
			},
			{
				period:                  periods.period2,
				usagePerPhaseCumulative: []float64{20, 20},
				expectedDetailedLines:   []ratingtestutils.ExpectedDetailedLine{},
				expectedTotals:          ratingtestutils.ExpectedTotals{},
			},
			{
				period:                  periods.period3,
				usagePerPhaseCumulative: []float64{21, 21, 21},
				expectedDetailedLines: []ratingtestutils.ExpectedDetailedLine{
					{
						ChildUniqueReferenceID: formatCorrectionDetailedLineChildUniqueReferenceID(billingrating.VolumeUnitPriceChildUniqueReferenceID, "phase-1-line-1", periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						CorrectsRunID:          lo.ToPtr(period1RunID),
						PerUnitAmount:          10,
						Quantity:               -15,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: -150,
							Total:  -150,
						},
					},
					{
						ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.VolumeUnitPriceChildUniqueReferenceID, periods.period1),
						Category:               stddetailedline.CategoryRegular,
						ServicePeriod:          lo.ToPtr(periods.period1),
						CorrectsRunID:          lo.ToPtr(period1RunID),
						PerUnitAmount:          5,
						Quantity:               16,
						Totals: ratingtestutils.ExpectedTotals{
							Amount: 80,
							Total:  80,
						},
					},
				},
				expectedTotals: ratingtestutils.ExpectedTotals{
					Amount: -70,
					Total:  -70,
				},
			},
		},
	})
}

func TestRateRejectsOverlappingPriorPeriods(t *testing.T) {
	t.Parallel()

	periods := lateEventRatingTestPeriods()
	intent := ratingtestutils.NewIntentForTest(t,
		timeutil.ClosedPeriod{From: periods.period1.From, To: periods.period3.To},
		*productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
			Multiplier: alpacadecimal.NewFromInt(1),
		}),
		productcatalog.Discounts{},
	)

	_, err := New(billingratingservice.New()).Rate(t.Context(), Input{
		Intent: intent,
		PriorPeriods: []PriorPeriod{
			{
				RunID:           usagebased.RealizationRunID{Namespace: "ns", ID: "run-1"},
				ServicePeriod:   periods.period1,
				MeteredQuantity: alpacadecimal.NewFromInt(10),
			},
			{
				RunID: usagebased.RealizationRunID{Namespace: "ns", ID: "run-2"},
				ServicePeriod: timeutil.ClosedPeriod{
					From: periods.period1.From.Add(24 * time.Hour),
					To:   periods.period1.To,
				},
				MeteredQuantity: alpacadecimal.NewFromInt(11),
			},
		},
		CurrentPeriod: CurrentPeriod{
			ServicePeriod:   periods.period2,
			MeteredQuantity: alpacadecimal.NewFromInt(11),
		},
	})
	require.ErrorContains(t, err, "prior periods[0] service period overlaps prior periods[1] service period")
}

func TestRateRejectsPriorPeriodThatIsEmptyAtMinimumStreamingWindowSize(t *testing.T) {
	t.Parallel()

	periods := lateEventRatingTestPeriods()
	intent := ratingtestutils.NewIntentForTest(t,
		timeutil.ClosedPeriod{From: periods.period1.From, To: periods.period2.To},
		*productcatalog.NewPriceFrom(productcatalog.DynamicPrice{
			Multiplier: alpacadecimal.NewFromInt(1),
		}),
		productcatalog.Discounts{},
	)

	_, err := New(billingratingservice.New()).Rate(t.Context(), Input{
		Intent: intent,
		PriorPeriods: []PriorPeriod{
			{
				RunID: usagebased.RealizationRunID{Namespace: "ns", ID: "run-1"},
				ServicePeriod: timeutil.ClosedPeriod{
					From: periods.period1.From.Add(100 * time.Millisecond),
					To:   periods.period1.From.Add(200 * time.Millisecond),
				},
				MeteredQuantity: alpacadecimal.NewFromInt(10),
			},
		},
		CurrentPeriod: CurrentPeriod{
			ServicePeriod:   periods.period2,
			MeteredQuantity: alpacadecimal.NewFromInt(11),
		},
	})
	require.ErrorContains(t, err, "prior periods[0] service period must not be empty when truncated to minimum streaming window size")
}

func runLateEventRatingTestCase(t *testing.T, tc lateEventRatingTestCase) {
	t.Helper()

	require.NotEmpty(t, tc.phases)

	fullServicePeriod := timeutil.ClosedPeriod{
		From: tc.phases[0].period.From,
		To:   tc.phases[len(tc.phases)-1].period.To,
	}

	intent := ratingtestutils.NewIntentForTest(t, fullServicePeriod, tc.price, tc.discounts)

	engine := New(billingratingservice.New())
	bookedDetailedLinesByPhase := make([]usagebased.DetailedLines, len(tc.phases))
	phaseRunIDs := make([]usagebased.RealizationRunID, len(tc.phases))

	// Phase subtests must not call t.Parallel: each t.Run consumes prior bookedDetailedLinesByPhase and phaseRunIDs.
	for phaseIdx, phase := range tc.phases {
		runID := phase.runID
		if runID == "" {
			runID = ulid.Make().String()
		}
		phaseRunIDs[phaseIdx] = usagebased.RealizationRunID{
			Namespace: "ns",
			ID:        runID,
		}

		t.Run(tc.name+"/phase "+strconv.Itoa(phaseIdx+1), func(t *testing.T) {
			require.Len(t, phase.usagePerPhaseCumulative, phaseIdx+1)

			priorPeriods := make([]PriorPeriod, 0, phaseIdx)
			for priorPhaseIdx := 0; priorPhaseIdx < phaseIdx; priorPhaseIdx++ {
				priorPeriods = append(priorPeriods, PriorPeriod{
					RunID:           phaseRunIDs[priorPhaseIdx],
					MeteredQuantity: alpacadecimal.NewFromFloat(phase.usagePerPhaseCumulative[priorPhaseIdx]),
					ServicePeriod:   tc.phases[priorPhaseIdx].period,
					DetailedLines:   bookedDetailedLinesByPhase[priorPhaseIdx],
				})
			}

			out, err := engine.Rate(t.Context(), Input{
				Intent: intent,
				CurrentPeriod: CurrentPeriod{
					MeteredQuantity: alpacadecimal.NewFromFloat(phase.usagePerPhaseCumulative[phaseIdx]),
					ServicePeriod:   phase.period,
				},
				PriorPeriods: priorPeriods,
			})
			require.NoError(t, err)

			require.Equal(t, phase.expectedDetailedLines, ratingtestutils.ToExpectedDetailedLinesWithServicePeriod(out.DetailedLines))
			require.Equal(t, phase.expectedTotals, ratingtestutils.ToExpectedTotals(out.DetailedLines.SumTotals()))

			bookedDetailedLines := out.DetailedLines
			if phase.mutateBookedDetailedLines != nil {
				bookedDetailedLines = phase.mutateBookedDetailedLines(bookedDetailedLines)
			}
			for idx := range bookedDetailedLines {
				bookedDetailedLines[idx].ID = "phase-" + strconv.Itoa(phaseIdx+1) + "-line-" + strconv.Itoa(idx+1)
			}

			bookedDetailedLinesByPhase[phaseIdx] = bookedDetailedLines
		})
	}
}

type lateEventRatingTestPeriodsValue struct {
	period1 timeutil.ClosedPeriod
	period2 timeutil.ClosedPeriod
	period3 timeutil.ClosedPeriod
	period4 timeutil.ClosedPeriod
}

func lateEventRatingTestPeriods() lateEventRatingTestPeriodsValue {
	period1 := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC),
	}
	period2 := timeutil.ClosedPeriod{
		From: period1.To,
		To:   time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC),
	}
	period3 := timeutil.ClosedPeriod{
		From: period2.To,
		To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	period4 := timeutil.ClosedPeriod{
		From: period3.To,
		To:   time.Date(2025, 2, 10, 0, 0, 0, 0, time.UTC),
	}

	return lateEventRatingTestPeriodsValue{
		period1: period1,
		period2: period2,
		period3: period3,
		period4: period4,
	}
}

func formatCorrectionDetailedLineChildUniqueReferenceID(referenceID, detailedLineID string, servicePeriod timeutil.ClosedPeriod) string {
	return ratingtestutils.FormatDetailedLineChildUniqueReferenceID(
		referenceID+"#correction:detailed_line_id="+detailedLineID,
		servicePeriod,
	)
}
