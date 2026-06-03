# ingestnotification

<!-- archie:ai-start -->

> Post-flush bridge that transforms successfully flushed SinkMessages (handler.go) into EventBatchedIngest domain events (events/) and publishes them to Watermill's balance-worker Kafka topic, enabling downstream entitlement burn-down recalculation per subject. All logic is encapsulated behind the flushhandler.FlushEventHandler interface returned by NewHandler.

## Patterns

**FlushEventHandler wrapping via flushhandler.NewFlushEventHandler** — The internal handler struct is never returned directly. NewHandler always wraps OnFlushSuccess with flushhandler.NewFlushEventHandler using FlushEventHandlerOptions{Name, Callback, Logger, MetricMeter}; callers receive flushhandler.FlushEventHandler. (`return flushhandler.NewFlushEventHandler(flushhandler.FlushEventHandlerOptions{Name: "ingest_notification", Callback: handler.OnFlushSuccess, Logger: logger, MetricMeter: metricMeter})`)
**HandlerConfig.Validate() before construction** — NewHandler calls config.Validate() as its first action and propagates any error; MaxEventsInBatch <= 0 must be rejected. Never assign config fields before validation. (`if err := config.Validate(); err != nil { return nil, err }`)
**Subject-level grouping + lo.Uniq merge before publish** — OnFlushSuccess groups events by namespace+subjectKey via lo.GroupBy, merges MeterSlugs with lo.Uniq, and chunks RawEvents to MaxEventsInBatch via lo.Chunk. Never publish one Kafka message per raw SinkMessage. (`iEventsBySubject := lo.GroupBy(iEvents, func(event ingestevents.EventBatchedIngest) string { return event.Namespace.ID + "/" + event.SubjectKey })`)
**StoredAt set once per flush batch** — time.Now() is called once at the top of OnFlushSuccess and reused as StoredAt for all events; never call time.Now() inside the per-event loop. (`now := time.Now() // used as StoredAt for all events in the batch`)
**Nil Serialized guard before event construction** — SinkMessages are filtered with lo.Filter to exclude those where Serialized == nil before any mapping; any code path building EventBatchedIngest must apply this guard. (`filtered := lo.Filter(events, func(event sinkmodels.SinkMessage, _ int) bool { return event.Serialized != nil })`)
**Publish via eventbus.Publisher only** — All publishing goes through h.publisher.Publish(ctx, event) where publisher is eventbus.Publisher; never import raw Watermill message types in this package. (`if err := h.publisher.Publish(ctx, event); err != nil { finalErr = errors.Join(finalErr, err) }`)
**MeterSlugs use meter.Key not meter ID** — getMeterSlugsFromMeters extracts meter.Meter.Key (the slug); open-source downstream consumers have no access to internal IDs, so always use Key. (`slugs[i] = meter.Key`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Owns all grouping, merging, chunking, and publishing logic; NewHandler is the sole public constructor returning (flushhandler.FlushEventHandler, error). | MaxEventsInBatch must be validated before use — zero/negative values cause infinite loops in lo.Chunk; StoredAt is approximated as time.Now() after flush, not the actual ClickHouse write time; errors.Join accumulates per-event publish errors and continues publishing the rest. |
| `events/events.go` | Defines EventBatchedIngest and its marshaler.Event compliance; exports EventVersionSubsystem so balance-worker consumers subscribe by subsystem prefix. | MeterSlugs must remain slugs not IDs; keep the var _ marshaler.Event compile-time assertion when adding new event structs; the event must stay a pure value type — no pointers to mutable shared state. |

## Anti-Patterns

- Publishing one Kafka message per SinkMessage instead of grouping by namespace+subject and merging MeterSlugs with lo.Uniq.
- Using meter IDs instead of meter.Key (slug) in EventBatchedIngest.MeterSlugs — downstream open-source consumers cannot resolve IDs.
- Calling time.Now() inside the per-event loop — StoredAt must be set once per flush batch for consistency.
- Bypassing flushhandler.NewFlushEventHandler and returning the internal *handler struct directly.
- Importing raw Watermill message types directly in this package — always publish through eventbus.Publisher.

## Decisions

- **Subject-level grouping with lo.GroupBy before publish.** — Balance-worker processes recalculations per subject; batching all meter slugs for a subject into one Kafka message reduces message count and ensures atomicity of per-subject recalculation triggers.
- **MaxEventsInBatch chunking to cap RawEvents per Kafka message.** — Kafka enforces a max message size; unbounded RawEvents per subject on high-volume traffic would exceed it. Chunking keeps each message within safe bounds.
- **StoredAt approximated from time.Now() post-flush rather than per-event timestamps.** — ClickHouse write completion is the semantically correct stored-at for downstream balance recalculation; a single now per batch is a fair approximation avoiding per-row timestamp overhead.

## Example: NewHandler signature pattern for adding a FlushEventHandler in this package

```
import (
  "log/slog"
  "go.opentelemetry.io/otel/metric"
  "github.com/openmeterio/openmeter/openmeter/sink/flushhandler"
  "github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

func NewHandler(logger *slog.Logger, metricMeter metric.Meter, publisher eventbus.Publisher, config HandlerConfig) (flushhandler.FlushEventHandler, error) {
  if err := config.Validate(); err != nil { return nil, err }
  handler := &handler{publisher: publisher, logger: logger, config: config}
  return flushhandler.NewFlushEventHandler(flushhandler.FlushEventHandlerOptions{Name: "ingest_notification", Callback: handler.OnFlushSuccess, Logger: logger, MetricMeter: metricMeter})
}
```

<!-- archie:ai-end -->
