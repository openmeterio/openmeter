package clickhouse

import (
	_ "embed"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/huandu/go-sqlbuilder"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type queryMeter struct {
	Database        string
	EventsTableName string
	Namespace       string
	Meter           meterpkg.Meter
	Subject         []string
	FilterGroupBy   map[string][]string
	From            *time.Time
	To              *time.Time
	GroupBy         []string
	WindowSize      *meterpkg.WindowSize
	WindowTimeZone  *time.Location
	QuerySettings   map[string]string
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
	getColumn := columnFactory(d.EventsTableName)
	timeColumn := getColumn("time")

	query := sqlbuilder.ClickHouse.NewSelectBuilder()
	query.Select("count() AS total")
	query.From(tableName)

	// The event table is ordered by namespace, type
	query.Where(query.Equal("namespace", d.Namespace))
	query.Where(query.Equal("type", d.Meter.EventType))

	if len(d.Subject) > 0 {
		mapFunc := func(subject string) string {
			return query.Equal(getColumn("subject"), subject)
		}

		query.Where(query.Or(slicesx.Map(d.Subject, mapFunc)...))
	}

	// The event table is partitioned by time
	from := d.from()

	if from != nil {
		query.Where(query.GreaterEqualThan(timeColumn, from.Unix()))
	}

	if d.To != nil {
		query.Where(query.LessThan(timeColumn, d.To.Unix()))
	}

	sql, args := query.Build()
	return sql, args
}

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
				fmt.Sprintf("tumbleStart(%s, toIntervalMonth(1), '%s') AS windowstart", timeColumn, tz),
				fmt.Sprintf("tumbleEnd(%s, toIntervalMonth(1), '%s') AS windowend", timeColumn, tz),
			)

		default:
			return "", nil, fmt.Errorf("invalid window size type: %s", *d.WindowSize)
		}

		groupByColumns = append(groupByColumns, "windowstart", "windowend")
	} else {
		// TODO: remove this when we don't round to the nearest minute anymore
		// We round them to the nearest minute to ensure the result is the same as with
		// streaming connector using materialized views with per minute windows
		selectColumn := fmt.Sprintf("toStartOfMinute(min(%s)) AS windowstart, toStartOfMinute(max(%s)) + INTERVAL 1 MINUTE AS windowend", timeColumn, timeColumn)
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
		return "", []interface{}{}, fmt.Errorf("invalid aggregation type: %s", d.Meter.Aggregation)
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

	// Prewhere clauses

	query.Where(query.Equal(getColumn("namespace"), d.Namespace))
	query.Where(query.Equal(getColumn("type"), d.Meter.EventType))

	if len(d.Subject) > 0 {
		mapFunc := func(subject string) string {
			return query.Equal(getColumn("subject"), subject)
		}

		query.Where(query.Or(slicesx.Map(d.Subject, mapFunc)...))
	}

	// Apply the time where clause
	from := d.from()

	if from != nil {
		query.Where(query.GreaterEqualThan(timeColumn, from.Unix()))
	}

	if d.To != nil {
		query.Where(query.LessThan(timeColumn, d.To.Unix()))
	}

	var sqlPreWhere string

	if len(d.FilterGroupBy) > 0 {
		sqlPreWhere, _ = query.Build()
		dataColumn := getColumn("data")

		// We sort the group bys to ensure the query is deterministic
		filterGroupByKeys := make([]string, 0, len(d.FilterGroupBy))
		for k := range d.FilterGroupBy {
			filterGroupByKeys = append(filterGroupByKeys, k)
		}
		sort.Strings(filterGroupByKeys)

		// Where clauses
		for _, groupByKey := range filterGroupByKeys {
			if _, ok := d.Meter.GroupBy[groupByKey]; !ok {
				return "", nil, fmt.Errorf("meter does not have group by: %s", groupByKey)
			}

			groupByJSONPath := sqlbuilder.Escape(d.Meter.GroupBy[groupByKey])

			values := d.FilterGroupBy[groupByKey]
			if len(values) == 0 {
				return "", nil, fmt.Errorf("empty filter for group by: %s", groupByKey)
			}
			mapFunc := func(value string) string {
				return fmt.Sprintf("JSON_VALUE(%s, '%s') = '%s'", dataColumn, groupByJSONPath, sqlbuilder.Escape((value)))
			}

			query.Where(query.Or(slicesx.Map(values, mapFunc)...))
		}
	}

	// Group by
	query.GroupBy(groupByColumns...)

	if groupByWindowSize {
		query.OrderBy("windowstart")
	}

	sql, args := query.Build()

	// Only add prewhere if there are filters on JSON data
	if sqlPreWhere != "" {
		sqlParts := strings.Split(sql, sqlPreWhere)
		sqlAfter := sqlParts[1]

		if strings.HasPrefix(sqlAfter, " AND") {
			sqlAfter = strings.Replace(sqlAfter, "AND", "WHERE", 1)
		}

		sqlPreWhere = strings.Replace(sqlPreWhere, "WHERE", "PREWHERE", 1)
		sql = fmt.Sprintf("%s%s", sqlPreWhere, sqlAfter)
	}

	// Add settings
	settings := []string{
		"optimize_move_to_prewhere = 1",
		"allow_reorder_prewhere_conditions = 1",
	}
	for key, value := range d.QuerySettings {
		settings = append(settings, fmt.Sprintf("%s = %s", key, value))
	}

	sql = sql + fmt.Sprintf(" SETTINGS %s", strings.Join(settings, ", "))

	return sql, args, nil
}

// scanRows scans the rows from the query and returns a list of meter query rows.
func (queryMeter queryMeter) scanRows(rows driver.Rows) ([]meterpkg.MeterQueryRow, error) {
	values := []meterpkg.MeterQueryRow{}

	for rows.Next() {
		row := meterpkg.MeterQueryRow{
			GroupBy: map[string]*string{},
		}

		var value *float64
		args := []interface{}{&row.WindowStart, &row.WindowEnd, &value}
		argCount := len(args)

		for range queryMeter.GroupBy {
			tmp := ""
			args = append(args, &tmp)
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

		for i, key := range queryMeter.GroupBy {
			if s, ok := args[i+argCount].(*string); ok {
				// Subject is a top level field
				if key == "subject" {
					row.Subject = s
					continue
				}

				// We treat empty string as nil
				if s != nil && *s == "" {
					row.GroupBy[key] = nil
				} else {
					row.GroupBy[key] = s
				}
			}
		}

		// an empty row is returned when there are no values for the meter
		if row.WindowStart.IsZero() && row.WindowEnd.IsZero() && row.Value == 0 {
			continue
		}

		values = append(values, row)
	}

	err := rows.Err()
	if err != nil {
		return values, fmt.Errorf("rows error: %w", err)
	}

	return values, nil
}

type listMeterSubjectsQuery struct {
	Database        string
	EventsTableName string
	Namespace       string
	Meter           meterpkg.Meter
	From            *time.Time
	To              *time.Time
}

func (d listMeterSubjectsQuery) toSQL() (string, []interface{}) {
	tableName := getTableName(d.Database, d.EventsTableName)

	sb := sqlbuilder.ClickHouse.NewSelectBuilder()
	sb.Select("DISTINCT subject")
	sb.Where(sb.Equal("namespace", d.Namespace))
	sb.Where(sb.Equal("type", d.Meter.EventType))
	sb.From(tableName)
	sb.OrderBy("subject")

	if d.From != nil {
		sb.Where(sb.GreaterEqualThan("time", d.From.Unix()))
	}

	if d.To != nil {
		sb.Where(sb.LessThan("time", d.To.Unix()))
	}

	sql, args := sb.Build()
	return sql, args
}

func columnFactory(alias string) func(string) string {
	return func(column string) string {
		return fmt.Sprintf("%s.%s", alias, column)
	}
}
