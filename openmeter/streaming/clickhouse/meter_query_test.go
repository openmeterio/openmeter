package clickhouse

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestQueryMeter(t *testing.T) {
	subject := "subject1"
	from, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00.001Z")
	to, _ := time.Parse(time.RFC3339, "2023-01-02T00:00:00Z")
	storedAtOffset, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00.001Z")
	tz, _ := time.LoadLocation("Asia/Shanghai")
	windowSize := meter.WindowSizeHour

	tests := []struct {
		name     string
		query    queryMeter
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			name: "basic query",
			query: queryMeter{
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
				FilterSubject: []string{subject},
				From:          &from,
				To:            &to,
				GroupBy:       []string{"subject", "group1", "group2"},
				WindowSize:    &windowSize,
			},
			wantSQL:  "SELECT tumbleStart(om_events.time, toIntervalHour(1), 'UTC') AS windowstart, tumbleEnd(om_events.time, toIntervalHour(1), 'UTC') AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, ?)), null)) AS value, om_events.subject, JSON_VALUE(om_events.data, ?) as group1, JSON_VALUE(om_events.data, ?) as group2 FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.subject IN (?) AND om_events.time >= ? AND om_events.time < ? GROUP BY windowstart, windowend, subject, group1, group2 ORDER BY windowstart",
			wantArgs: []interface{}{"$.value", "$.group1", "$.group2", "my_namespace", "event1", []string{"subject1"}, from.Unix(), to.Unix()},
		},
		{
			name: "basic query with decimal precision",
			query: queryMeter{
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
				FilterSubject:          []string{subject},
				From:                   &from,
				To:                     &to,
				GroupBy:                []string{"subject", "group1", "group2"},
				WindowSize:             &windowSize,
				EnableDecimalPrecision: true,
			},
			wantSQL:  "SELECT tumbleStart(om_events.time, toIntervalHour(1), 'UTC') AS windowstart, tumbleEnd(om_events.time, toIntervalHour(1), 'UTC') AS windowend, sum(toDecimal128OrNull(nullIf(JSON_VALUE(om_events.data, ?), 'null'), 19)) AS value, om_events.subject, JSON_VALUE(om_events.data, ?) as group1, JSON_VALUE(om_events.data, ?) as group2 FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.subject IN (?) AND om_events.time >= ? AND om_events.time < ? GROUP BY windowstart, windowend, subject, group1, group2 ORDER BY windowstart",
			wantArgs: []interface{}{"$.value", "$.group1", "$.group2", "my_namespace", "event1", []string{"subject1"}, from.Unix(), to.Unix()},
		},
		{
			name: "basic query with decimal stored at offset",
			query: queryMeter{
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
				FilterStoredAt: &filter.FilterTimeUnix{
					FilterTime: filter.FilterTime{
						Lt: &storedAtOffset,
					},
				},
				FilterSubject:          []string{subject},
				From:                   &from,
				To:                     &to,
				GroupBy:                []string{"subject", "group1", "group2"},
				WindowSize:             &windowSize,
				EnableDecimalPrecision: true,
			},
			wantSQL:  "SELECT tumbleStart(om_events.time, toIntervalHour(1), 'UTC') AS windowstart, tumbleEnd(om_events.time, toIntervalHour(1), 'UTC') AS windowend, sum(toDecimal128OrNull(nullIf(JSON_VALUE(om_events.data, ?), 'null'), 19)) AS value, om_events.subject, JSON_VALUE(om_events.data, ?) as group1, JSON_VALUE(om_events.data, ?) as group2 FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.subject IN (?) AND om_events.time >= ? AND om_events.time < ? AND om_events.stored_at < ? GROUP BY windowstart, windowend, subject, group1, group2 ORDER BY windowstart",
			wantArgs: []interface{}{"$.value", "$.group1", "$.group2", "my_namespace", "event1", []string{"subject1"}, from.Unix(), to.Unix(), storedAtOffset.Unix()},
		},
		{
			name: "Aggregate all available data",
			query: queryMeter{
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
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, ?)), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ?",
			wantArgs: []interface{}{"$.value", "my_namespace", "event1"},
		},
		{
			name: "Aggregate with count aggregation",
			query: queryMeter{
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
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, count(*) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ?",
			wantArgs: []interface{}{"my_namespace", "event1"},
		},
		{
			name: "Aggregate with count aggregation with decimal precision",
			query: queryMeter{
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
				EnableDecimalPrecision: true,
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, count(*) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ?",
			wantArgs: []interface{}{"my_namespace", "event1"},
		},
		{
			name: "Aggregate with unique count aggregation",
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationUniqueCount,
					ValueProperty: lo.ToPtr("$.value"),
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, uniqExact(nullIf(JSON_VALUE(om_events.data, ?), 'null')) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ?",
			wantArgs: []interface{}{"$.value", "my_namespace", "event1"},
		},
		{
			name: "Aggregate with unique count aggregation with decimal precision",
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationUniqueCount,
					ValueProperty: lo.ToPtr("$.value"),
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
				EnableDecimalPrecision: true,
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, uniqExact(nullIf(JSON_VALUE(om_events.data, ?), 'null')) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ?",
			wantArgs: []interface{}{"$.value", "my_namespace", "event1"},
		},
		{
			name: "Aggregate with AVG aggregation",
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationAvg,
					ValueProperty: lo.ToPtr("$.value"),
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, avg(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, ?)), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ?",
			wantArgs: []interface{}{"$.value", "my_namespace", "event1"},
		},
		{
			name: "Aggregate with AVG aggregation with decimal precision",
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationAvg,
					ValueProperty: lo.ToPtr("$.value"),
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
				EnableDecimalPrecision: true,
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, avg(toDecimal128OrNull(nullIf(JSON_VALUE(om_events.data, ?), 'null'), 19)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ?",
			wantArgs: []interface{}{"$.value", "my_namespace", "event1"},
		},
		{
			name: "Aggregate with MIN aggregation",
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationMin,
					ValueProperty: lo.ToPtr("$.value"),
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, min(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, ?)), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ?",
			wantArgs: []interface{}{"$.value", "my_namespace", "event1"},
		},
		{
			name: "Aggregate with MIN aggregation with decimal precision",
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationMin,
					ValueProperty: lo.ToPtr("$.value"),
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
				EnableDecimalPrecision: true,
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, min(toDecimal128OrNull(nullIf(JSON_VALUE(om_events.data, ?), 'null'), 19)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ?",
			wantArgs: []interface{}{"$.value", "my_namespace", "event1"},
		},
		{
			name: "Aggregate with MAX aggregation",
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationMax,
					ValueProperty: lo.ToPtr("$.value"),
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, max(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, ?)), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ?",
			wantArgs: []interface{}{"$.value", "my_namespace", "event1"},
		},
		{
			name: "Aggregate with MAX aggregation with decimal precision",
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationMax,
					ValueProperty: lo.ToPtr("$.value"),
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
				EnableDecimalPrecision: true,
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, max(toDecimal128OrNull(nullIf(JSON_VALUE(om_events.data, ?), 'null'), 19)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ?",
			wantArgs: []interface{}{"$.value", "my_namespace", "event1"},
		},
		{
			name: "Aggregate with LATEST aggregation",
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationLatest,
					ValueProperty: lo.ToPtr("$.value"),
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, argMax(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, ?)), null), om_events.time) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ?",
			wantArgs: []interface{}{"$.value", "my_namespace", "event1"},
		},
		{
			name: "Aggregate with LATEST aggregation with decimal precision",
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationLatest,
					ValueProperty: lo.ToPtr("$.value"),
					GroupBy: map[string]string{
						"group1": "$.group1",
						"group2": "$.group2",
					},
				},
				EnableDecimalPrecision: true,
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, argMax(toDecimal128OrNull(nullIf(JSON_VALUE(om_events.data, ?), 'null'), 19), om_events.time) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ?",
			wantArgs: []interface{}{"$.value", "my_namespace", "event1"},
		},
		{
			name: "Aggregate data from start",
			query: queryMeter{
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
				From: &from,
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, ?)), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.time >= ?",
			wantArgs: []interface{}{"$.value", "my_namespace", "event1", from.Unix()},
		},
		{
			name: "Aggregate data between period",
			query: queryMeter{
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
				From: &from,
				To:   &to,
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, ?)), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.time >= ? AND om_events.time < ?",
			wantArgs: []interface{}{"$.value", "my_namespace", "event1", from.Unix(), to.Unix()},
		},
		{
			name: "Aggregate data between period, groupped by window size",
			query: queryMeter{
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
				From:       &from,
				To:         &to,
				WindowSize: &windowSize,
			},
			wantSQL:  "SELECT tumbleStart(om_events.time, toIntervalHour(1), 'UTC') AS windowstart, tumbleEnd(om_events.time, toIntervalHour(1), 'UTC') AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, ?)), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.time >= ? AND om_events.time < ? GROUP BY windowstart, windowend ORDER BY windowstart",
			wantArgs: []interface{}{"$.value", "my_namespace", "event1", from.Unix(), to.Unix()},
		},
		{
			name: "Aggregate data between period in a different timezone, groupped by window size",
			query: queryMeter{
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
				From:           &from,
				To:             &to,
				WindowSize:     &windowSize,
				WindowTimeZone: tz,
			},
			wantSQL:  "SELECT tumbleStart(om_events.time, toIntervalHour(1), 'Asia/Shanghai') AS windowstart, tumbleEnd(om_events.time, toIntervalHour(1), 'Asia/Shanghai') AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, ?)), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.time >= ? AND om_events.time < ? GROUP BY windowstart, windowend ORDER BY windowstart",
			wantArgs: []interface{}{"$.value", "my_namespace", "event1", from.Unix(), to.Unix()},
		},
		{
			name: "Aggregate data between period, groupped by DAY window size",
			query: queryMeter{
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
				From:       &from,
				To:         &to,
				WindowSize: lo.ToPtr(meter.WindowSizeDay),
			},
			wantSQL:  "SELECT tumbleStart(om_events.time, toIntervalDay(1), 'UTC') AS windowstart, windowstart + toIntervalDay(1) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, ?)), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.time >= ? AND om_events.time < ? GROUP BY windowstart, windowend ORDER BY windowstart",
			wantArgs: []interface{}{"$.value", "my_namespace", "event1", from.Unix(), to.Unix()},
		},
		{
			name: "Aggregate data between period in a different timezone, groupped by DAY window size",
			query: queryMeter{
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
				From:           &from,
				To:             &to,
				WindowSize:     lo.ToPtr(meter.WindowSizeDay),
				WindowTimeZone: tz,
			},
			wantSQL:  "SELECT tumbleStart(om_events.time, toIntervalDay(1), 'Asia/Shanghai') AS windowstart, windowstart + toIntervalDay(1) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, ?)), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.time >= ? AND om_events.time < ? GROUP BY windowstart, windowend ORDER BY windowstart",
			wantArgs: []interface{}{"$.value", "my_namespace", "event1", from.Unix(), to.Unix()},
		},
		{
			name: "Aggregate data for a single subject",
			query: queryMeter{
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
				FilterSubject: []string{subject},
				GroupBy:       []string{"subject"},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, ?)), null)) AS value, om_events.subject FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.subject IN (?) GROUP BY subject",
			wantArgs: []interface{}{"$.value", "my_namespace", "event1", []string{"subject1"}},
		},
		{
			name: "Aggregate data for a single subject and group by additional fields",
			query: queryMeter{
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
				FilterSubject: []string{subject},
				GroupBy:       []string{"subject", "group1", "group2"},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, ?)), null)) AS value, om_events.subject, JSON_VALUE(om_events.data, ?) as group1, JSON_VALUE(om_events.data, ?) as group2 FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.subject IN (?) GROUP BY subject, group1, group2",
			wantArgs: []interface{}{"$.value", "$.group1", "$.group2", "my_namespace", "event1", []string{"subject1"}},
		},
		{
			name: "Aggregate data for a multiple subjects",
			query: queryMeter{
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
				FilterSubject: []string{subject, "subject2"},
				GroupBy:       []string{"subject"},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, ?)), null)) AS value, om_events.subject FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.subject IN (?) GROUP BY subject",
			wantArgs: []interface{}{"$.value", "my_namespace", "event1", []string{"subject1", "subject2"}},
		},
		{
			name: "Select customer ID",
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationSum,
					ValueProperty: lo.ToPtr("$.value"),
				},
				FilterCustomer: []streaming.Customer{
					customer.Customer{
						ManagedResource: models.ManagedResource{
							NamespacedModel: models.NamespacedModel{
								Namespace: "my_namespace",
							},
							ID: "customer1",
						},
						UsageAttribution: &customer.CustomerUsageAttribution{
							SubjectKeys: []string{"subject1"},
						},
					},
					customer.Customer{
						ManagedResource: models.ManagedResource{
							NamespacedModel: models.NamespacedModel{
								Namespace: "my_namespace",
							},
							ID: "customer2",
						},
						UsageAttribution: &customer.CustomerUsageAttribution{
							SubjectKeys: []string{"subject2"},
						},
					},
				},
				GroupBy: []string{"customer_id"},
			},
			wantSQL: "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, ?)), null)) AS value, mapFromArrays(?, ?)[om_events.subject] AS customer_id FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.subject IN (?) GROUP BY customer_id",
			wantArgs: []interface{}{
				"$.value",
				[]string{"subject1", "subject2"},
				[]string{"customer1", "customer2"},
				"my_namespace",
				"event1",
				[]string{"subject1", "subject2"},
			},
		},
		{
			name: "Filter by customer ID without group by",
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationSum,
					ValueProperty: lo.ToPtr("$.value"),
				},
				FilterCustomer: []streaming.Customer{
					customer.Customer{
						ManagedResource: models.ManagedResource{
							NamespacedModel: models.NamespacedModel{
								Namespace: "my_namespace",
							},
							ID: "customer1",
						},
						Key: lo.ToPtr("customer-key-1"),
						UsageAttribution: &customer.CustomerUsageAttribution{
							SubjectKeys: []string{"subject1"},
						},
					},
					customer.Customer{
						ManagedResource: models.ManagedResource{
							NamespacedModel: models.NamespacedModel{
								Namespace: "my_namespace",
							},
							ID: "customer2",
						},
						UsageAttribution: &customer.CustomerUsageAttribution{
							SubjectKeys: []string{"subject2"},
						},
					},
				},
			},
			wantSQL: "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, ?)), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.subject IN (?)",
			wantArgs: []interface{}{"$.value", "my_namespace", "event1", []string{
				// Only the first customer has a key
				"customer-key-1",
				// Usage attribution subjects of the first customer
				"subject1",
				// Usage attribution subjects of the second customer
				"subject2",
			}},
		},
		{ // Filter by both customer and subject
			name: "Filter by both customer and subject",
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationSum,
					ValueProperty: lo.ToPtr("$.value"),
				},
				FilterCustomer: []streaming.Customer{
					customer.Customer{
						ManagedResource: models.ManagedResource{
							NamespacedModel: models.NamespacedModel{
								Namespace: "my_namespace",
							},
							ID: "customer1",
						},
						UsageAttribution: &customer.CustomerUsageAttribution{
							SubjectKeys: []string{"subject1", "subject2"},
						},
					},
				},
				FilterSubject: []string{"subject1"},
				GroupBy:       []string{"customer_id"},
			},
			wantSQL: "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, ?)), null)) AS value, mapFromArrays(?, ?)[om_events.subject] AS customer_id FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.subject IN (?) AND om_events.subject IN (?) GROUP BY customer_id",
			wantArgs: []interface{}{
				"$.value",
				[]string{"subject1", "subject2"},
				[]string{"customer1", "customer1"},
				"my_namespace",
				"event1",
				[]string{"subject1", "subject2"},
				[]string{"subject1"},
			},
		},
		{
			name: "Aggregate data with filtering for a single group and single value",
			query: queryMeter{
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
				FilterGroupBy: map[string]filter.FilterString{"g1": {Eq: lo.ToPtr("g1v1")}},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, ?)), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND JSON_VALUE(om_events.data, ?) = ?",
			wantArgs: []interface{}{"$.value", "my_namespace", "event1", "$.group1", "g1v1"},
		},
		{
			name: "Aggregate data with filtering for a single group and multiple values",
			query: queryMeter{
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
				FilterGroupBy: map[string]filter.FilterString{"g1": {In: lo.ToPtr([]string{"g1v1", "g1v2"})}},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, ?)), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND JSON_VALUE(om_events.data, ?) IN (?)",
			wantArgs: []interface{}{"$.value", "my_namespace", "event1", "$.group1", []string{"g1v1", "g1v2"}},
		},
		{
			name: "Aggregate data with filtering for multiple groups and multiple values",
			query: queryMeter{
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
				FilterGroupBy: map[string]filter.FilterString{
					"g1": {In: lo.ToPtr([]string{"g1v1", "g1v2"})},
					"g2": {In: lo.ToPtr([]string{"g2v1", "g2v2"})},
				},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, ?)), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND JSON_VALUE(om_events.data, ?) IN (?) AND JSON_VALUE(om_events.data, ?) IN (?)",
			wantArgs: []interface{}{"$.value", "my_namespace", "event1", "$.group1", []string{"g1v1", "g1v2"}, "$.group2", []string{"g2v1", "g2v2"}},
		},
		{
			name: "Aggregate all available data, prewhere enabled (should not move anything to prewhere)",
			query: queryMeter{
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
				EnablePrewhere: true,
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, ?)), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ?",
			wantArgs: []interface{}{"$.value", "my_namespace", "event1"},
		},
		{
			name: "Aggregate data with with filtering for multiple groups and multiple values prewhere enabled",
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				EnablePrewhere:  true,
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
				FilterGroupBy: map[string]filter.FilterString{
					"g1": {In: lo.ToPtr([]string{"g1v1", "g1v2"})},
					"g2": {In: lo.ToPtr([]string{"g2v1", "g2v2"})},
				},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, ?)), null)) AS value FROM openmeter.om_events PREWHERE om_events.namespace = ? AND om_events.type = ? WHERE JSON_VALUE(om_events.data, ?) IN (?) AND JSON_VALUE(om_events.data, ?) IN (?) SETTINGS optimize_move_to_prewhere = 1, allow_reorder_prewhere_conditions = 1",
			wantArgs: []interface{}{"$.value", "my_namespace", "event1", "$.group1", []string{"g1v1", "g1v2"}, "$.group2", []string{"g2v1", "g2v2"}},
		},
		{
			name: "Add query settings",
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationSum,
					ValueProperty: lo.ToPtr("$.value"),
				},
				QuerySettings: map[string]string{"foo": "1"},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, ?)), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? SETTINGS foo = 1",
			wantArgs: []interface{}{"$.value", "my_namespace", "event1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestQueryMeterOmitsEmptyLogicalFilterExpressions(t *testing.T) {
	baseSQL := "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, count(*) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ?"
	equalValue := "group-value"
	emptyAnd := []filter.FilterString{}
	emptyOr := []filter.FilterString{}
	mixedAnd := []filter.FilterString{
		{
			Or: &[]filter.FilterString{
				{},
				{Eq: &equalValue},
			},
		},
		{And: &emptyAnd},
		{},
	}

	tests := []struct {
		name     string
		filter   filter.FilterString
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			name:     "empty and",
			filter:   filter.FilterString{And: &emptyAnd},
			wantSQL:  baseSQL,
			wantArgs: []interface{}{"my_namespace", "event1"},
		},
		{
			name:     "empty or",
			filter:   filter.FilterString{Or: &emptyOr},
			wantSQL:  baseSQL,
			wantArgs: []interface{}{"my_namespace", "event1"},
		},
		{
			name:     "nested mixed operands",
			filter:   filter.FilterString{And: &mixedAnd},
			wantSQL:  baseSQL + " AND ((JSON_VALUE(om_events.data, ?) = ?))",
			wantArgs: []interface{}{"my_namespace", "event1", "$.group", "group-value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:         "meter1",
					EventType:   "event1",
					Aggregation: meter.MeterAggregationCount,
					GroupBy: map[string]string{
						"group": "$.group",
					},
				},
				FilterGroupBy: map[string]filter.FilterString{
					"group": tt.filter,
				},
			}

			gotSQL, gotArgs, err := query.toSQL()
			assert.NoError(t, err)
			assert.Equal(t, tt.wantSQL, gotSQL)
			assert.Equal(t, tt.wantArgs, gotArgs)
		})
	}
}

func TestQueryMeterBindsJSONPaths(t *testing.T) {
	valuePath := "$.value'), (SELECT sleep(3)), ('"
	groupPath := "$.group'), (SELECT sleep(3)), ('"

	query := queryMeter{
		Database:        "openmeter",
		EventsTableName: "om_events",
		Namespace:       "my_namespace",
		Meter: meter.Meter{
			Key:           "meter1",
			EventType:     "event1",
			Aggregation:   meter.MeterAggregationSum,
			ValueProperty: &valuePath,
			GroupBy: map[string]string{
				"group": groupPath,
			},
		},
		GroupBy: []string{"group"},
		FilterGroupBy: map[string]filter.FilterString{
			"group": {Eq: lo.ToPtr("group-value")},
		},
	}

	gotSQL, gotArgs, err := query.toSQL()
	assert.NoError(t, err)
	assert.NotContains(t, gotSQL, valuePath)
	assert.NotContains(t, gotSQL, groupPath)
	assert.Equal(t,
		[]interface{}{valuePath, groupPath, "my_namespace", "event1", groupPath, "group-value"},
		gotArgs,
	)
}

// The group-by key reaches identifier position and cannot be bound. Meter validation
// blocks such keys at write time; this guards a queryMeter from an untrusted source.
func TestQueryMeterRejectsInvalidGroupByKey(t *testing.T) {
	tests := []struct {
		name  string
		query queryMeter
	}{
		{
			name: "group by key not present on the meter",
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:         "meter1",
					EventType:   "event1",
					Aggregation: meter.MeterAggregationCount,
				},
				GroupBy: []string{"unknown"},
			},
		},
		{
			name: "group by key is not a safe identifier",
			query: queryMeter{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:         "meter1",
					EventType:   "event1",
					Aggregation: meter.MeterAggregationCount,
					GroupBy: map[string]string{
						"g) FROM system.numbers --": "$.value",
					},
				},
				GroupBy: []string{"g) FROM system.numbers --"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, _, err := tt.query.toSQL()

			assert.Error(t, err)
			assert.Empty(t, gotSQL)
		})
	}
}
