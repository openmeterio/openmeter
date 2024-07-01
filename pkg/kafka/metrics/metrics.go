// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/pkg/kafka/metrics/stats"
)

// Metrics stores set of Kafka client related metrics
// See: https://github.com/confluentinc/librdkafka/blob/v2.4.0/STATISTICS.md
type Metrics struct {
	*BrokerMetrics
	*TopicMetrics
	*ConsumerGroupMetrics

	// Time since this client instance was created (microseconds)
	Age metric.Int64Counter
	// Number of ops (callbacks, events, etc) waiting in queue for application to serve with rd_kafka_poll()
	ReplyQueue metric.Int64Gauge
	// Current number of messages in producer queues
	MessageCount metric.Int64Gauge
	// Current total size of messages in producer queues
	MessageSize metric.Int64Gauge
	// Total number of requests sent to Kafka brokers
	RequestsSent metric.Int64Counter
	// Total number of bytes transmitted to Kafka brokers
	RequestsBytesSent metric.Int64Counter
	// Total number of responses received from Kafka brokers
	RequestsReceived metric.Int64Counter
	// Total number of bytes received from Kafka brokers
	RequestsBytesReceived metric.Int64Counter
	// Total number of messages transmitted (produced) to Kafka brokers
	MessagesProduced metric.Int64Counter
	// Total number of message bytes (including framing, such as per-Message framing and MessageSet/batch framing) transmitted to Kafka brokers
	MessagesBytesProduced metric.Int64Counter
	// Total number of messages consumed, not including ignored messages (due to offset, etc), from Kafka brokers.
	MessagesConsumed metric.Int64Counter
	// Total number of message bytes (including framing) received from Kafka brokers
	MessagesBytesConsumed metric.Int64Counter
	// Number of topics in the metadata cache
	TopicsInMetadataCache metric.Int64Gauge
}

func (m *Metrics) Add(ctx context.Context, stats *stats.Stats, attrs ...attribute.KeyValue) {
	attrs = append(attrs, []attribute.KeyValue{
		attribute.String("name", stats.Name),
		attribute.String("client_id", stats.ClientID),
		attribute.String("type", stats.Type),
	}...)

	m.Age.Add(ctx, stats.Age, metric.WithAttributes(attrs...))
	m.ReplyQueue.Record(ctx, stats.ReplyQueue, metric.WithAttributes(attrs...))
	m.MessageCount.Record(ctx, stats.MessageCount, metric.WithAttributes(attrs...))
	m.MessageSize.Record(ctx, stats.MessageSize, metric.WithAttributes(attrs...))
	m.RequestsSent.Add(ctx, stats.RequestsSent, metric.WithAttributes(attrs...))
	m.RequestsBytesSent.Add(ctx, stats.RequestsBytesSent, metric.WithAttributes(attrs...))
	m.RequestsReceived.Add(ctx, stats.RequestsReceived, metric.WithAttributes(attrs...))
	m.RequestsBytesReceived.Add(ctx, stats.RequestsBytesReceived, metric.WithAttributes(attrs...))
	m.MessagesProduced.Add(ctx, stats.MessagesProduced, metric.WithAttributes(attrs...))
	m.MessagesBytesProduced.Add(ctx, stats.MessagesBytesProduced, metric.WithAttributes(attrs...))
	m.MessagesConsumed.Add(ctx, stats.MessagesConsumed, metric.WithAttributes(attrs...))
	m.MessagesBytesConsumed.Add(ctx, stats.MessagesBytesConsumed, metric.WithAttributes(attrs...))
	m.TopicsInMetadataCache.Record(ctx, stats.TopicsInMetadataCache, metric.WithAttributes(attrs...))

	if m.BrokerMetrics != nil {
		for _, broker := range stats.Brokers {
			// Skip bootstrap nodes
			if broker.NodeID < 0 {
				continue
			}

			m.BrokerMetrics.Add(ctx, &broker, attrs...)
		}
	}

	if m.TopicMetrics != nil {
		for _, topic := range stats.Topics {
			m.TopicMetrics.Add(ctx, &topic, attrs...)
		}
	}

	if m.ConsumerGroupMetrics != nil {
		m.ConsumerGroupMetrics.Add(ctx, &stats.ConsumerGroup, attrs...)
	}
}

func New(meter metric.Meter) (*Metrics, error) {
	var err error
	m := &Metrics{}

	m.BrokerMetrics, err = NewBrokerMetrics(meter)
	if err != nil {
		return nil, fmt.Errorf("failed to create broker metrics: %w", err)
	}

	m.TopicMetrics, err = NewTopicMetrics(meter)
	if err != nil {
		return nil, fmt.Errorf("failed to create topic metrics: %w", err)
	}

	m.ConsumerGroupMetrics, err = NewConsumerGroupMetrics(meter)
	if err != nil {
		return nil, fmt.Errorf("failed to create topic metrics: %w", err)
	}

	m.Age, err = meter.Int64Counter(
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

	m.RequestsSent, err = meter.Int64Counter(
		"kafka.requests_sent_count",
		metric.WithDescription("Total number of requests sent to Kafka brokers"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.requests_sent_count: %w", err)
	}

	m.RequestsBytesSent, err = meter.Int64Counter(
		"kafka.request_sent_bytes",
		metric.WithDescription("Total number of bytes transmitted to Kafka brokers"),
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.request_sent_bytes: %w", err)
	}

	m.RequestsReceived, err = meter.Int64Counter(
		"kafka.requests_received_count",
		metric.WithDescription("Total number of responses received from Kafka brokers"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.requests_received_count: %w", err)
	}

	m.RequestsBytesReceived, err = meter.Int64Counter(
		"kafka.requests_received_bytes",
		metric.WithDescription("Total number of bytes received from Kafka brokers"),
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.requests-sent: %w", err)
	}

	m.MessagesProduced, err = meter.Int64Counter(
		"kafka.messages_produced_count",
		metric.WithDescription("Total number of messages transmitted (produced) to Kafka brokers"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.messages_produced_count: %w", err)
	}

	m.MessagesBytesProduced, err = meter.Int64Counter(
		"kafka.messages_produced_bytes",
		metric.WithDescription("Total number of message bytes (including framing, such as per-Message framing and MessageSet/batch framing) transmitted to Kafka brokers"),
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.messages_produced_bytes: %w", err)
	}

	m.MessagesConsumed, err = meter.Int64Counter(
		"kafka.messages_consumed_count",
		metric.WithDescription("Total number of messages consumed, not including ignored messages (due to offset, etc), from Kafka brokers."),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.messages_consumed_count: %w", err)
	}

	m.MessagesBytesConsumed, err = meter.Int64Counter(
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
