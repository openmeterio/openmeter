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
	QueryUsage(ctx context.Context, ownerID models.NamespacedID, period timeutil.ClosedPeriod) (float64, error)
}

// PhaseUsageReader queries the same customer, meter and filters for a half-open interval.
// Value also reports whether the scalar read suggested an empty interval. Only
// HasEvents can certify that no matching source rows exist.
type PhaseUsageReader struct {
	Value     func(timeutil.ClosedPeriod) (value float64, empty bool, err error)
	HasEvents func(timeutil.ClosedPeriod) (bool, error)
}

// PhaseUsageQuery returns the phase's raw usage. The owner applies unit conversion once.
type PhaseUsageQuery func(timeutil.ClosedPeriod, PhaseUsageReader) (float64, error)

type UsageQuerierConfig struct {
	PhaseUsageQuery       PhaseUsageQuery
	StreamingConnector    streaming.Connector
	DescribeOwner         func(ctx context.Context, id models.NamespacedID) (grant.Owner, error)
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

func (u *usageQuerier) QueryUsage(ctx context.Context, ownerID models.NamespacedID, period timeutil.ClosedPeriod) (float64, error) {
	owner, err := u.DescribeOwner(ctx, ownerID)
	if err != nil {
		return 0.0, err
	}

	params := owner.DefaultQueryParams

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

		return owner.ConvertUsage(vTo.Sub(vFrom).InexactFloat64()), nil

	// For other aggregation types we can simply query the meter
	case meter.MeterAggregationSum, meter.MeterAggregationCount, meter.MeterAggregationLatest:
		// If the two times are the same we can return 0.0 as there's no usage
		if period.From.Equal(period.To) {
			return 0.0, nil
		}

		usage, err := u.queryPhase(ctx, ownerID.Namespace, owner, period)
		if err != nil {
			return 0.0, err
		}

		// Convert raw metered usage into the entitlement's billing unit (OM-400).
		// Identity when the owner has no UnitConfig. Both the engine's usage source
		// and the snapshot service's backfill querier route through here, so converted
		// and raw usage can never be mixed for a given owner.
		return owner.ConvertUsage(usage), nil
	default:
		return 0.0, fmt.Errorf("unsupported aggregation %s", owner.Meter.Aggregation)
	}
}

func (u *usageQuerier) queryPhase(ctx context.Context, namespace string, owner grant.Owner, period timeutil.ClosedPeriod) (float64, error) {
	read := func(p timeutil.ClosedPeriod) (float64, bool, error) {
		query := owner.DefaultQueryParams
		query.From, query.To = &p.From, &p.To
		rows, err := u.StreamingConnector.QueryMeter(ctx, namespace, owner.Meter, query)
		if err != nil {
			return 0, false, fmt.Errorf("failed to query meter %s: %w", owner.Meter.Key, err)
		}

		value, err := u.getValueFromRows(rows)
		empty := len(rows) == 0 || (owner.Meter.Aggregation == meter.MeterAggregationCount && value == 0)

		return value, empty, err
	}

	if u.PhaseUsageQuery == nil {
		value, _, err := read(period)
		return value, err
	}

	reader := PhaseUsageReader{Value: read}
	switch owner.Meter.Aggregation {
	case meter.MeterAggregationCount:
		reader.HasEvents = func(p timeutil.ClosedPeriod) (bool, error) {
			count, _, err := read(p)
			return count != 0, err
		}
	case meter.MeterAggregationSum:
		source, ok := u.StreamingConnector.(streaming.MeterEventChecker)
		if !ok {
			value, _, err := read(period)
			return value, err
		}

		reader.HasEvents = func(p timeutil.ClosedPeriod) (bool, error) {
			query := owner.DefaultQueryParams
			query.From, query.To = &p.From, &p.To

			return source.HasMeterEvents(ctx, namespace, owner.Meter, query)
		}
	default:
		value, _, err := read(period)
		return value, err
	}

	return u.PhaseUsageQuery(period, reader)
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
