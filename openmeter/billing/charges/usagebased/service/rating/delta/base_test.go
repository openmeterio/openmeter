package delta

import (
	"strconv"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	ratingtestutils "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating/testutils"
	billingratingservice "github.com/openmeterio/openmeter/openmeter/billing/rating/service"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type deltaRatingTestPeriodsValue struct {
	period1 timeutil.ClosedPeriod
	period2 timeutil.ClosedPeriod
	period3 timeutil.ClosedPeriod
}

type deltaRatingTestCase struct {
	price     productcatalog.Price
	discounts productcatalog.Discounts
	phases    []deltaRatingPhase
}

type deltaRatingPhase struct {
	period                timeutil.ClosedPeriod
	meteredQuantity       float64
	expectedDetailedLines []ratingtestutils.ExpectedDetailedLine
	expectedTotals        ratingtestutils.ExpectedTotals
}

func runDeltaRatingTestCase(t *testing.T, tc deltaRatingTestCase) {
	t.Helper()

	require.NotEmpty(t, tc.phases)

	fullServicePeriod := timeutil.ClosedPeriod{
		From: tc.phases[0].period.From,
		To:   tc.phases[len(tc.phases)-1].period.To,
	}

	intent := ratingtestutils.NewIntentForTest(t, fullServicePeriod, tc.price, tc.discounts)
	engine := New(billingratingservice.New())
	bookedDetailedLinesByPhase := make([]usagebased.DetailedLines, len(tc.phases))

	for phaseIdx, phase := range tc.phases {
		t.Run("phase "+strconv.Itoa(phaseIdx+1), func(t *testing.T) {
			alreadyBilledDetailedLines := make(usagebased.DetailedLines, 0, 32)
			for priorPhaseIdx := 0; priorPhaseIdx < phaseIdx; priorPhaseIdx++ {
				alreadyBilledDetailedLines = append(alreadyBilledDetailedLines, bookedDetailedLinesByPhase[priorPhaseIdx]...)
			}

			out, err := engine.Rate(t.Context(), Input{
				Intent: intent,
				CurrentPeriod: CurrentPeriod{
					MeteredQuantity: alpacadecimal.NewFromFloat(phase.meteredQuantity),
					ServicePeriod:   phase.period,
				},
				AlreadyBilledDetailedLines: alreadyBilledDetailedLines,
			})
			require.NoError(t, err)
			require.Equal(t, phase.expectedDetailedLines, ratingtestutils.ToExpectedDetailedLinesWithServicePeriod(out.DetailedLines))
			require.Equal(t, phase.expectedTotals, ratingtestutils.ToExpectedTotals(out.DetailedLines.SumTotals()))

			bookedDetailedLinesByPhase[phaseIdx] = detailedLinesBookedForDeltaTest(phaseIdx, out.DetailedLines)
		})
	}
}

func deltaRatingTestPeriods() deltaRatingTestPeriodsValue {
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

	return deltaRatingTestPeriodsValue{
		period1: period1,
		period2: period2,
		period3: period3,
	}
}
