package rate_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/rating/service/rate"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestProgressiveBillingMeteredPricerResolveBillablePeriod(t *testing.T) {
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC),
	}
	line := billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ServicePeriod: period,
		},
	}
	pricer := rate.ProgressiveBillingMeteredPricer{}

	t.Run("not billable at service period start", func(t *testing.T) {
		result, err := pricer.ResolveBillablePeriod(rating.ResolveBillablePeriodInput{
			Line:               line,
			AsOf:               period.From,
			ProgressiveBilling: true,
		})
		require.NoError(t, err)
		require.Equal(t, billing.IsLineBillableAsOfResult{}, result)
		require.NoError(t, result.Validate())
	})

	t.Run("billable for elapsed partial period", func(t *testing.T) {
		asOf := period.From.Add(time.Hour)
		result, err := pricer.ResolveBillablePeriod(rating.ResolveBillablePeriodInput{
			Line:               line,
			AsOf:               asOf,
			ProgressiveBilling: true,
		})
		require.NoError(t, err)
		require.Equal(t, billing.IsLineBillableAsOfResult{
			Billable: true,
			BillablePeriod: timeutil.ClosedPeriod{
				From: period.From,
				To:   asOf,
			},
		}, result)
		require.NoError(t, result.Validate())
	})
}
