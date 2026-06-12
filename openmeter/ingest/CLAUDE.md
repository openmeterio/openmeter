# ingest

<!-- archie:ai-start -->

> Usage-event intake side of the metering pipeline. The root defines ingest.Service and the Collector abstraction with two implementations/decorators (InMemoryCollector for tests, DeduplicatingCollector for production); subpackages publish to Kafka (kafkaingest), expose the v1 HTTP endpoint (httpdriver), and adapt collectors (ingestadapter).

## Patterns

**Collector is the swappable downstream sink** — Collector{ Ingest(ctx, namespace, event.Event) error; Close() } is the seam; implementations (in-memory, Kafka via ingestadapter) and decorators (DeduplicatingCollector) compose over it. (`type Collector interface { Ingest(ctx context.Context, namespace string, ev event.Event) error; Close() }`)
**Deduplication as a wrapping decorator** — DeduplicatingCollector embeds a Collector + dedupe.Deduplicator and only forwards events IsUnique reports as new — dedup is layered, not baked into a concrete collector. (`if isUnique { return d.Collector.Ingest(ctx, namespace, ev) }
return nil`)
**Service validates config and normalizes event time** — ingest.Config{Collector, Logger} has Validate (both required, no slog.Default fallback); processEvent forces UTC and defaults a zero timestamp to time.Now().UTC() before forwarding, logging with structured event fields. (`func NewService(config Config) (Service, error) { if err := config.Validate(); err != nil { return nil, err }; ... }`)
**CloudEvents as the wire type** — Events are github.com/cloudevents/sdk-go/v2 event.Event end to end; the service operates on event.Event and sets Time on it directly. (`if event.Time().IsZero() { event.SetTime(time.Now().UTC()) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `ingest.go` | Collector interface (the central abstraction) | All sinks and decorators must implement Ingest + Close; keep it minimal. |
| `service.go` | Service interface, Config (+Validate), service impl with per-event time normalization and structured logging | Logger is required (no slog.Default fallback); IngestEvents fails fast on the first event error. |
| `dedupe.go` | DeduplicatingCollector decorator over a Collector + dedupe.Deduplicator | Non-unique events are silently dropped (returns nil), not errored. |
| `inmemory.go` | InMemoryCollector test sink with lazy init + mutex, Events/Namespaces accessors | Test-only; events stored per-namespace map, not persisted or forwarded. |

## Anti-Patterns

- Baking deduplication or Kafka publishing into a concrete Collector instead of composing a decorator/adapter over the Collector interface.
- Constructing the service with a nil Logger or falling back to slog.Default() — Config.Validate forbids it.
- Forwarding events without normalizing Time to UTC (downstream metering assumes UTC event time).
- Using InMemoryCollector outside tests.

## Decisions

- **Ingestion is structured as a Collector interface with composable decorators.** — Lets the same service pipe events to in-memory (tests), Kafka (prod), and through deduplication without changing the ingest service.

## Example: Deduplicating before forwarding to the wrapped collector

```
func (d DeduplicatingCollector) Ingest(ctx context.Context, namespace string, ev event.Event) error {
	isUnique, err := d.Deduplicator.IsUnique(ctx, namespace, ev)
	if err != nil { return fmt.Errorf("checking event uniqueness: %w", err) }
	if isUnique { return d.Collector.Ingest(ctx, namespace, ev) }
	return nil
}
```

<!-- archie:ai-end -->
