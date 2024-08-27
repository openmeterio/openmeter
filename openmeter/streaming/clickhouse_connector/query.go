package clickhouse_connector

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/huandu/go-sqlbuilder"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type column struct {
	Name string
	Type string
}

// Create Events Table
type createEventsTable struct {
	Database string
}

func (d createEventsTable) toSQL() string {
	tableName := GetEventsTableName(d.Database)

	sb := sqlbuilder.ClickHouse.NewCreateTableBuilder()
	sb.CreateTable(tableName)
	sb.IfNotExists()
	sb.Define("namespace", "String")
	sb.Define("validation_error", "String")
	sb.Define("id", "String")
	sb.Define("type", "LowCardinality(String)")
	sb.Define("subject", "String")
	sb.Define("source", "String")
	sb.Define("time", "DateTime")
	sb.Define("data", "String")
	sb.Define("ingested_at", "DateTime")
	sb.Define("created_at", "DateTime")
	sb.SQL("ENGINE = MergeTree")
	sb.SQL("PARTITION BY toYYYYMM(time)")
	sb.SQL("ORDER BY (namespace, time, type, subject)")

	sql, _ := sb.Build()
	return sql
}

type queryEventsTable struct {
	Database  string
	Namespace string
	From      *time.Time
	To        *time.Time
	Limit     int
}

func (d queryEventsTable) toSQL() (string, []interface{}) {
	tableName := GetEventsTableName(d.Database)
	where := []string{}

	query := sqlbuilder.ClickHouse.NewSelectBuilder()
	query.Select("id", "type", "subject", "source", "time", "data", "validation_error", "ingested_at", "created_at")
	query.From(tableName)

	where = append(where, query.Equal("namespace", d.Namespace))
	if d.From != nil {
		where = append(where, query.GreaterEqualThan("time", d.From.Unix()))
	}
	if d.To != nil {
		where = append(where, query.LessEqualThan("time", d.To.Unix()))
	}
	query.Where(where...)

	query.Desc().OrderBy("time")
	query.Limit(d.Limit)

	sql, args := query.Build()
	return sql, args
}

type queryCountEvents struct {
	Database  string
	Namespace string
	From      time.Time
}

func (d queryCountEvents) toSQL() (string, []interface{}) {
	tableName := GetEventsTableName(d.Database)

	query := sqlbuilder.ClickHouse.NewSelectBuilder()
	query.Select("count() as count", "subject", "notEmpty(validation_error) as is_error")
	query.From(tableName)

	query.Where(query.Equal("namespace", d.Namespace))
	query.Where(query.GreaterEqualThan("time", d.From.Unix()))
	query.GroupBy("subject", "is_error")

	sql, args := query.Build()
	return sql, args
}

type createMeterView struct {
	Database      string
	Aggregation   models.MeterAggregation
	Namespace     string
	MeterSlug     string
	EventType     string
	ValueProperty string
	GroupBy       map[string]string
	// Populate creates the materialized view with data from the events table
	// This is not safe to use in production as requires to stop ingestion
	Populate bool
}

func (d createMeterView) toSQL() (string, []interface{}, error) {
	viewName := GetMeterViewName(d.Database, d.Namespace, d.MeterSlug)
	columns := []column{
		{Name: "subject", Type: "String"},
		{Name: "windowstart", Type: "DateTime"},
		{Name: "windowend", Type: "DateTime"},
	}

	// Value
	agg := ""

	switch d.Aggregation {
	case models.MeterAggregationSum:
		agg = "sum"
	case models.MeterAggregationAvg:
		agg = "avg"
	case models.MeterAggregationMin:
		agg = "min"
	case models.MeterAggregationMax:
		agg = "max"
	case models.MeterAggregationCount:
		agg = "count"
	case models.MeterAggregationUniqueCount:
		agg = "uniq"
	default:
		return "", nil, fmt.Errorf("invalid aggregation type: %s", d.Aggregation)
	}

	if d.Aggregation == models.MeterAggregationUniqueCount {
		columns = append(columns, column{Name: "value", Type: fmt.Sprintf("AggregateFunction(%s, String)", agg)})
	} else {
		columns = append(columns, column{Name: "value", Type: fmt.Sprintf("AggregateFunction(%s, Float64)", agg)})
	}

	// Group by
	orderBy := []string{"windowstart", "windowend", "subject"}
	sortedGroupBy := sortedKeys(d.GroupBy)
	for _, k := range sortedGroupBy {
		columnName := sqlbuilder.Escape(k)
		orderBy = append(orderBy, sqlbuilder.Escape(columnName))
		columns = append(columns, column{Name: columnName, Type: "String"})
	}

	sb := sqlbuilder.ClickHouse.NewCreateTableBuilder()
	sb.CreateTable(viewName)
	sb.IfNotExists()
	for _, column := range columns {
		sb.Define(column.Name, column.Type)
	}
	sb.SQL("ENGINE = AggregatingMergeTree()")
	sb.SQL(fmt.Sprintf("ORDER BY (%s)", strings.Join(orderBy, ", ")))
	if d.Populate {
		sb.SQL("POPULATE")
	}
	sb.SQL("AS")

	selectQuery, err := d.toSelectSQL()
	if err != nil {
		return "", nil, err
	}

	sb.SQL(selectQuery)
	sql, args := sb.Build()

	// TODO: can we do it differently?
	sql = strings.Replace(sql, "CREATE TABLE", "CREATE MATERIALIZED VIEW", 1)

	return sql, args, nil
}

func (d createMeterView) toSelectSQL() (string, error) {
	eventsTableName := GetEventsTableName(d.Database)

	aggStateFn := ""
	switch d.Aggregation {
	case models.MeterAggregationSum:
		aggStateFn = "sumState"
	case models.MeterAggregationAvg:
		aggStateFn = "avgState"
	case models.MeterAggregationMin:
		aggStateFn = "minState"
	case models.MeterAggregationMax:
		aggStateFn = "maxState"
	case models.MeterAggregationUniqueCount:
		aggStateFn = "uniqState"
	case models.MeterAggregationCount:
		aggStateFn = "countState"
	default:
		return "", fmt.Errorf("invalid aggregation type: %s", d.Aggregation)
	}

	// Selects
	selects := []string{
		"subject",
		"tumbleStart(time, toIntervalMinute(1)) AS windowstart",
		"tumbleEnd(time, toIntervalMinute(1)) AS windowend",
	}
	if d.ValueProperty == "" && d.Aggregation == models.MeterAggregationCount {
		selects = append(selects, fmt.Sprintf("%s(*) AS value", aggStateFn))
	} else if d.Aggregation == models.MeterAggregationUniqueCount {
		selects = append(selects, fmt.Sprintf("%s(JSON_VALUE(data, '%s')) AS value", aggStateFn, sqlbuilder.Escape(d.ValueProperty)))
	} else {
		selects = append(selects, fmt.Sprintf("%s(cast(JSON_VALUE(data, '%s'), 'Float64')) AS value", aggStateFn, sqlbuilder.Escape(d.ValueProperty)))
	}

	// Group by
	orderBy := []string{"windowstart", "windowend", "subject"}
	sortedGroupBy := sortedKeys(d.GroupBy)
	for _, k := range sortedGroupBy {
		v := d.GroupBy[k]
		columnName := sqlbuilder.Escape(k)
		orderBy = append(orderBy, sqlbuilder.Escape(columnName))
		selects = append(selects, fmt.Sprintf("JSON_VALUE(data, '%s') as %s", sqlbuilder.Escape(v), sqlbuilder.Escape(k)))
	}

	query := sqlbuilder.ClickHouse.NewSelectBuilder()
	query.Select(selects...)
	query.From(eventsTableName)
	// We use absolute paths to avoid shadowing in the case the materialized view have a `namespace` or `type` group by
	query.Where(fmt.Sprintf("%s.namespace = '%s'", eventsTableName, sqlbuilder.Escape(d.Namespace)))
	query.Where(fmt.Sprintf("empty(%s.validation_error) = 1", eventsTableName))
	query.Where(fmt.Sprintf("%s.type = '%s'", eventsTableName, sqlbuilder.Escape(d.EventType)))
	query.GroupBy(orderBy...)

	return query.String(), nil
}

type deleteMeterView struct {
	Database  string
	Namespace string
	MeterSlug string
}

func (d deleteMeterView) toSQL() string {
	viewName := GetMeterViewName(d.Database, d.Namespace, d.MeterSlug)

	return fmt.Sprintf("DROP VIEW %s", viewName)
}

type queryMeterView struct {
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

func (d queryMeterView) toSQL() (string, []interface{}, error) {
	viewName := GetMeterViewName(d.Database, d.Namespace, d.MeterSlug)

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
				fmt.Sprintf("tumbleStart(windowstart, toIntervalMinute(1), '%s') AS windowstart", tz),
				fmt.Sprintf("tumbleEnd(windowstart, toIntervalMinute(1), '%s') AS windowend", tz),
			)

		case models.WindowSizeHour:
			selectColumns = append(
				selectColumns,
				fmt.Sprintf("tumbleStart(windowstart, toIntervalHour(1), '%s') AS windowstart", tz),
				fmt.Sprintf("tumbleEnd(windowstart, toIntervalHour(1), '%s') AS windowend", tz),
			)

		case models.WindowSizeDay:
			selectColumns = append(
				selectColumns,
				fmt.Sprintf("tumbleStart(windowstart, toIntervalDay(1), '%s') AS windowstart", tz),
				fmt.Sprintf("tumbleEnd(windowstart, toIntervalDay(1), '%s') AS windowend", tz),
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
		selectColumns = append(selectColumns, "sumMerge(value) AS value")
	case models.MeterAggregationAvg:
		selectColumns = append(selectColumns, "avgMerge(value) AS value")
	case models.MeterAggregationMin:
		selectColumns = append(selectColumns, "minMerge(value) AS value")
	case models.MeterAggregationMax:
		selectColumns = append(selectColumns, "maxMerge(value) AS value")
	case models.MeterAggregationUniqueCount:
		selectColumns = append(selectColumns, "toFloat64(uniqMerge(value)) AS value")
	case models.MeterAggregationCount:
		selectColumns = append(selectColumns, "toFloat64(countMerge(value)) AS value")
	default:
		return "", nil, fmt.Errorf("invalid aggregation type: %s", d.Aggregation)
	}

	for _, column := range d.GroupBy {
		c := sqlbuilder.Escape(column)
		selectColumns = append(selectColumns, c)
		groupByColumns = append(groupByColumns, c)
	}

	queryView := sqlbuilder.ClickHouse.NewSelectBuilder()
	queryView.Select(selectColumns...)
	queryView.From(viewName)

	if len(d.Subject) > 0 {
		mapFunc := func(subject string) string {
			return queryView.Equal("subject", subject)
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
				return queryView.Equal(sqlbuilder.Escape(column), value)
			}

			where = append(where, queryView.Or(slicesx.Map(values, mapFunc)...))
		}
	}

	if d.From != nil {
		where = append(where, queryView.GreaterEqualThan("windowstart", d.From.Unix()))
	}

	if d.To != nil {
		where = append(where, queryView.LessEqualThan("windowend", d.To.Unix()))
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

type listMeterViewSubjects struct {
	Database  string
	Namespace string
	MeterSlug string
	From      *time.Time
	To        *time.Time
}

func (d listMeterViewSubjects) toSQL() (string, []interface{}) {
	viewName := GetMeterViewName(d.Database, d.Namespace, d.MeterSlug)

	var where []string
	sb := sqlbuilder.ClickHouse.NewSelectBuilder()
	sb.Select("DISTINCT subject")
	sb.From(viewName)

	if d.From != nil {
		where = append(where, sb.GreaterEqualThan("windowstart", d.From.Unix()))
	}

	if d.To != nil {
		where = append(where, sb.LessEqualThan("windowend", d.To.Unix()))
	}

	if len(where) > 0 {
		sb.Where(where...)
	}

	sb.OrderBy("subject")

	sql, args := sb.Build()
	return sql, args
}

func GetEventsTableName(database string) string {
	return fmt.Sprintf("%s.%s%s", sqlbuilder.Escape(database), tablePrefix, EventsTableName)
}

func GetMeterViewName(database string, namespace string, meterSlug string) string {
	meterViewName := fmt.Sprintf("%s%s_%s", tablePrefix, namespace, meterSlug)
	return fmt.Sprintf("%s.%s", sqlbuilder.Escape(database), sqlbuilder.Escape(meterViewName))
}
