# metrics

<!-- archie:ai-start -->

> Public facade for librdkafka OTel metrics: wraps internal domain-specific Int64Gauge structs (broker, topic, consumer group) behind a single Metrics type whose Add method fans out stats to all enabled sub-metric registries. Primary constraint: all metrics are point-in-time gauges derived from librdkafka's rolling-window JSON stats snapshot — never counters or histograms.

## Patterns

**Functional options for sub-metric gating** — Sub-metric categories are toggled via Option funcs (WithBrokerMetricsDisabled, WithTopicMetricsDisabled, WithConsumerGroupMetricsDisabled, WithExtendedMetrics) applied to an Options struct in New. Nil pointer guards in Add skip disabled categories. (`m, err := metrics.New(meter, metrics.WithExtendedMetrics(), metrics.WithBrokerMetricsDisabled())`)
**Sequential fail-fast error returns in New** — New allocates each Int64Gauge sequentially with an immediate err != nil check and wraps with fmt.Errorf. Sub-metric structs (internal.NewBrokerMetrics etc.) are constructed before top-level gauges. (`m.Age, err = meter.Int64Gauge("kafka.age_microseconds", ...); if err != nil { return nil, fmt.Errorf("failed to create metric: kafka.age: %w", err) }`)
**Nil guard at top of Add** — Add returns immediately if stats == nil before touching any field, matching the internal sub-metric pattern where a nil pointer check precedes all field reads. (`func (m *Metrics) Add(ctx context.Context, stats *stats.Stats, attrs ...attribute.KeyValue) { if stats == nil { return } }`)
**Attribute append before delegation** — Top-level client attributes (name, client_id, type) are appended to attrs before calling sub-metric Add methods. Sub-metrics append their own domain attributes inside their own Add. The attrs slice is mutated — callers must not reuse the slice after passing it. (`attrs = append(attrs, attribute.String("name", stats.Name), attribute.String("client_id", stats.ClientID), attribute.String("type", stats.Type))`)
**Skip negative-ID nodes in loops** — The broker loop in Add skips brokers with NodeID < 0 (bootstrap/internal nodes). The internal partition layer similarly skips negative partition IDs. (`for _, broker := range stats.Brokers { if broker.NodeID < 0 { continue } }`)
**All metrics are Int64Gauge** — Every metric in this package uses metric.Int64Gauge. librdkafka stats are point-in-time rolling-window snapshots, not cumulative increments — Int64Counter or Histogram would double-count across poll intervals. (`m.Age, err = meter.Int64Gauge("kafka.age_microseconds", metric.WithDescription(...), metric.WithUnit(...))`)
**Test via embedded stats.json with noop meter** — Tests unmarshal the canonical stats/testdata/stats.json fixture and call Add against noop.NewMeterProvider().Meter to verify no panic/error for all option combinations. Uses t.Context() not context.Background(). (`//go:embed stats/testdata/stats.json; testMeter := noop.NewMeterProvider().Meter("test"); kafkaMetrics.Add(t.Context(), testStats)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `metrics.go` | Public API: Metrics struct, New factory with functional options, Add fan-out to sub-metrics. Entry point for all callers. | Do not change metric types from Int64Gauge; do not add aggregation logic in Add; attrs slice is appended in-place — sub-metric Add calls must not retain or mutate the slice after returning. |
| `metrics_test.go` | Smoke tests for all option combos using noop meter and embedded stats fixture. Catches nil-deref regressions across Add paths. | Uses t.Context() not context.Background(); embeds stats/testdata/stats.json directly — keep the embed path in sync if testdata moves. |

## Anti-Patterns

- Using Int64Counter or Float64Histogram for any metric — librdkafka stats are point-in-time snapshots, not cumulative deltas
- Accessing stats fields in Add without the nil guard at the top — librdkafka JSON can omit entire sections
- Adding business logic or aggregation inside Add — it must only translate stats fields to OTel Record calls
- Constructing internal sub-metric structs outside the disabled-flag guard in New — optional sub-metrics must stay behind their disable option
- Retaining or mutating the attrs variadic slice after passing it to a sub-metric Add — the method appends to it, corrupting the caller's slice in a loop

## Decisions

- **Int64Gauge for all metrics rather than counters or histograms** — librdkafka emits rolling-window JSON snapshots, not incrementing counters; Gauge semantics correctly represent the current point-in-time value without double-counting across poll intervals.
- **Separate internal/ and stats/ sub-packages with a thin public facade** — stats/ owns JSON unmarshalling and enum semantics; internal/ owns OTel instrument construction per domain; the public package composes them, keeping each layer independently testable without exposing instrument construction to callers.
- **Functional options pattern for sub-metric categories** — Different Kafka client roles (producer vs consumer) expose different stat sections; disabling irrelevant categories avoids registering metrics that will never be recorded, reducing OTel cardinality overhead.

## Example: Construct Metrics with extended broker data disabled and record a stats snapshot

```
import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/kafka/metrics"
	"github.com/openmeterio/openmeter/pkg/kafka/metrics/stats"
	"go.opentelemetry.io/otel/attribute"
)

m, err := metrics.New(meter, metrics.WithExtendedMetrics(), metrics.WithBrokerMetricsDisabled())
if err != nil {
	return fmt.Errorf("kafka metrics: %w", err)
}
// On each librdkafka stats callback:
// ...
```

<!-- archie:ai-end -->
