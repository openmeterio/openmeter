package serializer

import (
	_ "embed"

	"github.com/openmeterio/openmeter/internal/ingest/kafkaingest/serializer"
)

type Serializer = serializer.Serializer

type CloudEventsKafkaPayload = serializer.CloudEventsKafkaPayload
