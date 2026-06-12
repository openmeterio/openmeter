# ingestadapter

<!-- archie:ai-start -->

> Adapter layer for the ingest pipeline that decorates an ingest.Collector with OpenTelemetry instrumentation. Its single export, WithTelemetry, wraps a collector to emit event/error counters and spans without altering ingest behavior.

## Patterns

**Decorator over ingest.Collector** — collectorTelemetry embeds a wrapped ingest.Collector and re-implements Ingest/Close, delegating to the inner collector and adding telemetry around the call. (`func WithTelemetry(collector ingest.Collector, metricMeter metric.Meter, tracer trace.Tracer) (ingest.Collector, error)`)
**Counters created in the constructor, returning error** — Int64Counters (openmeter.ingest.events, openmeter.ingest.errors) are created up front in WithTelemetry; any creation failure is wrapped with fmt.Errorf and returned rather than panicking. (`ingestEventsCounter, err := metricMeter.Int64Counter("openmeter.ingest.events", metric.WithDescription(...), metric.WithUnit("{event}")); if err != nil { return nil, fmt.Errorf("failed to create events counter: %w", err) }`)
**Span-and-count around each Ingest** — Ingest starts a span with namespace + event-id attributes, records error + increments the errors counter on failure, and increments the events counter on success, always tagging with the namespace attribute. (`ctx, span := c.tracer.Start(ctx, "openmeter.ingest.events", trace.WithAttributes(namespaceAttr, attribute.String("openmeter.event.id", ev.ID()))); defer span.End()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `telemetry.go` | Defines collectorTelemetry and WithTelemetry; the only adapter in this folder, instrumenting the Collector interface. | Preserve the Collector contract exactly — delegate to c.collector for both Ingest and Close. On error, record the span error AND increment ingestErrorsCounter before returning; on success increment ingestEventsCounter. Use the namespace attribute consistently on all metrics. |

## Anti-Patterns

- Mutating or dropping events in the decorator — telemetry must be transparent and only observe the inner Collector.
- Panicking on counter-creation failure instead of returning the wrapped error from WithTelemetry.
- Forgetting to call c.collector.Close() in Close(), leaking the wrapped collector.
- Adding business logic here; this adapter exists solely to attach OTel metrics/spans.

## Decisions

- **Telemetry is a separate adapter wrapping ingest.Collector rather than baked into the collector implementations.** — Keeps in-memory/dedupe/kafka collectors free of observability concerns and lets DI (app/common) opt into instrumentation by composition.

## Example: Wrapping a collector with OTel telemetry

```
func WithTelemetry(collector ingest.Collector, metricMeter metric.Meter, tracer trace.Tracer) (ingest.Collector, error) {
  ingestEventsCounter, err := metricMeter.Int64Counter("openmeter.ingest.events", metric.WithDescription("Number of events ingested"), metric.WithUnit("{event}"))
  if err != nil { return nil, fmt.Errorf("failed to create events counter: %w", err) }
  ingestErrorsCounter, err := metricMeter.Int64Counter("openmeter.ingest.errors", metric.WithDescription("Number of failed event ingests"), metric.WithUnit("{error}"))
  if err != nil { return nil, fmt.Errorf("failed to create errors counter: %w", err) }
  return &collectorTelemetry{collector: collector, tracer: tracer, ingestEventsCounter: ingestEventsCounter, ingestErrorsCounter: ingestErrorsCounter}, nil
}
```

<!-- archie:ai-end -->
