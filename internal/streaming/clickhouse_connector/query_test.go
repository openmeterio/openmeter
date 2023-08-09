package clickhouse_connector

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/pkg/models"
)

func TestCreateEventsTable(t *testing.T) {
	tests := []struct {
		data createEventsTable
		want string
	}{
		{
			data: createEventsTable{
				Database:        "openmeter",
				EventsTableName: "meter_events",
			},
			want: "CREATE TABLE IF NOT EXISTS openmeter.meter_events (id String, type LowCardinality(String), subject String, source String, time DateTime, data String) ENGINE = MergeTree PARTITION BY toYYYYMM(time) ORDER BY (time, type, subject)",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			got := tt.data.toSQL()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCreateMeterView(t *testing.T) {
	tests := []struct {
		query    createMeterView
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			query: createMeterView{
				Database:        "openmeter",
				EventsTableName: "meter_events",
				Aggregation:     models.MeterAggregationSum,
				EventType:       "myevent",
				MeterViewName:   "meter_meter1",
				ValueProperty:   "$.duration_ms",
				GroupBy:         map[string]string{"group1": "$.group1", "group2": "$.group2"},
			},
			wantSQL:  "CREATE MATERIALIZED VIEW IF NOT EXISTS openmeter.meter_meter1 (subject String, windowstart DateTime, windowend DateTime, value AggregateFunction(sum, Float64), group1 String, group2 String) ENGINE = AggregatingMergeTree() ORDER BY (windowstart, windowend, subject, group1, group2) AS SELECT subject, tumbleStart(time, toIntervalMinute(1)) AS windowstart, tumbleEnd(time, toIntervalMinute(1)) AS windowend, sumState(cast(JSON_VALUE(data, '$.duration_ms'), 'Float64')) AS value, JSON_VALUE(data, '$.group1') as group1, JSON_VALUE(data, '$.group2') as group2 FROM openmeter.meter_events WHERE type = 'myevent' GROUP BY windowstart, windowend, subject, group1, group2",
			wantArgs: nil,
		},
		{
			query: createMeterView{
				Database:        "openmeter",
				EventsTableName: "meter_events",
				Aggregation:     models.MeterAggregationAvg,
				EventType:       "myevent",
				MeterViewName:   "meter_meter1",
				ValueProperty:   "$.token_count",
				GroupBy:         map[string]string{},
			},
			wantSQL:  "CREATE MATERIALIZED VIEW IF NOT EXISTS openmeter.meter_meter1 (subject String, windowstart DateTime, windowend DateTime, value AggregateFunction(avg, Float64)) ENGINE = AggregatingMergeTree() ORDER BY (windowstart, windowend, subject) AS SELECT subject, tumbleStart(time, toIntervalMinute(1)) AS windowstart, tumbleEnd(time, toIntervalMinute(1)) AS windowend, avgState(cast(JSON_VALUE(data, '$.token_count'), 'Float64')) AS value FROM openmeter.meter_events WHERE type = 'myevent' GROUP BY windowstart, windowend, subject",
			wantArgs: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			gotSql, gotArgs, err := tt.query.toSQL()
			if err != nil {
				t.Error(err)
				return
			}

			assert.Equal(t, tt.wantSQL, gotSql)
			assert.Equal(t, tt.wantArgs, gotArgs)
		})
	}
}

func TestDeleteMeterView(t *testing.T) {
	tests := []struct {
		data     deleteMeterView
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			data: deleteMeterView{
				Database:      "openmeter",
				MeterViewName: "meter_meter1",
			},
			wantSQL:  "DROP VIEW openmeter.meter_meter1",
			wantArgs: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			gotSql, gotArgs := tt.data.toSQL()

			assert.Equal(t, tt.wantSQL, gotSql)
			assert.Equal(t, tt.wantArgs, gotArgs)
		})
	}
}

func TestDescribeMeterView(t *testing.T) {
	tests := []struct {
		data     describeMeterView
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			data: describeMeterView{
				Database:      "openmeter",
				MeterViewName: "meter_meter1",
			},
			wantSQL:  "DESCRIBE openmeter.meter_meter1",
			wantArgs: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			gotSql, gotArgs := tt.data.toSQL()

			assert.Equal(t, tt.wantSQL, gotSql)
			assert.Equal(t, tt.wantArgs, gotArgs)
		})
	}
}

func TestQueryMeterView(t *testing.T) {
	subject := "subject1"
	from, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00.001Z")
	to, _ := time.Parse(time.RFC3339, "2023-01-02T00:00:00Z")
	windowSize := models.WindowSizeHour

	tests := []struct {
		query    queryMeterView
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			query: queryMeterView{
				Database:      "openmeter",
				MeterViewName: "meter_meter1",
				Aggregation:   models.MeterAggregationSum,
				Subject:       &subject,
				From:          &from,
				To:            &to,
				GroupBy:       []string{"group1", "group2"},
				WindowSize:    &windowSize,
			},
			wantSQL:  "SELECT windowstart, windowend, subject, sumMerge(value) AS value, group1, group2 FROM openmeter.meter_meter1 WHERE subject = ? AND windowstart >= ? AND windowend <= ? GROUP BY windowstart, windowend, subject, group1, group2 ORDER BY windowstart",
			wantArgs: []interface{}{"subject1", "toDateTime(1672531200001)", "toDateTime(1672617600000)"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			gotSql, gotArgs, err := tt.query.toSQL()
			if err != nil {
				t.Error(err)
				return
			}

			assert.Equal(t, tt.wantSQL, gotSql)
			assert.Equal(t, tt.wantArgs, gotArgs)
		})
	}
}
