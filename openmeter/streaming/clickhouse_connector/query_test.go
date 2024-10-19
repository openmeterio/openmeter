package clickhouse_connector

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/pkg/models"
)

func TestQueryMeterView(t *testing.T) {
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
				Database:    "openmeter",
				Namespace:   "my_namespace",
				MeterSlug:   "meter1",
				Aggregation: models.MeterAggregationSum,
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
			query: queryMeter{
				Database:    "openmeter",
				Namespace:   "my_namespace",
				MeterSlug:   "meter1",
				Aggregation: models.MeterAggregationSum,
			},
			wantSQL:  "SELECT min(windowstart), max(windowend), sumMerge(value) AS value FROM openmeter.om_my_namespace_meter1 meter",
			wantArgs: nil,
		},
		{ // Aggregate with count aggregation
			query: queryMeter{
				Database:    "openmeter",
				Namespace:   "my_namespace",
				MeterSlug:   "meter1",
				Aggregation: models.MeterAggregationCount,
			},
			wantSQL:  "SELECT min(windowstart), max(windowend), toFloat64(countMerge(value)) AS value FROM openmeter.om_my_namespace_meter1 meter",
			wantArgs: nil,
		},
		{ // Aggregate data from start
			query: queryMeter{
				Database:    "openmeter",
				Namespace:   "my_namespace",
				MeterSlug:   "meter1",
				Aggregation: models.MeterAggregationSum,
				From:        &from,
			},
			wantSQL:  "SELECT min(windowstart), max(windowend), sumMerge(value) AS value FROM openmeter.om_my_namespace_meter1 meter WHERE meter.windowstart >= ?",
			wantArgs: []interface{}{from.Unix()},
		},
		{ // Aggregate data between period
			query: queryMeter{
				Database:    "openmeter",
				Namespace:   "my_namespace",
				MeterSlug:   "meter1",
				Aggregation: models.MeterAggregationSum,
				From:        &from,
				To:          &to,
			},
			wantSQL:  "SELECT min(windowstart), max(windowend), sumMerge(value) AS value FROM openmeter.om_my_namespace_meter1 meter WHERE meter.windowstart >= ? AND meter.windowend <= ?",
			wantArgs: []interface{}{from.Unix(), to.Unix()},
		},
		{ // Aggregate data between period, groupped by window size
			query: queryMeter{
				Database:    "openmeter",
				Namespace:   "my_namespace",
				MeterSlug:   "meter1",
				Aggregation: models.MeterAggregationSum,
				From:        &from,
				To:          &to,
				WindowSize:  &windowSize,
			},
			wantSQL:  "SELECT tumbleStart(windowstart, toIntervalHour(1), 'UTC') AS windowstart, tumbleEnd(windowstart, toIntervalHour(1), 'UTC') AS windowend, sumMerge(value) AS value FROM openmeter.om_my_namespace_meter1 meter WHERE meter.windowstart >= ? AND meter.windowend <= ? GROUP BY windowstart, windowend ORDER BY windowstart",
			wantArgs: []interface{}{from.Unix(), to.Unix()},
		},
		{ // Aggregate data between period in a different timezone, groupped by window size
			query: queryMeter{
				Database:       "openmeter",
				Namespace:      "my_namespace",
				MeterSlug:      "meter1",
				Aggregation:    models.MeterAggregationSum,
				From:           &from,
				To:             &to,
				WindowSize:     &windowSize,
				WindowTimeZone: tz,
			},
			wantSQL:  "SELECT tumbleStart(windowstart, toIntervalHour(1), 'Asia/Shanghai') AS windowstart, tumbleEnd(windowstart, toIntervalHour(1), 'Asia/Shanghai') AS windowend, sumMerge(value) AS value FROM openmeter.om_my_namespace_meter1 meter WHERE meter.windowstart >= ? AND meter.windowend <= ? GROUP BY windowstart, windowend ORDER BY windowstart",
			wantArgs: []interface{}{from.Unix(), to.Unix()},
		},
		{ // Aggregate data for a single subject
			query: queryMeter{
				Database:    "openmeter",
				Namespace:   "my_namespace",
				MeterSlug:   "meter1",
				Aggregation: models.MeterAggregationSum,
				Subject:     []string{subject},
				GroupBy:     []string{"subject"},
			},
			wantSQL:  "SELECT min(windowstart), max(windowend), sumMerge(value) AS value, subject FROM openmeter.om_my_namespace_meter1 meter WHERE (meter.subject = ?) GROUP BY subject",
			wantArgs: []interface{}{"subject1"},
		},
		{ // Aggregate data for a single subject and group by additional fields
			query: queryMeter{
				Database:    "openmeter",
				Namespace:   "my_namespace",
				MeterSlug:   "meter1",
				Aggregation: models.MeterAggregationSum,
				Subject:     []string{subject},
				GroupBy:     []string{"subject", "group1", "group2"},
			},
			wantSQL:  "SELECT min(windowstart), max(windowend), sumMerge(value) AS value, subject, group1, group2 FROM openmeter.om_my_namespace_meter1 meter WHERE (meter.subject = ?) GROUP BY subject, group1, group2",
			wantArgs: []interface{}{"subject1"},
		},
		{ // Aggregate data for a multiple subjects
			query: queryMeter{
				Database:    "openmeter",
				Namespace:   "my_namespace",
				MeterSlug:   "meter1",
				Aggregation: models.MeterAggregationSum,
				Subject:     []string{subject, "subject2"},
				GroupBy:     []string{"subject"},
			},
			wantSQL:  "SELECT min(windowstart), max(windowend), sumMerge(value) AS value, subject FROM openmeter.om_my_namespace_meter1 meter WHERE (meter.subject = ? OR meter.subject = ?) GROUP BY subject",
			wantArgs: []interface{}{"subject1", "subject2"},
		},
		{ // Aggregate data with filtering for a single group and single value
			query: queryMeter{
				Database:      "openmeter",
				Namespace:     "my_namespace",
				MeterSlug:     "meter1",
				Aggregation:   models.MeterAggregationSum,
				FilterGroupBy: map[string][]string{"g1": {"g1v1"}},
			},
			wantSQL:  "SELECT min(windowstart), max(windowend), sumMerge(value) AS value FROM openmeter.om_my_namespace_meter1 meter WHERE (meter.g1 = ?)",
			wantArgs: []interface{}{"g1v1"},
		},
		{ // Aggregate data with filtering for a single group and multiple values
			query: queryMeter{
				Database:      "openmeter",
				Namespace:     "my_namespace",
				MeterSlug:     "meter1",
				Aggregation:   models.MeterAggregationSum,
				FilterGroupBy: map[string][]string{"g1": {"g1v1", "g1v2"}},
			},
			wantSQL:  "SELECT min(windowstart), max(windowend), sumMerge(value) AS value FROM openmeter.om_my_namespace_meter1 meter WHERE (meter.g1 = ? OR meter.g1 = ?)",
			wantArgs: []interface{}{"g1v1", "g1v2"},
		},
		{ // Aggregate data with filtering for multiple groups and multiple values
			query: queryMeter{
				Database:      "openmeter",
				Namespace:     "my_namespace",
				MeterSlug:     "meter1",
				Aggregation:   models.MeterAggregationSum,
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
			wantSQL:  "SELECT DISTINCT subject FROM openmeter.om_my_namespace_meter1 ORDER BY subject",
			wantArgs: nil,
		},
		{
			query: listMeterSubjectsQuery{
				Database:  "openmeter",
				Namespace: "my_namespace",
				MeterSlug: "meter1",
				From:      &from,
			},
			wantSQL:  "SELECT DISTINCT subject FROM openmeter.om_my_namespace_meter1 WHERE windowstart >= ? ORDER BY subject",
			wantArgs: []interface{}{from.Unix()},
		},
		{
			query: listMeterSubjectsQuery{
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
