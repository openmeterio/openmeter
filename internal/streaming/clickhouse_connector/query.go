package clickhouse_connector

import (
	_ "embed"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

//go:embed sql/create_events_table.tpl.sql
var createEventsTableTemplate string

type createEventsTableData struct {
	Database        string
	EventsTableName string
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
