package delta

import (
	"strconv"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	ratingtestutils "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating/testutils"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestRateKeepsDetailedLineChildUniqueReferenceIDsWithoutServicePeriodSuffix(t *testing.T) {
	// Given:
	// - billing rating returns generated detailed lines without service-period suffixes
	// When:
	// - delta rating books the generated lines for the current run
	// Then:
	// - the output keeps the generated child unique reference IDs unchanged
	t.Parallel()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	ratingService := &stubRatingService{
		result: billingrating.GenerateDetailedLinesResult{
			DetailedLines: billingrating.DetailedLines{
				{
					Name:                   "Usage",
					Quantity:               alpacadecimal.NewFromInt(12),
					PerUnitAmount:          alpacadecimal.NewFromInt(3),
					ChildUniqueReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromInt(36),
						Total:  alpacadecimal.NewFromInt(36),
					},
				},
				{
					Name:                   "Discount",
					Quantity:               alpacadecimal.NewFromInt(1),
					PerUnitAmount:          alpacadecimal.NewFromInt(1),
					ChildUniqueReferenceID: "rateCardDiscount/correlationID=discount-50pct",
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromInt(1),
						Total:  alpacadecimal.NewFromInt(1),
					},
				},
			},
		},
	}

	out, err := New(ratingService).Rate(t.Context(), Input{
		Intent: newIntentForTest(t, servicePeriod),
		CurrentPeriod: CurrentPeriod{
			MeteredQuantity: alpacadecimal.NewFromInt(12),
			ServicePeriod:   servicePeriod,
		},
	})
	require.NoError(t, err)

	require.ElementsMatch(t, []string{
		"unit-price-usage",
		"rateCardDiscount/correlationID=discount-50pct",
	}, []string{
		out.DetailedLines[0].ChildUniqueReferenceID,
		out.DetailedLines[1].ChildUniqueReferenceID,
	})
	require.False(t, ratingService.lastOpts.IgnoreMinimumCommitment)
	require.True(t, ratingService.lastOpts.DisableCreditsMutator)
}

func TestRateIgnoresMinimumCommitmentForPartialRun(t *testing.T) {
	// Given:
	// - the current run covers only part of the charge service period
	// When:
	// - delta rating invokes billing rating
	// Then:
	// - billing rating is called with minimum commitment ignored
	t.Parallel()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	currentPeriod := timeutil.ClosedPeriod{
		From: servicePeriod.From,
		To:   servicePeriod.From.Add(24 * time.Hour),
	}
	ratingService := &stubRatingService{}

	_, err := New(ratingService).Rate(t.Context(), Input{
		Intent: newIntentForTest(t, servicePeriod),
		CurrentPeriod: CurrentPeriod{
			MeteredQuantity: alpacadecimal.NewFromInt(12),
			ServicePeriod:   currentPeriod,
		},
	})
	require.NoError(t, err)
	require.True(t, ratingService.lastOpts.IgnoreMinimumCommitment)
	require.True(t, ratingService.lastOpts.DisableCreditsMutator)
}

func TestRateSubtractsAlreadyBilledLinesAndBooksDeltaOnCurrentPeriod(t *testing.T) {
	// Given:
	// - billing rating returns a cumulative current usage line
	// - a prior run already booked part of the same usage line with a period suffix
	// When:
	// - delta rating subtracts the already billed line
	// Then:
	// - only the remaining quantity is booked on the current period
	t.Parallel()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	priorPeriod := timeutil.ClosedPeriod{
		From: servicePeriod.From,
		To:   servicePeriod.From.Add(24 * time.Hour),
	}
	currentPeriod := timeutil.ClosedPeriod{
		From: priorPeriod.To,
		To:   servicePeriod.To,
	}

	out, err := New(&stubRatingService{
		result: billingrating.GenerateDetailedLinesResult{
			DetailedLines: billingrating.DetailedLines{
				{
					Name:                   "Usage",
					Quantity:               alpacadecimal.NewFromInt(5),
					PerUnitAmount:          alpacadecimal.NewFromInt(10),
					ChildUniqueReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromInt(50),
						Total:  alpacadecimal.NewFromInt(50),
					},
				},
			},
		},
	}).Rate(t.Context(), Input{
		Intent: newIntentForTest(t, servicePeriod),
		CurrentPeriod: CurrentPeriod{
			MeteredQuantity: alpacadecimal.NewFromInt(5),
			ServicePeriod:   currentPeriod,
		},
		AlreadyBilledDetailedLines: usagebased.DetailedLines{
			{
				PricerReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID,
				Base: stddetailedline.Base{
					ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
						Name: "Usage",
					}),
					ServicePeriod:          priorPeriod,
					Currency:               currencyx.Code("USD"),
					ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, priorPeriod),
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					PerUnitAmount:          alpacadecimal.NewFromInt(10),
					Quantity:               alpacadecimal.NewFromInt(3),
					Category:               stddetailedline.CategoryRegular,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromInt(30),
						Total:  alpacadecimal.NewFromInt(30),
					},
				},
			},
		},
	})
	require.NoError(t, err)

	require.Equal(t, []ratingtestutils.ExpectedDetailedLine{
		{
			ChildUniqueReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID,
			Category:               stddetailedline.CategoryRegular,
			ServicePeriod:          lo.ToPtr(currentPeriod),
			PerUnitAmount:          10,
			Quantity:               2,
			Totals: ratingtestutils.ExpectedTotals{
				Amount: 20,
				Total:  20,
			},
		},
	}, ratingtestutils.ToExpectedDetailedLinesWithServicePeriod(out.DetailedLines))
}

func TestRateGeneratesCorrectionChildUniqueReferenceIDForPreviousOnlyReversal(t *testing.T) {
	// Given:
	// - a prior run booked a line that is absent from the current cumulative rating
	// When:
	// - delta rating subtracts already billed lines
	// Then:
	// - the previous-only reversal gets a deterministic correction child reference
	t.Parallel()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	priorPeriod := timeutil.ClosedPeriod{
		From: servicePeriod.From,
		To:   servicePeriod.From.Add(24 * time.Hour),
	}
	currentPeriod := timeutil.ClosedPeriod{
		From: priorPeriod.To,
		To:   servicePeriod.To,
	}

	out, err := New(&stubRatingService{}).Rate(t.Context(), Input{
		Intent: newIntentForTest(t, servicePeriod),
		CurrentPeriod: CurrentPeriod{
			MeteredQuantity: alpacadecimal.Zero,
			ServicePeriod:   currentPeriod,
		},
		AlreadyBilledDetailedLines: usagebased.DetailedLines{
			{
				PricerReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID,
				Base: stddetailedline.Base{
					ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
						ID:   "previous-line-id",
						Name: "Usage",
					}),
					ServicePeriod:          priorPeriod,
					Currency:               currencyx.Code("USD"),
					ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, priorPeriod),
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					PerUnitAmount:          alpacadecimal.NewFromInt(10),
					Quantity:               alpacadecimal.NewFromInt(3),
					Category:               stddetailedline.CategoryRegular,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromInt(30),
						Total:  alpacadecimal.NewFromInt(30),
					},
				},
			},
		},
	})
	require.NoError(t, err)

	require.Equal(t, []ratingtestutils.ExpectedDetailedLine{
		{
			ChildUniqueReferenceID: "unit-price-usage#correction:detailed_line_id=previous-line-id",
			Category:               stddetailedline.CategoryRegular,
			ServicePeriod:          lo.ToPtr(currentPeriod),
			PerUnitAmount:          10,
			Quantity:               -3,
			Totals: ratingtestutils.ExpectedTotals{
				Amount: -30,
				Total:  -30,
			},
		},
	}, ratingtestutils.ToExpectedDetailedLinesWithServicePeriod(out.DetailedLines))
}

func TestRateErrorsWhenPreviousOnlyReversalDetailedLineIDIsMissing(t *testing.T) {
	// Given:
	// - a previous-only reversal is needed
	// - the already billed detailed line has no persisted line ID
	// When:
	// - delta rating tries to generate the correction child reference
	// Then:
	// - rating fails because the correction reference cannot be stable
	t.Parallel()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	priorPeriod := timeutil.ClosedPeriod{
		From: servicePeriod.From,
		To:   servicePeriod.From.Add(24 * time.Hour),
	}
	currentPeriod := timeutil.ClosedPeriod{
		From: priorPeriod.To,
		To:   servicePeriod.To,
	}

	_, err := New(&stubRatingService{}).Rate(t.Context(), Input{
		Intent: newIntentForTest(t, servicePeriod),
		CurrentPeriod: CurrentPeriod{
			MeteredQuantity: alpacadecimal.Zero,
			ServicePeriod:   currentPeriod,
		},
		AlreadyBilledDetailedLines: usagebased.DetailedLines{
			{
				PricerReferenceID: billingrating.UnitPriceUsageChildUniqueReferenceID,
				Base: stddetailedline.Base{
					ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
						Name: "Usage",
					}),
					ServicePeriod:          priorPeriod,
					Currency:               currencyx.Code("USD"),
					ChildUniqueReferenceID: ratingtestutils.FormatDetailedLineChildUniqueReferenceID(billingrating.UnitPriceUsageChildUniqueReferenceID, priorPeriod),
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					PerUnitAmount:          alpacadecimal.NewFromInt(10),
					Quantity:               alpacadecimal.NewFromInt(3),
					Category:               stddetailedline.CategoryRegular,
					Totals: totals.Totals{
						Amount: alpacadecimal.NewFromInt(30),
						Total:  alpacadecimal.NewFromInt(30),
					},
				},
			},
		},
	})
	require.ErrorContains(t, err, "previous only reversal child unique reference id: detailed line id is required")
}

func detailedLinesBookedForDeltaTest(phaseIdx int, lines usagebased.DetailedLines) usagebased.DetailedLines {
	booked := lines.Clone()
	for idx := range booked {
		booked[idx].ID = "phase-" + strconv.Itoa(phaseIdx+1) + "-line-" + strconv.Itoa(idx+1)
	}

	return booked
}

func newIntentForTest(t testing.TB, servicePeriod timeutil.ClosedPeriod) usagebased.Intent {
	t.Helper()

	return ratingtestutils.NewUnitPriceIntentForTest(t, servicePeriod, alpacadecimal.NewFromInt(3))
}

type stubRatingService struct {
	result   billingrating.GenerateDetailedLinesResult
	lastOpts billingrating.GenerateDetailedLinesOptions
}

func (s *stubRatingService) ResolveBillablePeriod(in billingrating.ResolveBillablePeriodInput) (*timeutil.ClosedPeriod, error) {
	return nil, nil
}

func (s *stubRatingService) GenerateDetailedLines(in billingrating.StandardLineAccessor, opts ...billingrating.GenerateDetailedLinesOption) (billingrating.GenerateDetailedLinesResult, error) {
	s.lastOpts = billingrating.NewGenerateDetailedLinesOptions(opts...)
	return s.result, nil
}
