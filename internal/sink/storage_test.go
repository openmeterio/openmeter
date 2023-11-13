package sink_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/internal/sink"
)

func TestInsertEventsQuery(t *testing.T) {
	now := time.Now()

	query := sink.InsertEventsQuery{
		Database: "database",
		Messages: []sink.SinkMessage{
			{
				Namespace: "my_namespace",
				Serialized: &serializer.CloudEventsKafkaPayload{
					Id:      "1",
					Source:  "source",
					Subject: "subject-1",
					Time:    now.UnixMilli(),
					Type:    "api-calls",
					Data:    `{"duration_ms": 100, "method": "GET", "path": "/api/v1"}`,
				},
			},
			{
				Namespace: "my_namespace",
				Serialized: &serializer.CloudEventsKafkaPayload{
					Id:      "2",
					Source:  "source",
					Subject: "subject-2",
					Time:    now.UnixMilli(),
					Type:    "api-calls",
					Data:    `{"duration_ms": 80, "method": "GET", "path": "/api/v1"}`,
				},
			},
		},
	}

	sql, args, err := query.ToSQL()
	assert.NoError(t, err)
	assert.Equal(t, args, []interface{}{
		"my_namespace", "1", "api-calls", "source", "subject-1", now.UnixMilli(), `{"duration_ms": 100, "method": "GET", "path": "/api/v1"}`,
		"my_namespace", "2", "api-calls", "source", "subject-2", now.UnixMilli(), `{"duration_ms": 80, "method": "GET", "path": "/api/v1"}`,
	})
	assert.Equal(t, `INSERT INTO database.om_events (namespace, id, type, source, subject, time, data) VALUES (?, ?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?, ?)`, sql)

}
