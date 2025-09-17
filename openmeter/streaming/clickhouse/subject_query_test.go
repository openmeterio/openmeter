package clickhouse

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/meter"
)

func TestListSubjects(t *testing.T) {
	from, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00.001Z")
	to, _ := time.Parse(time.RFC3339, "2023-01-02T00:00:00Z")

	tests := []struct {
		name     string
		query    listSubjectsQuery
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			name: "basic query",
			query: listSubjectsQuery{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
			},
			wantSQL:  "SELECT DISTINCT subject FROM openmeter.om_events WHERE namespace = ? ORDER BY subject",
			wantArgs: []interface{}{"my_namespace"},
		},
		{
			name: "query with meter",
			query: listSubjectsQuery{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: &meter.Meter{
					Key:         "meter1",
					EventType:   "event1",
					Aggregation: meter.MeterAggregationSum,
				},
			},
			wantSQL:  "SELECT DISTINCT subject FROM openmeter.om_events WHERE namespace = ? AND type = ? ORDER BY subject",
			wantArgs: []interface{}{"my_namespace", "event1"},
		},
		{
			name: "query with from time",
			query: listSubjectsQuery{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: &meter.Meter{
					Key:         "meter1",
					EventType:   "event1",
					Aggregation: meter.MeterAggregationSum,
				},
				From: &from,
			},
			wantSQL:  "SELECT DISTINCT subject FROM openmeter.om_events WHERE namespace = ? AND type = ? AND time >= ? ORDER BY subject",
			wantArgs: []interface{}{"my_namespace", "event1", from.Unix()},
		},
		{
			name: "query with to time",
			query: listSubjectsQuery{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Meter: &meter.Meter{
					Key:         "meter1",
					EventType:   "event1",
					Aggregation: meter.MeterAggregationSum,
				},
				From: &from,
				To:   &to,
			},
			wantSQL:  "SELECT DISTINCT subject FROM openmeter.om_events WHERE namespace = ? AND type = ? AND time >= ? AND time < ? ORDER BY subject",
			wantArgs: []interface{}{"my_namespace", "event1", from.Unix(), to.Unix()},
		},
		{
			name: "query with search",
			query: listSubjectsQuery{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Search:          lo.ToPtr("arch"),
			},
			wantSQL:  "SELECT DISTINCT subject FROM openmeter.om_events WHERE namespace = ? AND positionCaseInsensitive(subject, ?) > 0 ORDER BY subject",
			wantArgs: []interface{}{"my_namespace", "arch"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			gotSql, gotArgs := tt.query.toSQL()

			assert.Equal(t, tt.wantArgs, gotArgs)
			assert.Equal(t, tt.wantSQL, gotSql)
		})
	}
}
