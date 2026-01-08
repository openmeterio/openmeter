package summeterv1

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/huandu/go-sqlbuilder"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

const TableName = "numeric_meter_v1"

func (e Engine) CreateTableSQL() string {
	tableName := fmt.Sprintf("%s.%s", e.database, TableName)

	sb := sqlbuilder.ClickHouse.NewCreateTableBuilder()
	sb.CreateTable(tableName)
	sb.IfNotExists()
	sb.Define("namespace", "String")
	sb.Define("meter_id", "LowCardinality(String)")
	sb.Define("subject", "String")
	sb.Define("time", "DateTime")
	sb.Define("value", "Decimal128(20)")
	sb.Define("group_by_filters", "Map(LowCardinality(String),String)")
	sb.Define("stored_at", "DateTime")
	sb.Define("store_row_id", "String")
	sb.SQL("ENGINE = MergeTree")
	sb.SQL("PARTITION BY toYYYYMM(time)")
	sb.SQL("ORDER BY (namespace, meter_id, subject,toStartOfHour(time))")

	sql, _ := sb.Build()
	return sql
}

type Record struct {
	Namespace      string
	MeterID        string
	Subject        string
	Time           time.Time
	Value          alpacadecimal.Decimal
	GroupByFilters map[string]string
	StoredAt       time.Time
	StoreRowID     string
}

// Insert Events Query
type InsertRecordsQuery struct {
	Database      string
	Records       []Record
	QuerySettings map[string]string
}

func (q InsertRecordsQuery) ToSQL() (string, []interface{}) {
	tableName := fmt.Sprintf("%s.%s", q.Database, TableName)

	query := sqlbuilder.ClickHouse.NewInsertBuilder()
	query.InsertInto(tableName)
	query.Cols("namespace", "meter_id", "subject", "time", "value", "group_by_filters", "stored_at", "store_row_id")

	// Add settings
	var settings []string
	for key, value := range q.QuerySettings {
		settings = append(settings, fmt.Sprintf("%s = %s", key, value))
	}

	if len(settings) > 0 {
		query.SQL(fmt.Sprintf("SETTINGS %s", strings.Join(settings, ", ")))
	}

	for _, event := range q.Records {
		query.Values(
			event.Namespace,
			event.MeterID,
			event.Subject,
			event.Time,
			event.Value,
			event.GroupByFilters,
			event.StoredAt,
			event.StoreRowID,
		)
	}

	sql, args := query.Build()
	return sql, args
}

func (e Engine) InsertRecords(ctx context.Context, records []Record) error {
	query := InsertRecordsQuery{
		Database:      e.database,
		Records:       records,
		QuerySettings: map[string]string{},
	}

	sql, args := query.ToSQL()
	// TODO: WithAsync?
	return e.clickhouse.AsyncInsert(ctx, sql, true, args...)
}

// QueryMinEventsStoredAt builds a query to get the minimum stored_at from the events table
// filtered by namespace and type.
type QueryMinEventsStoredAt struct {
	Database        string
	EventsTableName string
	Namespace       string
	EventType       string
}

func (q QueryMinEventsStoredAt) ToSQL() (string, []interface{}) {
	tableName := fmt.Sprintf("%s.%s", q.Database, q.EventsTableName)

	sb := sqlbuilder.ClickHouse.NewSelectBuilder()
	sb.Select("min(stored_at)")
	sb.From(tableName)
	sb.Where(sb.Equal("namespace", q.Namespace))
	sb.Where(sb.Equal("type", q.EventType))

	sql, args := sb.Build()
	return sql, args
}

// MinEventsStoredAt executes the min(stored_at) query against ClickHouse.
// It returns nil if there are no rows for the given filters.
func (e Engine) MinEventsStoredAt(ctx context.Context, eventsTableName string, namespace string, eventType string) (*time.Time, error) {
	q := QueryMinEventsStoredAt{
		Database:        e.database,
		EventsTableName: eventsTableName,
		Namespace:       namespace,
		EventType:       eventType,
	}
	sql, args := q.ToSQL()

	row := e.clickhouse.QueryRow(ctx, sql, args...)
	var ts *time.Time
	if err := row.Scan(&ts); err != nil {
		return nil, err
	}
	return ts, nil
}

// InsertFromEventsQuery builds an INSERT ... SELECT to backfill records from the events table.
type InsertFromEventsQuery struct {
	Database        string
	EventsTableName string
	Meter           meter.Meter
	Period          timeutil.ClosedPeriod
}

// ToSQL builds an INSERT ... SELECT statement that reads from the events table
// and writes into the numeric meter table.
//
// Example SELECT part (omitting the INSERT INTO ... prefix):
//
//	SELECT
//	  namespace,
//	  ? AS meter_id,
//	  subject,
//	  time,
//	  CAST(ifNotFinite(toFloat64OrNull(JSON_VALUE(data, '$.amount')), null) AS Decimal128(20)) AS value,
//	  map('g1', ifNull(JSON_VALUE(data, '$.a'), ''), 'g2', ifNull(JSON_VALUE(data, '$.b'), '')) AS group_by_filters,
//	  stored_at,
//	  toString(generateUUIDv4()) AS store_row_id
//	FROM <db>.om_events
//	WHERE namespace = ? AND type = ? AND stored_at >= ? AND stored_at < ?
//
// Notes:
// - The meter_id placeholder is bound as the first argument.
// - The value JSON path and group-by JSON paths are parameterized to avoid SQL injection and to match ClickHouse JSON_VALUE usage.
func (q InsertFromEventsQuery) ToSQL() (string, []interface{}, error) {
	if q.Meter.ValueProperty == nil || *q.Meter.ValueProperty == "" {
		return "", nil, fmt.Errorf("value property is required for numeric meter")
	}
	tableName := fmt.Sprintf("%s.%s", q.Database, TableName)
	eventsTable := fmt.Sprintf("%s.%s", q.Database, q.EventsTableName)

	// We need to build args in the exact order of appearance of placeholders:
	// 1) meter_id
	// 2) value path (in valueExpr)
	// 3) group-by paths (in groupExpr)
	// 4) where clause args (namespace, type, from, to)
	valueArgs := make([]interface{}, 0, 1)
	groupArgs := make([]interface{}, 0, len(q.Meter.GroupBy))

	// Build group_by_filters map expression with deterministic key order
	groupKeys := lo.Keys(q.Meter.GroupBy)
	sort.Strings(groupKeys)

	groupParts := make([]string, 0, len(groupKeys)*2)
	for _, k := range groupKeys {
		path := q.Meter.GroupBy[k]
		groupParts = append(groupParts, fmt.Sprintf("'%s'", k))
		groupParts = append(groupParts, "ifNull(JSON_VALUE(data, ?), '')")
		groupArgs = append(groupArgs, path)
	}
	groupExpr := "map(" + strings.Join(groupParts, ", ") + ")"

	// Value expression: parse decimal directly from JSON string to avoid float rounding
	valueExpr := "CAST(toDecimal128OrNull(JSON_VALUE(data, ?), 20) AS Decimal128(20))"
	valueArgs = append(valueArgs, *q.Meter.ValueProperty)

	// Insert columns
	cols := "namespace, meter_id, subject, time, value, group_by_filters, stored_at, store_row_id"

	// Build SELECT using sqlbuilder
	sel := sqlbuilder.ClickHouse.NewSelectBuilder()
	sel.Select(
		"namespace",
		"? AS meter_id",
		"subject",
		"time",
		valueExpr+" AS value",
		groupExpr+" AS group_by_filters",
		"stored_at",
		"toString(generateUUIDv4()) AS store_row_id",
	)
	sel.From(eventsTable)
	sel.Where(sel.Equal("namespace", q.Meter.Namespace))
	sel.Where(sel.Equal("type", q.Meter.EventType))
	sel.Where(sel.GreaterEqualThan("stored_at", q.Period.From))
	sel.Where(sel.LessThan("stored_at", q.Period.To))
	// Filter out rows where value cannot be parsed as decimal
	sel.Where(fmt.Sprintf("toDecimal128OrNull(JSON_VALUE(data, %s), 20) IS NOT NULL", sel.Var(*q.Meter.ValueProperty)))
	selectSQL, whereArgs := sel.Build()

	// Add meter_id and filters to args
	args := make([]interface{}, 0, 1+len(valueArgs)+len(groupArgs)+len(whereArgs))
	args = append(args, q.Meter.ID)
	args = append(args, valueArgs...)
	args = append(args, groupArgs...)
	args = append(args, whereArgs...)

	finalSQL := fmt.Sprintf("INSERT INTO %s (%s) %s", tableName, cols, selectSQL)
	return finalSQL, args, nil
}

func (e Engine) InsertFromEvents(ctx context.Context, eventsTableName string, m meter.Meter, period timeutil.ClosedPeriod) error {
	q := InsertFromEventsQuery{
		Database:        e.database,
		EventsTableName: eventsTableName,
		Meter:           m,
		Period:          period,
	}
	sql, args, err := q.ToSQL()
	if err != nil {
		return err
	}

	return e.clickhouse.Exec(ctx, sql, args...)
}
