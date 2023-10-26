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

func TestQueryEventsTable(t *testing.T) {
	tests := []struct {
		query    queryEventsTable
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			query: queryEventsTable{
				Database:        "openmeter",
				EventsTableName: "meter_events",
				Limit:           100,
			},
			wantSQL:  "SELECT id, type, subject, source, time, data FROM openmeter.meter_events ORDER BY time DESC LIMIT 100",
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

			assert.Equal(t, tt.wantArgs, gotArgs)
			assert.Equal(t, tt.wantSQL, gotSql)
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
			wantSQL:  "CREATE MATERIALIZED VIEW IF NOT EXISTS openmeter.meter_meter1 (subject String, windowstart DateTime, windowend DateTime, value AggregateFunction(sum, Float64), group1 String, group2 String) ENGINE = AggregatingMergeTree() ORDER BY (windowstart, windowend, subject, group1, group2) AS SELECT subject, tumbleStart(time, toIntervalMinute(1)) AS windowstart, tumbleEnd(time, toIntervalMinute(1)) AS windowend, sumState(cast(JSON_VALUE(data, '$.duration_ms'), 'Float64')) AS value, JSON_VALUE(data, '$.group1') as group1, JSON_VALUE(data, '$.group2') as group2 FROM openmeter.meter_events WHERE openmeter.meter_events.type = 'myevent' GROUP BY windowstart, windowend, subject, group1, group2",
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
			wantSQL:  "CREATE MATERIALIZED VIEW IF NOT EXISTS openmeter.meter_meter1 (subject String, windowstart DateTime, windowend DateTime, value AggregateFunction(avg, Float64)) ENGINE = AggregatingMergeTree() ORDER BY (windowstart, windowend, subject) AS SELECT subject, tumbleStart(time, toIntervalMinute(1)) AS windowstart, tumbleEnd(time, toIntervalMinute(1)) AS windowend, avgState(cast(JSON_VALUE(data, '$.token_count'), 'Float64')) AS value FROM openmeter.meter_events WHERE openmeter.meter_events.type = 'myevent' GROUP BY windowstart, windowend, subject",
			wantArgs: nil,
		},
		{
			query: createMeterView{
				Database:        "openmeter",
				EventsTableName: "meter_events",
				Aggregation:     models.MeterAggregationCount,
				EventType:       "myevent",
				MeterViewName:   "meter_meter1",
				ValueProperty:   "",
				GroupBy:         map[string]string{},
			},
			wantSQL:  "CREATE MATERIALIZED VIEW IF NOT EXISTS openmeter.meter_meter1 (subject String, windowstart DateTime, windowend DateTime, value AggregateFunction(count, Float64)) ENGINE = AggregatingMergeTree() ORDER BY (windowstart, windowend, subject) AS SELECT subject, tumbleStart(time, toIntervalMinute(1)) AS windowstart, tumbleEnd(time, toIntervalMinute(1)) AS windowend, countState(*) AS value FROM openmeter.meter_events WHERE openmeter.meter_events.type = 'myevent' GROUP BY windowstart, windowend, subject",
			wantArgs: nil,
		},
		{
			query: createMeterView{
				Database:        "openmeter",
				EventsTableName: "meter_events",
				Aggregation:     models.MeterAggregationCount,
				EventType:       "myevent",
				MeterViewName:   "meter_meter1",
				ValueProperty:   "",
				GroupBy:         map[string]string{},
			},
			wantSQL:  "CREATE MATERIALIZED VIEW IF NOT EXISTS openmeter.meter_meter1 (subject String, windowstart DateTime, windowend DateTime, value AggregateFunction(count, Float64)) ENGINE = AggregatingMergeTree() ORDER BY (windowstart, windowend, subject) AS SELECT subject, tumbleStart(time, toIntervalMinute(1)) AS windowstart, tumbleEnd(time, toIntervalMinute(1)) AS windowend, countState(*) AS value FROM openmeter.meter_events WHERE openmeter.meter_events.type = 'myevent' GROUP BY windowstart, windowend, subject",
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
				Subject:       []string{subject},
				From:          &from,
				To:            &to,
				GroupBy:       []string{"group1", "group2"},
				WindowSize:    &windowSize,
			},
			wantSQL:  "SELECT tumbleStart(windowstart, toIntervalHour(1)) AS windowstart, tumbleEnd(windowstart, toIntervalHour(1)) AS windowend, subject, sumMerge(value) AS value, group1, group2 FROM openmeter.meter_meter1 WHERE (subject = ?) AND windowstart >= ? AND windowend <= ? GROUP BY windowstart, windowend, subject, group1, group2 ORDER BY windowstart",
			wantArgs: []interface{}{"subject1", int64(1672531200), int64(1672617600)},
		},
		{ // Aggregate all available data
			query: queryMeterView{
				Database:      "openmeter",
				MeterViewName: "meter_meter1",
				Aggregation:   models.MeterAggregationSum,
			},
			wantSQL:  "SELECT min(windowstart), max(windowend), sumMerge(value) AS value FROM openmeter.meter_meter1",
			wantArgs: nil,
		},
		{ // Aggregate with count aggregation
			query: queryMeterView{
				Database:      "openmeter",
				MeterViewName: "meter_meter1",
				Aggregation:   models.MeterAggregationCount,
			},
			wantSQL:  "SELECT min(windowstart), max(windowend), toFloat64(countMerge(value)) AS value FROM openmeter.meter_meter1",
			wantArgs: nil,
		},
		{ // Aggregate data from start
			query: queryMeterView{
				Database:      "openmeter",
				MeterViewName: "meter_meter1",
				Aggregation:   models.MeterAggregationSum,
				From:          &from,
			},
			wantSQL:  "SELECT min(windowstart), max(windowend), sumMerge(value) AS value FROM openmeter.meter_meter1 WHERE windowstart >= ?",
			wantArgs: []interface{}{int64(1672531200)},
		},
		{ // Aggregate data between interval
			query: queryMeterView{
				Database:      "openmeter",
				MeterViewName: "meter_meter1",
				Aggregation:   models.MeterAggregationSum,
				From:          &from,
				To:            &to,
			},
			wantSQL:  "SELECT min(windowstart), max(windowend), sumMerge(value) AS value FROM openmeter.meter_meter1 WHERE windowstart >= ? AND windowend <= ?",
			wantArgs: []interface{}{int64(1672531200), int64(1672617600)},
		},
		{ // Aggregate data between interval, groupped by window size
			query: queryMeterView{
				Database:      "openmeter",
				MeterViewName: "meter_meter1",
				Aggregation:   models.MeterAggregationSum,
				From:          &from,
				To:            &to,
				WindowSize:    &windowSize,
			},
			wantSQL:  "SELECT tumbleStart(windowstart, toIntervalHour(1)) AS windowstart, tumbleEnd(windowstart, toIntervalHour(1)) AS windowend, sumMerge(value) AS value FROM openmeter.meter_meter1 WHERE windowstart >= ? AND windowend <= ? GROUP BY windowstart, windowend ORDER BY windowstart",
			wantArgs: []interface{}{int64(1672531200), int64(1672617600)},
		},
		{ // Aggregate data for a single subject
			query: queryMeterView{
				Database:      "openmeter",
				MeterViewName: "meter_meter1",
				Aggregation:   models.MeterAggregationSum,
				Subject:       []string{subject},
			},
			wantSQL:  "SELECT min(windowstart), max(windowend), subject, sumMerge(value) AS value FROM openmeter.meter_meter1 WHERE (subject = ?) GROUP BY subject",
			wantArgs: []interface{}{"subject1"},
		},
		{ // Aggregate data for a single subject and group by additional fields
			query: queryMeterView{
				Database:      "openmeter",
				MeterViewName: "meter_meter1",
				Aggregation:   models.MeterAggregationSum,
				Subject:       []string{subject},
				GroupBy:       []string{"group1", "group2"},
			},
			wantSQL:  "SELECT min(windowstart), max(windowend), subject, sumMerge(value) AS value, group1, group2 FROM openmeter.meter_meter1 WHERE (subject = ?) GROUP BY subject, group1, group2",
			wantArgs: []interface{}{"subject1"},
		},
		{ // Aggregate data for a multiple subjects
			query: queryMeterView{
				Database:      "openmeter",
				MeterViewName: "meter_meter1",
				Aggregation:   models.MeterAggregationSum,
				Subject:       []string{subject, "subject2"},
			},
			wantSQL:  "SELECT min(windowstart), max(windowend), subject, sumMerge(value) AS value FROM openmeter.meter_meter1 WHERE (subject = ? OR subject = ?) GROUP BY subject",
			wantArgs: []interface{}{"subject1", "subject2"},
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
