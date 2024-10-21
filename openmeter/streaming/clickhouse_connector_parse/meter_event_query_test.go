package clickhouse_connector_parse

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/streaming"
)

func TestInsertMeterEventsQuery(t *testing.T) {
	now := time.Now()

	rawEvent := streaming.RawEvent{
		Namespace:  "my_namespace",
		ID:         "1",
		Source:     "source",
		Subject:    "subject-1",
		Time:       now,
		StoredAt:   now,
		IngestedAt: now,
		Type:       "api-calls",
		Data:       `{"duration_ms": 100, "method": "GET", "path": "/api/v1"}`,
	}

	query := InsertMeterEventsQuery{
		Database: "database",
		QuerySettings: map[string]string{
			"parallel_view_processing": "1",
			"max_insert_threads":       "2",
		},
		MeterEvents: []streaming.MeterEvent{
			{
				RawEvent: rawEvent,
				Meter:    "api_request_duration",
				Value:    100.0,
				GroupBy: map[string]string{
					"method": "GET",
					"path":   "/api/v1",
				},
			},
			{
				RawEvent: rawEvent,
				Meter:    "api_request_total",
				Value:    1.0,
				GroupBy: map[string]string{
					"method": "GET",
					"path":   "/api/v1",
				},
			},
		},
	}

	sql, args := query.ToSQL()

	assert.Equal(t, []interface{}{
		// First Meter Event
		"my_namespace", now, "api_request_duration", "subject-1", 100.0, "",
		map[string]string{"method": "GET", "path": "/api/v1"},
		"1", "source", "api-calls", now, now,
		// Second Meter Event
		"my_namespace", now, "api_request_total", "subject-1", 1.0, "",
		map[string]string{"method": "GET", "path": "/api/v1"},
		"1", "source", "api-calls", now, now,
	}, args)
	assert.Equal(t, `INSERT INTO database.om_meter_events (namespace, time, meter, subject, value, value_str, group_by, event_id, event_source, event_type, ingested_at, stored_at) SETTINGS parallel_view_processing = 1, max_insert_threads = 2 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, sql)
}
