package clickhouse_connector

import (
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/huandu/go-sqlbuilder"
)

const (
	MeterEventTableName = "om_meter_events"
)

// Meter Event represents a single meter event in ClickHouse
type CHMeterEvent struct {
	// Identifiers
	Namespace string    `ch:"namespace"`
	Time      time.Time `ch:"time"`
	Meter     string    `ch:"meter"`
	Subject   string    `ch:"subject"`

	// Usage
	Value   float64           `ch:"value"`
	GroupBy map[string]string `ch:"group_by"`

	// Metadata
	EventID     string    `ch:"event_id"`
	EventSource string    `ch:"event_source"`
	EventType   string    `ch:"event_type"`
	IngestedAt  time.Time `ch:"ingested_at"`
	StoredAt    time.Time `ch:"stored_at"`
}

// Create Meter Event Table
type createMeterEventTable struct {
	Database string
}

func (d createMeterEventTable) toSQL() string {
	tableName := GetMeterEventsTableName(d.Database)

	sb := sqlbuilder.ClickHouse.NewCreateTableBuilder()
	sb.CreateTable(tableName)
	sb.IfNotExists()

	// Identifiers
	sb.Define("namespace", "String")
	sb.Define("time", "DateTime")
	sb.Define("meter", "LowCardinality(String)")
	sb.Define("subject", "String")

	// Usage
	sb.Define("value", "Decimal(14, 4)")
	sb.Define("group_by", "Map(String, String)")

	// Metadata
	sb.Define("event_id", "String")
	sb.Define("event_type", "LowCardinality(String)")
	sb.Define("event_source", "String")
	sb.Define("ingested_at", "DateTime")
	sb.Define("stored_at", "DateTime")
	sb.SQL("ENGINE = MergeTree")
	sb.SQL("PARTITION BY toYYYYMM(time)")
	sb.SQL("ORDER BY (namespace, time, meter, subject)")

	sql, _ := sb.Build()
	return sql
}

// Insert Meter Events Query
type InsertMeterEventsQuery struct {
	Database      string
	MeterEvents   []CHMeterEvent
	QuerySettings map[string]string
}

func (q InsertMeterEventsQuery) ToSQL() (string, []interface{}) {
	tableName := GetMeterEventsTableName(q.Database)

	query := sqlbuilder.ClickHouse.NewInsertBuilder()
	query.InsertInto(tableName)
	query.Cols(
		"namespace",
		"time",
		"meter",
		"subject",
		"value",
		"group_by",
		"event_id",
		"event_source",
		"event_type",
		"ingested_at",
		"stored_at",
	)

	// Add settings
	var settings []string
	for key, value := range q.QuerySettings {
		settings = append(settings, fmt.Sprintf("%s = %s", key, value))
	}

	if len(settings) > 0 {
		query.SQL(fmt.Sprintf("SETTINGS %s", strings.Join(settings, ", ")))
	}

	for _, meterEvent := range q.MeterEvents {
		query.Values(
			meterEvent.Namespace,
			meterEvent.Time,
			meterEvent.Meter,
			meterEvent.Subject,
			meterEvent.Value,
			meterEvent.GroupBy,
			meterEvent.EventID,
			meterEvent.EventSource,
			meterEvent.EventType,
			meterEvent.IngestedAt,
			meterEvent.StoredAt,
		)
	}

	sql, args := query.Build()
	return sql, args
}

// Get Meter Events Table Name
func GetMeterEventsTableName(database string) string {
	return fmt.Sprintf("%s.%s", sqlbuilder.Escape(database), MeterEventTableName)
}
