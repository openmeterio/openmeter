package kafkaingest

import (
	"context"
	"log/slog"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"

	"github.com/openmeterio/openmeter/internal/ingest/kafkaingest"
)

// Collector is a receiver of events that handles sending those events to a downstream Kafka broker.
type Collector = kafkaingest.Collector

func KafkaProducerGroup(ctx context.Context, producer *kafka.Producer, logger *slog.Logger) (execute func() error, interrupt func(error)) {
	return kafkaingest.KafkaProducerGroup(ctx, producer, logger)
}
