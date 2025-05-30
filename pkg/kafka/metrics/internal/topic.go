package internal

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/pkg/kafka/metrics/stats"
)

type TopicBatchMetrics struct {
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

func (m *TopicBatchMetrics) Add(ctx context.Context, stats *stats.TopicStats, attrs ...attribute.KeyValue) {
	if stats == nil {
		return
	}

	attrs = append(attrs, []attribute.KeyValue{
		attribute.String("topic", stats.Topic),
	}...)

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
}

func NewTopicBatchMetrics(meter metric.Meter) (*TopicBatchMetrics, error) {
	var err error
	m := &TopicBatchMetrics{}

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

type TopicMetrics struct {
	batchMetrics     *TopicBatchMetrics
	partitionMetrics *PartitionMetrics

	// Age of client's topic object (milliseconds)
	Age metric.Int64Gauge `json:"age"`
	// Age of metadata from broker for this topic (milliseconds)
	MetadataAge metric.Int64Gauge `json:"metadata_age"`
}

func (m *TopicMetrics) Add(ctx context.Context, stats *stats.TopicStats, attrs ...attribute.KeyValue) {
	if stats == nil {
		return
	}

	attrs = append(attrs, []attribute.KeyValue{
		attribute.String("topic", stats.Topic),
	}...)

	m.Age.Record(ctx, stats.Age, metric.WithAttributes(attrs...))
	m.MetadataAge.Record(ctx, stats.MetadataAge, metric.WithAttributes(attrs...))

	if m.partitionMetrics != nil {
		for _, partition := range stats.Partitions {
			// Skip internal partition
			if partition.Partition < 0 {
				continue
			}

			m.partitionMetrics.Add(ctx, &partition, attrs...)
		}
	}

	if m.batchMetrics != nil {
		m.batchMetrics.Add(ctx, stats, attrs...)
	}
}

func NewTopicMetrics(meter metric.Meter, extended bool) (*TopicMetrics, error) {
	var err error
	m := &TopicMetrics{}

	if extended {
		m.batchMetrics, err = NewTopicBatchMetrics(meter)
		if err != nil {
			return nil, fmt.Errorf("failed to create topic batch metrics: %w", err)
		}
	}

	m.partitionMetrics, err = NewPartitionMetrics(meter, extended)
	if err != nil {
		return nil, fmt.Errorf("failed to create partition metrics: %w", err)
	}

	m.Age, err = meter.Int64Gauge(
		"kafka.topic.age",
		metric.WithUnit("{milliseconds}"),
		metric.WithDescription("age of client's topic object (milliseconds)"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.topic.age: %w", err)
	}

	m.MetadataAge, err = meter.Int64Gauge(
		"kafka.topic.metadata_age",
		metric.WithUnit("{milliseconds}"),
		metric.WithDescription("age of metadata from broker for this topic (milliseconds)"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.topic.metadata_age: %w", err)
	}

	return m, nil
}
