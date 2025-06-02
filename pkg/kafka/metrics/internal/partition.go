package internal

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/pkg/kafka/metrics/stats"
)

type PartitionMessageMetrics struct {
	// Number of messages waiting to be produced in first-level queue
	MessagesInQueue metric.Int64Gauge
	// Number of bytes in first-level queue
	MessageBytesInQueue metric.Int64Gauge
	// Number of messages ready to be produced in transmit queue
	MessagesReadyToTransmit metric.Int64Gauge
	// Number of bytes in transmit queue
	MessageBytesReadyToTransmit metric.Int64Gauge
	// Number of pre-fetched messages in fetch queue
	MessagesInFetchQueue metric.Int64Gauge
	// Bytes in fetch queue
	MessageBytesInFetchQueue metric.Int64Gauge
	// Current number of messages in-flight to/from broker
	MessagesInflight metric.Int64Gauge
	// Total number of messages received (consumer), or total number of messages produced (possibly not yet transmitted) (producer).
	TotalNumOfMessages metric.Int64Gauge
}

func (m *PartitionMessageMetrics) Add(ctx context.Context, stats *stats.Partition, attrs ...attribute.KeyValue) {
	if stats == nil {
		return
	}

	attrs = append(attrs, []attribute.KeyValue{
		attribute.Int64("partition", stats.Partition),
	}...)

	m.MessagesInQueue.Record(ctx, stats.MessagesInQueue, metric.WithAttributes(attrs...))
	m.MessageBytesInQueue.Record(ctx, stats.MessageBytesInQueue, metric.WithAttributes(attrs...))
	m.MessagesReadyToTransmit.Record(ctx, stats.MessagesReadyToTransmit, metric.WithAttributes(attrs...))
	m.MessageBytesReadyToTransmit.Record(ctx, stats.MessageBytesReadyToTransmit, metric.WithAttributes(attrs...))
	m.MessagesInFetchQueue.Record(ctx, stats.MessagesInFetchQueue, metric.WithAttributes(attrs...))
	m.MessageBytesInFetchQueue.Record(ctx, stats.MessageBytesInFetchQueue, metric.WithAttributes(attrs...))
	m.TotalNumOfMessages.Record(ctx, stats.TotalNumOfMessages, metric.WithAttributes(attrs...))
	m.MessagesInflight.Record(ctx, stats.MessagesInflight, metric.WithAttributes(attrs...))
}

func NewPartitionMessageMetrics(meter metric.Meter) (*PartitionMessageMetrics, error) {
	var err error
	m := &PartitionMessageMetrics{}

	m.MessagesInQueue, err = meter.Int64Gauge(
		"kafka.partition.messages_in_queue",
		metric.WithDescription("Number of messages waiting to be produced in first-level queue"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.messages_in_queue: %w", err)
	}

	m.MessageBytesInQueue, err = meter.Int64Gauge(
		"kafka.partition.message_bytes_in_queue",
		metric.WithDescription("Number of bytes in first-level queue"),
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.message_bytes_in_queue: %w", err)
	}

	m.MessagesReadyToTransmit, err = meter.Int64Gauge(
		"kafka.partition.messages_ready_to_send",
		metric.WithDescription("Number of messages ready to be produced in transmit queue"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.messages_ready_to_send: %w", err)
	}

	m.MessageBytesReadyToTransmit, err = meter.Int64Gauge(
		"kafka.partition.message_bytes_ready_to_send",
		metric.WithDescription("Number of bytes ready to be produced in transmit queue"),
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.message_bytes_ready_to_send: %w", err)
	}

	m.MessagesInFetchQueue, err = meter.Int64Gauge(
		"kafka.partition.message_in_fetch_queue",
		metric.WithDescription("Number of pre-fetched messages in fetch queue"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.message_in_fetch_queue: %w", err)
	}

	m.MessageBytesInFetchQueue, err = meter.Int64Gauge(
		"kafka.partition.message_bytes_in_fetch_queue",
		metric.WithDescription("Number of message bytes pre-fetched in fetch queue"),
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.message_bytes_in_fetch_queue: %w", err)
	}

	m.TotalNumOfMessages, err = meter.Int64Gauge(
		"kafka.partition.total_num_of_messages",
		metric.WithDescription("Total number of messages received (consumer, same as MessageBytesReceived), or total number of messages produced (possibly not yet transmitted) (producer)"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.total_num_of_messages: %w", err)
	}

	m.MessagesInflight, err = meter.Int64Gauge(
		"kafka.partition.messages_inflight",
		metric.WithDescription("Current number of messages in-flight to/from broker"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.messages_inflight: %w", err)
	}

	return m, nil
}

type PartitionOffsetMetrics struct {
	// Current/Last logical offset query
	QueryOffset metric.Int64Gauge
	// Next offset to fetch
	NextOffset metric.Int64Gauge
	// Offset of last message passed to application + 1
	AppOffset metric.Int64Gauge
	// Offset to be committed
	StoredOffset metric.Int64Gauge
	// Last committed offset
	CommittedOffset metric.Int64Gauge
	// Last PARTITION_EOF signaled offset
	EOFOffset metric.Int64Gauge
	// Partition's low watermark offset on broker
	LowWatermarkOffset metric.Int64Gauge
	// Partition's high watermark offset on broker
	HighWatermarkOffset metric.Int64Gauge
	// Partition's last stable offset on broker, or same as hi_offset is broker version is less than 0.11.0.0.
	LastStableOffsetOnBroker metric.Int64Gauge
}

func (m *PartitionOffsetMetrics) Add(ctx context.Context, stats *stats.Partition, attrs ...attribute.KeyValue) {
	if stats == nil {
		return
	}

	attrs = append(attrs, []attribute.KeyValue{
		attribute.Int64("partition", stats.Partition),
	}...)

	m.QueryOffset.Record(ctx, stats.QueryOffset, metric.WithAttributes(attrs...))
	m.NextOffset.Record(ctx, stats.NextOffset, metric.WithAttributes(attrs...))
	m.AppOffset.Record(ctx, stats.AppOffset, metric.WithAttributes(attrs...))
	m.StoredOffset.Record(ctx, stats.StoredOffset, metric.WithAttributes(attrs...))
	m.CommittedOffset.Record(ctx, stats.CommittedOffset, metric.WithAttributes(attrs...))
	m.EOFOffset.Record(ctx, stats.EOFOffset, metric.WithAttributes(attrs...))
	m.LowWatermarkOffset.Record(ctx, stats.LowWatermarkOffset, metric.WithAttributes(attrs...))
	m.HighWatermarkOffset.Record(ctx, stats.HighWatermarkOffset, metric.WithAttributes(attrs...))
	m.LastStableOffsetOnBroker.Record(ctx, stats.LastStableOffsetOnBroker, metric.WithAttributes(attrs...))
}

func NewPartitionOffsetMetrics(meter metric.Meter) (*PartitionOffsetMetrics, error) {
	var err error
	m := &PartitionOffsetMetrics{}

	m.QueryOffset, err = meter.Int64Gauge(
		"kafka.partition.query_offset",
		metric.WithDescription("Current/Last logical offset query"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.query_offset: %w", err)
	}

	m.NextOffset, err = meter.Int64Gauge(
		"kafka.partition.next_offset",
		metric.WithDescription("Next offset to fetch"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.next_offset: %w", err)
	}

	m.AppOffset, err = meter.Int64Gauge(
		"kafka.partition.app_offset",
		metric.WithDescription("Offset of last message passed to application + 1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.app_offset: %w", err)
	}

	m.StoredOffset, err = meter.Int64Gauge(
		"kafka.partition.stored_offset",
		metric.WithDescription("Offset to be committed"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.stored_offset: %w", err)
	}

	m.CommittedOffset, err = meter.Int64Gauge(
		"kafka.partition.committed_offset",
		metric.WithDescription("Last committed offset"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.committed_offset: %w", err)
	}

	m.EOFOffset, err = meter.Int64Gauge(
		"kafka.partition.eof_offset",
		metric.WithDescription("Last PARTITION_EOF signaled offset"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.eof_offset: %w", err)
	}

	m.LowWatermarkOffset, err = meter.Int64Gauge(
		"kafka.partition.low_watermark_offset",
		metric.WithDescription("Partition's low watermark offset on broker"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.low_watermark_offset: %w", err)
	}

	m.HighWatermarkOffset, err = meter.Int64Gauge(
		"kafka.partition.high_watermark_offset",
		metric.WithDescription("Partition's high watermark offset on broker"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.high_watermark_offset: %w", err)
	}

	m.LastStableOffsetOnBroker, err = meter.Int64Gauge(
		"kafka.partition.last_stable_offset",
		metric.WithDescription("Partition's last stable offset on broker, or same as high_watermark_offset is broker version is less than 0.11.0.0."),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.last_stable_offset: %w", err)
	}

	return m, nil
}

type PartitionMetrics struct {
	messageMetrics *PartitionMessageMetrics
	offsetMetrics  *PartitionOffsetMetrics

	// The id of the broker that messages are currently being fetched from
	Broker metric.Int64Gauge
	// Current leader broker id
	Leader metric.Int64Gauge
	// Difference between (HighWatermarkOffset or LowWatermarkOffset) and CommittedOffset
	// HighWatermarkOffset is used when isolation.level=read_uncommitted, otherwise LowWatermarkOffset.
	ConsumerLag metric.Int64Gauge
	// Difference between (HighWatermarkOffset or LowWatermarkOffset) and StoredOffset. See ConsumerLag and StoredOffset.
	ConsumerLagStored metric.Int64Gauge
	// Total number of messages transmitted (produced)
	MessagesSent metric.Int64Gauge
	// Total number of bytes transmitted
	MessageBytesSent metric.Int64Gauge
	// Total number of messages consumed, not including ignored messages (due to offset, etc).
	MessagesReceived metric.Int64Gauge
	// Total number of bytes received
	MessageBytesReceived metric.Int64Gauge
}

func (m *PartitionMetrics) Add(ctx context.Context, stats *stats.Partition, attrs ...attribute.KeyValue) {
	if stats == nil {
		return
	}

	attrs = append(attrs, []attribute.KeyValue{
		attribute.Int64("partition", stats.Partition),
	}...)

	m.Broker.Record(ctx, stats.Broker, metric.WithAttributes(attrs...))
	m.Leader.Record(ctx, stats.Leader, metric.WithAttributes(attrs...))
	m.ConsumerLag.Record(ctx, stats.ConsumerLag, metric.WithAttributes(attrs...))
	m.ConsumerLagStored.Record(ctx, stats.ConsumerLagStored, metric.WithAttributes(attrs...))
	m.MessagesSent.Record(ctx, stats.MessagesSent, metric.WithAttributes(attrs...))
	m.MessageBytesSent.Record(ctx, stats.MessageBytesSent, metric.WithAttributes(attrs...))
	m.MessagesReceived.Record(ctx, stats.MessagesReceived, metric.WithAttributes(attrs...))
	m.MessageBytesReceived.Record(ctx, stats.MessageBytesReceived, metric.WithAttributes(attrs...))

	if m.messageMetrics != nil {
		m.messageMetrics.Add(ctx, stats, attrs...)
	}

	if m.offsetMetrics != nil {
		m.offsetMetrics.Add(ctx, stats, attrs...)
	}
}

func NewPartitionMetrics(meter metric.Meter, extended bool) (*PartitionMetrics, error) {
	var err error
	m := &PartitionMetrics{}

	if extended {
		m.messageMetrics, err = NewPartitionMessageMetrics(meter)
		if err != nil {
			return nil, fmt.Errorf("failed to create partition message metrics: %w", err)
		}

		m.offsetMetrics, err = NewPartitionOffsetMetrics(meter)
		if err != nil {
			return nil, fmt.Errorf("failed to create partition offset metrics: %w", err)
		}
	}

	m.Broker, err = meter.Int64Gauge(
		"kafka.partition.broker",
		metric.WithDescription("The id of the broker that messages are currently being fetched from"),
		metric.WithUnit("{broker}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.broker: %w", err)
	}

	m.Leader, err = meter.Int64Gauge(
		"kafka.partition.leader",
		metric.WithDescription("Current leader broker id"),
		metric.WithUnit("{broker}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.leader: %w", err)
	}

	m.ConsumerLag, err = meter.Int64Gauge(
		"kafka.partition.consumer_lag",
		metric.WithDescription("Difference between (high_watermark_offset or low_watermark_offset) and committed_offset). high_watermark_offset is used when isolation.level=read_uncommitted, otherwise last_stable_offset."),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.consumer_lag: %w", err)
	}

	m.ConsumerLagStored, err = meter.Int64Gauge(
		"kafka.partition.consumer_lag_stored",
		metric.WithDescription("Difference between (high_watermark_offset or last_stable_offset) and stored_offset. See consumer_lag and stored_offset."),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.consumer_lag_stored: %w", err)
	}

	m.MessagesSent, err = meter.Int64Gauge(
		"kafka.partition.messages_sent",
		metric.WithDescription("Total number of messages transmitted (produced)"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.messages_sent: %w", err)
	}

	m.MessageBytesSent, err = meter.Int64Gauge(
		"kafka.partition.message_bytes_sent",
		metric.WithDescription("Total number of bytes transmitted for messages_sent"),
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.message_bytes_sent: %w", err)
	}

	m.MessagesReceived, err = meter.Int64Gauge(
		"kafka.partition.messages_received",
		metric.WithDescription("Total number of messages transmitted (produced)"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.messages_received: %w", err)
	}

	m.MessageBytesReceived, err = meter.Int64Gauge(
		"kafka.partition.message_bytes_received",
		metric.WithDescription("Total number of bytes received for messages_received"),
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.message_bytes_received: %w", err)
	}

	return m, nil
}
