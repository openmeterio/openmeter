# metrics

<!-- archie:ai-start -->

> Public facade for librdkafka OTel metrics: the Metrics type (metrics.go) owns top-level client gauges and fans out to domain-specific Int64Gauge registries in internal/ (broker, topic, consumer group), which read typed librdkafka JSON stats from stats/. Primary constraint: all metrics are point-in-time Int64Gauge derived from librdkafka's rolling-window snapshot — never counters or histograms.

## Patterns

**Functional options for sub-metric gating** — Sub-metric categories are toggled via Option funcs (WithBrokerMetricsDisabled, WithTopicMetricsDisabled, WithConsumerGroupMetricsDisabled, WithExtendedMetrics) applied to an Options struct in New; nil-pointer guards in Add skip disabled categories. (`m, err := metrics.New(meter, metrics.WithExtendedMetrics(), metrics.WithBrokerMetricsDisabled())`)
**Sequential fail-fast error returns in New** — New allocates each Int64Gauge sequentially with an immediate err != nil check wrapped via fmt.Errorf; sub-metric structs (internal.NewBrokerMetrics etc.) are constructed before top-level gauges. (`m.Age, err = meter.Int64Gauge("kafka.age_microseconds", ...); if err != nil { return nil, fmt.Errorf("failed to create metric: kafka.age: %w", err) }`)
**Nil guard at top of Add** — Add returns immediately if stats == nil before touching any field, matching the internal sub-metric nil-check-before-field-read pattern. (`func (m *Metrics) Add(ctx context.Context, stats *stats.Stats, attrs ...attribute.KeyValue) { if stats == nil { return } }`)
**Attribute append before delegation** — Top-level client attributes (name, client_id, type) are appended to attrs before calling sub-metric Add methods; sub-metrics append their own domain attributes. The attrs slice is mutated — callers must not reuse it afterward. (`attrs = append(attrs, attribute.String("name", stats.Name), attribute.String("client_id", stats.ClientID), attribute.String("type", stats.Type))`)
**Skip negative-ID nodes in loops** — The broker loop in Add skips brokers with NodeID < 0 (bootstrap/internal nodes); the internal partition layer similarly skips negative partition IDs. (`for _, broker := range stats.Brokers { if broker.NodeID < 0 { continue } }`)
**All metrics are Int64Gauge** — Every metric uses metric.Int64Gauge because librdkafka stats are point-in-time rolling-window snapshots; Int64Counter or Histogram would double-count across poll intervals. (`m.Age, err = meter.Int64Gauge("kafka.age_microseconds", metric.WithDescription(...), metric.WithUnit("{microseconds}"))`)
**Test via embedded stats.json with noop meter** — Tests unmarshal the canonical stats/testdata/stats.json fixture and call Add against noop.NewMeterProvider().Meter for all option combinations, using t.Context() not context.Background(). (`//go:embed stats/testdata/stats.json; testMeter := noop.NewMeterProvider().Meter("test"); kafkaMetrics.Add(t.Context(), testStats)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `metrics.go` | Public API: Metrics struct, New factory with functional options, Add fan-out to sub-metrics. Entry point for all callers. | Do not change metric types from Int64Gauge or add aggregation in Add; the attrs slice is appended in-place — sub-metric Add calls must not retain or mutate it after returning. |
| `metrics_test.go` | Smoke tests across all option combos using the noop meter and embedded stats fixture; catches nil-deref regressions across Add paths. | Uses t.Context() not context.Background(); embeds stats/testdata/stats.json directly — keep the embed path in sync if testdata moves. |

## Anti-Patterns

- Using Int64Counter or Float64Histogram for any metric — librdkafka stats are point-in-time snapshots, not cumulative deltas.
- Accessing stats fields in Add without the nil guard at the top — librdkafka JSON can omit entire sections.
- Adding business logic or aggregation inside Add — it must only translate stats fields to OTel Record calls.
- Constructing internal sub-metric structs outside the disabled-flag guard in New — optional sub-metrics must stay behind their disable option.
- Retaining or mutating the attrs variadic slice after passing it to a sub-metric Add — the method appends to it, corrupting the caller's slice in a loop.

## Decisions

- **Int64Gauge for all metrics rather than counters or histograms.** — librdkafka emits rolling-window JSON snapshots, not incrementing counters; Gauge semantics represent the current point-in-time value without double-counting across poll intervals.
- **Separate internal/ and stats/ sub-packages with a thin public facade.** — stats/ owns JSON unmarshalling and enum semantics; internal/ owns OTel instrument construction per domain; the public package composes them, keeping each layer independently testable without exposing instrument construction.
- **Functional options pattern for sub-metric categories.** — Different Kafka client roles (producer vs consumer) expose different stat sections; disabling irrelevant categories avoids registering never-recorded metrics and reduces OTel cardinality overhead.

## Example: Construct Metrics with extended data but broker metrics disabled, then record a stats snapshot

```
import (
  "fmt"
  "github.com/openmeterio/openmeter/pkg/kafka/metrics"
  "github.com/openmeterio/openmeter/pkg/kafka/metrics/stats"
)

m, err := metrics.New(meter, metrics.WithExtendedMetrics(), metrics.WithBrokerMetricsDisabled())
if err != nil { return fmt.Errorf("kafka metrics: %w", err) }
// On each librdkafka stats callback:
m.Add(ctx, parsedStats, attribute.String("role", "consumer"))
```

<!-- archie:ai-end -->
