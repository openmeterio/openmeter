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
			wantSQL:  "SELECT tumbleStart(om_events.time, toIntervalHour(1), 'UTC') AS windowstart, tumbleEnd(om_events.time, toIntervalHour(1), 'UTC') AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value')), null)) AS value, om_events.subject, JSON_VALUE(om_events.data, '$.group1') as group1, JSON_VALUE(om_events.data, '$.group2') as group2 FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.subject IN (?) AND om_events.time >= ? AND om_events.time < ? GROUP BY windowstart, windowend, subject, group1, group2 ORDER BY windowstart",
			wantArgs: []interface{}{"my_namespace", "event1", []string{"subject1"}, from.Unix(), to.Unix()},
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
			wantSQL:  "SELECT tumbleStart(om_events.time, toIntervalHour(1), 'UTC') AS windowstart, tumbleEnd(om_events.time, toIntervalHour(1), 'UTC') AS windowend, sum(toDecimal128OrNull(JSON_VALUE(om_events.data, '$.value'), 19)) AS value, om_events.subject, JSON_VALUE(om_events.data, '$.group1') as group1, JSON_VALUE(om_events.data, '$.group2') as group2 FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.subject IN (?) AND om_events.time >= ? AND om_events.time < ? GROUP BY windowstart, windowend, subject, group1, group2 ORDER BY windowstart",
			wantArgs: []interface{}{"my_namespace", "event1", []string{"subject1"}, from.Unix(), to.Unix()},
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
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value')), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ?",
			wantArgs: []interface{}{"my_namespace", "event1"},
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
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, toUInt64(count(*)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ?",
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
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, toUInt64(count(*)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ?",
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
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, uniqExact(JSON_VALUE(om_events.data, '$.value')) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ?",
			wantArgs: []interface{}{"my_namespace", "event1"},
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
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, uniqExact(JSON_VALUE(om_events.data, '$.value')) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ?",
			wantArgs: []interface{}{"my_namespace", "event1"},
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
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, argMax(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value')), null), om_events.time) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ?",
			wantArgs: []interface{}{"my_namespace", "event1"},
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
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value')), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.time >= ?",
			wantArgs: []interface{}{"my_namespace", "event1", from.Unix()},
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
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value')), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.time >= ? AND om_events.time < ?",
			wantArgs: []interface{}{"my_namespace", "event1", from.Unix(), to.Unix()},
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
			wantSQL:  "SELECT tumbleStart(om_events.time, toIntervalHour(1), 'UTC') AS windowstart, tumbleEnd(om_events.time, toIntervalHour(1), 'UTC') AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value')), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.time >= ? AND om_events.time < ? GROUP BY windowstart, windowend ORDER BY windowstart",
			wantArgs: []interface{}{"my_namespace", "event1", from.Unix(), to.Unix()},
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
			wantSQL:  "SELECT tumbleStart(om_events.time, toIntervalHour(1), 'Asia/Shanghai') AS windowstart, tumbleEnd(om_events.time, toIntervalHour(1), 'Asia/Shanghai') AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value')), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.time >= ? AND om_events.time < ? GROUP BY windowstart, windowend ORDER BY windowstart",
			wantArgs: []interface{}{"my_namespace", "event1", from.Unix(), to.Unix()},
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
			wantSQL:  "SELECT tumbleStart(om_events.time, toIntervalDay(1), 'UTC') AS windowstart, windowstart + toIntervalDay(1) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value')), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.time >= ? AND om_events.time < ? GROUP BY windowstart, windowend ORDER BY windowstart",
			wantArgs: []interface{}{"my_namespace", "event1", from.Unix(), to.Unix()},
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
			wantSQL:  "SELECT tumbleStart(om_events.time, toIntervalDay(1), 'Asia/Shanghai') AS windowstart, windowstart + toIntervalDay(1) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value')), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.time >= ? AND om_events.time < ? GROUP BY windowstart, windowend ORDER BY windowstart",
			wantArgs: []interface{}{"my_namespace", "event1", from.Unix(), to.Unix()},
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
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value')), null)) AS value, om_events.subject FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.subject IN (?) GROUP BY subject",
			wantArgs: []interface{}{"my_namespace", "event1", []string{"subject1"}},
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
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value')), null)) AS value, om_events.subject, JSON_VALUE(om_events.data, '$.group1') as group1, JSON_VALUE(om_events.data, '$.group2') as group2 FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.subject IN (?) GROUP BY subject, group1, group2",
			wantArgs: []interface{}{"my_namespace", "event1", []string{"subject1"}},
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
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value')), null)) AS value, om_events.subject FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.subject IN (?) GROUP BY subject",
			wantArgs: []interface{}{"my_namespace", "event1", []string{"subject1", "subject2"}},
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
			wantSQL:  "WITH map('subject1', 'customer1', 'subject2', 'customer2') as subject_to_customer_id SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value')), null)) AS value, subject_to_customer_id[om_events.subject] AS customer_id FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.subject IN (?) GROUP BY customer_id",
			wantArgs: []interface{}{"my_namespace", "event1", []string{"subject1", "subject2"}},
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
			wantSQL: "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value')), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.subject IN (?)",
			wantArgs: []interface{}{"my_namespace", "event1", []string{
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
			wantSQL:  "WITH map('subject1', 'customer1', 'subject2', 'customer1') as subject_to_customer_id SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value')), null)) AS value, subject_to_customer_id[om_events.subject] AS customer_id FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND om_events.subject IN (?) AND om_events.subject IN (?) GROUP BY customer_id",
			wantArgs: []interface{}{"my_namespace", "event1", []string{"subject1", "subject2"}, []string{"subject1"}},
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
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value')), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND JSON_VALUE(om_events.data, '$.group1') = ?",
			wantArgs: []interface{}{"my_namespace", "event1", "g1v1"},
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
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value')), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND JSON_VALUE(om_events.data, '$.group1') IN (?)",
			wantArgs: []interface{}{"my_namespace", "event1", []string{"g1v1", "g1v2"}},
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
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value')), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? AND JSON_VALUE(om_events.data, '$.group1') IN (?) AND JSON_VALUE(om_events.data, '$.group2') IN (?)",
			wantArgs: []interface{}{"my_namespace", "event1", []string{"g1v1", "g1v2"}, []string{"g2v1", "g2v2"}},
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
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value')), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ?",
			wantArgs: []interface{}{"my_namespace", "event1"},
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
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value')), null)) AS value FROM openmeter.om_events PREWHERE om_events.namespace = ? AND om_events.type = ? WHERE JSON_VALUE(om_events.data, '$.group1') IN (?) AND JSON_VALUE(om_events.data, '$.group2') IN (?) SETTINGS optimize_move_to_prewhere = 1, allow_reorder_prewhere_conditions = 1",
			wantArgs: []interface{}{"my_namespace", "event1", []string{"g1v1", "g1v2"}, []string{"g2v1", "g2v2"}},
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
			wantSQL:  "SELECT tumbleStart(min(om_events.time), toIntervalMinute(1)) AS windowstart, tumbleEnd(max(om_events.time), toIntervalMinute(1)) AS windowend, sum(ifNotFinite(toFloat64OrNull(JSON_VALUE(om_events.data, '$.value')), null)) AS value FROM openmeter.om_events WHERE om_events.namespace = ? AND om_events.type = ? SETTINGS foo = 1",
			wantArgs: []interface{}{"my_namespace", "event1"},
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
