package clickhouse

import (
	"fmt"
	"sort"

	"github.com/huandu/go-sqlbuilder"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

// queryMeterTableV2 struct holds the parameters for v2 meter queries
type queryMeterTableV2 struct {
	Database        string
	EventsTableName string
	Namespace       string
	Meter           meter.Meter
	Params          streaming.QueryParamsV2
}

// toSQL generates the SQL query and arguments for fetching events with v2 filtering
func (q queryMeterTableV2) toSQL() (string, []interface{}, error) {
	tableName := getTableName(q.Database, q.EventsTableName)
	getColumn := columnFactory(q.EventsTableName)
	timeColumn := getColumn("time")

	var selectColumns, groupByColumns, where []string

	// Select windows if any
	groupByWindowSize := q.Params.WindowSize != nil

	tz := "UTC"
	if q.Params.WindowTimeZone != nil {
		tz = q.Params.WindowTimeZone.String()
	}

	if groupByWindowSize {
		switch *q.Params.WindowSize {
		case meter.WindowSizeMinute:
			selectColumns = append(
				selectColumns,
				fmt.Sprintf("tumbleStart(%s, toIntervalMinute(1), '%s') AS windowstart", timeColumn, tz),
				fmt.Sprintf("tumbleEnd(%s, toIntervalMinute(1), '%s') AS windowend", timeColumn, tz),
			)

		case meter.WindowSizeHour:
			selectColumns = append(
				selectColumns,
				fmt.Sprintf("tumbleStart(%s, toIntervalHour(1), '%s') AS windowstart", timeColumn, tz),
				fmt.Sprintf("tumbleEnd(%s, toIntervalHour(1), '%s') AS windowend", timeColumn, tz),
			)

		case meter.WindowSizeDay:
			selectColumns = append(
				selectColumns,
				fmt.Sprintf("tumbleStart(%s, toIntervalDay(1), '%s') AS windowstart", timeColumn, tz),
				fmt.Sprintf("tumbleEnd(%s, toIntervalDay(1), '%s') AS windowend", timeColumn, tz),
			)

		default:
			return "", nil, fmt.Errorf("invalid window size type: %s", *q.Params.WindowSize)
		}

		groupByColumns = append(groupByColumns, "windowstart", "windowend")
	} else {
		// TODO: remove this when we don't round to the nearest minute anymore
		// We round them to the nearest minute to ensure the result is the same as with
		// streaming connector using materialized views with per minute windows
		selectColumn := fmt.Sprintf("tumbleStart(min(%s), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(%s), toIntervalMinute(1)) AS windowend", timeColumn, timeColumn)
		selectColumns = append(selectColumns, selectColumn)
	}

	// Select Value
	sqlAggregation := ""
	switch q.Meter.Aggregation {
	case meter.MeterAggregationSum:
		sqlAggregation = "sum"
	case meter.MeterAggregationAvg:
		sqlAggregation = "avg"
	case meter.MeterAggregationMin:
		sqlAggregation = "min"
	case meter.MeterAggregationMax:
		sqlAggregation = "max"
	case meter.MeterAggregationUniqueCount:
		sqlAggregation = "uniq"
	case meter.MeterAggregationCount:
		sqlAggregation = "count"
	default:
		return "", []interface{}{}, fmt.Errorf("invalid aggregation type: %s", q.Meter.Aggregation)
	}

	switch q.Meter.Aggregation {
	case meter.MeterAggregationCount:
		selectColumns = append(selectColumns, fmt.Sprintf("toFloat64(%s(*)) AS value", sqlAggregation))
	case meter.MeterAggregationUniqueCount:
		selectColumns = append(selectColumns, fmt.Sprintf("toFloat64(%s(JSON_VALUE(%s, '%s'))) AS value", sqlAggregation, getColumn("data"), sqlbuilder.Escape(*q.Meter.ValueProperty)))
	default:
		// JSON_VALUE returns an empty string if the JSON Path is not found. With toFloat64OrNull we convert it to NULL so the aggregation function can handle it properly.
		selectColumns = append(selectColumns, fmt.Sprintf("%s(toFloat64OrNull(JSON_VALUE(%s, '%s'))) AS value", sqlAggregation, getColumn("data"), sqlbuilder.Escape(*q.Meter.ValueProperty)))
	}

	for _, groupByKey := range q.Params.GroupBy {
		// Subject is a special case as it's a top level column
		if groupByKey == "subject" {
			selectColumns = append(selectColumns, getColumn("subject"))
			groupByColumns = append(groupByColumns, "subject")
			continue
		}

		// Group by columns need to be parsed from the JSON data
		groupByColumn := sqlbuilder.Escape(groupByKey)
		groupByJSONPath := sqlbuilder.Escape(q.Meter.GroupBy[groupByKey])
		selectColumn := fmt.Sprintf("JSON_VALUE(%s, '%s') as %s", getColumn("data"), groupByJSONPath, groupByColumn)

		selectColumns = append(selectColumns, selectColumn)
		groupByColumns = append(groupByColumns, groupByColumn)
	}

	query := sqlbuilder.ClickHouse.NewSelectBuilder()
	query.Select(selectColumns...)
	query.From(tableName)
	query.Where(query.Equal(getColumn("namespace"), q.Namespace))
	query.Where(query.Equal(getColumn("type"), q.Meter.EventType))

	// If the meter has an event from time, we filter by it before applying the query params filters
	if q.Meter.EventFrom != nil {
		where = append(where, query.GreaterEqualThan(timeColumn, *q.Meter.EventFrom))
	}

	if q.Params.Filter != nil {
		if q.Params.Filter.Subject != nil {
			expr := q.Params.Filter.Subject.SelectWhereExpr(getColumn("subject"), query)
			if expr != "" {
				where = append(where, expr)
			}
		}

		if q.Params.Filter.Time != nil {
			expr := q.Params.Filter.Time.SelectWhereExpr(timeColumn, query)
			if expr != "" {
				where = append(where, expr)
			}
		}

		if q.Params.Filter.GroupBy != nil {
			// We sort the group by s to ensure the query is deterministic
			groupByKeys := make([]string, 0, len(*q.Params.Filter.GroupBy))
			for k := range *q.Params.Filter.GroupBy {
				groupByKeys = append(groupByKeys, k)
			}
			sort.Strings(groupByKeys)

			for _, k := range groupByKeys {
				f := (*q.Params.Filter.GroupBy)[k]
				groupByJSONPath := q.Meter.GroupBy[k]

				column := fmt.Sprintf("JSON_VALUE(%s, '%s')", getColumn("data"), groupByJSONPath)
				expr := f.SelectWhereExpr(column, query)
				if expr != "" {
					where = append(where, expr)
				}
			}
		}
	}

	if len(where) > 0 {
		query.Where(where...)
	}

	query.GroupBy(groupByColumns...)

	if groupByWindowSize {
		query.OrderBy("windowstart")
	}

	sql, args := query.Build()
	return sql, args, nil
}

// toCountRowSQL returns the SQL query for the estimated number of rows.
// This estimate is useful for query progress tracking.
// We only filter by columns that are in the ClickHouse table order.
func (q *queryMeterTableV2) toCountRowSQL() (string, []interface{}) {
	tableName := getTableName(q.Database, q.EventsTableName)
	getColumn := columnFactory(q.EventsTableName)
	timeColumn := getColumn("time")

	query := sqlbuilder.ClickHouse.NewSelectBuilder()
	query.Select("count() AS total")
	query.From(tableName)

	// The event table is ordered by namespace, type
	query.Where(query.Equal("namespace", q.Namespace))
	query.Where(query.Equal("type", q.Meter.EventType))

	if q.Params.Filter != nil {
		if q.Params.Filter.Subject != nil {
			expr := q.Params.Filter.Subject.SelectWhereExpr(getColumn("subject"), query)
			if expr != "" {
				query.Where(expr)
			}
		}

		if q.Params.Filter.Time != nil {
			expr := q.Params.Filter.Time.SelectWhereExpr(timeColumn, query)
			if expr != "" {
				query.Where(expr)
			}
		}
	}

	if q.Meter.EventFrom != nil {
		query.Where(query.GreaterEqualThan(timeColumn, *q.Meter.EventFrom))
	}

	sql, args := query.Build()
	return sql, args
}
