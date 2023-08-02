package clickhouse_connector

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestCreateEventsTable(t *testing.T) {
	tests := []struct {
		data createEventsTableData
		want string
	}{
		{
			data: createEventsTableData{
				Database:        "openmeter",
				EventsTableName: "meter_events",
			},
			want: "CREATE TABLE IF NOT EXISTS openmeter.meter_events ( id String, type LowCardinality(String), subject String, source String, time DateTime, data String ) ENGINE = MergeTree PARTITION BY toYYYYMM(time) ORDER BY (time, type, subject);",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			got, err := streaming.TemplateQuery(createEventsTableTemplate, tt.data)
			if err != nil {
				t.Error(err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCreateMeterView(t *testing.T) {
	tests := []struct {
		data createMeterViewData
		want string
	}{
		{
			data: createMeterViewData{
				Database:        "openmeter",
				EventsTableName: "meter_events",
				MeterViewName:   "meter_meter1",
				ValueProperty:   "$.duration_ms",
				GroupBy:         map[string]string{"group1": "$.group1", "group2": "$.group2"},
			},
			want: "CREATE MATERIALIZED VIEW IF NOT EXISTS openmeter.meter_meter1 ( subject String, windowstart DateTime, windowend DateTime, value AggregateFunction(sum, Float64), group1 String, group2 String ) ENGINE = AggregatingMergeTree() ORDER BY (windowstart, windowend, subject, group1, group2) AS SELECT subject, tumbleStart(time, toIntervalMinute(1)) AS windowstart, tumbleEnd(time, toIntervalMinute(1)) AS windowend, sumState(cast(JSON_VALUE(data, '$.duration_ms'), 'Float64')) AS value, JSON_VALUE(data, '$.group1') as group1, JSON_VALUE(data, '$.group2') as group2 FROM openmeter.meter_events WHERE type = '' GROUP BY windowstart, windowend, subject, group1, group2;",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			got, err := streaming.TemplateQuery(createMeterViewTemplate, tt.data)
			if err != nil {
				t.Error(err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDeleteMeterView(t *testing.T) {
	tests := []struct {
		data deleteMeterViewData
		want string
	}{
		{
			data: deleteMeterViewData{
				Database:      "openmeter",
				MeterViewName: "meter_meter1",
			},
			want: "DROP VIEW openmeter.meter_meter1;",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			got, err := streaming.TemplateQuery(deleteMeterViewTemplate, tt.data)
			if err != nil {
				t.Error(err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestQueryMeterView(t *testing.T) {
	subject := "subject1"
	from, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00.001Z")
	to, _ := time.Parse(time.RFC3339, "2023-01-02T00:00:00Z")
	windowSize := models.WindowSizeHour

	tests := []struct {
		data queryMeterViewData
		want string
	}{
		{
			data: queryMeterViewData{
				Database:      "openmeter",
				MeterViewName: "meter_meter1",
				Subject:       &subject,
				From:          &from,
				To:            &to,
				GroupBy:       []string{"group1", "group2"},
				WindowSize:    &windowSize,
			},
			want: "SELECT windowstart, windowend, subject, sumMerge(value) AS value, group1, group2 FROM openmeter.meter_meter1 WHERE subject = 'subject1' AND windowstart >= toDateTime(1672531200001) AND windowend <= toDateTime(1672617600000)GROUP BY windowstart, windowend, subject, group1, group2 ORDER BY windowstart;",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			got, err := streaming.TemplateQuery(queryMeterViewTemplate, tt.data)
			if err != nil {
				t.Error(err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
