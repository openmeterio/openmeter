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

type column struct {
	Name string
	Type string
}

type queryMeter struct {
	Database       string
	Namespace      string
	MeterSlug      string
	Aggregation    models.MeterAggregation
	Subject        []string
	FilterGroupBy  map[string][]string
	From           *time.Time
	To             *time.Time
	GroupBy        []string
	WindowSize     *models.WindowSize
	WindowTimeZone *time.Location
}

func (d queryMeter) toSQL() (string, []interface{}, error) {
	viewAlias := "meter"
	tableName := fmt.Sprintf("%s %s", GetMeterEventsTableName(d.Database), viewAlias)
	getColumn := columnFactory(viewAlias)

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
				fmt.Sprintf("tumbleStart(time, toIntervalMinute(1), '%s') AS windowstart", tz),
				fmt.Sprintf("tumbleEnd(time, toIntervalMinute(1), '%s') AS windowend", tz),
			)

		case models.WindowSizeHour:
			selectColumns = append(
				selectColumns,
				fmt.Sprintf("tumbleStart(time, toIntervalHour(1), '%s') AS windowstart", tz),
				fmt.Sprintf("tumbleEnd(time, toIntervalHour(1), '%s') AS windowend", tz),
			)

		case models.WindowSizeDay:
			selectColumns = append(
				selectColumns,
				fmt.Sprintf("tumbleStart(time, toIntervalDay(1), '%s') AS windowstart", tz),
				fmt.Sprintf("tumbleEnd(time, toIntervalDay(1), '%s') AS windowend", tz),
			)

		default:
			return "", nil, fmt.Errorf("invalid window size type: %s", *d.WindowSize)
		}

		groupByColumns = append(groupByColumns, "windowstart", "windowend")
	} else {
		selectColumns = append(selectColumns, "min(windowstart)", "max(windowend)")
	}

	switch d.Aggregation {
	case models.MeterAggregationSum:
		selectColumns = append(selectColumns, "sum(value) AS value")
	case models.MeterAggregationAvg:
		selectColumns = append(selectColumns, "avg(value) AS value")
	case models.MeterAggregationMin:
		selectColumns = append(selectColumns, "min(value) AS value")
	case models.MeterAggregationMax:
		selectColumns = append(selectColumns, "max(value) AS value")
	case models.MeterAggregationUniqueCount:
		// FIXME: value is a number, not a string
		selectColumns = append(selectColumns, "toFloat64(uniq(value)) AS value")
	case models.MeterAggregationCount:
		selectColumns = append(selectColumns, "sum(value) AS value")
	default:
		return "", nil, fmt.Errorf("invalid aggregation type: %s", d.Aggregation)
	}

	for _, column := range d.GroupBy {
		c := sqlbuilder.Escape(column)
		selectColumn := fmt.Sprintf("group_by['%s'] as %s", c, c)
		selectColumns = append(selectColumns, selectColumn)
		groupByColumns = append(groupByColumns, c)
	}

	queryView := sqlbuilder.ClickHouse.NewSelectBuilder()
	queryView.Select(selectColumns...)
	queryView.From(tableName)
	queryView.Where(queryView.Equal("namespace", d.Namespace))
	queryView.Where(queryView.Equal("meter", d.MeterSlug))

	if len(d.Subject) > 0 {
		mapFunc := func(subject string) string {
			return queryView.Equal(getColumn("subject"), subject)
		}

		where = append(where, queryView.Or(slicesx.Map(d.Subject, mapFunc)...))
	}

	if len(d.FilterGroupBy) > 0 {
		// We sort the columns to ensure the query is deterministic
		columns := make([]string, 0, len(d.FilterGroupBy))
		for k := range d.FilterGroupBy {
			columns = append(columns, k)
		}
		sort.Strings(columns)

		for _, column := range columns {
			values := d.FilterGroupBy[column]
			if len(values) == 0 {
				return "", nil, fmt.Errorf("empty filter for group by: %s", column)
			}
			mapFunc := func(value string) string {
				return queryView.Equal(sqlbuilder.Escape(getColumn(column)), value)
			}

			where = append(where, queryView.Or(slicesx.Map(values, mapFunc)...))
		}
	}

	if d.From != nil {
		where = append(where, queryView.GreaterEqualThan(getColumn("time"), d.From.Unix()))
	}

	if d.To != nil {
		where = append(where, queryView.LessEqualThan(getColumn("time"), d.To.Unix()))
	}

	if len(where) > 0 {
		queryView.Where(where...)
	}

	queryView.GroupBy(groupByColumns...)

	if groupByWindowSize {
		queryView.OrderBy("windowstart")
	}

	sql, args := queryView.Build()
	return sql, args, nil
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
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
