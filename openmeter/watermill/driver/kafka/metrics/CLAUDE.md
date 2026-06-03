# metrics

<!-- archie:ai-start -->

> Bridges go-metrics (used by Sarama/Kafka clients) to OpenTelemetry by wrapping go-metrics types so every metric mutation is forwarded to an otelmetric.Meter — exposing Kafka client telemetry in the OTel pipeline without changing Sarama internals.

## Patterns

**Wrap-and-delegate metric types** — Each wrapped type (wrappedMeter, wrappedCounter, wrappedGauge, wrappedGaugeFloat64, wrappedHistogram) embeds the original go-metrics interface and overrides only mutation methods (Mark, Inc, Dec, Update, Record) to call both the OTel instrument and the underlying go-metrics method. (`func (m *wrappedCounter) Inc(n int64) { m.otelMeter.Add(context.Background(), n, otelmetric.WithAttributeSet(m.attributes)); m.Counter.Inc(n) }`)
**Registry delegation via embedding** — registry embeds metrics.Registry for all read/list operations and overrides only Register and GetOrRegister to intercept and wrap new metric registrations. (`type registry struct { metrics.Registry; mu sync.Mutex; meticMeter otelmetric.Meter }`)
**NameTransformFn for renaming + drop filtering** — All metric names pass through nameTransformFn before OTel instrument creation; returning Drop:true bypasses wrapping and returns the raw go-metrics type unchanged. (`transfomedMetric := r.nameTransformFn(name); if transfomedMetric.Drop { return def, nil }`)
**Mutex-guarded registration** — GetOrRegister and Register acquire r.mu before reading/writing the embedded Registry, preventing races on concurrent Sarama metric registration. (`r.mu.Lock(); defer r.mu.Unlock()`)
**Reflect-based factory unwrapping** — def may be a zero-arg function returning the metric (Sarama pattern); reflect.ValueOf(def).Kind()==reflect.Func triggers a call to extract the metric before the type switch. (`if v := reflect.ValueOf(def); v.Kind() == reflect.Func { def = v.Call(nil)[0].Interface() }`)
**context.Background() for OTel calls (documented limitation)** — All OTel instrument calls use context.Background() because go-metrics mutation methods carry no context. This is an accepted limitation documented in README.md — one of the few legitimate exceptions to the no-Background rule. (`m.otelMeter.Add(context.Background(), n, otelmetric.WithAttributeSet(m.attributes))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Entire package: NewRegistry constructor, registry, wrapped metric structs and their OTel-forwarding mutation methods, getWrappedMeter type switch. | A new go-metrics type needs (1) a new wrapped struct embedding it, (2) a case in getWrappedMeter, (3) overrides of every mutation method — missing any leaves silent gaps where OTel receives no data. |
| `README.md` | Documents known limitations: context is always context.Background() and registration errors are only logged, never propagated. | OTel registration errors swallowed by errorHandler mean missing metrics with no panic surfaced to callers. |

## Anti-Patterns

- Adding a metric type without implementing ALL mutation methods on the wrapper.
- Calling r.Registry.Register directly inside getWrappedMeter instead of returning the wrapped value — causes double-registration.
- Removing reflect-based factory unwrapping — Sarama passes factories as zero-arg functions.
- Initialising NewRegistry without a MetricMeter — constructor errors; unchecked means no OTel forwarding.
- Using a real propagated context instead of context.Background() — go-metrics interfaces have no context parameter.

## Decisions

- **Embed metrics.Registry rather than re-implementing it.** — Only the write path (Register, GetOrRegister) needs interception to wrap new metrics; read/iterate/unregister stay unchanged, minimizing surface area.
- **NameTransformFn injectable with Drop support.** — Sarama emits dozens of internal metrics; callers filter noise and rename to OTel conventions without forking this package.
- **context.Background() in all OTel calls.** — go-metrics mutation methods carry no context; propagating caller context is impossible without forking Sarama.

## Example: Register a go-metrics registry forwarding to OTel with noise filtering

```
import (
	"github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka/metrics"
	"go.opentelemetry.io/otel/attribute"
)

reg, err := metrics.NewRegistry(metrics.NewRegistryOptions{
	MetricMeter: meter,
	NameTransformFn: func(name string) metrics.TransformedMetric {
		if name == "incoming-byte-rate" { return metrics.TransformedMetric{Drop: true} }
		return metrics.TransformedMetric{Name: "kafka." + name, Attributes: attribute.NewSet()}
	},
})
```

<!-- archie:ai-end -->
