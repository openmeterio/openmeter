package kafkaingest

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/internal/ingest/kafkaingest/serializer"
)

// Collector is a receiver of events that handles sending those events to a downstream Kafka broker.
type Collector struct {
	Producer   *kafka.Producer
	Serializer serializer.Serializer

	// NamespacedTopicTemplate needs to contain at least one string parameter passed to fmt.Sprintf.
	// For example: "om_%s_events"
	NamespacedTopicTemplate string

	ingestEventCounter metric.Int64Counter
}

func NewCollector(
	producer *kafka.Producer,
	serializer serializer.Serializer,
	namespacedTopicTemplate string,
	metricMeter metric.Meter,
) (*Collector, error) {
	if producer == nil {
		return nil, fmt.Errorf("producer is required")
	}
	if serializer == nil {
		return nil, fmt.Errorf("serializer is required")
	}
	if namespacedTopicTemplate == "" {
		return nil, fmt.Errorf("namespaced topic template is required")
	}
	if metricMeter == nil {
		return nil, fmt.Errorf("metric meter is required")
	}

	// Initialize OTel metrics
	ingestEventCounter, err := metricMeter.Int64Counter(
		"ingest.events",
		metric.WithDescription("The number of events ingested"),
		metric.WithUnit("{event}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create events counter: %w", err)
	}

	return &Collector{
		Producer:                producer,
		Serializer:              serializer,
		NamespacedTopicTemplate: namespacedTopicTemplate,
		ingestEventCounter:      ingestEventCounter,
	}, nil
}

// Ingest produces an event to a Kafka topic.
func (s Collector) Ingest(ctx context.Context, namespace string, ev event.Event) error {
	topic := fmt.Sprintf(s.NamespacedTopicTemplate, namespace)
	key, err := s.Serializer.SerializeKey(topic, ev)
	if err != nil {
		return fmt.Errorf("serialize event key: %w", err)
	}

	value, err := s.Serializer.SerializeValue(topic, ev)
	if err != nil {
		return fmt.Errorf("serialize event value: %w", err)
	}

	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Timestamp:      ev.Time(),
		Headers: []kafka.Header{
			{Key: "namespace", Value: []byte(namespace)},
			{Key: "specversion", Value: []byte(ev.SpecVersion())},
		},
		Key:   key,
		Value: value,
	}

	err = s.Producer.Produce(msg, nil)
	if err != nil {
		return fmt.Errorf("producing kafka message: %w", err)
	}

	// Increment the ingest event counter metric
	namespaceAttr := attribute.String("namespace", namespace)
	s.ingestEventCounter.Add(ctx, 1, metric.WithAttributes(namespaceAttr))

	return nil
}

// Close closes the underlying producer.
func (s Collector) Close() {
	s.Producer.Flush(30 * 1000)
	s.Producer.Close()
}

func KafkaProducerGroup(ctx context.Context, producer *kafka.Producer, logger *slog.Logger) (execute func() error, interrupt func(error)) {
	ctx, cancel := context.WithCancel(ctx)
	return func() error {
			for {
				select {
				case e := <-producer.Events():
					switch ev := e.(type) {
					case *kafka.Message:
						// The message delivery report, indicating success or
						// permanent failure after retries have been exhausted.
						// Application level retries won't help since the client
						// is already configured to do that.
						m := ev
						if m.TopicPartition.Error != nil {
							logger.Error("kafka delivery failed", "error", m.TopicPartition.Error)
						} else {
							logger.Debug("kafka message delivered", "topic", *m.TopicPartition.Topic, "partition", m.TopicPartition.Partition, "offset", m.TopicPartition.Offset)
						}
					case kafka.Error:
						// Generic client instance-level errors, such as
						// broker connection failures, authentication issues, etc.
						//
						// These errors should generally be considered informational
						// as the underlying client will automatically try to
						// recover from any errors encountered, the application
						// does not need to take action on them.
						logger.Error("kafka error", "error", ev)
					}
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		},
		func(error) {
			cancel()
		}
}
