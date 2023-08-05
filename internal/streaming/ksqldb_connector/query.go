package ksqldb_connector

import (
	_ "embed"

	. "github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

// https://github.com/cloudevents/spec/blob/main/cloudevents/formats/cloudevents.json
//
//go:embed sql/events_stream.tpl.sql
var cloudEventsStreamQueryTemplate string

type cloudEventsStreamQueryData struct {
	Format        string
	Namespace     string
	Topic         string
	KeySchemaId   int
	ValueSchemaId int
}

//go:embed sql/detected_events_table.tpl.sql
var detectedEventsTableQueryTemplate string

type detectedEventsTableQueryData struct {
	Format     string
	Namespace  string
	Topic      string
	Retention  int
	Partitions int
}

//go:embed sql/detected_events_stream.tpl.sql
var detectedEventsStreamQueryTemplate string

type detectedEventsStreamQueryData struct {
	Format    string
	Namespace string
	Topic     string
}

//go:embed sql/meter_table_describe.tpl.sql
var meterTableDescribeQueryTemplate string

//go:embed sql/values.tpl.sql
var meterValuesTemplate string

type meterValuesData struct {
	Namespace string
	Slug      string
	GroupBy   []string
	*QueryParams
}

type meterTableDescribeQueryData struct {
	Namespace string
	Slug      string
}

//go:embed sql/meter_table.tpl.sql
var meterTableQueryTemplate string

type meterTableQueryData struct {
	Format    string
	Namespace string
	*models.Meter
	WindowRetention string
	Partitions      int
}

//go:embed sql/delete_meter_table.tpl.sql
var deleteMeterTableQueryTemplate string

type deleteMeterTableQueryData struct {
	Namespace string
	Slug      string
}
