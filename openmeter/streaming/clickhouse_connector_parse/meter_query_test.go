package clickhouse_connector_parse

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
				Database:  "openmeter",
				Namespace: "my_namespace",
				Meter: models.Meter{
					Slug:        "meter1",
					Aggregation: models.MeterAggregationSum,
				},
				Subject:    []string{subject},
				From:       &from,
				To:         &to,
				GroupBy:    []string{"subject", "group1", "group2"},
				WindowSize: &windowSize,
			},
			wantSQL:  "SELECT tumbleStart(om_meter_events.time, toIntervalHour(1), 'UTC') AS windowstart, tumbleEnd(om_meter_events.time, toIntervalHour(1), 'UTC') AS windowend, sum(om_meter_events.value) AS value, om_meter_events.subject, om_meter_events.group_by['group1'] as group1, om_meter_events.group_by['group2'] as group2 FROM openmeter.om_meter_events WHERE om_meter_events.namespace = ? AND om_meter_events.meter = ? AND (om_meter_events.subject = ?) AND om_meter_events.time >= ? AND om_meter_events.time <= ? GROUP BY windowstart, windowend, subject, group1, group2 ORDER BY windowstart",
			wantArgs: []interface{}{"my_namespace", "meter1", "subject1", from.Unix(), to.Unix()},
		},
		{ // Aggregate all available data
			query: queryMeter{
				Database:  "openmeter",
				Namespace: "my_namespace",
				Meter: models.Meter{
					Slug:        "meter1",
					Aggregation: models.MeterAggregationSum,
				},
			},
			wantSQL:  "SELECT min(time) as windowstart, max(time) as windowend, sum(om_meter_events.value) AS value FROM openmeter.om_meter_events WHERE om_meter_events.namespace = ? AND om_meter_events.meter = ?",
			wantArgs: []interface{}{"my_namespace", "meter1"},
		},
		{ // Aggregate with count aggregation
			query: queryMeter{
				Database:  "openmeter",
				Namespace: "my_namespace",
				Meter: models.Meter{
					Slug:        "meter1",
					Aggregation: models.MeterAggregationCount,
				},
			},
			wantSQL:  "SELECT min(time) as windowstart, max(time) as windowend, sum(om_meter_events.value) AS value FROM openmeter.om_meter_events WHERE om_meter_events.namespace = ? AND om_meter_events.meter = ?",
			wantArgs: []interface{}{"my_namespace", "meter1"},
		},
		{ // Aggregate data from start
			query: queryMeter{
				Database:  "openmeter",
				Namespace: "my_namespace",
				Meter: models.Meter{
					Slug:        "meter1",
					Aggregation: models.MeterAggregationSum,
				},
				From: &from,
			},
			wantSQL:  "SELECT min(time) as windowstart, max(time) as windowend, sum(om_meter_events.value) AS value FROM openmeter.om_meter_events WHERE om_meter_events.namespace = ? AND om_meter_events.meter = ? AND om_meter_events.time >= ?",
			wantArgs: []interface{}{"my_namespace", "meter1", from.Unix()},
		},
		{ // Aggregate data between period
			query: queryMeter{
				Database:  "openmeter",
				Namespace: "my_namespace",
				Meter: models.Meter{
					Slug:        "meter1",
					Aggregation: models.MeterAggregationSum,
				},
				From: &from,
				To:   &to,
			},
			wantSQL:  "SELECT min(time) as windowstart, max(time) as windowend, sum(om_meter_events.value) AS value FROM openmeter.om_meter_events WHERE om_meter_events.namespace = ? AND om_meter_events.meter = ? AND om_meter_events.time >= ? AND om_meter_events.time <= ?",
			wantArgs: []interface{}{"my_namespace", "meter1", from.Unix(), to.Unix()},
		},
		{ // Aggregate data between period, groupped by window size
			query: queryMeter{
				Database:  "openmeter",
				Namespace: "my_namespace",
				Meter: models.Meter{
					Slug:        "meter1",
					Aggregation: models.MeterAggregationSum,
				},
				From:       &from,
				To:         &to,
				WindowSize: &windowSize,
			},
			wantSQL:  "SELECT tumbleStart(om_meter_events.time, toIntervalHour(1), 'UTC') AS windowstart, tumbleEnd(om_meter_events.time, toIntervalHour(1), 'UTC') AS windowend, sum(om_meter_events.value) AS value FROM openmeter.om_meter_events WHERE om_meter_events.namespace = ? AND om_meter_events.meter = ? AND om_meter_events.time >= ? AND om_meter_events.time <= ? GROUP BY windowstart, windowend ORDER BY windowstart",
			wantArgs: []interface{}{"my_namespace", "meter1", from.Unix(), to.Unix()},
		},
		{ // Aggregate data between period in a different timezone, groupped by window size
			query: queryMeter{
				Database:  "openmeter",
				Namespace: "my_namespace",
				Meter: models.Meter{
					Slug:        "meter1",
					Aggregation: models.MeterAggregationSum,
				},
				From:           &from,
				To:             &to,
				WindowSize:     &windowSize,
				WindowTimeZone: tz,
			},
			wantSQL:  "SELECT tumbleStart(om_meter_events.time, toIntervalHour(1), 'Asia/Shanghai') AS windowstart, tumbleEnd(om_meter_events.time, toIntervalHour(1), 'Asia/Shanghai') AS windowend, sum(om_meter_events.value) AS value FROM openmeter.om_meter_events WHERE om_meter_events.namespace = ? AND om_meter_events.meter = ? AND om_meter_events.time >= ? AND om_meter_events.time <= ? GROUP BY windowstart, windowend ORDER BY windowstart",
			wantArgs: []interface{}{"my_namespace", "meter1", from.Unix(), to.Unix()},
		},
		{ // Aggregate data for a single subject
			query: queryMeter{
				Database:  "openmeter",
				Namespace: "my_namespace",
				Meter: models.Meter{
					Slug:        "meter1",
					Aggregation: models.MeterAggregationSum,
				},
				Subject: []string{subject},
				GroupBy: []string{"subject"},
			},
			wantSQL:  "SELECT min(time) as windowstart, max(time) as windowend, sum(om_meter_events.value) AS value, om_meter_events.subject FROM openmeter.om_meter_events WHERE om_meter_events.namespace = ? AND om_meter_events.meter = ? AND (om_meter_events.subject = ?) GROUP BY subject",
			wantArgs: []interface{}{"my_namespace", "meter1", "subject1"},
		},
		{ // Aggregate data for a single subject and group by additional fields
			query: queryMeter{
				Database:  "openmeter",
				Namespace: "my_namespace",
				Meter: models.Meter{
					Slug:        "meter1",
					Aggregation: models.MeterAggregationSum,
				},
				Subject: []string{subject},
				GroupBy: []string{"subject", "group1", "group2"},
			},
			wantSQL:  "SELECT min(time) as windowstart, max(time) as windowend, sum(om_meter_events.value) AS value, om_meter_events.subject, om_meter_events.group_by['group1'] as group1, om_meter_events.group_by['group2'] as group2 FROM openmeter.om_meter_events WHERE om_meter_events.namespace = ? AND om_meter_events.meter = ? AND (om_meter_events.subject = ?) GROUP BY subject, group1, group2",
			wantArgs: []interface{}{"my_namespace", "meter1", "subject1"},
		},
		{ // Aggregate data for a multiple subjects
			query: queryMeter{
				Database:  "openmeter",
				Namespace: "my_namespace",
				Meter: models.Meter{
					Slug:        "meter1",
					Aggregation: models.MeterAggregationSum,
				},
				Subject: []string{subject, "subject2"},
				GroupBy: []string{"subject"},
			},
			wantSQL:  "SELECT min(time) as windowstart, max(time) as windowend, sum(om_meter_events.value) AS value, om_meter_events.subject FROM openmeter.om_meter_events WHERE om_meter_events.namespace = ? AND om_meter_events.meter = ? AND (om_meter_events.subject = ? OR om_meter_events.subject = ?) GROUP BY subject",
			wantArgs: []interface{}{"my_namespace", "meter1", "subject1", "subject2"},
		},
		{ // Aggregate data with filtering for a single group and single value
			query: queryMeter{
				Database:  "openmeter",
				Namespace: "my_namespace",
				Meter: models.Meter{
					Slug:        "meter1",
					Aggregation: models.MeterAggregationSum,
				},
				FilterGroupBy: map[string][]string{"g1": {"g1v1"}},
			},
			wantSQL:  "SELECT min(time) as windowstart, max(time) as windowend, sum(om_meter_events.value) AS value FROM openmeter.om_meter_events WHERE om_meter_events.namespace = ? AND om_meter_events.meter = ? AND (om_meter_events.group_by['g1'] = ?)",
			wantArgs: []interface{}{"my_namespace", "meter1", "g1v1"},
		},
		{ // Aggregate data with filtering for a single group and multiple values
			query: queryMeter{
				Database:  "openmeter",
				Namespace: "my_namespace",
				Meter: models.Meter{
					Slug:        "meter1",
					Aggregation: models.MeterAggregationSum,
				},
				FilterGroupBy: map[string][]string{"g1": {"g1v1", "g1v2"}},
			},
			wantSQL:  "SELECT min(time) as windowstart, max(time) as windowend, sum(om_meter_events.value) AS value FROM openmeter.om_meter_events WHERE om_meter_events.namespace = ? AND om_meter_events.meter = ? AND (om_meter_events.group_by['g1'] = ? OR om_meter_events.group_by['g1'] = ?)",
			wantArgs: []interface{}{"my_namespace", "meter1", "g1v1", "g1v2"},
		},
		{ // Aggregate data with filtering for multiple groups and multiple values
			query: queryMeter{
				Database:  "openmeter",
				Namespace: "my_namespace",
				Meter: models.Meter{
					Slug:        "meter1",
					Aggregation: models.MeterAggregationSum,
				},
				FilterGroupBy: map[string][]string{"g1": {"g1v1", "g1v2"}, "g2": {"g2v1", "g2v2"}},
			},
			wantSQL:  "SELECT min(time) as windowstart, max(time) as windowend, sum(om_meter_events.value) AS value FROM openmeter.om_meter_events WHERE om_meter_events.namespace = ? AND om_meter_events.meter = ? AND (om_meter_events.group_by['g1'] = ? OR om_meter_events.group_by['g1'] = ?) AND (om_meter_events.group_by['g2'] = ? OR om_meter_events.group_by['g2'] = ?)",
			wantArgs: []interface{}{"my_namespace", "meter1", "g1v1", "g1v2", "g2v1", "g2v2"},
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
				Database:  "openmeter",
				Namespace: "my_namespace",
				MeterSlug: "meter1",
			},
			wantSQL:  "SELECT DISTINCT subject FROM openmeter.om_meter_events WHERE namespace = ? AND meter = ? ORDER BY subject",
			wantArgs: []interface{}{"my_namespace", "meter1"},
		},
		{
			query: listMeterSubjectsQuery{
				Database:  "openmeter",
				Namespace: "my_namespace",
				MeterSlug: "meter1",
				From:      &from,
			},
			wantSQL:  "SELECT DISTINCT subject FROM openmeter.om_meter_events WHERE namespace = ? AND meter = ? AND time >= ? ORDER BY subject",
			wantArgs: []interface{}{"my_namespace", "meter1", from.Unix()},
		},
		{
			query: listMeterSubjectsQuery{
				Database:  "openmeter",
				Namespace: "my_namespace",
				MeterSlug: "meter1",
				From:      &from,
				To:        &to,
			},
			wantSQL:  "SELECT DISTINCT subject FROM openmeter.om_meter_events WHERE namespace = ? AND meter = ? AND time >= ? AND time <= ? ORDER BY subject",
			wantArgs: []interface{}{"my_namespace", "meter1", from.Unix(), to.Unix()},
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
