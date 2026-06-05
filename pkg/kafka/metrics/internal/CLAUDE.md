# internal

<!-- archie:ai-start -->

> Internal OpenTelemetry metric-instrument wrappers that translate librdkafka statistics (from the sibling stats package) into OTel Int64Gauge recordings for brokers, consumer groups, partitions, and topics. Not importable outside pkg/kafka/metrics (Go internal package).

## Patterns

**Struct-of-gauges + New + Add pair** — Each metric group is a struct holding metric.Int64Gauge fields, constructed by a NewXxxMetrics(meter metric.Meter, ...) (*Xxx, error) constructor and populated by an Add(ctx, stats *stats.Yyy, attrs ...attribute.KeyValue) method. New code must follow this exact triad. (`type BrokerMetrics struct { State metric.Int64Gauge; ... }; func NewBrokerMetrics(meter, extended) (*BrokerMetrics, error); func (m *BrokerMetrics) Add(ctx, stats, attrs...)`)
**Gauge creation always wraps errors with the metric name** — Every meter.Int64Gauge(...) call is immediately followed by an `if err != nil { return nil, fmt.Errorf("failed to create metric: <name>: %w", err) }` guard. Use the exact metric name string in the wrap. (`m.State, err = meter.Int64Gauge("kafka.broker.state", ...); if err != nil { return nil, fmt.Errorf("failed to create metric: kafka.broker.state: %w", err) }`)
**Add is nil-safe and attribute-additive** — Add returns early when stats == nil, then appends identity attributes (node_name/node_id, topic, partition) onto the incoming attrs slice before Record. Sub-metric structs (latency/throttle/message/offset) are only invoked when their pointer field is non-nil. (`if stats == nil { return }; attrs = append(attrs, attribute.String("topic", stats.Topic)); if m.batchMetrics != nil { m.batchMetrics.Add(ctx, stats, attrs...) }`)
**Extended-metrics gating via `extended bool`** — Constructors accept an `extended bool` and only build the heavy sub-metric structs (latency, throttle, batch, message, offset percentiles) when true, leaving those pointer fields nil otherwise. Add then skips nil sub-metrics. (`if extended { m.latencyMetrics, err = NewBrokerLatencyMetrics(meter); ... m.throttleMetrics, err = NewBrokerThrottleMetrics(meter) }`)
**Window/percentile metrics expand to min/max/avg/sum/stddev/p50..p9999** — stats.WindowStats fields (Latency, Throttle, BatchSize, BatchCount) map to a fixed set of 11 gauges named *_min/_max/_avg/_sum/_stddev/_p50/_p75/_p90/_p95/_p99/_p9999, each recorded from the matching WindowStats field. (`m.LatencyP9999.Record(ctx, stats.Latency.P9999, metric.WithAttributes(attrs...))`)
**Enum stats recorded via .Int64()** — String-enum stats (BrokerSource, BrokerState, ConsumerGroupState/JoinState) are recorded by calling their .Int64() projection from the stats package, not the raw string. (`m.State.Record(ctx, stats.State.Int64(), metric.WithAttributes(attrs...))`)
**Internal partitions are skipped** — When iterating topic partitions, partitions with Partition < 0 (internal UA/UnAssigned) are skipped before recording partition metrics. (`for _, partition := range stats.Partitions { if partition.Partition < 0 { continue }; m.partitionMetrics.Add(ctx, &partition, attrs...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `broker.go` | BrokerMetrics + BrokerLatencyMetrics + BrokerThrottleMetrics; per-broker request/response/connect counters plus optional rtt/throttle percentile windows. | Each new gauge needs a matching Record call in Add AND its own error-wrap on construction; the metric name in the fmt.Errorf wrap must match the gauge name (note: a couple of existing wraps reuse kafka.broker.state — keep names accurate for new code). |
| `consumergroup.go` | ConsumerGroupMetrics; consumer-group state/join-state/rebalance/assignment gauges. | State and JoinState are recorded via .Int64(); does not append node attributes (unlike broker). |
| `partition.go` | PartitionMetrics + PartitionMessageMetrics + PartitionOffsetMetrics; per-partition lag, offsets, queue depths, throughput. | Adds attribute.Int64("partition", stats.Partition); offset/message sub-metrics are extended-gated; Add takes *stats.Partition (singular type name), not a slice. |
| `topic.go` | TopicMetrics + TopicBatchMetrics; topic age/metadata-age plus optional batch size/count windows, and owns the PartitionMetrics for its partitions. | partitionMetrics is built unconditionally (NewPartitionMetrics is called regardless of extended); only batchMetrics is extended-gated. Iterates stats.Partitions and skips Partition < 0. |

## Anti-Patterns

- Constructing a meter.Int64Gauge without the immediate `if err != nil { return nil, fmt.Errorf("failed to create metric: <name>: %w", err) }` guard
- Recording an enum stat as a raw string instead of calling its .Int64() projection
- Calling a sub-metric struct's Add without a nil-pointer check, or forgetting the early `if stats == nil { return }` guard
- Exporting these helpers for use outside pkg/kafka/metrics — this is a Go internal package by design
- Recording the internal partition (Partition < 0) instead of skipping it

## Decisions

- **Metrics modeled as Int64Gauge wrappers over the stats package rather than computed in-line** — Cleanly separates librdkafka JSON parsing (stats) from OTel instrument lifecycle (internal), letting the parent metrics package register and feed them on each stats callback.
- **Extended/percentile metrics are opt-in via an `extended bool`** — Per-broker/topic/partition percentile histograms are high-cardinality and expensive; default deployments skip them and only build base counters.

## Example: Adding a new gauge to a metric group: construct with error-wrap, record nil-safely with attributes

```
func NewConsumerGroupMetrics(meter metric.Meter) (*ConsumerGroupMetrics, error) {
	var err error
	m := &ConsumerGroupMetrics{}
	m.State, err = meter.Int64Gauge(
		"kafka.consumer_group.state",
		metric.WithDescription("Local consumer group handler's state"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric: kafka.consumer_group.state: %w", err)
	}
	return m, nil
}

func (m *ConsumerGroupMetrics) Add(ctx context.Context, stats *stats.ConsumerGroupStats, attrs ...attribute.KeyValue) {
	if stats == nil {
// ...
```

<!-- archie:ai-end -->
