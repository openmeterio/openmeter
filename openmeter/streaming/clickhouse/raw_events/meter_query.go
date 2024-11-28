package raw_events

import (
	_ "embed"
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/huandu/go-sqlbuilder"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type queryMeter struct {
	Database        string
	EventsTableName string
	Namespace       string
	Meter           models.Meter
	Subject         []string
	FilterGroupBy   map[string][]string
	From            *time.Time
	To              *time.Time
	GroupBy         []string
	WindowSize      *models.WindowSize
	WindowTimeZone  *time.Location
}

func (d queryMeter) toSQL() (string, []interface{}, error) {
	tableName := getTableName(d.Database, d.EventsTableName)
	getColumn := columnFactory(d.EventsTableName)
	timeColumn := getColumn("time")

	var selectColumns, groupByColumns, where []string

	// Select windows if any
	groupByWindowSize := d.WindowSize != nil

	tz := "UTC"
	if d.WindowTimeZone != nil {
		tz = d.WindowTimeZone.String()
	}

	if groupByWindowSize {
		switch *d.WindowSize {
		case models.WindowSizeMinute:
			selectColumns = append(
				selectColumns,
				fmt.Sprintf("tumbleStart(%s, toIntervalMinute(1), '%s') AS windowstart", timeColumn, tz),
				fmt.Sprintf("tumbleEnd(%s, toIntervalMinute(1), '%s') AS windowend", timeColumn, tz),
			)

		case models.WindowSizeHour:
			selectColumns = append(
				selectColumns,
				fmt.Sprintf("tumbleStart(%s, toIntervalHour(1), '%s') AS windowstart", timeColumn, tz),
				fmt.Sprintf("tumbleEnd(%s, toIntervalHour(1), '%s') AS windowend", timeColumn, tz),
			)

		case models.WindowSizeDay:
			selectColumns = append(
				selectColumns,
				fmt.Sprintf("tumbleStart(%s, toIntervalDay(1), '%s') AS windowstart", timeColumn, tz),
				fmt.Sprintf("tumbleEnd(%s, toIntervalDay(1), '%s') AS windowend", timeColumn, tz),
			)

		default:
			return "", nil, fmt.Errorf("invalid window size type: %s", *d.WindowSize)
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
	case models.MeterAggregationSum:
		sqlAggregation = "sum"
	case models.MeterAggregationAvg:
		sqlAggregation = "avg"
	case models.MeterAggregationMin:
		sqlAggregation = "min"
	case models.MeterAggregationMax:
		sqlAggregation = "max"
	case models.MeterAggregationUniqueCount:
		sqlAggregation = "uniq"
	case models.MeterAggregationCount:
		sqlAggregation = "count"
	default:
		return "", []interface{}{}, fmt.Errorf("invalid aggregation type: %s", d.Meter.Aggregation)
	}

	if d.Meter.ValueProperty == "" && d.Meter.Aggregation == models.MeterAggregationCount {
		selectColumns = append(selectColumns, fmt.Sprintf("%s(*) AS value", sqlAggregation))
	} else if d.Meter.Aggregation == models.MeterAggregationUniqueCount {
		selectColumns = append(selectColumns, fmt.Sprintf("%s(JSON_VALUE(%s, '%s')) AS value", sqlAggregation, getColumn("data"), sqlbuilder.Escape(d.Meter.ValueProperty)))
	} else {
		selectColumns = append(selectColumns, fmt.Sprintf("%s(cast(JSON_VALUE(%s, '%s'), 'Float64')) AS value", sqlAggregation, getColumn("data"), sqlbuilder.Escape(d.Meter.ValueProperty)))
	}

	groupBys := make([]string, 0, len(d.GroupBy))

	for _, groupBy := range d.GroupBy {
		if groupBy == "subject" {
			selectColumns = append(selectColumns, getColumn("subject"))
			groupByColumns = append(groupByColumns, "subject")
			continue
		}

		groupBys = append(groupBys, groupBy)
	}

	// Select Group By
	slices.Sort(groupBys)

	for _, groupByKey := range groupBys {
		groupByColumn := sqlbuilder.Escape(groupByKey)
		groupByJSONPath := sqlbuilder.Escape(d.Meter.GroupBy[groupByKey])
		selectColumn := fmt.Sprintf("JSON_VALUE(%s, '%s') as %s", getColumn("data"), groupByJSONPath, groupByColumn)

		selectColumns = append(selectColumns, selectColumn)
		groupByColumns = append(groupByColumns, groupByColumn)
	}

	query := sqlbuilder.ClickHouse.NewSelectBuilder()
	query.Select(selectColumns...)
	query.From(tableName)
	query.Where(query.Equal(getColumn("namespace"), d.Namespace))
	query.Where(query.Equal(getColumn("type"), d.Meter.EventType))

	if len(d.Subject) > 0 {
		mapFunc := func(subject string) string {
			return query.Equal(getColumn("subject"), subject)
		}

		where = append(where, query.Or(slicesx.Map(d.Subject, mapFunc)...))
	}

	if len(d.FilterGroupBy) > 0 {
		// We sort the group by s to ensure the query is deterministic
		groupByKeys := make([]string, 0, len(d.FilterGroupBy))
		for k := range d.FilterGroupBy {
			groupByKeys = append(groupByKeys, k)
		}
		sort.Strings(groupByKeys)

		for _, groupByKey := range groupByKeys {
			if _, ok := d.Meter.GroupBy[groupByKey]; !ok {
				return "", nil, fmt.Errorf("meter does not have group by: %s", groupByKey)
			}

			groupByJSONPath := sqlbuilder.Escape(d.Meter.GroupBy[groupByKey])

			values := d.FilterGroupBy[groupByKey]
			if len(values) == 0 {
				return "", nil, fmt.Errorf("empty filter for group by: %s", groupByKey)
			}
			mapFunc := func(value string) string {
				column := fmt.Sprintf("JSON_VALUE(%s, '%s')", getColumn("data"), groupByJSONPath)

				// Subject is a special case
				if groupByKey == "subject" {
					column = "subject"
				}

				return fmt.Sprintf("%s = '%s'", column, sqlbuilder.Escape((value)))
			}

			where = append(where, query.Or(slicesx.Map(values, mapFunc)...))
		}
	}

	if d.From != nil {
		where = append(where, query.GreaterEqualThan(getColumn("time"), d.From.Unix()))
	}

	if d.To != nil {
		where = append(where, query.LessEqualThan(getColumn("time"), d.To.Unix()))
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

type listMeterSubjectsQuery struct {
	Database        string
	EventsTableName string
	Namespace       string
	Meter           models.Meter
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
		sb.Where(sb.LessEqualThan("time", d.To.Unix()))
	}

	sql, args := sb.Build()
	return sql, args
}

func columnFactory(alias string) func(string) string {
	return func(column string) string {
		return fmt.Sprintf("%s.%s", alias, column)
	}
}
