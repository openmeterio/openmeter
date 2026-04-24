# metrics

<!-- archie:ai-start -->

> Public facade for librdkafka OTel metrics: wraps internal domain-specific gauge structs (broker, topic, consumer group) behind a single Metrics type whose Add method fans out stats to all enabled sub-metric registries. Primary constraint: all metrics are point-in-time gauges derived from librdkafka's rolling-window JSON stats snapshot.

## Patterns

**Functional options for sub-metric gating** — Sub-metric categories (broker, topic, consumer group, extended) are toggled via Option funcs (WithBrokerMetricsDisabled, WithTopicMetricsDisabled, WithConsumerGroupMetricsDisabled, WithExtendedMetrics) applied to an Options struct in New. Nil pointer guards in Add skip disabled categories. (`kafkaMetrics, err := metrics.New(meter, metrics.WithExtendedMetrics(), metrics.WithBrokerMetricsDisabled())`)
**Sequential error returns in New factory** — New allocates each Int64Gauge sequentially with an immediate err != nil check and wraps errors with fmt.Errorf. Sub-metric structs are constructed first via internal.NewXxxMetrics, then top-level gauges. (`m.Age, err = meter.Int64Gauge("kafka.age_microseconds", ...); if err != nil { return nil, fmt.Errorf("failed to create metric: kafka.age: %w", err) }`)
**Nil guard at top of Add** — Add returns immediately if stats == nil before touching any field, matching the internal sub-metric pattern where a nil pointer check precedes all field reads. (`func (m *Metrics) Add(ctx context.Context, stats *stats.Stats, attrs ...attribute.KeyValue) { if stats == nil { return } }`)
**Attribute append before delegation** — Top-level client attributes (name, client_id, type) are appended to attrs before calling sub-metric Add methods; sub-metrics append their own domain attributes inside their own Add. (`attrs = append(attrs, attribute.String("name", stats.Name), attribute.String("client_id", stats.ClientID), attribute.String("type", stats.Type))`)
**Skip internal/bootstrap nodes in broker loop** — The broker loop in Add skips brokers with NodeID < 0, matching the internal pattern of skipping negative partition IDs. (`if broker.NodeID < 0 { continue }`)
**All metrics are Int64Gauge — no counters or histograms** — Every metric in this package (top-level and internal) uses metric.Int64Gauge. librdkafka stats are point-in-time snapshots, not cumulative increments. (`m.Age, err = meter.Int64Gauge("kafka.age_microseconds", metric.WithDescription(...), metric.WithUnit(...))`)
**Test via embedded testdata/stats.json with noop meter** — Tests unmarshal the canonical stats.json fixture and call Add against a noop.NewMeterProvider().Meter to verify no panic/error for all option combinations. (`//go:embed stats/testdata/stats.json; testMeter := noop.NewMeterProvider().Meter("test"); kafkaMetrics.Add(t.Context(), testStats)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `metrics.go` | Public API: Metrics struct, New factory, Add fan-out, Option funcs. Entry point for all callers. | Do not change metric types from Int64Gauge; do not add aggregation logic in Add; attrs slice is appended in-place — sub-metric Add calls must not retain or mutate the slice after returning. |
| `metrics_test.go` | Smoke tests for all option combos using noop meter and embedded stats fixture. Tests Add with real stats.Stats to catch nil-deref regressions. | Uses t.Context() not context.Background(); embeds stats/testdata/stats.json directly — keep the embed path in sync if testdata moves. |

## Anti-Patterns

- Using Int64Counter or Histogram for any metric — librdkafka stats are snapshots, not cumulative deltas
- Accessing stats fields in Add without the nil guard at the top — librdkafka JSON can omit sections
- Adding business logic or aggregation inside Add — it must only translate stats fields to OTel Record calls
- Constructing internal sub-metric structs outside the disabled-flag guard in New — all optional sub-metrics must remain behind their disable option
- Retaining or mutating the attrs variadic slice after passing it to a sub-metric Add — the method appends to it, corrupting the caller's slice in a loop

## Decisions

- **Int64Gauge for all metrics rather than counters or histograms** — librdkafka emits rolling-window JSON snapshots, not incrementing counters; Gauge semantics correctly represent the current point-in-time value without double-counting across poll intervals.
- **Separate internal/ and stats/ sub-packages with a thin public facade** — stats/ owns JSON unmarshalling and enum semantics; internal/ owns OTel instrument construction per domain; the public package composes them, keeping each layer testable independently without exposing instrument construction to callers.
- **Functional options pattern for sub-metric categories** — Different Kafka client roles (producer vs consumer) expose different stat sections; disabling irrelevant categories avoids registering metrics that will never be recorded, reducing cardinality overhead.

## Example: Construct Metrics with extended broker data disabled and record a stats snapshot

```
import (
	"context"
	"github.com/openmeterio/openmeter/pkg/kafka/metrics"
	"github.com/openmeterio/openmeter/pkg/kafka/metrics/stats"
	"go.opentelemetry.io/otel/attribute"
)

m, err := metrics.New(meter, metrics.WithExtendedMetrics(), metrics.WithBrokerMetricsDisabled())
if err != nil {
	return fmt.Errorf("kafka metrics: %w", err)
}
// Later, on each librdkafka stats callback:
var s stats.Stats
if err := json.Unmarshal(statsJSON, &s); err == nil {
	m.Add(ctx, &s, attribute.String("environment", "production"))
// ...
```

<!-- archie:ai-end -->
