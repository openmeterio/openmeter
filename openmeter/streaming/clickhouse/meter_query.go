package clickhouse

import (
	_ "embed"
	"fmt"
	"math"
	"slices"
	"sort"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/huandu/go-sqlbuilder"
	"github.com/samber/lo"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type queryMeter struct {
	Database        string
	EventsTableName string
	Namespace       string
	Meter           meterpkg.Meter
	FilterCustomer  []streaming.Customer
	FilterSubject   []string
	FilterGroupBy   map[string][]string
	From            *time.Time
	To              *time.Time
	GroupBy         []string
	WindowSize      *meterpkg.WindowSize
	WindowTimeZone  *time.Location
}

// from returns the from time for the query.
func (d *queryMeter) from() *time.Time {
	// If the query from time is set, use it
	from := d.From

	// If none of the from times are set, return nil
	if from == nil && d.Meter.EventFrom == nil {
		return nil
	}

	// If only the event from time is set, use it
	if from == nil && d.Meter.EventFrom != nil {
		return d.Meter.EventFrom
	}

	// If only the query from time is set, use it
	if from != nil && d.Meter.EventFrom == nil {
		return from
	}

	// If both the query from time and the event from time are set
	// use the query from time if it's after the event from time
	if from.After(*d.Meter.EventFrom) {
		return from
	}

	return d.Meter.EventFrom
}

// toCountRowSQL returns the SQL query for the estimated number of rows.
// This estimate is useful for query progress tracking.
// We only filter by columns that are in the ClickHouse table order.
func (d *queryMeter) toCountRowSQL() (string, []interface{}) {
	tableName := getTableName(d.Database, d.EventsTableName)

	query := sqlbuilder.ClickHouse.NewSelectBuilder()
	query.Select("count() AS total")
	query.From(tableName)

	// Where by ordered columns
	query = d.whereByOrderedColumns(query)

	sql, args := query.Build()
	return sql, args
}

// toSQL returns the SQL query for the meter query.
func (d *queryMeter) toSQL() (string, []interface{}, error) {
	tableName := getTableName(d.Database, d.EventsTableName)
	getColumn := columnFactory(d.EventsTableName)
	timeColumn := getColumn("time")

	var selectColumns, groupByColumns []string

	// Select windows if any
	groupByWindowSize := d.WindowSize != nil

	tz := "UTC"
	if d.WindowTimeZone != nil {
		tz = d.WindowTimeZone.String()
	}

	if groupByWindowSize {
		switch *d.WindowSize {
		case meterpkg.WindowSizeMinute:
			selectColumns = append(
				selectColumns,
				fmt.Sprintf("tumbleStart(%s, toIntervalMinute(1), '%s') AS windowstart", timeColumn, tz),
				fmt.Sprintf("tumbleEnd(%s, toIntervalMinute(1), '%s') AS windowend", timeColumn, tz),
			)

		case meterpkg.WindowSizeHour:
			selectColumns = append(
				selectColumns,
				fmt.Sprintf("tumbleStart(%s, toIntervalHour(1), '%s') AS windowstart", timeColumn, tz),
				fmt.Sprintf("tumbleEnd(%s, toIntervalHour(1), '%s') AS windowend", timeColumn, tz),
			)

		case meterpkg.WindowSizeDay:
			selectColumns = append(
				selectColumns,
				fmt.Sprintf("tumbleStart(%s, toIntervalDay(1), '%s') AS windowstart", timeColumn, tz),
				fmt.Sprintf("tumbleEnd(%s, toIntervalDay(1), '%s') AS windowend", timeColumn, tz),
			)

		case meterpkg.WindowSizeMonth:
			selectColumns = append(
				selectColumns,
				// We need to convert the tumbleStart and tumbleEnd to DateTime, as otherwise we got a Date type. Given
				// we are scanning the result into a time.Time, we will end up with the correct date in UTC. In case the timezone
				// is not UTC, the returned values will be offset by the timezone difference.
				//
				// e.g.:
				//  if timezone is Europe/Budapest, then if we are not casting to DateTime, then:
				// 	 tumbleStart will return 2025-01-01 which will become 2025-01-01 00:00:00 in UTC
				//   this is wrong, as in CET this is 2024-12-31 23:00:00
				//  if we are casting to DateTime, then:
				// 	 tumbleStart will return 2025-01-01 00:00:00 in Europe/Budapest

				// Other queries are not affected by this, as for anything < Month, the result is always a DateTime (most probably due to
				// DST changes).
				fmt.Sprintf("toDateTime(tumbleStart(%s, toIntervalMonth(1), '%s'), '%s') AS windowstart", timeColumn, tz, tz),
				fmt.Sprintf("toDateTime(tumbleEnd(%s, toIntervalMonth(1), '%s'), '%s') AS windowend", timeColumn, tz, tz),
			)

		default:
			return "", nil, models.NewGenericValidationError(
				fmt.Errorf("invalid window size type: %s", *d.WindowSize),
			)
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
	switch d.Meter.Aggregation {
	case meterpkg.MeterAggregationSum:
		sqlAggregation = "sum"
	case meterpkg.MeterAggregationAvg:
		sqlAggregation = "avg"
	case meterpkg.MeterAggregationMin:
		sqlAggregation = "min"
	case meterpkg.MeterAggregationMax:
		sqlAggregation = "max"
	case meterpkg.MeterAggregationUniqueCount:
		sqlAggregation = "uniq"
	case meterpkg.MeterAggregationCount:
		sqlAggregation = "count"
	case meterpkg.MeterAggregationLatest:
		sqlAggregation = "argMax"
	default:
		return "", []interface{}{}, models.NewGenericValidationError(
			fmt.Errorf("invalid aggregation type: %s", d.Meter.Aggregation),
		)
	}

	switch d.Meter.Aggregation {
	case meterpkg.MeterAggregationCount:
		selectColumns = append(selectColumns, fmt.Sprintf("toFloat64(%s(*)) AS value", sqlAggregation))
	case meterpkg.MeterAggregationUniqueCount:
		selectColumns = append(selectColumns, fmt.Sprintf("toFloat64(%s(JSON_VALUE(%s, '%s'))) AS value", sqlAggregation, getColumn("data"), sqlbuilder.Escape(*d.Meter.ValueProperty)))
	case meterpkg.MeterAggregationLatest:
		selectColumns = append(selectColumns, fmt.Sprintf("%s(ifNotFinite(toFloat64OrNull(JSON_VALUE(%s, '%s')), null), %s) AS value", sqlAggregation, getColumn("data"), sqlbuilder.Escape(*d.Meter.ValueProperty), timeColumn))
	default:
		// JSON_VALUE returns an empty string if the JSON Path is not found. With toFloat64OrNull we convert it to NULL so the aggregation function can handle it properly.
		selectColumns = append(selectColumns, fmt.Sprintf("%s(ifNotFinite(toFloat64OrNull(JSON_VALUE(%s, '%s')), null)) AS value", sqlAggregation, getColumn("data"), sqlbuilder.Escape(*d.Meter.ValueProperty)))
	}

	for _, groupByKey := range d.GroupBy {
		// Subject is a special case as it's a top level column
		if groupByKey == "subject" {
			selectColumns = append(selectColumns, getColumn("subject"))
			groupByColumns = append(groupByColumns, "subject")
			continue
		}

		// Customer ID is a special case as it's a top level column
		if groupByKey == "customer_id" {
			groupByColumns = append(groupByColumns, "customer_id")
			continue
		}

		// Group by columns need to be parsed from the JSON data
		groupByColumn := sqlbuilder.Escape(groupByKey)
		groupByJSONPath := sqlbuilder.Escape(d.Meter.GroupBy[groupByKey])
		selectColumn := fmt.Sprintf("JSON_VALUE(%s, '%s') as %s", getColumn("data"), groupByJSONPath, groupByColumn)

		selectColumns = append(selectColumns, selectColumn)
		groupByColumns = append(groupByColumns, groupByColumn)
	}

	query := sqlbuilder.ClickHouse.NewSelectBuilder()
	query.Select(selectColumns...)
	query.From(tableName)

	// Select customer id column if it's in the group by
	if slices.Contains(d.GroupBy, "customer_id") {
		query = selectCustomerIdColumn(d.EventsTableName, d.FilterCustomer, query)
	}

	// Where by ordered columns
	query = d.whereByOrderedColumns(query)

	// Where by columns not in the order of the event table
	if len(d.FilterGroupBy) > 0 {
		// We sort the group by s to ensure the query is deterministic
		groupByKeys := make([]string, 0, len(d.FilterGroupBy))
		for k := range d.FilterGroupBy {
			groupByKeys = append(groupByKeys, k)
		}
		sort.Strings(groupByKeys)

		for _, groupByKey := range groupByKeys {
			if _, ok := d.Meter.GroupBy[groupByKey]; !ok {
				return "", nil, models.NewGenericValidationError(
					fmt.Errorf("meter does not have group by: %s", groupByKey),
				)
			}

			groupByJSONPath := sqlbuilder.Escape(d.Meter.GroupBy[groupByKey])

			values := d.FilterGroupBy[groupByKey]
			if len(values) == 0 {
				return "", nil, models.NewGenericValidationError(
					fmt.Errorf("empty filter for group by: %s", groupByKey),
				)
			}
			mapFunc := func(value string) string {
				column := fmt.Sprintf("JSON_VALUE(%s, '%s')", getColumn("data"), groupByJSONPath)

				// Subject is a special case
				if groupByKey == "subject" {
					column = "subject"
				}

				// Customer ID is a special case
				if groupByKey == "customer_id" {
					column = "customer_id"
				}

				return fmt.Sprintf("%s = '%s'", column, sqlbuilder.Escape((value)))
			}

			query = query.Where(query.Or(slicesx.Map(values, mapFunc)...))
		}
	}

	// Group by
	query = query.GroupBy(groupByColumns...)

	// Order by
	if groupByWindowSize {
		query = query.OrderBy("windowstart")
	}

	sql, args := query.Build()
	return sql, args, nil
}

// whereByOrderedColumns applies the where clause to the query for columns that are ordered by the event table.
// The event table is ordered by namespace, type, subject, time.
func (d *queryMeter) whereByOrderedColumns(query *sqlbuilder.SelectBuilder) *sqlbuilder.SelectBuilder {
	getColumn := columnFactory(d.EventsTableName)

	query.Where(query.Equal(getColumn("namespace"), d.Namespace))
	query.Where(query.Equal(getColumn("type"), d.Meter.EventType))
	query = customersWhere(d.EventsTableName, d.FilterCustomer, query)
	query = subjectWhere(d.EventsTableName, d.FilterSubject, query)
	query = d.timeWhere(query)

	return query
}

// timeWhere applies the time filter to the query.
func (d *queryMeter) timeWhere(query *sqlbuilder.SelectBuilder) *sqlbuilder.SelectBuilder {
	getColumn := columnFactory(d.EventsTableName)
	timeColumn := getColumn("time")

	from := d.from()

	if from != nil {
		query = query.Where(query.GreaterEqualThan(timeColumn, from.Unix()))
	}

	if d.To != nil {
		query = query.Where(query.LessThan(timeColumn, d.To.Unix()))
	}

	return query
}

// scanRows scans the rows from the query and returns a list of meter query rows.
func (queryMeter queryMeter) scanRows(rows driver.Rows) ([]meterpkg.MeterQueryRow, error) {
	values := []meterpkg.MeterQueryRow{}

	// Get the columns from the query
	columns := rows.Columns()

	if columns[0] != "windowstart" {
		return values, fmt.Errorf("first column is not windowstart")
	}

	if columns[1] != "windowend" {
		return values, fmt.Errorf("second column is not windowend")
	}

	if columns[2] != "value" {
		return values, fmt.Errorf("third column is not value")
	}

	// Scan the rows
	for rows.Next() {
		row := meterpkg.MeterQueryRow{
			GroupBy: map[string]*string{},
		}

		var value *float64
		args := []interface{}{&row.WindowStart, &row.WindowEnd, &value}
		argCount := len(args)

		if len(columns) > argCount {
			for range columns[argCount:] {
				args = append(args, lo.ToPtr(""))
			}
		}

		if err := rows.Scan(args...); err != nil {
			return values, fmt.Errorf("query meter view row scan: %w", err)
		}

		// If there is no value for the period, we skip the row
		// This can happen when the event doesn't have the value field.
		if value == nil {
			continue
		}

		// TODO: should we use decima all the way?
		row.Value = *value

		if math.IsNaN(row.Value) {
			return values, fmt.Errorf("value is NaN")
		}

		if math.IsInf(row.Value, 0) {
			return values, fmt.Errorf("value is infinite")
		}

		for i, column := range columns[argCount:] {
			if s, ok := args[i+argCount].(*string); ok {
				// Subject is a top level field
				if column == "subject" {
					row.Subject = s
					continue
				}

				// Customer ID is a top level field
				if column == "customer_id" {
					row.CustomerID = s
					continue
				}

				// Consistency check
				if !slices.Contains(queryMeter.GroupBy, column) {
					return values, fmt.Errorf("column %s is not a valid group by", column)
				}

				row.GroupBy[column] = s
			}
		}

		values = append(values, row)
	}

	err := rows.Err()
	if err != nil {
		return values, fmt.Errorf("rows error: %w", err)
	}

	return values, nil
}
