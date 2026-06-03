# internal

<!-- archie:ai-start -->

> OTel metric instrument registry for librdkafka statistics: each domain struct (BrokerMetrics, TopicMetrics, PartitionMetrics, ConsumerGroupMetrics) owns one metric.Int64Gauge per stat field and records point-in-time values from the stats sub-structs with context-propagated OTel attributes. All gauges reflect librdkafka's rolling-window snapshot semantics.

## Patterns

**Metric struct per stats domain with optional sub-structs** — Each Kafka stats domain gets a *Metrics struct with one metric.Int64Gauge per stat. Sub-categories (latency, throttle, message, offset, batch) are private pointer fields built only when extended=true and nil-guarded in Add. TopicMetrics always builds PartitionMetrics — only batchMetrics is extended-gated. (`type BrokerMetrics struct { latencyMetrics *BrokerLatencyMetrics; throttleMetrics *BrokerThrottleMetrics; Source metric.Int64Gauge }`)
**NewXxxMetrics factory with fail-fast error returns** — Constructors call meter.Int64Gauge per field sequentially, wrapping each error with fmt.Errorf("failed to create metric: <name>: %w", err) and returning on first failure. Extended sub-metrics are built first inside the extended branch. (`m.Source, err = meter.Int64Gauge("kafka.broker.source", ...); if err != nil { return nil, fmt.Errorf("failed to create metric: kafka.broker.source: %w", err) }`)
**Add: nil guard, inject attrs, Record, delegate to sub-metrics** — Each Add starts with 'if stats == nil { return }', appends domain attrs to the caller's attrs slice (node_name/node_id, partition, topic), records each gauge, then nil-guards sub-metric Add calls with the augmented attrs. (`func (m *BrokerMetrics) Add(ctx context.Context, stats *stats.BrokerStats, attrs ...attribute.KeyValue) { if stats == nil { return }; attrs = append(attrs, attribute.String("node_name", stats.NodeName)); m.Source.Record(ctx, stats.Source.Int64(), metric.WithAttributes(attrs...)) }`)
**Metric naming kafka.<domain>.<stat> with OTel units** — Names are dot-separated kafka.<domain>.<stat>; units use OTel brace notation: {microsecond}, {byte}, {message}, {partition}, {broker}. (`meter.Int64Gauge("kafka.broker.latency_p99", metric.WithUnit("{microsecond}"))`)
**Extended flag gates sub-metrics, not core counters** — New{Broker,Partition,Topic}Metrics take 'extended bool'; latency/throttle and message/offset sub-structs only build/run when extended=true. NewTopicMetrics always builds PartitionMetrics but gates batchMetrics. (`if extended { m.latencyMetrics, err = NewBrokerLatencyMetrics(meter) }`)
**Internal partition skip in TopicMetrics.Add** — TopicMetrics.Add skips partitions where Partition < 0 (librdkafka UA/UnAssigned internal partition); the guard lives here, not in PartitionMetrics.Add. (`for _, partition := range stats.Partitions { if partition.Partition < 0 { continue }; m.partitionMetrics.Add(ctx, &partition, attrs...) }`)
**attrs slice mutated in Add — don't reuse after the call** — Each Add appends domain attrs via append (may reallocate); callers iterating collections and reusing the original attrs slice after an Add will see corrupted attribute sets. (`attrs = append(attrs, attribute.String("node_name", stats.NodeName), attribute.Int64("node_id", stats.NodeID))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `broker.go` | BrokerLatencyMetrics (11 latency percentile gauges), BrokerThrottleMetrics (11 throttle gauges), BrokerMetrics (19 core gauges + optional sub-structs built only when extended). | Pre-existing bug: m.Source's error message duplicates m.State's text — do not copy; use the actual metric name in error strings. |
| `partition.go` | PartitionMessageMetrics (8 queue/inflight gauges), PartitionOffsetMetrics (9 offset gauges), PartitionMetrics (8 core gauges + optional sub-structs, extended-gated). | Both PartitionMessageMetrics.Add and PartitionOffsetMetrics.Add independently append the partition attr, so the caller's attrs accumulates partition twice if both run. |
| `topic.go` | TopicBatchMetrics (22 batch percentile gauges), TopicMetrics (2 core gauges + unconditional PartitionMetrics + extended-gated batchMetrics); holds the Partition<0 skip guard. | TopicMetrics always constructs PartitionMetrics regardless of extended — do not move it inside the extended block. |
| `consumergroup.go` | ConsumerGroupMetrics for 6 rebalance/state gauges; no sub-structs, no extended flag. | Add injects no domain attrs — it records with caller-supplied attrs only; callers own group-id attribution. |

## Anti-Patterns

- Using Int64Counter or Float64Histogram — librdkafka stats are rolling-window snapshots; only Int64Gauge represents them without accumulation.
- Accessing stats fields in Add without the 'if stats == nil { return }' guard — librdkafka JSON can omit whole sections.
- Adding business logic, aggregation, or branching inside Add — it must only translate stats fields to metric.Record calls.
- Constructing latency/throttle/message/offset/batch sub-metric structs outside the extended=true branch.
- Reusing or capturing the attrs slice passed into Add after it returns — append may have mutated the backing array.

## Decisions

- **Int64Gauge for all metrics, not counters or histograms.** — librdkafka exposes rolling-window snapshots; gauges represent instantaneous values and avoid double-counting when the stats callback fires repeatedly.
- **Extended flag separates core from verbose metrics.** — Latency/throttle percentile distributions and detailed offset/message metrics are costly to register/record; extended=false skips dozens of unused instruments without losing critical gauges.
- **Domain attributes appended inside Add, not by callers.** — Callers iterate collections and shouldn't know the per-entity attribute schema; centralizing node_name/node_id/topic/partition injection simplifies loops and prevents schema drift.

## Example: Add a new Int64Gauge to BrokerMetrics and record it in Add

```
import (
    "context"
    "fmt"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
    "github.com/openmeterio/openmeter/pkg/kafka/metrics/stats"
)
// 1. Add field:
type BrokerMetrics struct { /* ... */ NewGauge metric.Int64Gauge }
// 2. Register in NewBrokerMetrics after core gauges:
m.NewGauge, err = meter.Int64Gauge("kafka.broker.new_gauge", metric.WithUnit("{byte}"))
if err != nil { return nil, fmt.Errorf("failed to create metric: kafka.broker.new_gauge: %w", err) }
// 3. Record in Add (after nil guard + attrs append):
m.NewGauge.Record(ctx, stats.NewStat, metric.WithAttributes(attrs...))
```

<!-- archie:ai-end -->
