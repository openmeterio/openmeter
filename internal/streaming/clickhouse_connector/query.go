package clickhouse_connector

import (
	_ "embed"
	"fmt"
	"time"

	"github.com/huandu/go-sqlbuilder"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Create Events Table
type createEventsTable struct {
	Database        string
	EventsTableName string
}

func (d createEventsTable) toSQL() string {
	tableName := fmt.Sprintf("%s.%s", d.Database, d.EventsTableName)

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

//go:embed sql/create_meter_view.tpl.sql
var createMeterViewTemplate string

type createMeterViewData struct {
	Database        string
	EventsTableName string
	MeterViewName   string
	EventType       string
	ValueProperty   string
	GroupBy         map[string]string
}

//go:embed sql/delete_meter_view.tpl.sql
var deleteMeterViewTemplate string

type deleteMeterViewData struct {
	Database      string
	MeterViewName string
}

//go:embed sql/describe_meter_view.tpl.sql
var describeMeterViewTemplate string

type describeMeterViewData struct {
	Database      string
	MeterViewName string
}

//go:embed sql/query_meter_view.tpl.sql
var queryMeterViewTemplate string

type queryMeterViewData struct {
	Database      string
	MeterViewName string
	Subject       *string
	From          *time.Time
	To            *time.Time
	GroupBy       []string
	WindowSize    *models.WindowSize
}
