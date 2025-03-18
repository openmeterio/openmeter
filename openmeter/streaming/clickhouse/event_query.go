package clickhouse

import (
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/huandu/go-sqlbuilder"

	"github.com/openmeterio/openmeter/openmeter/streaming"
)

// Create Events Table
type createEventsTable struct {
	Database        string
	EventsTableName string
}

func (d createEventsTable) toSQL() string {
	tableName := getTableName(d.Database, d.EventsTableName)

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
	sb.Define("ingested_at", "DateTime")
	sb.Define("stored_at", "DateTime")
	sb.SQL("ENGINE = MergeTree")
	sb.SQL("PARTITION BY toYYYYMM(time)")
	// Lowest cardinality columns we always filter on goes to the most left.
	// ClickHouse always picks partition first so we always filter time by month.
	// Theoretically we could add toStartOfHour(time) to the order sooner than subject
	// but we bet on that a typical namespace has more subjects than hours in a month.
	// Subject is an optional filter so it won't always help to reduce number of rows scanned.
	// Finally we add time not just to speed up queries but also to keep data on the disk together.
	sb.SQL("ORDER BY (namespace, type, subject, toStartOfHour(time))")

	sql, _ := sb.Build()
	return sql
}

// Query Events Table
type queryEventsTable struct {
	Database        string
	EventsTableName string
	Namespace       string
	From            time.Time
	To              *time.Time
	IngestedAtFrom  *time.Time
	IngestedAtTo    *time.Time
	ID              *string
	Subject         *string
	Limit           int
}

// toCountRowSQL returns the SQL query for the estimated number of rows.
// This estimate is useful for query progress tracking.
// We only filter by columns that are in the ClickHouse table order.
func (d queryEventsTable) toCountRowSQL() (string, []interface{}) {
	tableName := getTableName(d.Database, d.EventsTableName)

	query := sqlbuilder.ClickHouse.NewSelectBuilder()
	query.Select("count() as total")
	query.From(tableName)

	query.Where(query.Equal("namespace", d.Namespace))
	query.Where(query.GreaterEqualThan("time", d.From.Unix()))

	if d.To != nil {
		query.Where(query.LessEqualThan("time", d.To.Unix()))
	}
	if d.Subject != nil {
		query.Where(query.Equal("subject", *d.Subject))
	}

	sql, args := query.Build()
	return sql, args
}

func (d queryEventsTable) toSQL() (string, []interface{}) {
	tableName := getTableName(d.Database, d.EventsTableName)

	query := sqlbuilder.ClickHouse.NewSelectBuilder()
	query.Select("id", "type", "subject", "source", "time", "data", "ingested_at", "stored_at")
	query.From(tableName)

	query.Where(query.Equal("namespace", d.Namespace))
	query.Where(query.GreaterEqualThan("time", d.From.Unix()))

	if d.To != nil {
		query.Where(query.LessEqualThan("time", d.To.Unix()))
	}
	if d.IngestedAtFrom != nil {
		query.Where(query.GreaterEqualThan("ingested_at", d.IngestedAtFrom.Unix()))
	}
	if d.IngestedAtTo != nil {
		query.Where(query.LessEqualThan("ingested_at", d.IngestedAtTo.Unix()))
	}
	if d.ID != nil {
		query.Where(query.Like("id", fmt.Sprintf("%%%s%%", *d.ID)))
	}
	if d.Subject != nil {
		query.Where(query.Equal("subject", *d.Subject))
	}

	query.Desc().OrderBy("time")
	query.Limit(d.Limit)

	sql, args := query.Build()
	return sql, args
}

type queryCountEvents struct {
	Database        string
	EventsTableName string
	Namespace       string
	From            time.Time
}

func (d queryCountEvents) toSQL() (string, []interface{}) {
	tableName := getTableName(d.Database, d.EventsTableName)

	query := sqlbuilder.ClickHouse.NewSelectBuilder()
	query.Select("count() as count", "subject")
	query.From(tableName)

	query.Where(query.Equal("namespace", d.Namespace))
	query.Where(query.GreaterEqualThan("time", d.From.Unix()))
	query.GroupBy("subject")

	sql, args := query.Build()
	return sql, args
}

// Insert Events Query
type InsertEventsQuery struct {
	Database        string
	EventsTableName string
	Events          []streaming.RawEvent
	QuerySettings   map[string]string
}

func (q InsertEventsQuery) ToSQL() (string, []interface{}) {
	tableName := getTableName(q.Database, q.EventsTableName)

	query := sqlbuilder.ClickHouse.NewInsertBuilder()
	query.InsertInto(tableName)
	query.Cols("namespace", "id", "type", "source", "subject", "time", "data", "ingested_at", "stored_at")

	// Add settings
	var settings []string
	for key, value := range q.QuerySettings {
		settings = append(settings, fmt.Sprintf("%s = %s", key, value))
	}

	if len(settings) > 0 {
		query.SQL(fmt.Sprintf("SETTINGS %s", strings.Join(settings, ", ")))
	}

	for _, event := range q.Events {
		query.Values(
			event.Namespace,
			event.ID,
			event.Type,
			event.Source,
			event.Subject,
			event.Time,
			event.Data,
			event.IngestedAt,
			event.StoredAt,
		)
	}

	sql, args := query.Build()
	return sql, args
}

func getTableName(database string, tableName string) string {
	return fmt.Sprintf("%s.%s", database, tableName)
}
