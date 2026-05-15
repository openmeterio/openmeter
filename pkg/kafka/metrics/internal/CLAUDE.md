# internal

<!-- archie:ai-start -->

> OTel metric instrument registry for librdkafka statistics: each domain struct (BrokerMetrics, TopicMetrics, PartitionMetrics, ConsumerGroupMetrics) owns one metric.Int64Gauge per stat field and records point-in-time values from the corresponding stats sub-structs with context-propagated OTel attributes. All metrics are Int64Gauge reflecting librdkafka's rolling-window snapshot semantics.

## Patterns

**Metric struct per stats domain with optional sub-structs** — Each Kafka stats domain gets its own *Metrics struct with one metric.Int64Gauge per measurable stat. Sub-categories (latency, throttle, message, offset, batch) are private pointer fields initialized only when extended=true; nil-guarded in Add. TopicMetrics always creates PartitionMetrics — only batchMetrics is extended-gated. (`type BrokerMetrics struct { latencyMetrics *BrokerLatencyMetrics; throttleMetrics *BrokerThrottleMetrics; Source metric.Int64Gauge; ... }`)
**NewXxxMetrics factory with sequential fail-fast error returns** — Constructors call meter.Int64Gauge for each field sequentially, wrapping every error with fmt.Errorf('failed to create metric: <name>: %w', err) and returning on first failure. Extended sub-metrics are constructed before core metrics inside the extended=true branch. (`m.Source, err = meter.Int64Gauge("kafka.broker.source", ...); if err != nil { return nil, fmt.Errorf("failed to create metric: kafka.broker.source: %w", err) }`)
**Add method: nil guard, inject domain attrs, Record each gauge, delegate to sub-metrics** — Every *Metrics.Add starts with 'if stats == nil { return }', appends domain-specific OTel attributes to the caller-supplied attrs slice (node_name/node_id for brokers, partition for partitions, topic for topics), records each gauge, then nil-guards calls to sub-metric Add methods passing the now-augmented attrs slice. (`func (m *BrokerMetrics) Add(ctx context.Context, stats *stats.BrokerStats, attrs ...attribute.KeyValue) { if stats == nil { return }; attrs = append(attrs, attribute.String("node_name", stats.NodeName), ...); m.Source.Record(ctx, stats.Source.Int64(), metric.WithAttributes(attrs...)); if m.latencyMetrics != nil { m.latencyMetrics.Add(ctx, stats, attrs...) } }`)
**Metric naming: kafka.<domain>.<stat> with OTel semantic units** — All metric names follow dot-separated kafka.<domain>.<stat> (e.g. kafka.broker.latency_min, kafka.partition.consumer_lag, kafka.topic.batch_size_p99). Units use OTel brace notation: {microsecond}, {byte}, {message}, {partition}, {broker}. (`meter.Int64Gauge("kafka.broker.latency_p99", metric.WithUnit("{microsecond}"))`)
**Extended flag gates sub-metric construction, not core counters** — NewBrokerMetrics, NewPartitionMetrics, and NewTopicMetrics accept 'extended bool'; latency/throttle (broker) and message/offset (partition) sub-structs are only constructed and called when extended=true. NewTopicMetrics always constructs PartitionMetrics but gates batchMetrics behind extended. (`if extended { m.latencyMetrics, err = NewBrokerLatencyMetrics(meter); ... } // core gauges always registered below`)
**Internal partition skip in TopicMetrics.Add** — TopicMetrics.Add loops over stats.Partitions and skips entries where partition.Partition < 0 (librdkafka UA/UnAssigned internal partition). The skip guard lives in TopicMetrics.Add, not in PartitionMetrics.Add. (`for _, partition := range stats.Partitions { if partition.Partition < 0 { continue }; m.partitionMetrics.Add(ctx, &partition, attrs...) }`)
**attrs slice is mutated in Add — callers must not reuse after the call** — Each Add method appends domain-specific attributes to the variadic attrs slice via append, which may or may not reallocate. Callers iterating over collections (brokers, partitions) and reusing the original attrs slice after an Add call will see corrupted attribute sets. (`attrs = append(attrs, attribute.String("node_name", stats.NodeName), attribute.Int64("node_id", stats.NodeID))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `broker.go` | Defines BrokerLatencyMetrics (11 latency percentile gauges), BrokerThrottleMetrics (11 throttle percentile gauges), and BrokerMetrics (19 core gauges + optional sub-structs). Both sub-structs are private fields initialized only when extended=true. | Pre-existing bug: the error message for m.Source ('failed to create metric: kafka.broker.state') duplicates the message for m.State — do not copy this pattern to new gauges. Use the actual metric name in the error string. |
| `partition.go` | Defines PartitionMessageMetrics (8 queue/inflight message gauges), PartitionOffsetMetrics (9 offset gauges), and PartitionMetrics (8 core gauges + optional sub-structs). Both sub-structs gated by extended=true. | Partition attrs (attribute.Int64('partition', stats.Partition)) are appended inside each Add method on the stats.Partition struct — both PartitionMessageMetrics.Add and PartitionOffsetMetrics.Add append partition attrs independently, so the caller-supplied attrs slice accumulates partition twice if both run. |
| `topic.go` | Defines TopicBatchMetrics (22 batch size/count percentile gauges) and TopicMetrics (2 core gauges + unconditional PartitionMetrics + extended-gated batchMetrics). Internal partition guard (Partition < 0) lives here. | TopicMetrics always constructs PartitionMetrics regardless of extended flag — only batchMetrics is extended-gated. Do not move PartitionMetrics construction inside the extended block. |
| `consumergroup.go` | Defines ConsumerGroupMetrics for 6 consumer group rebalance/state gauges. No sub-structs, no extended flag. | ConsumerGroupMetrics.Add does not append any domain-specific attributes — it records with the caller-supplied attrs only (no group-id attribute injected here). Consistent with design: callers are responsible for group-id attribution. |

## Anti-Patterns

- Using Int64Counter or Float64Histogram for any stat — all librdkafka stats are point-in-time rolling-window snapshots; only Int64Gauge correctly represents them without accumulation
- Accessing stats fields in Add without the 'if stats == nil { return }' nil guard at the top — librdkafka JSON can omit entire sections
- Adding business logic, aggregation, or conditional branching inside Add methods — they must only translate stats fields to metric.Record calls
- Constructing sub-metric structs (latency, throttle, message, offset, batch) outside the extended=true branch in factory functions
- Reusing or capturing the attrs slice passed into Add after the call returns — append may have mutated the backing array

## Decisions

- **Int64Gauge for all metrics rather than counters or histograms** — librdkafka exposes rolling-window snapshots, not monotonic counters; gauges correctly represent instantaneous values and avoid double-counting when the stats callback fires repeatedly at configurable intervals.
- **Extended flag separates core from verbose metrics** — Latency percentile distributions (11 gauges each for latency and throttle) and detailed offset/message metrics are costly to register and record; extended=false lets operators skip dozens of unused instruments in production without losing the most operationally critical gauges.
- **Domain-specific attributes appended inside Add rather than by callers** — Callers iterate over collections (brokers, topics, partitions) and should not need to know the attribute schema per entity type; centralizing node_name/node_id/topic/partition injection in each Add simplifies collection loops and prevents attribute schema drift.

## Example: Add a new Int64Gauge to BrokerMetrics and record it in Add

```
import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"github.com/openmeterio/openmeter/pkg/kafka/metrics/stats"
)

// 1. Add field to struct:
type BrokerMetrics struct {
	// ... existing fields ...
	NewGauge metric.Int64Gauge
}

// 2. Register in NewBrokerMetrics (after other core gauges):
// ...
```

<!-- archie:ai-end -->
