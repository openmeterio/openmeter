# ingestadapter

<!-- archie:ai-start -->

> OTel instrumentation decorator for ingest.Collector. Wraps any Collector implementation to emit spans and two Int64Counters (events ingested, errors) without changing ingest behavior. Applied at wiring time in app/common.

## Patterns

**Decorator / wrapper pattern over ingest.Collector** — collectorTelemetry embeds an ingest.Collector and implements the same interface. WithTelemetry() is the sole constructor — callers pass in the inner collector and receive an instrumented collector back. (`return &collectorTelemetry{collector: collector, tracer: tracer, ingestEventsCounter: c1, ingestErrorsCounter: c2}, nil`)
**Span-per-call with RecordError on failure** — Ingest() opens a trace span, delegates to the inner collector, then calls span.RecordError + span.SetStatus on error before incrementing the errors counter. Always defer span.End(). (`ctx, span := c.tracer.Start(ctx, "openmeter.ingest.events", ...); defer span.End(); err = c.collector.Ingest(...); if err != nil { span.RecordError(err); ... }`)
**Metric naming convention openmeter.<domain>.<noun>** — Counter names follow openmeter.<domain>.<noun> (e.g. openmeter.ingest.events, openmeter.ingest.errors) with unit strings in UCUM {event} / {error} notation. (`metricMeter.Int64Counter("openmeter.ingest.events", metric.WithDescription(...), metric.WithUnit("{event}"))`)
**Attribute scoping by namespace** — Both counters and the span carry attribute.String("namespace", namespace) so metrics are filterable per tenant without additional label cardinality. (`namespaceAttr := attribute.String("namespace", namespace); c.ingestEventsCounter.Add(ctx, 1, metric.WithAttributes(namespaceAttr))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `telemetry.go` | Sole file. Defines collectorTelemetry struct, WithTelemetry constructor, and implements Ingest/Close. Metric counters are pre-allocated in the constructor, not per-call. | Counters must be created once in WithTelemetry and stored on the struct — metric.Meter.Int64Counter is not cheap to call on every event. Close() must delegate to the inner collector. |

## Anti-Patterns

- Creating metric instruments inside Ingest() instead of in the constructor — causes allocation and registration overhead on every event
- Swallowing the inner collector's error — always propagate it after recording the span error
- Adding business logic or event transformation here — this package must remain a pure observability decorator
- Omitting the namespace attribute from counters — tenant-scoped metrics are a hard requirement

## Decisions

- **Separate ingestadapter package instead of embedding telemetry in kafkaingest** — Keeps observability concerns decoupled from transport; any Collector implementation (kafkaingest, in-memory, test stub) can be wrapped without modification.

<!-- archie:ai-end -->
