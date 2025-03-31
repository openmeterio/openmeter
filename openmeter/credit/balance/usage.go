package balance

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// UsageQuerier is a helper for querying usage for a given owner and period.
type UsageQuerier interface {
	QueryUsage(ctx context.Context, ownerID models.NamespacedID, period timeutil.Period) (float64, error)
}

type UsageQuerierConfig struct {
	StreamingConnector    streaming.Connector
	DescribeOwner         func(ctx context.Context, id models.NamespacedID) (grant.Owner, error)
	GetDefaultParams      func(ctx context.Context, ownerID models.NamespacedID) (streaming.QueryParams, error)
	GetUsagePeriodStartAt func(ctx context.Context, ownerID models.NamespacedID, at time.Time) (time.Time, error)
}

type usageQuerier struct {
	UsageQuerierConfig
}

func NewUsageQuerier(conf UsageQuerierConfig) UsageQuerier {
	return &usageQuerier{
		UsageQuerierConfig: conf,
	}
}

var _ UsageQuerier = (*usageQuerier)(nil)

func (u *usageQuerier) QueryUsage(ctx context.Context, ownerID models.NamespacedID, period timeutil.Period) (float64, error) {
	params, err := u.GetDefaultParams(ctx, ownerID)
	if err != nil {
		return 0.0, err
	}

	owner, err := u.DescribeOwner(ctx, ownerID)
	if err != nil {
		return 0.0, err
	}

	// Let's query the meter based on the aggregation
	switch owner.Meter.Aggregation {
	case meter.MeterAggregationUniqueCount:
		periodStart, err := u.GetUsagePeriodStartAt(ctx, ownerID, period.From)
		if err != nil {
			return 0.0, err
		}

		// To get the UNIQUE_COUNT value between `from` and `to` we need to:
		// 1. Query between the period start and `to` to get the unique count at `to`
		// 2. Query between the period start and `from` to get the unique count at `from`
		// 3. Subtract the two values
		params.From = &periodStart
		params.To = &period.To

		var (
			valueTo   = 0.0
			valueFrom = 0.0
		)

		if !periodStart.Equal(period.To) {
			rows, err := u.StreamingConnector.QueryMeter(ctx, ownerID.Namespace, owner.Meter, params)
			if err != nil {
				return 0.0, fmt.Errorf("failed to query meter %s: %w", owner.Meter.Key, err)
			}

			valueTo, err = u.getValueFromRows(rows)
			if err != nil {
				return 0.0, err
			}
		}

		params.To = &period.From

		// If the two times are different we need to query the value at `from`
		if !params.From.Equal(*params.To) && !periodStart.Equal(period.From) {
			rows, err := u.StreamingConnector.QueryMeter(ctx, ownerID.Namespace, owner.Meter, params)
			if err != nil {
				return 0.0, fmt.Errorf("failed to query meter %s: %w", owner.Meter.Key, err)
			}

			valueFrom, err = u.getValueFromRows(rows)
			if err != nil {
				return 0.0, err
			}
		}

		// Let's do an accurate subsctraction
		vTo := alpacadecimal.NewFromFloat(valueTo)
		vFrom := alpacadecimal.NewFromFloat(valueFrom)

		return vTo.Sub(vFrom).InexactFloat64(), nil

	// For SUM and COUNT we can simply query the meter
	case meter.MeterAggregationSum, meter.MeterAggregationCount:
		// If the two times are the same we can return 0.0 as there's no usage
		if period.From.Equal(period.To) {
			return 0.0, nil
		}

		params.From = &period.From
		params.To = &period.To

		// Let's query the meter
		rows, err := u.StreamingConnector.QueryMeter(ctx, ownerID.Namespace, owner.Meter, params)
		if err != nil {
			return 0.0, fmt.Errorf("failed to query meter %s: %w", owner.Meter.Key, err)
		}

		return u.getValueFromRows(rows)
	default:
		return 0.0, fmt.Errorf("unsupported aggregation %s", owner.Meter.Aggregation)
	}
}

func (u *usageQuerier) getValueFromRows(rows []meter.MeterQueryRow) (float64, error) {
	// We expect only one row
	if len(rows) > 1 {
		return 0.0, fmt.Errorf("expected 1 row, got %d", len(rows))
	}
	if len(rows) == 0 {
		return 0.0, nil
	}
	return rows[0].Value, nil
}
