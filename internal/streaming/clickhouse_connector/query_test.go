package clickhouse_connector

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
			got, err := templateQuery(createEventsTableTemplate, tt.data)
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
			got, err := templateQuery(createMeterViewTemplate, tt.data)
			if err != nil {
				t.Error(err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
