package clickhouse

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/filter"
)

func TestQueryMeterV2(t *testing.T) {
	subject := "subject1"
	from, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00.001Z")
	to, _ := time.Parse(time.RFC3339, "2023-01-02T00:00:00Z")
	tz, _ := time.LoadLocation("Asia/Shanghai")
	windowSize := meter.WindowSizeHour

	tests := []struct {
		query    queryMeterTableV2
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			query: queryMeterTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationSum,
					ValueProperty: lo.ToPtr("$.value"),
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
				Params: streaming.QueryParamsV2{
					Filter: &streaming.Filter{
						Subject: &filter.FilterString{
							Eq: &subject,
						},
						Time: &filter.FilterTime{
							And: &[]filter.FilterTime{
								{
									Gte: &from,
								},
								{
									Lte: &to,
								},
							},
						},
					},
					GroupBy:    []string{"subject", "group1", "group2"},
					WindowSize: &windowSize,
				},
			},
			wantSQL:  "SELECT tumbleStart(om_events.time, toIntervalHour(1), 'UTC') AS windowstart, tumbleEnd(om_events.time, toIntervalHour(1), 'UTC') AS windowend, sum(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value'))) AS value, om_events.subject, JSON_VALUE(om_events.data, '$.group1') as group1, JSON_VALUE(om_events.data, '$.group2') as group2 FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.subject = ? AND (om_events.time >= ? AND om_events.time <= ?) GROUP BY windowstart, windowend, subject, group1, group2 ORDER BY windowstart",
			wantArgs: []interface{}{"my_namespace", "event1", "subject1", from, to},
		},
		{ // Aggregate all available data
			query: queryMeterTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationSum,
					ValueProperty: lo.ToPtr("$.value"),
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
				Params: streaming.QueryParamsV2{},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value'))) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ?",
			wantArgs: []interface{}{"my_namespace", "event1"},
		},
		{ // Aggregate with count aggregation
			query: queryMeterTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:         "meter1",
					EventType:   "event1",
					Aggregation: meter.MeterAggregationCount,
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
				Params: streaming.QueryParamsV2{},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, toFloat64(count(*)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ?",
			wantArgs: []interface{}{"my_namespace", "event1"},
		},
		{ // Aggregate data from start
			query: queryMeterTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationSum,
					ValueProperty: lo.ToPtr("$.value"),
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
				Params: streaming.QueryParamsV2{
					Filter: &streaming.Filter{
						Time: &filter.FilterTime{
							Gte: &from,
						},
					},
				},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value'))) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.time >= ?",
			wantArgs: []interface{}{"my_namespace", "event1", from},
		},
		{ // Aggregate data between period
			query: queryMeterTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationSum,
					ValueProperty: lo.ToPtr("$.value"),
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
				Params: streaming.QueryParamsV2{
					Filter: &streaming.Filter{
						Time: &filter.FilterTime{
							And: &[]filter.FilterTime{
								{
									Gte: &from,
								},
								{
									Lte: &to,
								},
							},
						},
					},
				},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value'))) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND (om_events.time >= ? AND om_events.time <= ?)",
			wantArgs: []interface{}{"my_namespace", "event1", from, to},
		},
		{ // Aggregate data between period, groupped by window size
			query: queryMeterTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationSum,
					ValueProperty: lo.ToPtr("$.value"),
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
				Params: streaming.QueryParamsV2{
					Filter: &streaming.Filter{
						Time: &filter.FilterTime{
							And: &[]filter.FilterTime{
								{
									Gte: &from,
								},
								{
									Lte: &to,
								},
							},
						},
					},
					WindowSize: &windowSize,
				},
			},
			wantSQL:  "SELECT tumbleStart(om_events.time, toIntervalHour(1), 'UTC') AS windowstart, tumbleEnd(om_events.time, toIntervalHour(1), 'UTC') AS windowend, sum(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value'))) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND (om_events.time >= ? AND om_events.time <= ?) GROUP BY windowstart, windowend ORDER BY windowstart",
			wantArgs: []interface{}{"my_namespace", "event1", from, to},
		},
		{ // Aggregate data between period in a different timezone, groupped by window size
			query: queryMeterTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationSum,
					ValueProperty: lo.ToPtr("$.value"),
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
				Params: streaming.QueryParamsV2{
					Filter: &streaming.Filter{
						Time: &filter.FilterTime{
							And: &[]filter.FilterTime{
								{
									Gte: &from,
								},
								{
									Lte: &to,
								},
							},
						},
					},
					WindowSize:     &windowSize,
					WindowTimeZone: tz,
				},
			},
			wantSQL:  "SELECT tumbleStart(om_events.time, toIntervalHour(1), 'Asia/Shanghai') AS windowstart, tumbleEnd(om_events.time, toIntervalHour(1), 'Asia/Shanghai') AS windowend, sum(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value'))) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND (om_events.time >= ? AND om_events.time <= ?) GROUP BY windowstart, windowend ORDER BY windowstart",
			wantArgs: []interface{}{"my_namespace", "event1", from, to},
		},
		{ // Aggregate data for a single subject
			query: queryMeterTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationSum,
					ValueProperty: lo.ToPtr("$.value"),
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
				Params: streaming.QueryParamsV2{
					Filter: &streaming.Filter{
						Subject: &filter.FilterString{
							Eq: &subject,
						},
					},
					GroupBy: []string{"subject"},
				},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value'))) AS value, om_events.subject FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.subject = ? GROUP BY subject",
			wantArgs: []interface{}{"my_namespace", "event1", "subject1"},
		},
		{ // Aggregate data for a single subject and group by additional fields
			query: queryMeterTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationSum,
					ValueProperty: lo.ToPtr("$.value"),
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
				Params: streaming.QueryParamsV2{
					Filter: &streaming.Filter{
						Subject: &filter.FilterString{
							Eq: &subject,
						},
					},
					GroupBy: []string{"subject", "group1", "group2"},
				},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value'))) AS value, om_events.subject, JSON_VALUE(om_events.data, '$.group1') as group1, JSON_VALUE(om_events.data, '$.group2') as group2 FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.subject = ? GROUP BY subject, group1, group2",
			wantArgs: []interface{}{"my_namespace", "event1", "subject1"},
		},
		{ // Aggregate data for a multiple subjects
			query: queryMeterTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationSum,
					ValueProperty: lo.ToPtr("$.value"),
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
				Params: streaming.QueryParamsV2{
					Filter: &streaming.Filter{
						Subject: &filter.FilterString{
							In: &[]string{subject, "subject2"},
						},
					},
					GroupBy: []string{"subject"},
				},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value'))) AS value, om_events.subject FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.subject IN (?) GROUP BY subject",
			wantArgs: []interface{}{"my_namespace", "event1", []string{"subject1", "subject2"}},
		},
		{ // Aggregate data with filtering for a single group and single value
			query: queryMeterTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationSum,
					ValueProperty: lo.ToPtr("$.value"),
					GroupBy: map[string]string{
						"g1": "$.group1",
						"g2": "$.group2",
					},
				},
				Params: streaming.QueryParamsV2{
					Filter: &streaming.Filter{
						GroupBy: &map[string]filter.FilterString{
							"g1": {
								Eq: lo.ToPtr("g1v1"),
							},
						},
					},
				},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value'))) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND JSON_VALUE(om_events.data, '$.group1') = ?",
			wantArgs: []interface{}{"my_namespace", "event1", "g1v1"},
		},
		{ // Aggregate data with filtering for a single group and multiple values
			query: queryMeterTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationSum,
					ValueProperty: lo.ToPtr("$.value"),
					GroupBy: map[string]string{
						"g1": "$.group1",
						"g2": "$.group2",
					},
				},
				Params: streaming.QueryParamsV2{
					Filter: &streaming.Filter{
						GroupBy: &map[string]filter.FilterString{
							"g1": {
								In: &[]string{"g1v1", "g1v2"},
							},
						},
					},
				},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value'))) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND JSON_VALUE(om_events.data, '$.group1') IN (?)",
			wantArgs: []interface{}{"my_namespace", "event1", []string{"g1v1", "g1v2"}},
		},
		{ // Aggregate data with filtering for multiple groups and multiple values
			query: queryMeterTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationSum,
					ValueProperty: lo.ToPtr("$.value"),
					GroupBy: map[string]string{
						"g1": "$.group1",
						"g2": "$.group2",
					},
				},
				Params: streaming.QueryParamsV2{
					Filter: &streaming.Filter{
						GroupBy: &map[string]filter.FilterString{
							"g1": {
								In: &[]string{"g1v1", "g1v2"},
							},
							"g2": {
								In: &[]string{"g2v1", "g2v2"},
							},
						},
					},
				},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value'))) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND JSON_VALUE(om_events.data, '$.group1') IN (?) AND JSON_VALUE(om_events.data, '$.group2') IN (?)",
			wantArgs: []interface{}{"my_namespace", "event1", []string{"g1v1", "g1v2"}, []string{"g2v1", "g2v2"}},
		},
		{ // Aggregate data from the meter's event from time if from time is before the meter's event from time
			query: queryMeterTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationSum,
					ValueProperty: lo.ToPtr("$.value"),
					EventFrom:     lo.ToPtr(from.Add(time.Minute * 10)),
				},
				Params: streaming.QueryParamsV2{
					Filter: &streaming.Filter{
						Time: &filter.FilterTime{
							Gte: &from,
						},
					},
				},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value'))) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.time >= ? AND om_events.time >= ?",
			wantArgs: []interface{}{"my_namespace", "event1", from.Add(time.Minute * 10), from},
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

func TestQueryMeterTableV2_ToCountRowSQL(t *testing.T) {
	subject := "subject1"
	from, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00.001Z")
	to, _ := time.Parse(time.RFC3339, "2023-01-02T00:00:00Z")
	eventFrom, _ := time.Parse(time.RFC3339, "2023-01-01T10:00:00Z")

	tests := []struct {
		name     string
		query    queryMeterTableV2
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			name: "basic count query",
			query: queryMeterTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:       "meter1",
					EventType: "event1",
				},
				Params: streaming.QueryParamsV2{},
			},
			wantSQL:  "SELECT count() AS total FROM openmeter.om_events WHERE namespace = ? AND type = ?",
			wantArgs: []interface{}{"my_namespace", "event1"},
		},
		{
			name: "count query with subject filter",
			query: queryMeterTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:       "meter1",
					EventType: "event1",
				},
				Params: streaming.QueryParamsV2{
					Filter: &streaming.Filter{
						Subject: &filter.FilterString{
							Eq: &subject,
						},
					},
				},
			},
			wantSQL:  "SELECT count() AS total FROM openmeter.om_events WHERE namespace = ? AND type = ? AND om_events.subject = ?",
			wantArgs: []interface{}{"my_namespace", "event1", "subject1"},
		},
		{
			name: "count query with time filter",
			query: queryMeterTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:       "meter1",
					EventType: "event1",
				},
				Params: streaming.QueryParamsV2{
					Filter: &streaming.Filter{
						Time: &filter.FilterTime{
							Gte: &from,
							Lte: &to,
						},
					},
				},
			},
			wantSQL:  "SELECT count() AS total FROM openmeter.om_events WHERE namespace = ? AND type = ? AND om_events.time >= ?",
			wantArgs: []interface{}{"my_namespace", "event1", from},
		},
		{
			name: "count query with time filter using And operator",
			query: queryMeterTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:       "meter1",
					EventType: "event1",
				},
				Params: streaming.QueryParamsV2{
					Filter: &streaming.Filter{
						Time: &filter.FilterTime{
							And: &[]filter.FilterTime{
								{
									Gte: &from,
								},
								{
									Lte: &to,
								},
							},
						},
					},
				},
			},
			wantSQL:  "SELECT count() AS total FROM openmeter.om_events WHERE namespace = ? AND type = ? AND (om_events.time >= ? AND om_events.time <= ?)",
			wantArgs: []interface{}{"my_namespace", "event1", from, to},
		},
		{
			name: "count query with subject and time filters",
			query: queryMeterTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:       "meter1",
					EventType: "event1",
				},
				Params: streaming.QueryParamsV2{
					Filter: &streaming.Filter{
						Subject: &filter.FilterString{
							Eq: &subject,
						},
						Time: &filter.FilterTime{
							Gte: &from,
							Lte: &to,
						},
					},
				},
			},
			wantSQL:  "SELECT count() AS total FROM openmeter.om_events WHERE namespace = ? AND type = ? AND om_events.subject = ? AND om_events.time >= ?",
			wantArgs: []interface{}{"my_namespace", "event1", "subject1", from},
		},
		{
			name: "count query with meter event from time",
			query: queryMeterTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:       "meter1",
					EventType: "event1",
					EventFrom: &eventFrom,
				},
				Params: streaming.QueryParamsV2{},
			},
			wantSQL:  "SELECT count() AS total FROM openmeter.om_events WHERE namespace = ? AND type = ? AND om_events.time >= ?",
			wantArgs: []interface{}{"my_namespace", "event1", eventFrom},
		},
		{
			name: "count query with meter event from time and time filter",
			query: queryMeterTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:       "meter1",
					EventType: "event1",
					EventFrom: &eventFrom,
				},
				Params: streaming.QueryParamsV2{
					Filter: &streaming.Filter{
						Time: &filter.FilterTime{
							Gte: &from,
							Lte: &to,
						},
					},
				},
			},
			wantSQL:  "SELECT count() AS total FROM openmeter.om_events WHERE namespace = ? AND type = ? AND om_events.time >= ? AND om_events.time >= ?",
			wantArgs: []interface{}{"my_namespace", "event1", from, eventFrom},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, gotArgs := tt.query.toCountRowSQL()
			assert.Equal(t, tt.wantSQL, gotSQL)
			assert.Equal(t, tt.wantArgs, gotArgs)
		})
	}
}
