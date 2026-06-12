package clickhouse

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/pagination/v2"
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

func TestListSubjectsV2(t *testing.T) {
	tests := []struct {
		name     string
		query    listSubjectsV2Query
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			name: "basic query",
			query: listSubjectsV2Query{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Params: streaming.ListSubjectsV2Params{
					Namespace: "my_namespace",
				},
			},
			wantSQL:  "SELECT subject FROM openmeter.om_events WHERE namespace = ? AND subject <> ? GROUP BY namespace, subject ORDER BY namespace, subject LIMIT ?",
			wantArgs: []interface{}{"my_namespace", "", 100},
		},
		{
			name: "query with key filter",
			query: listSubjectsV2Query{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Params: streaming.ListSubjectsV2Params{
					Namespace: "my_namespace",
					Key:       &filter.FilterString{Ilike: lo.ToPtr("%customer%")},
				},
			},
			wantSQL:  "SELECT subject FROM openmeter.om_events WHERE namespace = ? AND subject <> ? AND LOWER(subject) LIKE LOWER(?) GROUP BY namespace, subject ORDER BY namespace, subject LIMIT ?",
			wantArgs: []interface{}{"my_namespace", "", "%customer%", 100},
		},
		{
			name: "query with cursor and limit",
			query: listSubjectsV2Query{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Params: streaming.ListSubjectsV2Params{
					Namespace: "my_namespace",
					Cursor:    lo.ToPtr(pagination.NewCursor(time.Time{}, "customer-1")),
					Limit:     lo.ToPtr(10),
				},
			},
			wantSQL:  "SELECT subject FROM openmeter.om_events WHERE namespace = ? AND subject <> ? AND subject > ? GROUP BY namespace, subject ORDER BY namespace, subject LIMIT ?",
			wantArgs: []interface{}{"my_namespace", "", "customer-1", 10},
		},
		{
			name: "query with settings",
			query: listSubjectsV2Query{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Params: streaming.ListSubjectsV2Params{
					Namespace: "my_namespace",
				},
				QuerySettings: map[string]string{"max_execution_time": "60"},
			},
			wantSQL:  "SELECT subject FROM openmeter.om_events WHERE namespace = ? AND subject <> ? GROUP BY namespace, subject ORDER BY namespace, subject LIMIT ? SETTINGS max_execution_time = 60",
			wantArgs: []interface{}{"my_namespace", "", 100},
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

func TestCreateEventsSubjectsProjection(t *testing.T) {
	projection := createEventsSubjectsProjection{
		Database:        "openmeter",
		EventsTableName: "om_events",
	}

	assert.Equal(
		t,
		"ALTER TABLE openmeter.om_events ADD PROJECTION IF NOT EXISTS prj_namespace_subject (SELECT namespace, subject GROUP BY namespace, subject)",
		projection.toSQL(),
	)
}
