package raw_events

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/pkg/models"
)

func TestQueryMeter(t *testing.T) {
	subject := "subject1"
	from, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00.001Z")
	to, _ := time.Parse(time.RFC3339, "2023-01-02T00:00:00Z")
	tz, _ := time.LoadLocation("Asia/Shanghai")
	windowSize := models.WindowSizeHour

	tests := []struct {
		query    queryMeter
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: models.Meter{
					Slug:          "meter1",
					EventType:     "event1",
					Aggregation:   models.MeterAggregationSum,
					ValueProperty: "$.value",
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
				Subject:    []string{subject},
				From:       &from,
				To:         &to,
				GroupBy:    []string{"subject", "group1", "group2"},
				WindowSize: &windowSize,
			},
			wantSQL:  "SELECT tumbleStart(om_events.time, toIntervalHour(1), 'UTC') AS windowstart, tumbleEnd(om_events.time, toIntervalHour(1), 'UTC') AS windowend, sum(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value'))) AS value, om_events.subject, JSON_VALUE(om_events.data, '$.group1') as group1, JSON_VALUE(om_events.data, '$.group2') as group2 FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND (om_events.subject = ?) AND om_events.time >= ? AND om_events.time <= ? GROUP BY windowstart, windowend, subject, group1, group2 ORDER BY windowstart",
			wantArgs: []interface{}{"my_namespace", "event1", "subject1", from.Unix(), to.Unix()},
		},
		{ // Aggregate all available data
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: models.Meter{
					Slug:          "meter1",
					EventType:     "event1",
					Aggregation:   models.MeterAggregationSum,
					ValueProperty: "$.value",
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value'))) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ?",
			wantArgs: []interface{}{"my_namespace", "event1"},
		},
		{ // Aggregate with count aggregation
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: models.Meter{
					Slug:        "meter1",
					EventType:   "event1",
					Aggregation: models.MeterAggregationCount,
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, toFloat64(count(*)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ?",
			wantArgs: []interface{}{"my_namespace", "event1"},
		},
		{ // Aggregate data from start
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: models.Meter{
					Slug:          "meter1",
					EventType:     "event1",
					Aggregation:   models.MeterAggregationSum,
					ValueProperty: "$.value",
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
				From: &from,
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value'))) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.time >= ?",
			wantArgs: []interface{}{"my_namespace", "event1", from.Unix()},
		},
		{ // Aggregate data between period
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: models.Meter{
					Slug:          "meter1",
					EventType:     "event1",
					Aggregation:   models.MeterAggregationSum,
					ValueProperty: "$.value",
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
				From: &from,
				To:   &to,
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value'))) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.time >= ? AND om_events.time <= ?",
			wantArgs: []interface{}{"my_namespace", "event1", from.Unix(), to.Unix()},
		},
		{ // Aggregate data between period, groupped by window size
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: models.Meter{
					Slug:          "meter1",
					EventType:     "event1",
					Aggregation:   models.MeterAggregationSum,
					ValueProperty: "$.value",
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
				From:       &from,
				To:         &to,
				WindowSize: &windowSize,
			},
			wantSQL:  "SELECT tumbleStart(om_events.time, toIntervalHour(1), 'UTC') AS windowstart, tumbleEnd(om_events.time, toIntervalHour(1), 'UTC') AS windowend, sum(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value'))) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.time >= ? AND om_events.time <= ? GROUP BY windowstart, windowend ORDER BY windowstart",
			wantArgs: []interface{}{"my_namespace", "event1", from.Unix(), to.Unix()},
		},
		{ // Aggregate data between period in a different timezone, groupped by window size
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: models.Meter{
					Slug:          "meter1",
					EventType:     "event1",
					Aggregation:   models.MeterAggregationSum,
					ValueProperty: "$.value",
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
				From:           &from,
				To:             &to,
				WindowSize:     &windowSize,
				WindowTimeZone: tz,
			},
			wantSQL:  "SELECT tumbleStart(om_events.time, toIntervalHour(1), 'Asia/Shanghai') AS windowstart, tumbleEnd(om_events.time, toIntervalHour(1), 'Asia/Shanghai') AS windowend, sum(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value'))) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.time >= ? AND om_events.time <= ? GROUP BY windowstart, windowend ORDER BY windowstart",
			wantArgs: []interface{}{"my_namespace", "event1", from.Unix(), to.Unix()},
		},
		{ // Aggregate data for a single subject
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: models.Meter{
					Slug:          "meter1",
					EventType:     "event1",
					Aggregation:   models.MeterAggregationSum,
					ValueProperty: "$.value",
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
				Subject: []string{subject},
				GroupBy: []string{"subject"},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value'))) AS value, om_events.subject FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND (om_events.subject = ?) GROUP BY subject",
			wantArgs: []interface{}{"my_namespace", "event1", "subject1"},
		},
		{ // Aggregate data for a single subject and group by additional fields
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: models.Meter{
					Slug:          "meter1",
					EventType:     "event1",
					Aggregation:   models.MeterAggregationSum,
					ValueProperty: "$.value",
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
				Subject: []string{subject},
				GroupBy: []string{"subject", "group1", "group2"},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value'))) AS value, om_events.subject, JSON_VALUE(om_events.data, '$.group1') as group1, JSON_VALUE(om_events.data, '$.group2') as group2 FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND (om_events.subject = ?) GROUP BY subject, group1, group2",
			wantArgs: []interface{}{"my_namespace", "event1", "subject1"},
		},
		{ // Aggregate data for a multiple subjects
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: models.Meter{
					Slug:          "meter1",
					EventType:     "event1",
					Aggregation:   models.MeterAggregationSum,
					ValueProperty: "$.value",
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
				Subject: []string{subject, "subject2"},
				GroupBy: []string{"subject"},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value'))) AS value, om_events.subject FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND (om_events.subject = ? OR om_events.subject = ?) GROUP BY subject",
			wantArgs: []interface{}{"my_namespace", "event1", "subject1", "subject2"},
		},
		{ // Aggregate data with filtering for a single group and single value
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: models.Meter{
					Slug:          "meter1",
					EventType:     "event1",
					Aggregation:   models.MeterAggregationSum,
					ValueProperty: "$.value",
					GroupBy: map[string]string{
						"g1": "$.group1",
						"g2": "$.group2",
					},
				},
				FilterGroupBy: map[string][]string{"g1": {"g1v1"}},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value'))) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND (JSON_VALUE(om_events.data, '$.group1') = 'g1v1')",
			wantArgs: []interface{}{"my_namespace", "event1"},
		},
		{ // Aggregate data with filtering for a single group and multiple values
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: models.Meter{
					Slug:          "meter1",
					EventType:     "event1",
					Aggregation:   models.MeterAggregationSum,
					ValueProperty: "$.value",
					GroupBy: map[string]string{
						"g1": "$.group1",
						"g2": "$.group2",
					},
				},
				FilterGroupBy: map[string][]string{"g1": {"g1v1", "g1v2"}},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value'))) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND (JSON_VALUE(om_events.data, '$.group1') = 'g1v1' OR JSON_VALUE(om_events.data, '$.group1') = 'g1v2')",
			wantArgs: []interface{}{"my_namespace", "event1"},
		},
		{ // Aggregate data with filtering for multiple groups and multiple values
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: models.Meter{
					Slug:          "meter1",
					EventType:     "event1",
					Aggregation:   models.MeterAggregationSum,
					ValueProperty: "$.value",
					GroupBy: map[string]string{
						"g1": "$.group1",
						"g2": "$.group2",
					},
				},
				FilterGroupBy: map[string][]string{"g1": {"g1v1", "g1v2"}, "g2": {"g2v1", "g2v2"}},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value'))) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND (JSON_VALUE(om_events.data, '$.group1') = 'g1v1' OR JSON_VALUE(om_events.data, '$.group1') = 'g1v2') AND (JSON_VALUE(om_events.data, '$.group2') = 'g2v1' OR JSON_VALUE(om_events.data, '$.group2') = 'g2v2')",
			wantArgs: []interface{}{"my_namespace", "event1"},
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

func TestListMeterSubjects(t *testing.T) {
	from, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00.001Z")
	to, _ := time.Parse(time.RFC3339, "2023-01-02T00:00:00Z")

	tests := []struct {
		query    listMeterSubjectsQuery
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			query: listMeterSubjectsQuery{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: models.Meter{
					Slug:        "meter1",
					EventType:   "event1",
					Aggregation: models.MeterAggregationSum,
				},
			},
			wantSQL:  "SELECT DISTINCT subject FROM openmeter.om_events WHERE namespace = ? AND type = ? ORDER BY subject",
			wantArgs: []interface{}{"my_namespace", "event1"},
		},
		{
			query: listMeterSubjectsQuery{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: models.Meter{
					Slug:        "meter1",
					EventType:   "event1",
					Aggregation: models.MeterAggregationSum,
				},
				From: &from,
			},
			wantSQL:  "SELECT DISTINCT subject FROM openmeter.om_events WHERE namespace = ? AND type = ? AND time >= ? ORDER BY subject",
			wantArgs: []interface{}{"my_namespace", "event1", from.Unix()},
		},
		{
			query: listMeterSubjectsQuery{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: models.Meter{
					Slug:        "meter1",
					EventType:   "event1",
					Aggregation: models.MeterAggregationSum,
				},
				From: &from,
				To:   &to,
			},
			wantSQL:  "SELECT DISTINCT subject FROM openmeter.om_events WHERE namespace = ? AND type = ? AND time >= ? AND time <= ? ORDER BY subject",
			wantArgs: []interface{}{"my_namespace", "event1", from.Unix(), to.Unix()},
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
