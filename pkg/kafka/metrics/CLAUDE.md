# metrics

<!-- archie:ai-start -->

> Top-level OTel metrics facade for librdkafka client statistics: a single Metrics struct holds the top-level Int64Gauges plus broker/topic/consumer-group sub-metric structs from internal/, and Add() walks a parsed stats.Stats (from stats/) recording every gauge. New(meter, ...Option) is the only constructor; wired into app/common, cmd/server, ingest/kafkaingest, and sink.

## Patterns

**Single Metrics struct of Int64Gauges + sub-metric pointers** — Metrics embeds top-level metric.Int64Gauge fields plus *internal.BrokerMetrics/TopicMetrics/ConsumerGroupMetrics; New populates all of them and returns (*Metrics, error). (`type Metrics struct { brokerMetrics *internal.BrokerMetrics; Age metric.Int64Gauge; ... }`)
**New is the only constructor; every gauge creation wraps the error** — Each meter.Int64Gauge(...) call is immediately followed by `if err != nil { return nil, fmt.Errorf("failed to create metric: <name>: %w", err) }`. Sub-metric constructors wrap with "failed to create <broker|topic|consumer group> metrics: %w". (`m.Age, err = meter.Int64Gauge("kafka.age_microseconds", ...); if err != nil { return nil, fmt.Errorf("failed to create metric: kafka.age: %w", err) }`)
**Add is nil-safe and attribute-additive** — Add(ctx, stats, attrs...) returns early on `stats == nil`, appends name/client_id/type attributes, records every top-level gauge, then dispatches to non-nil sub-metric structs. (`func (m *Metrics) Add(ctx context.Context, stats *stats.Stats, attrs ...attribute.KeyValue) { if stats == nil { return } ... }`)
**Functional Options gate sub-metrics** — Options{extendedMetrics, brokerMetricsDisabled, topicMetricsDisabled, consumerGroupMetricsDisabled} are set via Option funcs (WithExtendedMetrics, WithBrokerMetricsDisabled, WithTopicMetricsDisabled, WithConsumerGroupMetricsDisabled); disabled sub-metrics stay nil so Add skips them. (`func WithExtendedMetrics() Option { return func(o *Options) { o.extendedMetrics = true } }`)
**Bootstrap broker filtering on the facade side** — When iterating stats.Brokers, brokers with NodeID < 0 (bootstrap nodes) are skipped before calling brokerMetrics.Add. (`for _, broker := range stats.Brokers { if broker.NodeID < 0 { continue }; m.brokerMetrics.Add(ctx, &broker, attrs...) }`)
**kafka.* metric names with description + unit** — Every Int64Gauge uses a dotted name (kafka.message_count, kafka.message_size_bytes) plus metric.WithDescription and metric.WithUnit ("{message}", "{byte}", "{microseconds}", "{topic}"). (`meter.Int64Gauge("kafka.message_size_bytes", metric.WithDescription(...), metric.WithUnit("{byte}"))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `metrics.go` | Defines Metrics, Options/Option (+With* funcs), New constructor, and Add recorder; the public surface of the package. | A gauge needs THREE coordinated edits: struct field, New() block with the error-wrap guard, and a m.<Field>.Record(...) line in Add — missing any one silently drops the metric. Existing copy-paste artifact: kafka.requests_received_bytes wraps its error string as "kafka.requests-sent"; do not propagate that — use the real metric name in new code. |
| `metrics_test.go` | Table test TestWithMetrics exercising New with each Option combo against a noop meter, plus NewTestStats helper unmarshalling the embedded stats/testdata/stats.json fixture. | Uses noop.NewMeterProvider() so it only asserts construction/recording does not error, not metric values; uses t.Context(). New Options must be added to the test table or they go untested. |

## Anti-Patterns

- Adding a gauge to New() without the immediate `if err != nil { return nil, fmt.Errorf("failed to create metric: <name>: %w", err) }` guard
- Adding a struct field + New() block but forgetting the matching m.<Field>.Record(...) call in Add (or vice versa)
- Recording sub-metrics in Add without the `if m.<sub>Metrics != nil` guard — disabled sub-metrics are intentionally nil
- Recording bootstrap brokers (NodeID < 0) instead of skipping them
- Parsing/computing librdkafka stats here instead of consuming the typed structs from the stats/ sibling package

## Decisions

- **Stats parsing (stats/), metric instrument wrappers (internal/), and the recording facade (this package) are split into three layers.** — Keeps pure-data parsing dependency-light, the internal instrument wrappers reusable per-entity, and this package the single wiring/Add entry point.
- **Sub-metrics and extended/percentile metrics are opt-in/opt-out via functional Options.** — Per-broker/per-topic/per-partition percentile metrics are high-cardinality; callers (sink, ingest) can disable them or enable extended metrics per deployment.

## Example: Add a new top-level gauge end-to-end (field + New guard + Record)

```
// in metrics.go
// 1) struct field
type Metrics struct { /* ... */ NewGauge metric.Int64Gauge }
// 2) in New(...), after existing gauges
m.NewGauge, err = meter.Int64Gauge(
	"kafka.new_gauge_count",
	metric.WithDescription("..."),
	metric.WithUnit("{message}"),
)
if err != nil {
	return nil, fmt.Errorf("failed to create metric: kafka.new_gauge_count: %w", err)
}
// 3) in Add(...)
m.NewGauge.Record(ctx, stats.NewGauge, metric.WithAttributes(attrs...))
```

<!-- archie:ai-end -->
