package metrics

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/pkg/kafka/metrics/stats"
)

type TopicMetrics struct {
	*PartitionMetrics

	// Age of client's topic object (milliseconds)
	Age metric.Int64Gauge `json:"age"`
	// Age of metadata from broker for this topic (milliseconds)
	MetadataAge metric.Int64Gauge `json:"metadata_age"`

	// Batch sizes in bytes

	// Smallest value
	BatchSizeMin metric.Int64Gauge
	// Largest value
	BatchSizeMax metric.Int64Gauge
	// Average value
	BatchSizeAvg metric.Int64Gauge
	// Sum of values
	BatchSizeSum metric.Int64Gauge
	// Standard deviation (based on histogram)
	BatchSizeStdDev metric.Int64Gauge
	// 50th percentile
	BatchSizeP50 metric.Int64Gauge
	// 75th percentile
	BatchSizeP75 metric.Int64Gauge
	// 90th percentile
	BatchSizeP90 metric.Int64Gauge
	// 95th percentile
	BatchSizeP95 metric.Int64Gauge
	// 99th percentile
	BatchSizeP99 metric.Int64Gauge
	// 99.99th percentile
	BatchSizeP9999 metric.Int64Gauge

	// Batch message counts

	// Smallest value
	BatchCountMin metric.Int64Gauge
	// Largest value
	BatchCountMax metric.Int64Gauge
	// Average value
	BatchCountAvg metric.Int64Gauge
	// Sum of values
	BatchCountSum metric.Int64Gauge
	// Standard deviation (based on histogram)
	BatchCountStdDev metric.Int64Gauge
	// Memory size of Hdr Histogram
	BatchCountHdrSize metric.Int64Gauge
	// 50th percentile
	BatchCountP50 metric.Int64Gauge
	// 75th percentile
	BatchCountP75 metric.Int64Gauge
	// 90th percentile
	BatchCountP90 metric.Int64Gauge
	// 95th percentile
	BatchCountP95 metric.Int64Gauge
	// 99th percentile
	BatchCountP99 metric.Int64Gauge
	// 99.99th percentile
	BatchCountP9999 metric.Int64Gauge
}

func (m *TopicMetrics) Add(ctx context.Context, stats *stats.TopicStats, attrs ...attribute.KeyValue) {
	attrs = append(attrs, []attribute.KeyValue{
		attribute.String("topic", stats.Topic),
	}...)

	m.Age.Record(ctx, stats.Age, metric.WithAttributes(attrs...))
	m.MetadataAge.Record(ctx, stats.MetadataAge, metric.WithAttributes(attrs...))

	m.BatchSizeMin.Record(ctx, stats.BatchSize.Min, metric.WithAttributes(attrs...))
	m.BatchSizeMax.Record(ctx, stats.BatchSize.Max, metric.WithAttributes(attrs...))
	m.BatchSizeAvg.Record(ctx, stats.BatchSize.Avg, metric.WithAttributes(attrs...))
	m.BatchSizeSum.Record(ctx, stats.BatchSize.Sum, metric.WithAttributes(attrs...))
	m.BatchSizeStdDev.Record(ctx, stats.BatchSize.StdDev, metric.WithAttributes(attrs...))
	m.BatchSizeP50.Record(ctx, stats.BatchSize.P50, metric.WithAttributes(attrs...))
	m.BatchSizeP75.Record(ctx, stats.BatchSize.P75, metric.WithAttributes(attrs...))
	m.BatchSizeP90.Record(ctx, stats.BatchSize.P90, metric.WithAttributes(attrs...))
	m.BatchSizeP95.Record(ctx, stats.BatchSize.P95, metric.WithAttributes(attrs...))
	m.BatchSizeP99.Record(ctx, stats.BatchSize.P99, metric.WithAttributes(attrs...))
	m.BatchSizeP9999.Record(ctx, stats.BatchSize.P9999, metric.WithAttributes(attrs...))

	m.BatchCountMin.Record(ctx, stats.BatchCount.Min, metric.WithAttributes(attrs...))
	m.BatchCountMax.Record(ctx, stats.BatchCount.Max, metric.WithAttributes(attrs...))
	m.BatchCountAvg.Record(ctx, stats.BatchCount.Avg, metric.WithAttributes(attrs...))
	m.BatchCountSum.Record(ctx, stats.BatchCount.Sum, metric.WithAttributes(attrs...))
	m.BatchCountStdDev.Record(ctx, stats.BatchCount.StdDev, metric.WithAttributes(attrs...))
	m.BatchCountP50.Record(ctx, stats.BatchCount.P50, metric.WithAttributes(attrs...))
	m.BatchCountP75.Record(ctx, stats.BatchCount.P75, metric.WithAttributes(attrs...))
	m.BatchCountP90.Record(ctx, stats.BatchCount.P90, metric.WithAttributes(attrs...))
	m.BatchCountP95.Record(ctx, stats.BatchCount.P95, metric.WithAttributes(attrs...))
	m.BatchCountP99.Record(ctx, stats.BatchCount.P99, metric.WithAttributes(attrs...))
	m.BatchCountP9999.Record(ctx, stats.BatchCount.P9999, metric.WithAttributes(attrs...))

	if m.PartitionMetrics != nil {
		for _, partition := range stats.Partitions {
			// Skip internal partition
			if partition.Partition < 0 {
				continue
			}

			m.PartitionMetrics.Add(ctx, &partition, attrs...)
		}
	}
}

func NewTopicMetrics(meter metric.Meter) (*TopicMetrics, error) {
	var err error
	m := &TopicMetrics{}

	m.PartitionMetrics, err = NewPartitionMetrics(meter)
	if err != nil {
		return nil, fmt.Errorf("failed to create partition metrics: %w", err)
	}

	m.Age, err = meter.Int64Gauge(
		"kafka.topic.age",
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.topic.age: %w", err)
	}

	m.MetadataAge, err = meter.Int64Gauge(
		"kafka.topic.metadata_age",
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.topic.metadata_age: %w", err)
	}

	m.BatchSizeMin, err = meter.Int64Gauge(
		"kafka.topic.batch_size_min",
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.topic.batch_size_min: %w", err)
	}

	m.BatchSizeMax, err = meter.Int64Gauge(
		"kafka.topic.batch_size_max",
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.topic.batch_size_max: %w", err)
	}

	m.BatchSizeAvg, err = meter.Int64Gauge(
		"kafka.topic.batch_size_avg",
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.topic.batch_size_avg: %w", err)
	}

	m.BatchSizeSum, err = meter.Int64Gauge(
		"kafka.topic.batch_size_sum",
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.topic.batch_size_sum: %w", err)
	}

	m.BatchSizeStdDev, err = meter.Int64Gauge(
		"kafka.topic.batch_size_stddev",
		metric.WithDescription("Standard deviation (based on histogram)"),
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.topic.batch_size_stddev: %w", err)
	}

	m.BatchSizeP50, err = meter.Int64Gauge(
		"kafka.topic.batch_size_p50",
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.topic.batch_size_p50: %w", err)
	}

	m.BatchSizeP75, err = meter.Int64Gauge(
		"kafka.topic.batch_size_p75",
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.topic.batch_size_p75: %w", err)
	}

	m.BatchSizeP90, err = meter.Int64Gauge(
		"kafka.topic.batch_size_p90",
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.topic.batch_size_p90: %w", err)
	}

	m.BatchSizeP95, err = meter.Int64Gauge(
		"kafka.topic.batch_size_p95",
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.topic.batch_size_p95: %w", err)
	}

	m.BatchSizeP99, err = meter.Int64Gauge(
		"kafka.topic.batch_size_p99",
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.topic.batch_size_p99: %w", err)
	}

	m.BatchSizeP9999, err = meter.Int64Gauge(
		"kafka.topic.batch_size_p9999",
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.topic.batch_size_p9999: %w", err)
	}

	m.BatchCountMin, err = meter.Int64Gauge(
		"kafka.topic.batch_count_min",
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.topic.batch_count_min: %w", err)
	}

	m.BatchCountMax, err = meter.Int64Gauge(
		"kafka.topic.batch_count_max",
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.topic.batch_count_max: %w", err)
	}

	m.BatchCountAvg, err = meter.Int64Gauge(
		"kafka.topic.batch_count_avg",
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.topic.batch_count_avg: %w", err)
	}

	m.BatchCountSum, err = meter.Int64Gauge(
		"kafka.topic.batch_count_sum",
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.topic.batch_count_sum: %w", err)
	}

	m.BatchCountStdDev, err = meter.Int64Gauge(
		"kafka.topic.batch_count_stddev",
		metric.WithDescription("Standard deviation (based on histogram)"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.topic.batch_count_stddev: %w", err)
	}

	m.BatchCountP50, err = meter.Int64Gauge(
		"kafka.topic.batch_count_p50",
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.topic.batch_count_p50: %w", err)
	}

	m.BatchCountP75, err = meter.Int64Gauge(
		"kafka.topic.batch_count_p75",
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.topic.batch_count_p75: %w", err)
	}

	m.BatchCountP90, err = meter.Int64Gauge(
		"kafka.topic.batch_count_p90",
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.topic.batch_count_p90: %w", err)
	}

	m.BatchCountP95, err = meter.Int64Gauge(
		"kafka.topic.batch_count_p95",
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.topic.batch_count_p95: %w", err)
	}

	m.BatchCountP99, err = meter.Int64Gauge(
		"kafka.topic.batch_count_p99",
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.topic.batch_count_p99: %w", err)
	}

	m.BatchCountP9999, err = meter.Int64Gauge(
		"kafka.topic.batch_count_p9999",
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.topic.batch_count_p9999: %w", err)
	}

	return m, nil
}

type PartitionMetrics struct {
	// The id of the broker that messages are currently being fetched from
	Broker metric.Int64Gauge
	// Current leader broker id
	Leader metric.Int64Gauge
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
	// Difference between (HighWatermarkOffset or LowWatermarkOffset) and CommittedOffset
	// HighWatermarkOffset is used when isolation.level=read_uncommitted, otherwise LowWatermarkOffset.
	ConsumerLag metric.Int64Gauge
	// Difference between (HighWatermarkOffset or LowWatermarkOffset) and StoredOffset. See ConsumerLag and StoredOffset.
	ConsumerLagStored metric.Int64Gauge
	// Total number of messages transmitted (produced)
	MessagesSent metric.Int64Counter
	// Total number of bytes transmitted
	MessageBytesSent metric.Int64Counter
	// Total number of messages consumed, not including ignored messages (due to offset, etc).
	MessagesReceived metric.Int64Counter
	// Total number of bytes received
	MessageBytesReceived metric.Int64Counter
	// Total number of messages received (consumer), or total number of messages produced (possibly not yet transmitted) (producer).
	TotalNumOfMessages metric.Int64Counter
	// Current number of messages in-flight to/from broker
	MessagesInflight metric.Int64Gauge
}

func (m *PartitionMetrics) Add(ctx context.Context, stats *stats.Partition, attrs ...attribute.KeyValue) {
	attrs = append(attrs, []attribute.KeyValue{
		attribute.Int64("partition", stats.Partition),
	}...)

	m.Broker.Record(ctx, stats.Broker, metric.WithAttributes(attrs...))
	m.Leader.Record(ctx, stats.Leader, metric.WithAttributes(attrs...))
	m.MessagesInQueue.Record(ctx, stats.MessagesInQueue, metric.WithAttributes(attrs...))
	m.MessageBytesInQueue.Record(ctx, stats.MessageBytesInQueue, metric.WithAttributes(attrs...))
	m.MessagesReadyToTransmit.Record(ctx, stats.MessagesReadyToTransmit, metric.WithAttributes(attrs...))
	m.MessageBytesReadyToTransmit.Record(ctx, stats.MessageBytesReadyToTransmit, metric.WithAttributes(attrs...))
	m.MessagesInFetchQueue.Record(ctx, stats.MessagesInFetchQueue, metric.WithAttributes(attrs...))
	m.MessageBytesInFetchQueue.Record(ctx, stats.MessageBytesInFetchQueue, metric.WithAttributes(attrs...))
	m.QueryOffset.Record(ctx, stats.QueryOffset, metric.WithAttributes(attrs...))
	m.NextOffset.Record(ctx, stats.NextOffset, metric.WithAttributes(attrs...))
	m.AppOffset.Record(ctx, stats.AppOffset, metric.WithAttributes(attrs...))
	m.StoredOffset.Record(ctx, stats.StoredOffset, metric.WithAttributes(attrs...))
	m.CommittedOffset.Record(ctx, stats.CommittedOffset, metric.WithAttributes(attrs...))
	m.EOFOffset.Record(ctx, stats.EOFOffset, metric.WithAttributes(attrs...))
	m.LowWatermarkOffset.Record(ctx, stats.LowWatermarkOffset, metric.WithAttributes(attrs...))
	m.HighWatermarkOffset.Record(ctx, stats.HighWatermarkOffset, metric.WithAttributes(attrs...))
	m.LastStableOffsetOnBroker.Record(ctx, stats.LastStableOffsetOnBroker, metric.WithAttributes(attrs...))
	m.ConsumerLag.Record(ctx, stats.ConsumerLag, metric.WithAttributes(attrs...))
	m.ConsumerLagStored.Record(ctx, stats.ConsumerLagStored, metric.WithAttributes(attrs...))
	m.MessagesSent.Add(ctx, stats.MessagesSent, metric.WithAttributes(attrs...))
	m.MessageBytesSent.Add(ctx, stats.MessageBytesSent, metric.WithAttributes(attrs...))
	m.MessagesReceived.Add(ctx, stats.MessagesReceived, metric.WithAttributes(attrs...))
	m.MessageBytesReceived.Add(ctx, stats.MessageBytesReceived, metric.WithAttributes(attrs...))
	m.TotalNumOfMessages.Add(ctx, stats.TotalNumOfMessages, metric.WithAttributes(attrs...))
	m.MessagesInflight.Record(ctx, stats.MessagesInflight, metric.WithAttributes(attrs...))
}

func NewPartitionMetrics(meter metric.Meter) (*PartitionMetrics, error) {
	var err error
	m := &PartitionMetrics{}

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

	m.Leader, err = meter.Int64Gauge(
		"kafka.partition.leader",
		metric.WithDescription("Current leader broker id"),
		metric.WithUnit("{broker}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.leader: %w", err)
	}

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

	m.MessagesSent, err = meter.Int64Counter(
		"kafka.partition.messages_sent",
		metric.WithDescription("Total number of messages transmitted (produced)"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.messages_sent: %w", err)
	}

	m.MessageBytesSent, err = meter.Int64Counter(
		"kafka.partition.message_bytes_sent",
		metric.WithDescription("Total number of bytes transmitted for messages_sent"),
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.message_bytes_sent: %w", err)
	}

	m.MessagesReceived, err = meter.Int64Counter(
		"kafka.partition.messages_received",
		metric.WithDescription("Total number of messages transmitted (produced)"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.messages_received: %w", err)
	}

	m.MessageBytesReceived, err = meter.Int64Counter(
		"kafka.partition.message_bytes_received",
		metric.WithDescription("Total number of bytes received for messages_received"),
		metric.WithUnit("{byte}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.partition.message_bytes_received: %w", err)
	}

	m.TotalNumOfMessages, err = meter.Int64Counter(
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
