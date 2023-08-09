package clickhouse_connector

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/huandu/go-sqlbuilder"

	"github.com/openmeterio/openmeter/pkg/models"
)

// Create Events Table
type createEventsTable struct {
	Database        string
	EventsTableName string
}

type column struct {
	Name string
	Type string
}

func (d createEventsTable) toSQL() string {
	tableName := fmt.Sprintf("%s.%s", sqlbuilder.Escape(d.Database), sqlbuilder.Escape(d.EventsTableName))

	sb := sqlbuilder.ClickHouse.NewCreateTableBuilder()
	sb.CreateTable(tableName)
	sb.IfNotExists()
	sb.Define("id", "String")
	sb.Define("type", "LowCardinality(String)")
	sb.Define("subject", "String")
	sb.Define("source", "String")
	sb.Define("time", "DateTime")
	sb.Define("data", "String")
	sb.SQL("ENGINE = MergeTree")
	sb.SQL("PARTITION BY toYYYYMM(time)")
	sb.SQL("ORDER BY (time, type, subject)")

	sql, _ := sb.Build()
	return sql
}

type createMeterView struct {
	Database        string
	Aggregation     models.MeterAggregation
	EventsTableName string
	MeterViewName   string
	EventType       string
	ValueProperty   string
	GroupBy         map[string]string
}

func (d createMeterView) toSQL() (string, []interface{}, error) {
	viewName := fmt.Sprintf("%s.%s", sqlbuilder.Escape(d.Database), sqlbuilder.Escape(d.MeterViewName))
	eventsTableName := fmt.Sprintf("%s.%s", sqlbuilder.Escape(d.Database), sqlbuilder.Escape(d.EventsTableName))
	columns := []column{
		{Name: "subject", Type: "String"},
		{Name: "windowstart", Type: "DateTime"},
		{Name: "windowend", Type: "DateTime"},
	}
	asSelects := []string{
		"subject",
		"tumbleStart(time, toIntervalMinute(1)) AS windowstart",
		"tumbleEnd(time, toIntervalMinute(1)) AS windowend",
	}

	// Value
	agg := ""
	aggStateFn := ""

	switch d.Aggregation {
	case models.MeterAggregationSum:
		agg = "sum"
		aggStateFn = "sumState"
	case models.MeterAggregationAvg:
		agg = "avg"
		aggStateFn = "avgState"
	case models.MeterAggregationMin:
		agg = "min"
		aggStateFn = "minState"
	case models.MeterAggregationMax:
		agg = "max"
		aggStateFn = "maxState"
	case models.MeterAggregationCount:
		agg = "count"
		aggStateFn = "countState"
	default:
		return "", nil, fmt.Errorf("invalid aggregation type: %s", d.Aggregation)
	}

	columns = append(columns, column{Name: "value", Type: fmt.Sprintf("AggregateFunction(%s, Float64)", agg)})
	asSelects = append(asSelects, fmt.Sprintf("%s(cast(JSON_VALUE(data, '%s'), 'Float64')) AS value", aggStateFn, sqlbuilder.Escape(d.ValueProperty)))

	// Group by
	orderBy := []string{"windowstart", "windowend", "subject"}
	sortedGroupBy := sortedKeys(d.GroupBy)
	for _, k := range sortedGroupBy {
		v := d.GroupBy[k]
		columnName := sqlbuilder.Escape(k)
		orderBy = append(orderBy, sqlbuilder.Escape(columnName))
		columns = append(columns, column{Name: columnName, Type: "String"})
		asSelects = append(asSelects, fmt.Sprintf("JSON_VALUE(data, '%s') as %s", sqlbuilder.Escape(v), sqlbuilder.Escape(k)))
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

	sbAs := sqlbuilder.ClickHouse.NewSelectBuilder()
	sbAs.Select(asSelects...)
	sbAs.From(eventsTableName)
	sbAs.Where(fmt.Sprintf("type = '%s'", sqlbuilder.Escape(d.EventType)))
	sbAs.GroupBy(orderBy...)
	sb.SQL(sbAs.String())
	sql, args := sb.Build()

	// TODO: can we do it differently?
	sql = strings.Replace(sql, "CREATE TABLE", "CREATE MATERIALIZED VIEW", 1)

	return sql, args, nil
}

type deleteMeterView struct {
	Database      string
	MeterViewName string
}

func (d deleteMeterView) toSQL() (string, []interface{}) {
	viewName := fmt.Sprintf("%s.%s", sqlbuilder.Escape(d.Database), sqlbuilder.Escape(d.MeterViewName))
	return fmt.Sprintf("DROP VIEW %s", viewName), nil
}

type describeMeterView struct {
	Database      string
	MeterViewName string
}

func (d describeMeterView) toSQL() (string, []interface{}) {
	viewName := fmt.Sprintf("%s.%s", sqlbuilder.Escape(d.Database), sqlbuilder.Escape(d.MeterViewName))
	return fmt.Sprintf("DESCRIBE %s", viewName), nil
}

type queryMeterView struct {
	Database      string
	MeterViewName string
	Aggregation   models.MeterAggregation
	Subject       *string
	From          *time.Time
	To            *time.Time
	GroupBy       []string
	WindowSize    *models.WindowSize
}

func (d queryMeterView) toSQL() (string, []interface{}, error) {
	viewName := fmt.Sprintf("%s.%s", sqlbuilder.Escape(d.Database), sqlbuilder.Escape(d.MeterViewName))
	selectColumns := []string{"windowstart", "windowend", "subject"}
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
		selectColumns = append(selectColumns, "countMerge(value) AS value")
	default:
		return "", nil, fmt.Errorf("invalid aggregation type: %s", d.Aggregation)
	}

	groupByColumns := []string{"windowstart", "windowend", "subject"}
	where := []string{}

	for _, column := range d.GroupBy {
		c := sqlbuilder.Escape(column)
		selectColumns = append(selectColumns, c)
		groupByColumns = append(groupByColumns, c)
	}

	queryView := sqlbuilder.ClickHouse.NewSelectBuilder()
	queryView.Select(selectColumns...)
	queryView.From(viewName)

	if d.Subject != nil {
		where = append(where, queryView.Equal("subject", *d.Subject))
	}
	if d.From != nil {
		from := fmt.Sprintf("toDateTime(%d)", d.From.UnixMilli())
		where = append(where, queryView.GreaterEqualThan("windowstart", from))
	}
	if d.To != nil {
		to := fmt.Sprintf("toDateTime(%d)", d.To.UnixMilli())
		where = append(where, queryView.LessEqualThan("windowend", to))
	}
	if len(where) > 0 {
		queryView.Where(where...)
	}

	queryView.GroupBy(groupByColumns...)
	queryView.OrderBy("windowstart")

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
