package kafkaingest

import (
	"context"
	"log/slog"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/internal/ingest/kafkaingest"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
)

// Collector is a receiver of events that handles sending those events to a downstream Kafka broker.
type Collector = kafkaingest.Collector

func KafkaProducerGroup(ctx context.Context, producer *kafka.Producer, logger *slog.Logger) (execute func() error, interrupt func(error)) {
	return kafkaingest.KafkaProducerGroup(ctx, producer, logger)
}

func NewCollector(
	producer *kafka.Producer,
	serializer serializer.Serializer,
	namespacedTopicTemplate string,
	metricMeter metric.Meter,
) (*Collector, error) {
	return kafkaingest.NewCollector(producer, serializer, namespacedTopicTemplate, metricMeter)
}
