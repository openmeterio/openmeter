package summeterv1

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
				Database:     "openmeter",
				SumTableName: "om_events",
				Namespace:    "my_namespace",
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
				QueryParams: streaming.QueryParams{
					FilterSubject: []string{subject},
					From:          &from,
					To:            &to,
					GroupBy:       []string{"subject", "group1", "group2"},
					WindowSize:    &windowSize,
				},
			},
			wantSQL:  "SELECT tumbleStart(om_events.time, toIntervalHour(1), 'UTC') AS windowstart, tumbleEnd(om_events.time, toIntervalHour(1), 'UTC') AS windowend, SUM(om_events.value) AS value, om_events.subject, om_events.group_by_filters['group1'] as group1, om_events.group_by_filters['group2'] as group2 FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.subject IN (?) AND om_events.time >= ? AND om_events.time < ? GROUP BY windowstart, windowend, subject, group1, group2 ORDER BY windowstart",
			wantArgs: []interface{}{"my_namespace", "event1", []string{"subject1"}, from.Unix(), to.Unix()},
		},
		{
			name: "Aggregate all available data",
			query: queryMeter{
				Database:     "openmeter",
				SumTableName: "om_events",
				Namespace:    "my_namespace",
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
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, SUM(om_events.value) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ?",
			wantArgs: []interface{}{"my_namespace", "event1"},
		},
		{
			name: "Aggregate data from start",
			query: queryMeter{
				Database:     "openmeter",
				SumTableName: "om_events",
				Namespace:    "my_namespace",
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
				QueryParams: streaming.QueryParams{
					From: &from,
				},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, SUM(om_events.value) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.time >= ?",
			wantArgs: []interface{}{"my_namespace", "event1", from.Unix()},
		},
		{
			name: "Aggregate data between period",
			query: queryMeter{
				Database:     "openmeter",
				SumTableName: "om_events",
				Namespace:    "my_namespace",
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
				QueryParams: streaming.QueryParams{
					From: &from,
					To:   &to,
				},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, SUM(om_events.value) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.time >= ? AND om_events.time < ?",
			wantArgs: []interface{}{"my_namespace", "event1", from.Unix(), to.Unix()},
		},
		{
			name: "Aggregate data between period, groupped by window size",
			query: queryMeter{
				Database:     "openmeter",
				SumTableName: "om_events",
				Namespace:    "my_namespace",
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
				QueryParams: streaming.QueryParams{
					From:       &from,
					To:         &to,
					WindowSize: &windowSize,
				},
			},
			wantSQL:  "SELECT tumbleStart(om_events.time, toIntervalHour(1), 'UTC') AS windowstart, tumbleEnd(om_events.time, toIntervalHour(1), 'UTC') AS windowend, SUM(om_events.value) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.time >= ? AND om_events.time < ? GROUP BY windowstart, windowend ORDER BY windowstart",
			wantArgs: []interface{}{"my_namespace", "event1", from.Unix(), to.Unix()},
		},
		{
			name: "Aggregate data between period in a different timezone, groupped by window size",
			query: queryMeter{
				Database:     "openmeter",
				SumTableName: "om_events",
				Namespace:    "my_namespace",
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
				QueryParams: streaming.QueryParams{
					From:           &from,
					To:             &to,
					WindowSize:     &windowSize,
					WindowTimeZone: tz,
				},
			},
			wantSQL:  "SELECT tumbleStart(om_events.time, toIntervalHour(1), 'Asia/Shanghai') AS windowstart, tumbleEnd(om_events.time, toIntervalHour(1), 'Asia/Shanghai') AS windowend, SUM(om_events.value) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.time >= ? AND om_events.time < ? GROUP BY windowstart, windowend ORDER BY windowstart",
			wantArgs: []interface{}{"my_namespace", "event1", from.Unix(), to.Unix()},
		},
		{
			name: "Aggregate data for a single subject",
			query: queryMeter{
				Database:     "openmeter",
				SumTableName: "om_events",
				Namespace:    "my_namespace",
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
				QueryParams: streaming.QueryParams{
					FilterSubject: []string{subject},
					GroupBy:       []string{"subject"},
				},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, SUM(om_events.value) AS value, om_events.subject FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.subject IN (?) GROUP BY subject",
			wantArgs: []interface{}{"my_namespace", "event1", []string{"subject1"}},
		},
		{
			name: "Aggregate data for a single subject and group by additional fields",
			query: queryMeter{
				Database:     "openmeter",
				SumTableName: "om_events",
				Namespace:    "my_namespace",
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
				QueryParams: streaming.QueryParams{
					FilterSubject: []string{subject},
					GroupBy:       []string{"subject", "group1", "group2"},
				},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, SUM(om_events.value) AS value, om_events.subject, om_events.group_by_filters['group1'] as group1, om_events.group_by_filters['group2'] as group2 FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.subject IN (?) GROUP BY subject, group1, group2",
			wantArgs: []interface{}{"my_namespace", "event1", []string{"subject1"}},
		},
		{
			name: "Select customer ID",
			query: queryMeter{
				Database:     "openmeter",
				SumTableName: "om_events",
				Namespace:    "my_namespace",
				Meter: meter.Meter{
					Key:           "meter1",
					EventType:     "event1",
					Aggregation:   meter.MeterAggregationSum,
					ValueProperty: lo.ToPtr("$.value"),
				},
				QueryParams: streaming.QueryParams{
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
			},
			wantSQL:  "WITH map('subject1', 'customer1', 'subject2', 'customer2') as subject_to_customer_id SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, SUM(om_events.value) AS value, subject_to_customer_id[om_events.subject] AS customer_id FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.subject IN (?) GROUP BY customer_id",
			wantArgs: []interface{}{"my_namespace", "event1", []string{"subject1", "subject2"}},
		},
		{
			name: "Aggregate data with filtering for a single group and single value",
			query: queryMeter{
				Database:     "openmeter",
				SumTableName: "om_events",
				Namespace:    "my_namespace",
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
				QueryParams: streaming.QueryParams{
					FilterGroupBy: map[string]filter.FilterString{"g1": {Eq: lo.ToPtr("g1v1")}},
				},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, SUM(om_events.value) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.group_by_filters['g1'] = ?",
			wantArgs: []interface{}{"my_namespace", "event1", "g1v1"},
		},
		{
			name: "Aggregate data with filtering for a single group and multiple values",
			query: queryMeter{
				Database:     "openmeter",
				SumTableName: "om_events",
				Namespace:    "my_namespace",
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
				QueryParams: streaming.QueryParams{
					FilterGroupBy: map[string]filter.FilterString{"g1": {In: lo.ToPtr([]string{"g1v1", "g1v2"})}},
				},
			},
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, SUM(om_events.value) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.group_by_filters['g1'] IN (?)",
			wantArgs: []interface{}{"my_namespace", "event1", []string{"g1v1", "g1v2"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSql, gotArgs, err := tt.query.ToSQL()
			if err != nil {
				t.Error(err)
				return
			}

			assert.Equal(t, tt.wantSQL, gotSql)
			assert.Equal(t, tt.wantArgs, gotArgs)
		})
	}
}
