package raw_events

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/streaming"
)

func TestCreateEventsTable(t *testing.T) {
	tests := []struct {
		data createEventsTable
		want string
	}{
		{
			data: createEventsTable{
				Database:        "openmeter",
				EventsTableName: "om_events",
			},
			want: "CREATE TABLE IF NOT EXISTS openmeter.om_events (namespace String, validation_error String, id String, type LowCardinality(String), subject String, source String, time DateTime, data String, ingested_at DateTime, stored_at DateTime) ENGINE = MergeTree PARTITION BY toYYYYMM(time) ORDER BY (namespace, type, subject, toStartOfHour(time))",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			got := tt.data.toSQL()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestQueryEventsTable(t *testing.T) {
	subjectFilter := "customer-1"
	idFilter := "event-id-1"
	hasErrorTrue := true
	hasErrorFalse := false

	tests := []struct {
		query    queryEventsTable
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			query: queryEventsTable{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Limit:           100,
			},
			wantSQL:  "SELECT id, type, subject, source, time, data, validation_error, ingested_at, stored_at FROM openmeter.om_events WHERE namespace = ? ORDER BY time DESC LIMIT 100",
			wantArgs: []interface{}{"my_namespace"},
		},
		{
			query: queryEventsTable{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Limit:           100,
				Subject:         &subjectFilter,
			},
			wantSQL:  "SELECT id, type, subject, source, time, data, validation_error, ingested_at, stored_at FROM openmeter.om_events WHERE namespace = ? AND subject = ? ORDER BY time DESC LIMIT 100",
			wantArgs: []interface{}{"my_namespace", subjectFilter},
		},
		{
			query: queryEventsTable{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Limit:           100,
				ID:              &idFilter,
			},
			wantSQL:  "SELECT id, type, subject, source, time, data, validation_error, ingested_at, stored_at FROM openmeter.om_events WHERE namespace = ? AND id LIKE ? ORDER BY time DESC LIMIT 100",
			wantArgs: []interface{}{"my_namespace", "%event-id-1%"},
		},
		{
			query: queryEventsTable{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Limit:           100,
				HasError:        &hasErrorTrue,
			},
			wantSQL:  "SELECT id, type, subject, source, time, data, validation_error, ingested_at, stored_at FROM openmeter.om_events WHERE namespace = ? AND notEmpty(validation_error) = 1 ORDER BY time DESC LIMIT 100",
			wantArgs: []interface{}{"my_namespace"},
		},
		{
			query: queryEventsTable{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				Limit:           100,
				HasError:        &hasErrorFalse,
			},
			wantSQL:  "SELECT id, type, subject, source, time, data, validation_error, ingested_at, stored_at FROM openmeter.om_events WHERE namespace = ? AND empty(validation_error) = 1 ORDER BY time DESC LIMIT 100",
			wantArgs: []interface{}{"my_namespace"},
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

func TestQueryEventsCount(t *testing.T) {
	from, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00.001Z")
	tests := []struct {
		query    queryCountEvents
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			query: queryCountEvents{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Namespace:       "my_namespace",
				From:            from,
			},
			wantSQL:  "SELECT count() as count, subject, notEmpty(validation_error) as is_error FROM openmeter.om_events WHERE namespace = ? AND time >= ? GROUP BY subject, is_error",
			wantArgs: []interface{}{"my_namespace", from.Unix()},
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

func TestInsertEventsQuery(t *testing.T) {
	now := time.Now()

	query := InsertEventsQuery{
		Database:        "database",
		EventsTableName: "om_events",
		Events: []streaming.RawEvent{
			{
				Namespace:  "my_namespace",
				ID:         "1",
				Source:     "source",
				Subject:    "subject-1",
				Time:       now,
				StoredAt:   now,
				IngestedAt: now,
				Type:       "api-calls",
				Data:       `{"duration_ms": 100, "method": "GET", "path": "/api/v1"}`,
			},
			{
				Namespace:  "my_namespace",
				ID:         "2",
				Source:     "source",
				Subject:    "subject-2",
				Time:       now,
				StoredAt:   now,
				IngestedAt: now,
				Type:       "api-calls",
				Data:       `{"duration_ms": 80, "method": "GET", "path": "/api/v1"}`,
			},
			{
				Namespace:       "my_namespace",
				ValidationError: "event data value cannot be parsed as float64: not a number",
				ID:              "3",
				Source:          "source",
				Subject:         "subject-2",
				Time:            now,
				StoredAt:        now,
				IngestedAt:      now,
				Type:            "api-calls",
				Data:            `{"duration_ms": "foo", "method": "GET", "path": "/api/v1"}`,
			},
		},
	}

	sql, args := query.ToSQL()

	assert.Equal(t, []interface{}{
		"my_namespace", "", "1", "api-calls", "source", "subject-1", now, `{"duration_ms": 100, "method": "GET", "path": "/api/v1"}`, now, now,
		"my_namespace", "", "2", "api-calls", "source", "subject-2", now, `{"duration_ms": 80, "method": "GET", "path": "/api/v1"}`, now, now,
		"my_namespace", "event data value cannot be parsed as float64: not a number", "3", "api-calls", "source", "subject-2", now, `{"duration_ms": "foo", "method": "GET", "path": "/api/v1"}`, now, now,
	}, args)
	assert.Equal(t, `INSERT INTO database.om_events (namespace, validation_error, id, type, source, subject, time, data, ingested_at, stored_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, sql)
}