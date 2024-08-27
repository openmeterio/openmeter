package sink_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/openmeter/sink"
	sinkmodels "github.com/openmeterio/openmeter/openmeter/sink/models"
)

func TestInsertEventsQuery(t *testing.T) {
	now := time.Now()

	query := sink.InsertEventsQuery{
		Database: "database",
		Messages: []sinkmodels.SinkMessage{
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
			{
				Namespace: "my_namespace",
				Status: sinkmodels.ProcessingStatus{
					State: sinkmodels.INVALID,
					Error: errors.New("event data value cannot be parsed as float64: not a number"),
				},
				Serialized: &serializer.CloudEventsKafkaPayload{
					Id:      "3",
					Source:  "source",
					Subject: "subject-2",
					Time:    now.UnixMilli(),
					Type:    "api-calls",
					Data:    `{"duration_ms": "foo", "method": "GET", "path": "/api/v1"}`,
				},
			},
		},
	}

	sql, args, err := query.ToSQL()
	assert.NoError(t, err)

	// Remove the ingested_at and stored_at fields from the args to make the comparison easier
	argsWithoutIngestTimes := []interface{}{}
	for idx, arg := range args {
		// Skip the ingested_at and stored_at fields at indexes 8 and 9
		if idx != 0 && (idx%8 == 0 || idx%9 == 0) {
			continue
		}
		argsWithoutIngestTimes = append(argsWithoutIngestTimes, arg)
	}

	assert.Equal(t, []interface{}{
		"my_namespace", "", "1", "api-calls", "source", "subject-1", now.UnixMilli(), `{"duration_ms": 100, "method": "GET", "path": "/api/v1"}`,
		"my_namespace", "", "2", "api-calls", "source", "subject-2", now.UnixMilli(), `{"duration_ms": 80, "method": "GET", "path": "/api/v1"}`,
		"my_namespace", "event data value cannot be parsed as float64: not a number", "3", "api-calls", "source", "subject-2", now.UnixMilli(), `{"duration_ms": "foo", "method": "GET", "path": "/api/v1"}`,
	}, argsWithoutIngestTimes)
	assert.Equal(t, `INSERT INTO database.om_events (namespace, validation_error, id, type, source, subject, time, data, ingested_at, stored_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?, ?, ?, ?)`, sql)
}
