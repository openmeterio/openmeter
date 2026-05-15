# ingestadapter

<!-- archie:ai-start -->

> OTel instrumentation decorator for ingest.Collector. Wraps any Collector implementation to emit trace spans and two Int64Counters (events ingested, errors) without altering ingest behavior. Applied at wiring time in app/common.

## Patterns

**Decorator / wrapper pattern over ingest.Collector** — collectorTelemetry embeds an ingest.Collector and implements the same interface. WithTelemetry() is the sole constructor — callers pass in the inner collector and receive an instrumented collector back. (`return &collectorTelemetry{collector: collector, tracer: tracer, ingestEventsCounter: c1, ingestErrorsCounter: c2}, nil`)
**Pre-allocated metric instruments in constructor** — Counters are created once in WithTelemetry and stored on the struct. Never call metric.Meter.Int64Counter inside Ingest() — instrument registration is not cheap. (`ingestEventsCounter, err := metricMeter.Int64Counter("openmeter.ingest.events", metric.WithDescription(...), metric.WithUnit("{event}"))`)
**Span-per-call with RecordError on failure** — Ingest() opens a trace span, delegates to the inner collector, then calls span.RecordError + span.SetStatus on error before incrementing the errors counter. Always defer span.End(). (`ctx, span := c.tracer.Start(ctx, "openmeter.ingest.events", ...); defer span.End(); err = c.collector.Ingest(...); if err != nil { span.RecordError(err); span.SetStatus(otelcodes.Error, ...) }`)
**Namespace attribute on every metric and span** — Both counters and the span carry attribute.String("namespace", namespace) so metrics are filterable per tenant without additional label cardinality. (`namespaceAttr := attribute.String("namespace", namespace); c.ingestEventsCounter.Add(ctx, 1, metric.WithAttributes(namespaceAttr))`)
**Metric naming convention openmeter.<domain>.<noun>** — Counter names follow openmeter.<domain>.<noun> (e.g. openmeter.ingest.events, openmeter.ingest.errors) with unit strings in UCUM {event}/{error} notation. (`metricMeter.Int64Counter("openmeter.ingest.errors", metric.WithDescription("Number of failed event ingests"), metric.WithUnit("{error}"))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `telemetry.go` | Sole file. Defines collectorTelemetry struct, WithTelemetry constructor, and implements Ingest/Close. Metric counters are pre-allocated in the constructor, not per-call. | Close() must delegate to the inner collector. Never swallow the inner collector's error — always propagate after recording the span error. |

## Anti-Patterns

- Creating metric instruments inside Ingest() instead of in the constructor — causes allocation and registration overhead on every event
- Swallowing the inner collector's error — always propagate it after recording the span error
- Adding business logic or event transformation here — this package must remain a pure observability decorator
- Omitting the namespace attribute from counters — tenant-scoped metrics are required for per-namespace observability

## Decisions

- **Separate ingestadapter package instead of embedding telemetry in kafkaingest** — Keeps observability concerns decoupled from transport — any Collector implementation (kafkaingest, in-memory, test stub) can be wrapped without modification.

## Example: Wrapping a collector with telemetry at wiring time

```
// app/common/ingest.go (wiring side)
import (
	"github.com/openmeterio/openmeter/openmeter/ingest/ingestadapter"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

func NewInstrumentedCollector(
	collector ingest.Collector,
	metricMeter metric.Meter,
	tracer trace.Tracer,
) (ingest.Collector, error) {
	return ingestadapter.WithTelemetry(collector, metricMeter, tracer)
}
```

<!-- archie:ai-end -->
