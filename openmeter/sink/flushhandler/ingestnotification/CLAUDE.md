# ingestnotification

<!-- archie:ai-start -->

> Post-flush handler that transforms successfully flushed SinkMessages into EventBatchedIngest domain events and publishes them to Watermill's balance topic, enabling the balance-worker to recalculate entitlement burn-down per subject. This is the bridge between the ClickHouse write path and downstream entitlement processing.

## Patterns

**FlushEventHandler wrapping** — The handler struct is never exposed directly as flushhandler.FlushEventHandler. NewHandler wraps the internal OnFlushSuccess callback using flushhandler.NewFlushEventHandler with an options struct that declares the handler name, callback, logger, and metric.Meter. (`flushhandler.NewFlushEventHandler(flushhandler.FlushEventHandlerOptions{Name: "ingest_notification", Callback: handler.OnFlushSuccess, Logger: logger, MetricMeter: metricMeter})`)
**HandlerConfig validation before construction** — HandlerConfig.Validate() is called in NewHandler before any field assignment. NewHandler returns (flushhandler.FlushEventHandler, error) and propagates validation errors — callers must check the error. (`if err := config.Validate(); err != nil { return nil, err }`)
**Subject-level grouping and deduplication before publish** — OnFlushSuccess groups EventBatchedIngest by namespace+subject key using lo.GroupBy, merges MeterSlugs with lo.Uniq, and chunks merged RawEvents to MaxEventsInBatch before publishing. Never publish one Kafka message per raw SinkMessage. (`iEventsBySubject := lo.GroupBy(iEvents, func(event ingestevents.EventBatchedIngest) string { return event.Namespace.ID + "/" + event.SubjectKey })`)
**MeterSlugs not MeterIDs in events** — getMeterSlugsFromMeters extracts meter.Meter.Key (the slug) not the ID. Downstream open-source consumers have no access to IDs; always use Key. (`slugs[i] = meter.Key`)
**Publish via eventbus.Publisher, not raw Watermill** — All event publishing goes through h.publisher.Publish(ctx, event) where publisher is eventbus.Publisher. Never import watermill message types directly in this package. (`if err := h.publisher.Publish(ctx, event); err != nil { finalErr = errors.Join(finalErr, err) }`)
**StoredAt set once per flush batch** — time.Now() is called once at the top of OnFlushSuccess and reused as StoredAt for all events in the batch, ensuring all events in the same flush have a consistent stored-at timestamp. (`now := time.Now() // ... StoredAt: now`)
**Nil Serialized guard before event construction** — Events are filtered with lo.Filter to exclude SinkMessages where Serialized == nil before mapping. Any code path that builds EventBatchedIngest must guard against nil Serialized. (`filtered := lo.Filter(events, func(event sinkmodels.SinkMessage, _ int) bool { return event.Serialized != nil })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Implements the ingest notification flush handler. Owns the grouping, merging, chunking, and publishing logic. NewHandler is the only public constructor. | MaxEventsInBatch must be validated before use; missing validation lets zero/negative values cause infinite loops in lo.Chunk. StoredAt is approximated as time.Now() after flush — it is not the actual ClickHouse write timestamp. |
| `events/events.go` | Defines EventBatchedIngest and its marshaler.Event compliance. EventVersionSubsystem is exported for use by balance-worker subscribers. | MeterSlugs must remain slugs not IDs. The var _ marshaler.Event compile-time assertion must be kept when adding new event structs. Event struct must stay a pure value type — no pointers to mutable shared state. |

## Anti-Patterns

- Publishing one Kafka message per SinkMessage instead of grouping by namespace+subject and merging MeterSlugs
- Using meter IDs instead of meter.Key (slugs) in EventBatchedIngest.MeterSlugs
- Calling time.Now() inside the per-event loop — StoredAt must be set once per flush batch for consistency
- Bypassing flushhandler.NewFlushEventHandler and returning the internal handler struct directly as flushhandler.FlushEventHandler
- Importing raw Watermill message types directly — always publish through eventbus.Publisher

## Decisions

- **Subject-level grouping with lo.GroupBy before publish** — Balance-worker processes recalculations per subject; batching all meter slugs for a subject into one message reduces Kafka message count and ensures atomicity of balance recalculation triggers.
- **MaxEventsInBatch chunking to cap RawEvents per message** — Kafka has a max message size limit; unbounded RawEvents accumulation per subject would exceed it on high-volume subjects. Chunking keeps messages within safe size bounds.
- **StoredAt approximated from time.Now() post-flush rather than per-event timestamps** — ClickHouse write completion time is the semantically correct stored-at for downstream balance recalculation. Using a single now per batch is a fair approximation and avoids per-row timestamp tracking overhead.

## Example: Adding a new FlushEventHandler in this package

```
import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/sink/flushhandler"
	sinkmodels "github.com/openmeterio/openmeter/openmeter/sink/models"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

func NewHandler(logger *slog.Logger, metricMeter metric.Meter, publisher eventbus.Publisher, config HandlerConfig) (flushhandler.FlushEventHandler, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
// ...
```

<!-- archie:ai-end -->
