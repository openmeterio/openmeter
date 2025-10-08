package clickhouse

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/meter"
)

func TestListGroupByValues(t *testing.T) {
	from, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00.001Z")
	to, _ := time.Parse(time.RFC3339, "2023-01-02T00:00:00Z")

	tests := []struct {
		name     string
		query    listGroupByValuesQuery
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			name: "basic query",
			query: listGroupByValuesQuery{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:         "meter1",
					EventType:   "event1",
					Aggregation: meter.MeterAggregationSum,
					GroupBy: map[string]string{
						"group1": "$.group1",
					},
				},
				GroupByKey: "group1",
			},
			wantSQL:  "SELECT DISTINCT JSON_VALUE(om_events.data, ?) AS group_by_values FROM openmeter.om_events WHERE namespace = ? AND type = ? ORDER BY group_by_values",
			wantArgs: []interface{}{"$.group1", "my_namespace", "event1"},
		},
		{
			name: "query with meter",
			query: listGroupByValuesQuery{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:         "meter1",
					EventType:   "event1",
					Aggregation: meter.MeterAggregationSum,
					GroupBy: map[string]string{
						"group1": "$.group1",
					},
				},
				GroupByKey: "group1",
			},
			wantSQL:  "SELECT DISTINCT JSON_VALUE(om_events.data, ?) AS group_by_values FROM openmeter.om_events WHERE namespace = ? AND type = ? ORDER BY group_by_values",
			wantArgs: []interface{}{"$.group1", "my_namespace", "event1"},
		},
		{
			name: "query with from time",
			query: listGroupByValuesQuery{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:         "meter1",
					EventType:   "event1",
					Aggregation: meter.MeterAggregationSum,
					GroupBy: map[string]string{
						"group1": "$.group1",
					},
				},
				GroupByKey: "group1",
				From:       &from,
			},
			wantSQL:  "SELECT DISTINCT JSON_VALUE(om_events.data, ?) AS group_by_values FROM openmeter.om_events WHERE namespace = ? AND type = ? AND time >= ? ORDER BY group_by_values",
			wantArgs: []interface{}{"$.group1", "my_namespace", "event1", from.Unix()},
		},
		{
			name: "query with to time",
			query: listGroupByValuesQuery{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: meter.Meter{
					Key:         "meter1",
					EventType:   "event1",
					Aggregation: meter.MeterAggregationSum,
					GroupBy: map[string]string{
						"group1": "$.group1",
					},
				},
				From:       &from,
				To:         &to,
				GroupByKey: "group1",
			},
			wantSQL:  "SELECT DISTINCT JSON_VALUE(om_events.data, ?) AS group_by_values FROM openmeter.om_events WHERE namespace = ? AND type = ? AND time >= ? AND time < ? ORDER BY group_by_values",
			wantArgs: []interface{}{"$.group1", "my_namespace", "event1", from.Unix(), to.Unix()},
		},
		{
			name: "query with search",
			query: listGroupByValuesQuery{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Search:          lo.ToPtr("arch"),
				GroupByKey:      "group1",
				Meter: meter.Meter{
					Key:         "meter1",
					EventType:   "event1",
					Aggregation: meter.MeterAggregationSum,
					GroupBy: map[string]string{
						"group1": "$.group1",
					},
				},
			},
			wantSQL:  "SELECT DISTINCT JSON_VALUE(om_events.data, ?) AS group_by_values FROM openmeter.om_events WHERE namespace = ? AND type = ? AND positionCaseInsensitive(group_by_values, ?) > 0 ORDER BY group_by_values",
			wantArgs: []interface{}{"$.group1", "my_namespace", "event1", "arch"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSql, gotArgs := tt.query.toSQL()

			assert.Equal(t, tt.wantArgs, gotArgs)
			assert.Equal(t, tt.wantSQL, gotSql)
		})
	}
}
