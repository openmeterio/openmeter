package kafkaingest

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/topicresolver"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
	kafkametrics "github.com/openmeterio/openmeter/pkg/kafka/metrics"
	kafkastats "github.com/openmeterio/openmeter/pkg/kafka/metrics/stats"
	"github.com/openmeterio/openmeter/pkg/otelx"
)

const (
	HeaderKeyNamespace = "namespace"
)

// Collector is a receiver of events that handles sending those events to a downstream Kafka broker.
type Collector struct {
	Producer         *kafka.Producer
	Serializer       serializer.Serializer
	TopicResolver    topicresolver.Resolver
	TopicProvisioner pkgkafka.TopicProvisioner
	TopicPartitions  int

	Logger *slog.Logger
	Tracer trace.Tracer
}

func NewCollector(
	producer *kafka.Producer,
	serializer serializer.Serializer,
	resolver topicresolver.Resolver,
	provisioner pkgkafka.TopicProvisioner,
	partitions int,
	logger *slog.Logger,
	tracer trace.Tracer,
) (*Collector, error) {
	if producer == nil {
		return nil, fmt.Errorf("producer is required")
	}
	if serializer == nil {
		return nil, fmt.Errorf("serializer is required")
	}
	if resolver == nil {
		return nil, fmt.Errorf("topic name resolver is required")
	}

	if provisioner == nil {
		return nil, fmt.Errorf("topic provisioner is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if tracer == nil {
		return nil, fmt.Errorf("tracer is required")
	}

	return &Collector{
		Producer:         producer,
		Serializer:       serializer,
		TopicResolver:    resolver,
		TopicProvisioner: provisioner,
		TopicPartitions:  partitions,
		Logger:           logger,
		Tracer:           tracer,
	}, nil
}

// Ingest produces an event to a Kafka topic.
func (s Collector) Ingest(ctx context.Context, namespace string, ev event.Event) error {
	var err error

	ctx, span := s.Tracer.Start(ctx, "openmeter.ingest.process.event", trace.WithAttributes(
		attribute.String("event.id", ev.ID()),
		attribute.String("event.namespace", namespace),
	))
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(otelcodes.Error, err.Error())
		}

		span.End()
	}()

	span.AddEvent("resolved namespace to kafka topic")
	topicName, err := s.TopicResolver.Resolve(ctx, namespace)
	if err != nil {
		err = fmt.Errorf("failed to resolve namespace to topic name: %w", err)
		return err
	}
	span.SetAttributes(semconv.MessagingDestinationName(topicName))

	// Make sure topic is provisioned
	span.AddEvent("provisioning kafka topic")
	err = s.TopicProvisioner.Provision(ctx, pkgkafka.TopicConfig{
		Name:       topicName,
		Partitions: s.TopicPartitions,
	})
	if err != nil {
		err = fmt.Errorf("failed to provision topic: %w", err)
		return err
	}

	key, err := s.Serializer.SerializeKey(topicName, ev)
	if err != nil {
		err = fmt.Errorf("serialize event key: %w", err)
		return err
	}

	value, err := s.Serializer.SerializeValue(topicName, ev)
	if err != nil {
		err = fmt.Errorf("serialize event value: %w", err)
		return err
	}

	spanCtx, err := otelx.SerializeSpanContext(span.SpanContext())
	if err != nil {
		s.Logger.WarnContext(ctx, "failed to serialize span context", "error", err)
	}

	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topicName, Partition: kafka.PartitionAny},
		Timestamp:      ev.Time(),
		Headers: []kafka.Header{
			{Key: HeaderKeyNamespace, Value: []byte(namespace)},
			{Key: "specversion", Value: []byte(ev.SpecVersion())},
			{Key: "ingested_at", Value: []byte(time.Now().UTC().Format(time.RFC3339))},
			{Key: otelx.OTelSpanContextKey, Value: spanCtx},
		},
		Key:   key,
		Value: value,
	}

	span.AddEvent("publishing event to kafka topic")
	err = s.Producer.Produce(msg, nil)
	if err != nil {
		err = fmt.Errorf("producing kafka message: %w", err)
		return err
	}

	return nil
}

// Close closes the underlying producer.
func (s Collector) Close() {
	s.Producer.Flush(30 * 1000)
	s.Producer.Close()
}

func KafkaProducerGroup(ctx context.Context, producer *kafka.Producer, logger *slog.Logger, kafkaMetrics *kafkametrics.Metrics) (execute func() error, interrupt func(error)) {
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
							logger.ErrorContext(ctx, "kafka delivery failed", "error", m.TopicPartition.Error)
						} else {
							logger.DebugContext(ctx, "kafka message delivered", "topic", *m.TopicPartition.Topic, "partition", m.TopicPartition.Partition, "offset", m.TopicPartition.Offset)
						}
					case *kafka.Stats:
						// Report Kafka client metrics
						if kafkaMetrics == nil {
							continue
						}

						go func() {
							var stats kafkastats.Stats

							if err := json.Unmarshal([]byte(e.String()), &stats); err != nil {
								logger.WarnContext(ctx, "failed to unmarshal Kafka client stats", slog.String("err", err.Error()))
							}

							ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
							defer cancel()

							kafkaMetrics.Add(ctx, &stats)
						}()
					case kafka.Error:
						// Generic client instance-level errors, such as
						// broker connection failures, authentication issues, etc.
						//
						// These errors should generally be considered informational
						// as the underlying client will automatically try to
						// recover from any errors encountered, the application
						// does not need to take action on them.
						attrs := []any{
							slog.Int("code", int(ev.Code())),
							slog.String("error", ev.Error()),
						}

						// Log Kafka client "local" errors on warning level as those are mostly informational and the client is
						// able to handle/recover from them automatically.
						// See: https://github.com/confluentinc/librdkafka/blob/master/src/rdkafka.h#L415
						if ev.Code() <= -100 {
							logger.WarnContext(ctx, "kafka local error", attrs...)
						} else {
							logger.ErrorContext(ctx, "kafka broker error", attrs...)
						}
					}
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		},
		func(error) {
			logger.DebugContext(ctx, "kafka ingest producer shutting down...")

			cancel()
		}
}
