package sink

import (
	"context"

	"github.com/openmeterio/openmeter/internal/ingest/kafkaingest/serializer"
)

type Storage interface {
	BatchInsert(ctx context.Context, events []serializer.CloudEventsKafkaPayload) error
}
