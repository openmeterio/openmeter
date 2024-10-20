package clickhouse_connector

import (
	_ "embed"
	"fmt"
	"sort"
	"time"

	"github.com/huandu/go-sqlbuilder"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type queryMeter struct {
	Database       string
	Namespace      string
	Meter          models.Meter
	Subject        []string
	FilterGroupBy  map[string][]string
	From           *time.Time
	To             *time.Time
	GroupBy        []string
	WindowSize     *models.WindowSize
	WindowTimeZone *time.Location
}

func (d queryMeter) toSQL() (string, []interface{}, error) {
	tableName := GetMeterEventsTableName(d.Database)
	getColumn := columnFactory(MeterEventTableName)
	timeColumn := getColumn("time")

	var selectColumns, groupByColumns, where []string

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
		selectColumns = append(selectColumns, "min(windowstart)", "max(windowend)")
	}

	switch d.Meter.Aggregation {
	case models.MeterAggregationSum:
		selectColumns = append(selectColumns, fmt.Sprintf("sum(%s) AS value", getColumn("value")))
	case models.MeterAggregationAvg:
		selectColumns = append(selectColumns, fmt.Sprintf("avg(%s) AS value", getColumn("value")))
	case models.MeterAggregationMin:
		selectColumns = append(selectColumns, fmt.Sprintf("min(%s) AS value", getColumn("value")))
	case models.MeterAggregationMax:
		selectColumns = append(selectColumns, fmt.Sprintf("max(%s) AS value", getColumn("value")))
	case models.MeterAggregationUniqueCount:
		selectColumns = append(selectColumns, fmt.Sprintf("toDecimal(uniq(%s)) AS value", getColumn("value_str")))
	case models.MeterAggregationCount:
		selectColumns = append(selectColumns, fmt.Sprintf("sum(%s) AS value", getColumn("value")))
	default:
		return "", nil, fmt.Errorf("invalid aggregation type: %s", d.Meter.Aggregation)
	}

	for _, groupByKey := range d.GroupBy {
		c := sqlbuilder.Escape(groupByKey)
		selectColumn := fmt.Sprintf("%s['%s'] as %s", getColumn("group_by"), c, c)

		// Subject is a special case
		if groupByKey == "subject" {
			selectColumn = getColumn("subject")
		}

		selectColumns = append(selectColumns, selectColumn)
		groupByColumns = append(groupByColumns, c)
	}

	query := sqlbuilder.ClickHouse.NewSelectBuilder()
	query.Select(selectColumns...)
	query.From(tableName)
	query.Where(query.Equal(getColumn("namespace"), d.Namespace))
	query.Where(query.Equal(getColumn("meter"), d.Meter.GetID()))

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
			values := d.FilterGroupBy[groupByKey]
			if len(values) == 0 {
				return "", nil, fmt.Errorf("empty filter for group by: %s", groupByKey)
			}
			mapFunc := func(value string) string {
				column := sqlbuilder.Escape(fmt.Sprintf("%s['%s']", getColumn("group_by"), groupByKey))

				// Subject is a special case
				if groupByKey == "subject" {
					column = "subject"
				}

				return query.Equal(column, value)
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
	Database  string
	Namespace string
	MeterSlug string
	From      *time.Time
	To        *time.Time
}

func (d listMeterSubjectsQuery) toSQL() (string, []interface{}) {
	tableName := GetMeterEventsTableName(d.Database)

	sb := sqlbuilder.ClickHouse.NewSelectBuilder()
	sb.Select("DISTINCT subject")
	sb.Where(sb.Equal("namespace", d.Namespace))
	sb.Where(sb.Equal("meter", d.MeterSlug))
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
