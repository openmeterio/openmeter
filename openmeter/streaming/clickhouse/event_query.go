package clickhouse

import (
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/huandu/go-sqlbuilder"

	"github.com/openmeterio/openmeter/openmeter/streaming"
)

// EventsTableEngineType is the storage engine used when creating the events table.
type EventsTableEngineType string

const (
	EventsTableEngineMergeTree           EventsTableEngineType = "MergeTree"
	EventsTableEngineReplicatedMergeTree EventsTableEngineType = "ReplicatedMergeTree"
)

// EventsTableEngine describes how the events table CREATE statement should be
// rendered. The zero value renders the legacy `ENGINE = MergeTree` form, which
// is rewritten transparently to SharedMergeTree by ClickHouse Cloud.
type EventsTableEngine struct {
	Type          EventsTableEngineType
	ZooKeeperPath string
	ReplicaName   string
	Cluster       string
}

func (e EventsTableEngine) resolvedType() EventsTableEngineType {
	if e.Type == "" {
		return EventsTableEngineMergeTree
	}
	return e.Type
}

// Validate ensures the engine configuration is internally consistent.
func (e EventsTableEngine) Validate() error {
	// We backtick-quote the cluster name in the emitted SQL, so any value
	// ClickHouse accepts as an identifier is fine here. We only reject
	// whitespace-only values (likely a config typo — explicit nothing should
	// be expressed by leaving the field empty).
	if e.Cluster != "" && strings.TrimSpace(e.Cluster) == "" {
		return fmt.Errorf("cluster name must not be whitespace-only")
	}

	switch e.resolvedType() {
	case EventsTableEngineMergeTree:
		return nil
	case EventsTableEngineReplicatedMergeTree:
		if strings.TrimSpace(e.ZooKeeperPath) == "" {
			return fmt.Errorf("zooKeeperPath is required for %s", EventsTableEngineReplicatedMergeTree)
		}
		if strings.TrimSpace(e.ReplicaName) == "" {
			return fmt.Errorf("replicaName is required for %s", EventsTableEngineReplicatedMergeTree)
		}
		return nil
	default:
		return fmt.Errorf("unsupported events table engine type %q", e.Type)
	}
}

// engineClause renders the ENGINE = ... fragment.
func (e EventsTableEngine) engineClause() string {
	switch e.resolvedType() {
	case EventsTableEngineReplicatedMergeTree:
		return fmt.Sprintf(
			"ENGINE = ReplicatedMergeTree('%s', '%s')",
			escapeSingleQuotes(e.ZooKeeperPath),
			escapeSingleQuotes(e.ReplicaName),
		)
	default:
		return "ENGINE = MergeTree"
	}
}

// escapeSingleQuotes doubles single quotes so the value can be embedded in a
// single-quoted SQL string literal without breaking out.
func escapeSingleQuotes(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// quoteClusterIdentifier wraps a cluster name in backticks, doubling any
// embedded backticks so the value is parsed by ClickHouse as a single
// identifier. Backtick-quoting accepts any cluster name ClickHouse permits in
// <remote_servers> (including hyphens, dots, etc.), without re-implementing
// ClickHouse's identifier-validation rules in our code.
func quoteClusterIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

// Create Events Table
type createEventsTable struct {
	Database        string
	EventsTableName string
	Engine          EventsTableEngine
}

func (d createEventsTable) toSQL() string {
	// The sqlbuilder treats the table name as a raw token, so we splice
	// "ON CLUSTER {cluster}" into it to position the clause between the
	// table name and the column definitions, which is the only place
	// ClickHouse accepts it.
	tableToken := getTableName(d.Database, d.EventsTableName)
	if d.Engine.Cluster != "" {
		tableToken = fmt.Sprintf("%s ON CLUSTER %s", tableToken, quoteClusterIdentifier(d.Engine.Cluster))
	}

	sb := sqlbuilder.ClickHouse.NewCreateTableBuilder()
	sb.CreateTable(tableToken)
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
	sb.Define(fmt.Sprintf("INDEX %s_stored_at stored_at TYPE minmax GRANULARITY 4", d.EventsTableName))
	sb.Define("store_row_id", "String")
	sb.SQL(d.Engine.engineClause())
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
	Customers       *[]streaming.Customer
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
		query.Where(query.LessThan("time", d.To.Unix()))
	}

	// If we have a customer filter, we add it to the query
	var customers []streaming.Customer

	if d.Customers != nil {
		customers = *d.Customers
	}

	// If we have a subject filter, we add it to the query
	var subjects []string

	if d.Subject != nil {
		subjects = append(subjects, *d.Subject)
	}

	query = subjectWhere(d.EventsTableName, subjects, query)
	query = customersWhere(d.EventsTableName, customers, query)

	sql, args := query.Build()
	return sql, args
}

func (d queryEventsTable) toSQL() (string, []interface{}) {
	tableName := getTableName(d.Database, d.EventsTableName)
	query := sqlbuilder.ClickHouse.NewSelectBuilder()

	// Select columns
	query.Select(
		"id",
		"type",
		"subject",
		"source",
		"time",
		"data",
		"ingested_at",
		"stored_at",
		"store_row_id",
	)

	// Select customer_id column if customer filter is provided
	if d.Customers != nil {
		query = selectCustomerIdColumn(d.EventsTableName, *d.Customers, query)
	}

	query.From(tableName)

	// Add where clauses
	query.Where(query.Equal("namespace", d.Namespace))
	query.Where(query.GreaterEqualThan("time", d.From.Unix()))

	if d.To != nil {
		query.Where(query.LessThan("time", d.To.Unix()))
	}
	if d.IngestedAtFrom != nil {
		query.Where(query.GreaterEqualThan("ingested_at", d.IngestedAtFrom.Unix()))
	}
	if d.IngestedAtTo != nil {
		query.Where(query.LessThan("ingested_at", d.IngestedAtTo.Unix()))
	}
	if d.ID != nil {
		query.Where(query.Like("id", fmt.Sprintf("%%%s%%", *d.ID)))
	}

	// If we have a customer filter, we add it to the query
	var customers []streaming.Customer

	if d.Customers != nil {
		customers = *d.Customers
	}

	// If we have a subject filter, we add it to the query
	var subjects []string

	if d.Subject != nil {
		subjects = append(subjects, *d.Subject)
	}

	query = subjectWhere(d.EventsTableName, subjects, query)
	query = customersWhere(d.EventsTableName, customers, query)

	// Order by time and limit the number of rows returned
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
	query.Cols("namespace", "id", "type", "source", "subject", "time", "data", "ingested_at", "stored_at", "store_row_id")

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
			event.StoreRowID,
		)
	}

	sql, args := query.Build()
	return sql, args
}

func getTableName(database string, tableName string) string {
	return fmt.Sprintf("%s.%s", database, tableName)
}
