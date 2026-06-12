# metrics

<!-- archie:ai-start -->

> Adapts the rcrowley/go-metrics registry used by Sarama into OpenTelemetry instruments, so Kafka client metrics are emitted as raw OTel events instead of relying on go-metrics' periodic scraping. Primary constraint: it must transparently wrap go-metrics types without breaking Sarama's expectations of the metrics.Registry interface.

## Patterns

**Embed-and-decorate go-metrics types** — Each wrapper struct embeds the original go-metrics interface (Meter, Counter, Gauge, GaugeFloat64, Histogram) and overrides only the mutating method to also write to an OTel instrument, then delegates to the embedded value. (`type wrappedCounter struct { metrics.Counter; otelMeter otelmetric.Int64UpDownCounter; attributes attribute.Set }; func (m *wrappedCounter) Inc(n int64) { m.otelMeter.Add(...); m.Counter.Inc(n) }`)
**Registry embeds metrics.Registry and overrides Register/GetOrRegister** — registry embeds metrics.Registry and overrides only GetOrRegister and Register to wrap defs before delegating to the embedded Registry; all other Registry methods pass through. (`type registry struct { metrics.Registry; mu sync.Mutex; meticMeter otelmetric.Meter; ... }`)
**go-metrics type → OTel instrument mapping** — getWrappedMeter type-switches on the go-metrics interface and picks the matching OTel instrument: Meter→Int64Counter, Counter→Int64UpDownCounter, GaugeFloat64→Float64Gauge, Gauge→Int64Gauge, Histogram→Int64Histogram. (`case metrics.Counter: otelMeter, err := r.meticMeter.Int64UpDownCounter(transfomedMetric.Name)`)
**Reflect-resolve function defs** — go-metrics may register a factory func instead of an instance; getWrappedMeter detects reflect.Func and calls it with no args to obtain the real metric before type-switching. (`if v := reflect.ValueOf(def); v.Kind() == reflect.Func { def = v.Call(nil)[0].Interface() }`)
**Name transform with drop support** — NameTransformFn maps a go-metrics name to a TransformedMetric (Name, Attributes, Drop). Drop==true returns the original unwrapped metric so uninteresting metrics incur no OTel cost. (`if transfomedMetric.Drop { return def, nil }`)
**Errors logged, never panicked** — Instrument-creation and registration failures are routed through an ErrorHandler (default no-op; LoggingErrorHandler wraps slog) and the original metric is returned, so metrics wiring never breaks Sarama. (`if err != nil { r.errorHandler(err); return def }`)
**Mutex-guarded register paths** — GetOrRegister and Register both take r.mu before checking/wrapping/registering to avoid duplicate concurrent registration of the same named metric. (`r.mu.Lock(); defer r.mu.Unlock()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Entire package: NewRegistry constructor, the registry wrapper over metrics.Registry, getWrappedMeter type dispatch, and the five wrapped* instrument decorators. | NewRegistry requires MetricMeter (returns error if nil) but defaults NameTransformFn and ErrorHandler. The Histogram case uses break (logs and falls through to return def) instead of returning early like the others — keep that asymmetry in mind when editing. |
| `README.md` | Documents why this adapter exists (no OTel connector for go-metrics; scraping vs raw events) and current limitations. | Documents the two intentional shortcuts: OTel calls use context.Background(), and registration errors are only logged — do not treat these as bugs to silently change. |

## Anti-Patterns

- Replacing the embedded go-metrics interface methods entirely instead of delegating to them — Sarama still reads values back through the original go-metrics API.
- Using slog.Default() inside the package; logging must come via LoggingErrorHandler(dest *slog.Logger) injected by the caller.
- Panicking or returning errors up Sarama's path on instrument-creation failure; route through errorHandler and return the original def instead.
- Adding a new go-metrics type without extending the getWrappedMeter type switch — it falls through to the default errorHandler 'unsupported metric type' branch and stays unwrapped.
- Wrapping metrics in GetOrRegister/Register without holding r.mu, risking duplicate registration races.

## Decisions

- **Wrap go-metrics types and emit raw OTel events rather than build a periodic scraper.** — go-metrics only supports periodic scraping; wrapping the mutating methods lets the adapter push real-time events into OpenTelemetry as they happen.
- **Use context.Background() for all OTel record/add calls.** — go-metrics' event interface carries no context, so there is no caller context to propagate at the metric-emission site (documented limitation in README).
- **Reflect-call func-typed defs before type-switching.** — go-metrics sometimes registers a factory function rather than a concrete metric; resolving it first lets the single type switch handle all registration shapes.

## Example: Building an OTel-backed go-metrics registry to hand to Sarama

```
import (
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
	"github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka/metrics"
)

reg, err := metrics.NewRegistry(metrics.NewRegistryOptions{
	MetricMeter: meter,
	NameTransformFn: func(name string) metrics.TransformedMetric {
		return metrics.TransformedMetric{Name: "kafka_" + name, Attributes: attribute.NewSet()}
	},
	ErrorHandler: metrics.LoggingErrorHandler(logger),
})
if err != nil {
	return err
// ...
```

<!-- archie:ai-end -->
