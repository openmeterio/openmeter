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
	Database    string
	TablePrefix string
}

func (d createEventsTable) toSQL() string {
	tableName := GetEventsTableName(d.Database)

	sb := sqlbuilder.ClickHouse.NewCreateTableBuilder()
	sb.CreateTable(tableName)
	sb.IfNotExists()
	sb.Define("namespace", "String")
	sb.Define("id", "String")
	sb.Define("type", "LowCardinality(String)")
	sb.Define("subject", "String")
	sb.Define("source", "String")
	sb.Define("time", "DateTime")
	sb.Define("data", "String")
	sb.SQL("ENGINE = MergeTree")
	sb.SQL("PARTITION BY toYYYYMM(time)")
	sb.SQL("ORDER BY (namespace, time, type, subject)")

	sql, _ := sb.Build()
	return sql
}

// Create Invalid Events Table
type createInvalidEventsTable struct {
	Database string
}

func (d createInvalidEventsTable) toSQL() string {
	tableName := GetInvalidEventsTableName(d.Database)

	sb := sqlbuilder.ClickHouse.NewCreateTableBuilder()
	sb.CreateTable(tableName)
	sb.IfNotExists()
	sb.Define("namespace", "String")
	sb.Define("time", "DateTime")
	sb.Define("error", "String")
	sb.Define("event", "String")
	sb.SQL("ENGINE = MergeTree")
	sb.SQL("PARTITION BY toYYYYMM(time)")
	sb.SQL("ORDER BY (namespace, time)")

	sql, _ := sb.Build()
	return sql
}

type queryEventsTable struct {
	Database  string
	Namespace string
	Limit     int
}

func (d queryEventsTable) toSQL() (string, []interface{}, error) {
	tableName := GetEventsTableName(d.Database)

	query := sqlbuilder.ClickHouse.NewSelectBuilder()
	query.Select("id", "type", "subject", "source", "time", "data")
	query.From(tableName)
	query.Where(query.Equal("namespace", d.Namespace))
	query.Desc().OrderBy("time")
	query.Limit(d.Limit)

	sql, args := query.Build()
	return sql, args, nil
}

type createMeterView struct {
	Database      string
	Aggregation   models.MeterAggregation
	Namespace     string
	MeterSlug     string
	EventType     string
	ValueProperty string
	GroupBy       map[string]string
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
	default:
		return "", nil, fmt.Errorf("invalid aggregation type: %s", d.Aggregation)
	}

	columns = append(columns, column{Name: "value", Type: fmt.Sprintf("AggregateFunction(%s, Float64)", agg)})

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

	asSelects := []string{
		"subject",
		"tumbleStart(time, toIntervalMinute(1)) AS windowstart",
		"tumbleEnd(time, toIntervalMinute(1)) AS windowend",
	}

	// Value
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
	case models.MeterAggregationCount:
		aggStateFn = "countState"
	default:
		return "", fmt.Errorf("invalid aggregation type: %s", d.Aggregation)
	}

	if d.ValueProperty == "" && d.Aggregation == models.MeterAggregationCount {
		asSelects = append(asSelects, fmt.Sprintf("%s(*) AS value", aggStateFn))
	} else {
		asSelects = append(asSelects, fmt.Sprintf("%s(cast(JSON_VALUE(data, '%s'), 'Float64')) AS value", aggStateFn, sqlbuilder.Escape(d.ValueProperty)))
	}

	// Group by
	orderBy := []string{"windowstart", "windowend", "subject"}
	sortedGroupBy := sortedKeys(d.GroupBy)
	for _, k := range sortedGroupBy {
		v := d.GroupBy[k]
		columnName := sqlbuilder.Escape(k)
		orderBy = append(orderBy, sqlbuilder.Escape(columnName))
		asSelects = append(asSelects, fmt.Sprintf("JSON_VALUE(data, '%s') as %s", sqlbuilder.Escape(v), sqlbuilder.Escape(k)))
	}

	query := sqlbuilder.ClickHouse.NewSelectBuilder()
	query.Select(asSelects...)
	query.From(eventsTableName)
	// We use absolute path for type to avoid shadowing in the case the materialized view have a `type` column due to group by
	query.Where(fmt.Sprintf("namespace = '%s'", sqlbuilder.Escape(d.Namespace)))
	query.Where(fmt.Sprintf("%s.type = '%s'", eventsTableName, sqlbuilder.Escape(d.EventType)))
	query.GroupBy(orderBy...)

	return query.String(), nil
}

type deleteMeterView struct {
	Database  string
	Namespace string
	MeterSlug string
}

func (d deleteMeterView) toSQL() (string, []interface{}) {
	viewName := GetMeterViewName(d.Database, d.Namespace, d.MeterSlug)
	return fmt.Sprintf("DROP VIEW %s", viewName), nil
}

type queryMeterView struct {
	Database       string
	Namespace      string
	MeterSlug      string
	Aggregation    models.MeterAggregation
	Subject        []string
	From           *time.Time
	To             *time.Time
	GroupBy        []string
	GroupBySubject bool
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

	// Grouping by subject is required when filtering for a subject
	// It is also a default grouping requirement in certain queries (eg. meter values)
	groupBySubject := d.GroupBySubject || len(d.Subject) > 0

	if groupBySubject {
		selectColumns = append(selectColumns, "subject")
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
	case models.MeterAggregationCount:
		selectColumns = append(selectColumns, "toFloat64(countMerge(value)) AS value")
	default:
		return "", nil, fmt.Errorf("invalid aggregation type: %s", d.Aggregation)
	}

	if groupBySubject {
		groupByColumns = append(groupByColumns, "subject")
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

func (d listMeterViewSubjects) toSQL() (string, []interface{}, error) {
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
	return sql, args, nil
}

func GetEventsTableName(database string) string {
	return fmt.Sprintf("%s.%s_%s", sqlbuilder.Escape(database), TablePrefix, EventsTableName)
}

func GetInvalidEventsTableName(database string) string {
	return fmt.Sprintf("%s.%s_%s", sqlbuilder.Escape(database), TablePrefix, InvalidEventsTableName)
}

func GetMeterViewName(database string, namespace string, meterSlug string) string {
	meterViewName := fmt.Sprintf("%s_%s_%s", TablePrefix, namespace, meterSlug)
	return fmt.Sprintf("%s.%s", sqlbuilder.Escape(database), sqlbuilder.Escape(meterViewName))
}
