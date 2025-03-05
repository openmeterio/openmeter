package materialized_view

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/huandu/go-sqlbuilder"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type column struct {
	Name string
	Type string
}

type createMeterView struct {
	Database        string
	EventsTableName string
	Aggregation     meter.MeterAggregation
	Namespace       string
	MeterSlug       string
	EventType       string
	ValueProperty   *string
	GroupBy         map[string]string
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
	case meter.MeterAggregationSum:
		agg = "sum"
	case meter.MeterAggregationAvg:
		agg = "avg"
	case meter.MeterAggregationMin:
		agg = "min"
	case meter.MeterAggregationMax:
		agg = "max"
	case meter.MeterAggregationCount:
		agg = "count"
	case meter.MeterAggregationUniqueCount:
		agg = "uniq"
	default:
		return "", nil, fmt.Errorf("invalid aggregation type: %s", d.Aggregation)
	}

	if d.Aggregation == meter.MeterAggregationUniqueCount {
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
	eventsTableName := getTableName(d.Database, d.EventsTableName)

	aggStateFn := ""
	switch d.Aggregation {
	case meter.MeterAggregationSum:
		aggStateFn = "sumState"
	case meter.MeterAggregationAvg:
		aggStateFn = "avgState"
	case meter.MeterAggregationMin:
		aggStateFn = "minState"
	case meter.MeterAggregationMax:
		aggStateFn = "maxState"
	case meter.MeterAggregationUniqueCount:
		aggStateFn = "uniqState"
	case meter.MeterAggregationCount:
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
	if d.Aggregation == meter.MeterAggregationCount {
		selects = append(selects, fmt.Sprintf("%s(*) AS value", aggStateFn))
	} else if d.Aggregation == meter.MeterAggregationUniqueCount {
		selects = append(selects, fmt.Sprintf("%s(JSON_VALUE(data, '%s')) AS value", aggStateFn, sqlbuilder.Escape(*d.ValueProperty)))
	} else {
		selects = append(selects, fmt.Sprintf("%s(cast(JSON_VALUE(data, '%s'), 'Float64')) AS value", aggStateFn, sqlbuilder.Escape(*d.ValueProperty)))
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
	Aggregation    meter.MeterAggregation
	Subject        []string
	FilterGroupBy  map[string][]string
	From           *time.Time
	To             *time.Time
	GroupBy        []string
	WindowSize     *meter.WindowSize
	WindowTimeZone *time.Location
}

func (d queryMeterView) toSQL() (string, []interface{}, error) {
	viewAlias := "meter"
	viewName := fmt.Sprintf("%s %s", GetMeterViewName(d.Database, d.Namespace, d.MeterSlug), viewAlias)
	getColumn := columnFactory(viewAlias)

	var selectColumns, groupByColumns, where []string

	groupByWindowSize := d.WindowSize != nil

	tz := "UTC"
	if d.WindowTimeZone != nil {
		tz = d.WindowTimeZone.String()
	}

	if groupByWindowSize {
		switch *d.WindowSize {
		case meter.WindowSizeMinute:
			selectColumns = append(
				selectColumns,
				fmt.Sprintf("tumbleStart(windowstart, toIntervalMinute(1), '%s') AS windowstart", tz),
				fmt.Sprintf("tumbleEnd(windowstart, toIntervalMinute(1), '%s') AS windowend", tz),
			)

		case meter.WindowSizeHour:
			selectColumns = append(
				selectColumns,
				fmt.Sprintf("tumbleStart(windowstart, toIntervalHour(1), '%s') AS windowstart", tz),
				fmt.Sprintf("tumbleEnd(windowstart, toIntervalHour(1), '%s') AS windowend", tz),
			)

		case meter.WindowSizeDay:
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
	case meter.MeterAggregationSum:
		selectColumns = append(selectColumns, "sumMerge(value) AS value")
	case meter.MeterAggregationAvg:
		selectColumns = append(selectColumns, "avgMerge(value) AS value")
	case meter.MeterAggregationMin:
		selectColumns = append(selectColumns, "minMerge(value) AS value")
	case meter.MeterAggregationMax:
		selectColumns = append(selectColumns, "maxMerge(value) AS value")
	case meter.MeterAggregationUniqueCount:
		selectColumns = append(selectColumns, "toFloat64(uniqMerge(value)) AS value")
	case meter.MeterAggregationCount:
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
		where = append(where, queryView.GreaterEqualThan(getColumn("windowstart"), d.From.Unix()))
	}

	if d.To != nil {
		where = append(where, queryView.LessEqualThan(getColumn("windowend"), d.To.Unix()))
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

func GetMeterViewName(database string, namespace string, meterSlug string) string {
	meterViewName := fmt.Sprintf("om_%s_%s", namespace, meterSlug)
	return fmt.Sprintf("%s.%s", sqlbuilder.Escape(database), sqlbuilder.Escape(meterViewName))
}

func columnFactory(alias string) func(string) string {
	return func(column string) string {
		return fmt.Sprintf("%s.%s", alias, column)
	}
}

func getTableName(database string, tableName string) string {
	return fmt.Sprintf("%s.%s", database, tableName)
}
