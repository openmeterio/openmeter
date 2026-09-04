package service_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	ratingservice "github.com/openmeterio/openmeter/openmeter/billing/rating/service"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestResolveBillablePeriodFallsBackToNonProgressiveBillingWithoutDependencies(t *testing.T) {
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	invoiceAt := period.To.Add(24 * time.Hour)
	line := billing.GatheringLine{GatheringLineBase: billing.GatheringLineBase{
		ServicePeriod: period,
		InvoiceAt:     invoiceAt,
		Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(1),
		}),
	}}
	service := ratingservice.New(ratingservice.Config{})

	t.Run("valid dependencies allow progressive billing", func(t *testing.T) {
		// Given a progressively billable metered line with its rating dependencies.
		asOf := period.From.Add(12 * time.Hour)

		// When billability is resolved before the normal invoice time.
		result, err := service.ResolveBillablePeriod(rating.ResolveBillablePeriodInput{
			AsOf:               asOf,
			ProgressiveBilling: true,
			Line:               line,
			Feature:            &feature.Feature{},
			Meter:              &meter.Meter{Aggregation: meter.MeterAggregationSum},
		})

		// Then the elapsed portion is billable progressively.
		require.NoError(t, err)
		require.Equal(t, billing.IsLineBillableAsOfResult{
			Billable: true,
			BillablePeriod: timeutil.ClosedPeriod{
				From: period.From,
				To:   asOf,
			},
		}, result)
	})

	t.Run("missing dependencies defer billing until invoice at", func(t *testing.T) {
		// Given the same line without a resolvable feature or meter.
		// When billability is resolved after service completion but before InvoiceAt.
		result, err := service.ResolveBillablePeriod(rating.ResolveBillablePeriodInput{
			AsOf:               period.To,
			ProgressiveBilling: true,
			Line:               line,
		})

		// Then progressive billing is disabled and the line is not yet billable.
		require.NoError(t, err)
		require.Equal(t, billing.IsLineBillableAsOfResult{}, result)
	})

	t.Run("missing dependencies yield the full period at invoice at", func(t *testing.T) {
		// Given the same line without a resolvable feature or meter.
		// When its normal invoice time is reached.
		result, err := service.ResolveBillablePeriod(rating.ResolveBillablePeriodInput{
			AsOf:               invoiceAt,
			ProgressiveBilling: true,
			Line:               line,
		})

		// Then the complete service period is billable non-progressively.
		require.NoError(t, err)
		require.Equal(t, billing.IsLineBillableAsOfResult{
			Billable:       true,
			BillablePeriod: period,
		}, result)
	})
}
