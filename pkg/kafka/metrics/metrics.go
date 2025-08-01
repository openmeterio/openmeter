package metrics

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/pkg/kafka/metrics/internal"
	"github.com/openmeterio/openmeter/pkg/kafka/metrics/stats"
)

// Metrics stores set of Kafka client related metrics
// See: https://github.com/confluentinc/librdkafka/blob/v2.4.0/STATISTICS.md
type Metrics struct {
	brokerMetrics        *internal.BrokerMetrics
	topicMetrics         *internal.TopicMetrics
	consumerGroupMetrics *internal.ConsumerGroupMetrics

	// Time since this client instance was created (microseconds)
	Age metric.Int64Gauge
	// Number of ops (callbacks, events, etc) waiting in queue for application to serve with rd_kafka_poll()
	ReplyQueue metric.Int64Gauge
	// Current number of messages in producer queues
	MessageCount metric.Int64Gauge
	// Current total size of messages in producer queues
	MessageSize metric.Int64Gauge
	// Total number of requests sent to Kafka brokers
	RequestsSent metric.Int64Gauge
	// Total number of bytes transmitted to Kafka brokers
	RequestsBytesSent metric.Int64Gauge
	// Total number of responses received from Kafka brokers
	RequestsReceived metric.Int64Gauge
	// Total number of bytes received from Kafka brokers
	RequestsBytesReceived metric.Int64Gauge
	// Total number of messages transmitted (produced) to Kafka brokers
	MessagesProduced metric.Int64Gauge
	// Total number of message bytes (including framing, such as per-Message framing and MessageSet/batch framing) transmitted to Kafka brokers
	MessagesBytesProduced metric.Int64Gauge
	// Total number of messages consumed, not including ignored messages (due to offset, etc), from Kafka brokers.
	MessagesConsumed metric.Int64Gauge
	// Total number of message bytes (including framing) received from Kafka brokers
	MessagesBytesConsumed metric.Int64Gauge
	// Number of topics in the metadata cache
	TopicsInMetadataCache metric.Int64Gauge
}

func (m *Metrics) Add(ctx context.Context, stats *stats.Stats, attrs ...attribute.KeyValue) {
	if stats == nil {
		return
	}

	attrs = append(attrs, []attribute.KeyValue{
		attribute.String("name", stats.Name),
		attribute.String("client_id", stats.ClientID),
		attribute.String("type", stats.Type),
	}...)

	m.Age.Record(ctx, stats.Age, metric.WithAttributes(attrs...))
	m.ReplyQueue.Record(ctx, stats.ReplyQueue, metric.WithAttributes(attrs...))
	m.MessageCount.Record(ctx, stats.MessageCount, metric.WithAttributes(attrs...))
	m.MessageSize.Record(ctx, stats.MessageSize, metric.WithAttributes(attrs...))
	m.RequestsSent.Record(ctx, stats.RequestsSent, metric.WithAttributes(attrs...))
	m.RequestsBytesSent.Record(ctx, stats.RequestsBytesSent, metric.WithAttributes(attrs...))
	m.RequestsReceived.Record(ctx, stats.RequestsReceived, metric.WithAttributes(attrs...))
	m.RequestsBytesReceived.Record(ctx, stats.RequestsBytesReceived, metric.WithAttributes(attrs...))
	m.MessagesProduced.Record(ctx, stats.MessagesProduced, metric.WithAttributes(attrs...))
	m.MessagesBytesProduced.Record(ctx, stats.MessagesBytesProduced, metric.WithAttributes(attrs...))
	m.MessagesConsumed.Record(ctx, stats.MessagesConsumed, metric.WithAttributes(attrs...))
	m.MessagesBytesConsumed.Record(ctx, stats.MessagesBytesConsumed, metric.WithAttributes(attrs...))
	m.TopicsInMetadataCache.Record(ctx, stats.TopicsInMetadataCache, metric.WithAttributes(attrs...))

	if m.brokerMetrics != nil {
		for _, broker := range stats.Brokers {
			// Skip bootstrap nodes
			if broker.NodeID < 0 {
				continue
			}

			m.brokerMetrics.Add(ctx, &broker, attrs...)
		}
	}

	if m.topicMetrics != nil {
		for _, topic := range stats.Topics {
			m.topicMetrics.Add(ctx, &topic, attrs...)
		}
	}

	if m.consumerGroupMetrics != nil {
		m.consumerGroupMetrics.Add(ctx, &stats.ConsumerGroup, attrs...)
	}
}

type Options struct {
	extendedMetrics              bool
	brokerMetricsDisabled        bool
	topicMetricsDisabled         bool
	consumerGroupMetricsDisabled bool
}
type Option func(*Options)

func WithExtendedMetrics() Option {
	return func(o *Options) {
		o.extendedMetrics = true
	}
}

func WithBrokerMetricsDisabled() Option {
	return func(o *Options) {
		o.brokerMetricsDisabled = true
	}
}

func WithTopicMetricsDisabled() Option {
	return func(o *Options) {
		o.topicMetricsDisabled = true
	}
}

func WithConsumerGroupMetricsDisabled() Option {
	return func(o *Options) {
		o.consumerGroupMetricsDisabled = true
	}
}

func New(meter metric.Meter, opts ...Option) (*Metrics, error) {
	o := &Options{}

	for _, opt := range opts {
		opt(o)
	}

	var err error

	m := &Metrics{}

	if !o.brokerMetricsDisabled {
		m.brokerMetrics, err = internal.NewBrokerMetrics(meter, o.extendedMetrics)
		if err != nil {
			return nil, fmt.Errorf("failed to create broker metrics: %w", err)
		}
	}

	if !o.topicMetricsDisabled {
		m.topicMetrics, err = internal.NewTopicMetrics(meter, o.extendedMetrics)
		if err != nil {
			return nil, fmt.Errorf("failed to create topic metrics: %w", err)
		}
	}

	if !o.consumerGroupMetricsDisabled {
		m.consumerGroupMetrics, err = internal.NewConsumerGroupMetrics(meter)
		if err != nil {
			return nil, fmt.Errorf("failed to create consumer group metrics: %w", err)
		}
	}

	m.Age, err = meter.Int64Gauge(
		"kafka.age_microseconds",
		metric.WithDescription("Time since this client instance was created (microseconds)"),
		metric.WithUnit("{microseconds}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.age: %w", err)
	}

	m.ReplyQueue, err = meter.Int64Gauge(
		"kafka.reply_queue_count",
		metric.WithDescription("Number of ops (callbacks, events, etc) waiting in queue for application to serve with rd_kafka_poll()"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.reply_queue_count: %w", err)
	}

	m.MessageCount, err = meter.Int64Gauge(
		"kafka.message_count",
		metric.WithDescription("Current number of messages in producer queues"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.message_count: %w", err)
	}

	m.MessageSize, err = meter.Int64Gauge(
		"kafka.message_size_bytes",
		metric.WithDescription("Current total size of messages in producer queues"),
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.message_size_bytes: %w", err)
	}

	m.RequestsSent, err = meter.Int64Gauge(
		"kafka.requests_sent_count",
		metric.WithDescription("Total number of requests sent to Kafka brokers"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.requests_sent_count: %w", err)
	}

	m.RequestsBytesSent, err = meter.Int64Gauge(
		"kafka.request_sent_bytes",
		metric.WithDescription("Total number of bytes transmitted to Kafka brokers"),
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.request_sent_bytes: %w", err)
	}

	m.RequestsReceived, err = meter.Int64Gauge(
		"kafka.requests_received_count",
		metric.WithDescription("Total number of responses received from Kafka brokers"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.requests_received_count: %w", err)
	}

	m.RequestsBytesReceived, err = meter.Int64Gauge(
		"kafka.requests_received_bytes",
		metric.WithDescription("Total number of bytes received from Kafka brokers"),
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.requests-sent: %w", err)
	}

	m.MessagesProduced, err = meter.Int64Gauge(
		"kafka.messages_produced_count",
		metric.WithDescription("Total number of messages transmitted (produced) to Kafka brokers"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.messages_produced_count: %w", err)
	}

	m.MessagesBytesProduced, err = meter.Int64Gauge(
		"kafka.messages_produced_bytes",
		metric.WithDescription("Total number of message bytes (including framing, such as per-Message framing and MessageSet/batch framing) transmitted to Kafka brokers"),
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.messages_produced_bytes: %w", err)
	}

	m.MessagesConsumed, err = meter.Int64Gauge(
		"kafka.messages_consumed_count",
		metric.WithDescription("Total number of messages consumed, not including ignored messages (due to offset, etc), from Kafka brokers."),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.messages_consumed_count: %w", err)
	}

	m.MessagesBytesConsumed, err = meter.Int64Gauge(
		"kafka.messages_consumed_bytes",
		metric.WithDescription("Total number of message bytes (including framing) received from Kafka brokers"),
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.messages_consumed_bytes: %w", err)
	}

	m.TopicsInMetadataCache, err = meter.Int64Gauge(
		"kafka.topics_in_metadata_cache_count",
		metric.WithDescription("Number of topics in the metadata cache"),
		metric.WithUnit("{topic}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.topics_in_metadata_cache_count: %w", err)
	}

	return m, nil
}
