package clickhouse_connector_parse

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/huandu/go-sqlbuilder"

	"github.com/openmeterio/openmeter/openmeter/streaming"
)

const (
	MeterEventTableName = "om_meter_events"
)

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
	// For unique aggregation we need to store the value as a string
	sb.Define("value_str", "String")
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
	MeterEvents   []streaming.MeterEvent
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
		"value_str",
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
			meterEvent.ValueString,
			meterEvent.GroupBy,
			meterEvent.RawEvent.ID,
			meterEvent.RawEvent.Source,
			meterEvent.RawEvent.Type,
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
