# metrics

<!-- archie:ai-start -->

> Bridges go-metrics (used by Sarama/Kafka clients) to OpenTelemetry by wrapping go-metrics types so every metric mutation is forwarded to an otelmetric.Meter. Enables Kafka client telemetry to appear in the OpenTelemetry pipeline without changing Sarama internals.

## Patterns

**Wrap-and-delegate metric types** — Each wrapped type (wrappedMeter, wrappedCounter, wrappedGauge, wrappedGaugeFloat64, wrappedHistogram) embeds the original go-metrics interface and overrides only the mutation methods (Mark, Inc, Dec, Update, Record) to call both the OTel instrument and the underlying go-metrics method. (`type wrappedCounter struct { metrics.Counter; otelMeter otelmetric.Int64UpDownCounter; attributes attribute.Set } — Inc calls otelMeter.Add then Counter.Inc`)
**Registry delegation via embedding** — registry embeds metrics.Registry for all read/list operations and only overrides Register and GetOrRegister to intercept new metric registrations and wrap them. (`type registry struct { metrics.Registry; mu sync.Mutex; meticMeter otelmetric.Meter; ... }`)
**TransformMetricsNameToOtel for name mapping and drop filtering** — All metric names pass through NameTransformFn before OTel instrument creation. Returning Drop:true bypasses OTel wrapping and returns the raw go-metrics type unchanged. (`transfomedMetric := r.nameTransformFn(name); if transfomedMetric.Drop { return def, nil }`)
**Mutex-guarded registration** — GetOrRegister and Register acquire r.mu before checking or writing to the embedded Registry, preventing races on concurrent Sarama metric registration. (`r.mu.Lock(); defer r.mu.Unlock() at the top of both methods`)
**Reflect-based metric factory unwrapping** — def passed to getWrappedMeter may be a zero-arg function returning the metric (Sarama pattern). reflect.ValueOf(def).Kind() == reflect.Func triggers a call to extract the actual metric before type-switching. (`if v := reflect.ValueOf(def); v.Kind() == reflect.Func { def = v.Call(nil)[0].Interface() }`)
**context.Background() for OTel calls (documented limitation)** — All OTel instrument calls use context.Background() because go-metrics mutation methods carry no context parameter. This is an accepted limitation documented in README.md. (`m.otelMeter.Add(context.Background(), n, otelmetric.WithAttributeSet(m.attributes))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Entire package implementation: Registry constructor, wrapped metric structs, and their OTel-forwarding mutation methods. | Adding a new go-metrics type requires: (1) a new wrapped struct embedding that type, (2) a new case in getWrappedMeter's type switch, (3) override of every mutation method. Missing any method leaves silent gaps where OTel receives no data. |
| `README.md` | Documents the known limitation that context is context.Background() and errors are only logged. | If OTel metric registration errors are silently swallowed by errorHandler, callers see no panic but metrics are missing. |

## Anti-Patterns

- Adding a new metric type without implementing ALL mutation methods on the wrapper — partial delegation silently drops OTel events.
- Calling r.Registry.Register directly inside getWrappedMeter instead of returning the wrapped value — double-registration under the same name will occur.
- Removing the reflect-based factory unwrapping — Sarama passes metric factories as zero-arg functions, not direct metric values; removing the reflect call breaks those registrations.
- Using a real context instead of context.Background() in wrapped methods — go-metrics interfaces have no context parameter so callers cannot propagate one.
- Initialising NewRegistry without a MetricMeter — constructor returns an error; callers must check it or OTel forwarding is entirely absent.

## Decisions

- **Embed metrics.Registry rather than re-implementing it** — All read/iteration/unregister operations on the registry are unchanged; only the write path (Register, GetOrRegister) needs interception to wrap new metrics.
- **TransformMetricsNameToOtel as an injectable function with Drop support** — Sarama emits dozens of internal metrics; callers need to filter noise and rename metrics to match OTel naming conventions without forking this package.
- **context.Background() in all OTel calls** — go-metrics mutation methods (Mark, Inc, Update) carry no context; propagating caller context is structurally impossible without changing the go-metrics interface.

## Example: Register a custom go-metrics registry that forwards to OTel

```
import (
	"github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka/metrics"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

reg, err := metrics.NewRegistry(metrics.NewRegistryOptions{
	MetricMeter: meter, // otelmetric.Meter from app wiring
	NameTransformFn: func(name string) metrics.TransformedMetric {
		// drop internal Sarama bookkeeping metrics
		if name == "incoming-byte-rate" {
			return metrics.TransformedMetric{Drop: true}
		}
		return metrics.TransformedMetric{
			Name:       "kafka." + name,
// ...
```

<!-- archie:ai-end -->
