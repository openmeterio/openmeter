package materialized_view

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/meter"
)

func TestCreateMeterView(t *testing.T) {
	tests := []struct {
		query    createMeterView
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			query: createMeterView{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				MeterSlug:       "meter1",
				Aggregation:     meter.MeterAggregationSum,
				EventType:       "myevent",
				ValueProperty:   lo.ToPtr("$.duration_ms"),
				GroupBy:         map[string]string{"group1": "$.group1", "group2": "$.group2"},
			},
			wantSQL:  "CREATE MATERIALIZED VIEW IF NOT EXISTS openmeter.om_my_namespace_meter1 (subject String, windowstart DateTime, windowend DateTime, value AggregateFunction(sum, Float64), group1 String, group2 String) ENGINE = AggregatingMergeTree() ORDER BY (windowstart, windowend, subject, group1, group2) AS SELECT subject, tumbleStart(time, toIntervalMinute(1)) AS windowstart, tumbleEnd(time, toIntervalMinute(1)) AS windowend, sumState(cast(JSON_VALUE(data, '$.duration_ms'), 'Float64')) AS value, JSON_VALUE(data, '$.group1') as group1, JSON_VALUE(data, '$.group2') as group2 FROM openmeter.om_events WHERE openmeter.om_events.namespace = 'my_namespace' AND empty(openmeter.om_events.validation_error) = 1 AND openmeter.om_events.type = 'myevent' GROUP BY windowstart, windowend, subject, group1, group2",
			wantArgs: nil,
		},
		{
			query: createMeterView{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				MeterSlug:       "meter1",
				Aggregation:     meter.MeterAggregationAvg,
				EventType:       "myevent",
				ValueProperty:   lo.ToPtr("$.token_count"),
				GroupBy:         map[string]string{},
			},
			wantSQL:  "CREATE MATERIALIZED VIEW IF NOT EXISTS openmeter.om_my_namespace_meter1 (subject String, windowstart DateTime, windowend DateTime, value AggregateFunction(avg, Float64)) ENGINE = AggregatingMergeTree() ORDER BY (windowstart, windowend, subject) AS SELECT subject, tumbleStart(time, toIntervalMinute(1)) AS windowstart, tumbleEnd(time, toIntervalMinute(1)) AS windowend, avgState(cast(JSON_VALUE(data, '$.token_count'), 'Float64')) AS value FROM openmeter.om_events WHERE openmeter.om_events.namespace = 'my_namespace' AND empty(openmeter.om_events.validation_error) = 1 AND openmeter.om_events.type = 'myevent' GROUP BY windowstart, windowend, subject",
			wantArgs: nil,
		},
		{
			query: createMeterView{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				MeterSlug:       "meter1",
				Aggregation:     meter.MeterAggregationCount,
				EventType:       "myevent",
				GroupBy:         map[string]string{},
			},
			wantSQL:  "CREATE MATERIALIZED VIEW IF NOT EXISTS openmeter.om_my_namespace_meter1 (subject String, windowstart DateTime, windowend DateTime, value AggregateFunction(count, Float64)) ENGINE = AggregatingMergeTree() ORDER BY (windowstart, windowend, subject) AS SELECT subject, tumbleStart(time, toIntervalMinute(1)) AS windowstart, tumbleEnd(time, toIntervalMinute(1)) AS windowend, countState(*) AS value FROM openmeter.om_events WHERE openmeter.om_events.namespace = 'my_namespace' AND empty(openmeter.om_events.validation_error) = 1 AND openmeter.om_events.type = 'myevent' GROUP BY windowstart, windowend, subject",
			wantArgs: nil,
		},
		{
			query: createMeterView{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				MeterSlug:       "meter1",
				Aggregation:     meter.MeterAggregationCount,
				EventType:       "myevent",
				GroupBy:         map[string]string{},
			},
			wantSQL:  "CREATE MATERIALIZED VIEW IF NOT EXISTS openmeter.om_my_namespace_meter1 (subject String, windowstart DateTime, windowend DateTime, value AggregateFunction(count, Float64)) ENGINE = AggregatingMergeTree() ORDER BY (windowstart, windowend, subject) AS SELECT subject, tumbleStart(time, toIntervalMinute(1)) AS windowstart, tumbleEnd(time, toIntervalMinute(1)) AS windowend, countState(*) AS value FROM openmeter.om_events WHERE openmeter.om_events.namespace = 'my_namespace' AND empty(openmeter.om_events.validation_error) = 1 AND openmeter.om_events.type = 'myevent' GROUP BY windowstart, windowend, subject",
			wantArgs: nil,
		},
		{
			query: createMeterView{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				MeterSlug:       "meter1",
				Aggregation:     meter.MeterAggregationUniqueCount,
				EventType:       "myevent",
				ValueProperty:   lo.ToPtr("$.trace_id"),
				GroupBy:         map[string]string{},
			},
			wantSQL:  "CREATE MATERIALIZED VIEW IF NOT EXISTS openmeter.om_my_namespace_meter1 (subject String, windowstart DateTime, windowend DateTime, value AggregateFunction(uniq, String)) ENGINE = AggregatingMergeTree() ORDER BY (windowstart, windowend, subject) AS SELECT subject, tumbleStart(time, toIntervalMinute(1)) AS windowstart, tumbleEnd(time, toIntervalMinute(1)) AS windowend, uniqState(JSON_VALUE(data, '$.trace_id')) AS value FROM openmeter.om_events WHERE openmeter.om_events.namespace = 'my_namespace' AND empty(openmeter.om_events.validation_error) = 1 AND openmeter.om_events.type = 'myevent' GROUP BY windowstart, windowend, subject",
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
				Database:  "openmeter",
				Namespace: "my_namespace",
				MeterSlug: "meter1",
			},
			wantSQL: "DROP VIEW openmeter.om_my_namespace_meter1",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			gotSql := tt.data.toSQL()

			assert.Equal(t, tt.wantSQL, gotSql)
		})
	}
}

func TestQueryMeterView(t *testing.T) {
	subject := "subject1"
	from, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00.001Z")
	to, _ := time.Parse(time.RFC3339, "2023-01-02T00:00:00Z")
	tz, _ := time.LoadLocation("Asia/Shanghai")
	windowSize := meter.WindowSizeHour

	tests := []struct {
		query    queryMeterView
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			query: queryMeterView{
				Database:    "openmeter",
				Namespace:   "my_namespace",
				MeterSlug:   "meter1",
				Aggregation: meter.MeterAggregationSum,
				Subject:     []string{subject},
				From:        &from,
				To:          &to,
				GroupBy:     []string{"subject", "group1", "group2"},
				WindowSize:  &windowSize,
			},
			wantSQL:  "SELECT tumbleStart(windowstart, toIntervalHour(1), 'UTC') AS windowstart, tumbleEnd(windowstart, toIntervalHour(1), 'UTC') AS windowend, sumMerge(value) AS value, subject, group1, group2 FROM openmeter.om_my_namespace_meter1 meter WHERE (meter.subject = ?) AND meter.windowstart >= ? AND meter.windowend <= ? GROUP BY windowstart, windowend, subject, group1, group2 ORDER BY windowstart",
			wantArgs: []interface{}{"subject1", from.Unix(), to.Unix()},
		},
		{ // Aggregate all available data
			query: queryMeterView{
				Database:    "openmeter",
				Namespace:   "my_namespace",
				MeterSlug:   "meter1",
				Aggregation: meter.MeterAggregationSum,
			},
			wantSQL:  "SELECT min(windowstart), max(windowend), sumMerge(value) AS value FROM openmeter.om_my_namespace_meter1 meter",
			wantArgs: nil,
		},
		{ // Aggregate with count aggregation
			query: queryMeterView{
				Database:    "openmeter",
				Namespace:   "my_namespace",
				MeterSlug:   "meter1",
				Aggregation: meter.MeterAggregationCount,
			},
			wantSQL:  "SELECT min(windowstart), max(windowend), toFloat64(countMerge(value)) AS value FROM openmeter.om_my_namespace_meter1 meter",
			wantArgs: nil,
		},
		{ // Aggregate data from start
			query: queryMeterView{
				Database:    "openmeter",
				Namespace:   "my_namespace",
				MeterSlug:   "meter1",
				Aggregation: meter.MeterAggregationSum,
				From:        &from,
			},
			wantSQL:  "SELECT min(windowstart), max(windowend), sumMerge(value) AS value FROM openmeter.om_my_namespace_meter1 meter WHERE meter.windowstart >= ?",
			wantArgs: []interface{}{from.Unix()},
		},
		{ // Aggregate data between period
			query: queryMeterView{
				Database:    "openmeter",
				Namespace:   "my_namespace",
				MeterSlug:   "meter1",
				Aggregation: meter.MeterAggregationSum,
				From:        &from,
				To:          &to,
			},
			wantSQL:  "SELECT min(windowstart), max(windowend), sumMerge(value) AS value FROM openmeter.om_my_namespace_meter1 meter WHERE meter.windowstart >= ? AND meter.windowend <= ?",
			wantArgs: []interface{}{from.Unix(), to.Unix()},
		},
		{ // Aggregate data between period, groupped by window size
			query: queryMeterView{
				Database:    "openmeter",
				Namespace:   "my_namespace",
				MeterSlug:   "meter1",
				Aggregation: meter.MeterAggregationSum,
				From:        &from,
				To:          &to,
				WindowSize:  &windowSize,
			},
			wantSQL:  "SELECT tumbleStart(windowstart, toIntervalHour(1), 'UTC') AS windowstart, tumbleEnd(windowstart, toIntervalHour(1), 'UTC') AS windowend, sumMerge(value) AS value FROM openmeter.om_my_namespace_meter1 meter WHERE meter.windowstart >= ? AND meter.windowend <= ? GROUP BY windowstart, windowend ORDER BY windowstart",
			wantArgs: []interface{}{from.Unix(), to.Unix()},
		},
		{ // Aggregate data between period in a different timezone, groupped by window size
			query: queryMeterView{
				Database:       "openmeter",
				Namespace:      "my_namespace",
				MeterSlug:      "meter1",
				Aggregation:    meter.MeterAggregationSum,
				From:           &from,
				To:             &to,
				WindowSize:     &windowSize,
				WindowTimeZone: tz,
			},
			wantSQL:  "SELECT tumbleStart(windowstart, toIntervalHour(1), 'Asia/Shanghai') AS windowstart, tumbleEnd(windowstart, toIntervalHour(1), 'Asia/Shanghai') AS windowend, sumMerge(value) AS value FROM openmeter.om_my_namespace_meter1 meter WHERE meter.windowstart >= ? AND meter.windowend <= ? GROUP BY windowstart, windowend ORDER BY windowstart",
			wantArgs: []interface{}{from.Unix(), to.Unix()},
		},
		{ // Aggregate data for a single subject
			query: queryMeterView{
				Database:    "openmeter",
				Namespace:   "my_namespace",
				MeterSlug:   "meter1",
				Aggregation: meter.MeterAggregationSum,
				Subject:     []string{subject},
				GroupBy:     []string{"subject"},
			},
			wantSQL:  "SELECT min(windowstart), max(windowend), sumMerge(value) AS value, subject FROM openmeter.om_my_namespace_meter1 meter WHERE (meter.subject = ?) GROUP BY subject",
			wantArgs: []interface{}{"subject1"},
		},
		{ // Aggregate data for a single subject and group by additional fields
			query: queryMeterView{
				Database:    "openmeter",
				Namespace:   "my_namespace",
				MeterSlug:   "meter1",
				Aggregation: meter.MeterAggregationSum,
				Subject:     []string{subject},
				GroupBy:     []string{"subject", "group1", "group2"},
			},
			wantSQL:  "SELECT min(windowstart), max(windowend), sumMerge(value) AS value, subject, group1, group2 FROM openmeter.om_my_namespace_meter1 meter WHERE (meter.subject = ?) GROUP BY subject, group1, group2",
			wantArgs: []interface{}{"subject1"},
		},
		{ // Aggregate data for a multiple subjects
			query: queryMeterView{
				Database:    "openmeter",
				Namespace:   "my_namespace",
				MeterSlug:   "meter1",
				Aggregation: meter.MeterAggregationSum,
				Subject:     []string{subject, "subject2"},
				GroupBy:     []string{"subject"},
			},
			wantSQL:  "SELECT min(windowstart), max(windowend), sumMerge(value) AS value, subject FROM openmeter.om_my_namespace_meter1 meter WHERE (meter.subject = ? OR meter.subject = ?) GROUP BY subject",
			wantArgs: []interface{}{"subject1", "subject2"},
		},
		{ // Aggregate data with filtering for a single group and single value
			query: queryMeterView{
				Database:      "openmeter",
				Namespace:     "my_namespace",
				MeterSlug:     "meter1",
				Aggregation:   meter.MeterAggregationSum,
				FilterGroupBy: map[string][]string{"g1": {"g1v1"}},
			},
			wantSQL:  "SELECT min(windowstart), max(windowend), sumMerge(value) AS value FROM openmeter.om_my_namespace_meter1 meter WHERE (meter.g1 = ?)",
			wantArgs: []interface{}{"g1v1"},
		},
		{ // Aggregate data with filtering for a single group and multiple values
			query: queryMeterView{
				Database:      "openmeter",
				Namespace:     "my_namespace",
				MeterSlug:     "meter1",
				Aggregation:   meter.MeterAggregationSum,
				FilterGroupBy: map[string][]string{"g1": {"g1v1", "g1v2"}},
			},
			wantSQL:  "SELECT min(windowstart), max(windowend), sumMerge(value) AS value FROM openmeter.om_my_namespace_meter1 meter WHERE (meter.g1 = ? OR meter.g1 = ?)",
			wantArgs: []interface{}{"g1v1", "g1v2"},
		},
		{ // Aggregate data with filtering for multiple groups and multiple values
			query: queryMeterView{
				Database:      "openmeter",
				Namespace:     "my_namespace",
				MeterSlug:     "meter1",
				Aggregation:   meter.MeterAggregationSum,
				FilterGroupBy: map[string][]string{"g1": {"g1v1", "g1v2"}, "g2": {"g2v1", "g2v2"}},
			},
			wantSQL:  "SELECT min(windowstart), max(windowend), sumMerge(value) AS value FROM openmeter.om_my_namespace_meter1 meter WHERE (meter.g1 = ? OR meter.g1 = ?) AND (meter.g2 = ? OR meter.g2 = ?)",
			wantArgs: []interface{}{"g1v1", "g1v2", "g2v1", "g2v2"},
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

func TestListMeterViewSubjects(t *testing.T) {
	from, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00.001Z")
	to, _ := time.Parse(time.RFC3339, "2023-01-02T00:00:00Z")

	tests := []struct {
		query    listMeterViewSubjects
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			query: listMeterViewSubjects{
				Database:  "openmeter",
				Namespace: "my_namespace",
				MeterSlug: "meter1",
			},
			wantSQL:  "SELECT DISTINCT subject FROM openmeter.om_my_namespace_meter1 ORDER BY subject",
			wantArgs: nil,
		},
		{
			query: listMeterViewSubjects{
				Database:  "openmeter",
				Namespace: "my_namespace",
				MeterSlug: "meter1",
				From:      &from,
			},
			wantSQL:  "SELECT DISTINCT subject FROM openmeter.om_my_namespace_meter1 WHERE windowstart >= ? ORDER BY subject",
			wantArgs: []interface{}{from.Unix()},
		},
		{
			query: listMeterViewSubjects{
				Database:  "openmeter",
				Namespace: "my_namespace",
				MeterSlug: "meter1",
				From:      &from,
				To:        &to,
			},
			wantSQL:  "SELECT DISTINCT subject FROM openmeter.om_my_namespace_meter1 WHERE windowstart >= ? AND windowend <= ? ORDER BY subject",
			wantArgs: []interface{}{from.Unix(), to.Unix()},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			gotSql, gotArgs := tt.query.toSQL()

			assert.Equal(t, tt.wantArgs, gotArgs)
			assert.Equal(t, tt.wantSQL, gotSql)
		})
	}
}
