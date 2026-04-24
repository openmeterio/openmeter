# internal

<!-- archie:ai-start -->

> OTel metric instrument registry for librdkafka statistics — each struct owns one Int64Gauge per stat field and records values from stats sub-structs with context-propagated attributes. All metrics are gauges derived from librdkafka's rolling-window JSON stats.

## Patterns

**Metric struct per stats domain** — Each Kafka stats domain (broker, topic, partition, consumer group) gets its own *Metrics struct with one metric.Int64Gauge field per measurable stat. Sub-categories (latency, throttle, message, offset, batch) are promoted to embedded optional sub-structs enabled only when extended=true. (`type BrokerMetrics struct { latencyMetrics *BrokerLatencyMetrics; throttleMetrics *BrokerThrottleMetrics; Source metric.Int64Gauge; ... }`)
**NewXxxMetrics factory with sequential error returns** — Constructors call meter.Int64Gauge for each field sequentially, wrapping every error with fmt.Errorf('failed to create metric: <name>: %w', err) and returning immediately on the first failure. Extended sub-metrics are constructed first before core metrics. (`func NewBrokerMetrics(meter metric.Meter, extended bool) (*BrokerMetrics, error) { if extended { m.latencyMetrics, err = NewBrokerLatencyMetrics(meter); ... } m.Source, err = meter.Int64Gauge("kafka.broker.source", ...); if err != nil { return nil, fmt.Errorf(...) } }`)
**Add method delegates to sub-metric structs** — Every *Metrics struct has an Add(ctx, *stats.XxxStats, ...attribute.KeyValue) method that appends domain-specific OTel attributes (node_name/node_id for brokers, partition id for partitions, topic name for topics), calls Record on each gauge, then nil-guards calls to sub-metric Add methods. (`func (m *BrokerMetrics) Add(ctx context.Context, stats *stats.BrokerStats, attrs ...attribute.KeyValue) { attrs = append(attrs, attribute.String("node_name", stats.NodeName), ...); m.Source.Record(ctx, stats.Source.Int64(), metric.WithAttributes(attrs...)); if m.latencyMetrics != nil { m.latencyMetrics.Add(ctx, stats, attrs...) } }`)
**Nil guard on stats pointer** — Every Add method starts with 'if stats == nil { return }' before accessing any field — prevents panics when a stats section is absent in the librdkafka JSON. (`func (m *BrokerLatencyMetrics) Add(ctx context.Context, stats *stats.BrokerStats, attrs ...attribute.KeyValue) { if stats == nil { return } ... }`)
**Metric naming: kafka.<domain>.<stat>** — All metric names follow the dot-separated schema kafka.<domain>.<stat> (e.g. kafka.broker.latency_min, kafka.partition.consumer_lag, kafka.topic.batch_size_p99). Units use OTel semantic units in braces: {microsecond}, {byte}, {message}, {partition}. (`meter.Int64Gauge("kafka.broker.latency_p99", metric.WithUnit("{microsecond}"))`)
**Extended flag gates sub-metric construction** — NewBrokerMetrics and NewPartitionMetrics and NewTopicMetrics accept an 'extended bool' parameter; latency/throttle/message/offset/batch sub-structs are only constructed (and later called) when extended=true. Core counters are always registered. (`func NewPartitionMetrics(meter metric.Meter, extended bool) (*PartitionMetrics, error) { if extended { m.messageMetrics, err = NewPartitionMessageMetrics(meter); m.offsetMetrics, err = NewPartitionOffsetMetrics(meter) } ... }`)
**Partition loop skips internal partitions** — TopicMetrics.Add loops over stats.Partitions and skips any entry where partition.Partition < 0 (the UA/UnAssigned internal partition used by librdkafka). (`for _, partition := range stats.Partitions { if partition.Partition < 0 { continue }; m.partitionMetrics.Add(ctx, &partition, attrs...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `broker.go` | Defines BrokerLatencyMetrics, BrokerThrottleMetrics, and BrokerMetrics. BrokerMetrics composes the two sub-structs as private fields initialized only when extended=true. | All broker gauge names must match the kafka.broker.* prefix. The 'failed to create metric: kafka.broker.state' error message appears twice (copy-paste from Source); it's a pre-existing bug — don't propagate it to new gauges. |
| `partition.go` | Defines PartitionMessageMetrics (queue/inflight message stats), PartitionOffsetMetrics (all offset variants), and PartitionMetrics which composes both as optional sub-structs. | Partition attrs are added in each Add method, not once at the top — attrs slice is modified in place so callers' original slice must not be reused. |
| `topic.go` | Defines TopicBatchMetrics (22 gauges for batch size and batch count percentile distributions) and TopicMetrics. TopicMetrics unconditionally constructs PartitionMetrics (not guarded by extended). | TopicMetrics always creates partition metrics; extended only gates batchMetrics. Partition internal partition guard is here, not in PartitionMetrics. |
| `consumergroup.go` | Defines ConsumerGroupMetrics for consumer group rebalance and state stats. No sub-structs; no extended flag. | ConsumerGroupMetrics.Add does not append any custom attributes — it records with the caller-supplied attrs only (no group-id attribute added here). |

## Anti-Patterns

- Using metric types other than Int64Gauge — all stats are point-in-time snapshots from librdkafka, not cumulative counters; using Int64Counter or Histogram would misrepresent the semantics
- Accessing stats fields without the nil guard at the top of Add — librdkafka JSON can omit sections
- Adding business logic or aggregation inside Add methods — they must only translate stats fields to OTel Record calls
- Constructing sub-metric structs outside the extended=true branch in factories — all optional metrics must stay behind the extended flag
- Reusing or mutating the attrs slice passed into Add after the call — the method appends to it, which would corrupt the caller's slice in a loop

## Decisions

- **Int64Gauge for all metrics rather than counters or histograms** — librdkafka exposes rolling-window snapshots not monotonic counters; gauges correctly represent the instantaneous view and avoid double-counting when the stats callback fires repeatedly.
- **Extended flag separates core from verbose metrics** — Latency percentile distributions (11 gauges each for latency and throttle) and detailed offset/message metrics are costly; the extended flag lets operators opt-in without registering dozens of unused instruments in production.
- **Domain-specific attributes appended inside Add rather than passed by caller** — Callers iterate over collections (brokers, topics, partitions) and should not need to know the attribute schema per entity; each Add centralizes the node_name/node_id/topic/partition attribute injection.

## Example: Add a new broker gauge and record it in Add

```
import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"github.com/openmeterio/openmeter/pkg/kafka/metrics/stats"
)

// In BrokerMetrics struct:
NewGauge metric.Int64Gauge

// In NewBrokerMetrics:
m.NewGauge, err = meter.Int64Gauge(
	"kafka.broker.new_gauge",
	metric.WithDescription("Description here"),
// ...
```

<!-- archie:ai-end -->
