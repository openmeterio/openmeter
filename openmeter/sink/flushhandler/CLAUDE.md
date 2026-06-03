# flushhandler

<!-- archie:ai-start -->

> Post-flush event-handler infrastructure for the sink worker: a buffered async callback framework (FlushEventHandler interface + flushEventHandler impl) and a fan-out multiplexer (FlushEventHandlers) that dispatches ClickHouse-flushed SinkMessages to downstream handlers. Primary constraint: graceful drain-before-exit so no committed-flush notifications are dropped on shutdown.

## Patterns

**NewFlushEventHandler with mandatory validation** — Construct every handler via NewFlushEventHandler(FlushEventHandlerOptions). Name, Callback, Logger, and MetricMeter are required (error otherwise); CallbackTimeout and DrainTimeout default to defaultCallbackTimeout (30s) when zero. (`handler, err := flushhandler.NewFlushEventHandler(flushhandler.FlushEventHandlerOptions{Name: "ingest-notification", Callback: cb, Logger: logger, MetricMeter: meter})`)
**Non-blocking Start launching an internal goroutine** — Start(ctx) must return immediately after go f.start(ctx). The internal loop defers drainDoneClose() so WaitForDrain unblocks after shutdown completes. (`func (f *flushEventHandler) Start(ctx context.Context) error { go f.start(ctx); return nil }`)
**Two-phase shutdown: Close then WaitForDrain** — Close() sets isShutdown (via Swap), signals stopChan, then closes the events channel under mu. WaitForDrain blocks on drainDone until the drain loop empties the buffered channel. Always call both in sequence before exit. (`handler.Close(); handler.WaitForDrain(ctx)`)
**FlushEventHandlers as the single fan-out entry point** — Compose multiple handlers via NewFlushEventHandlers() + AddHandler(). FlushEventHandlers itself implements FlushEventHandler so the sink worker holds one interface reference. Register post-drain callbacks with OnDrainComplete, not in Close. (`mux := flushhandler.NewFlushEventHandlers(); mux.AddHandler(ingestHandler); mux.OnDrainComplete(func() { close(done) })`)
**Trace parent span captured at Start, propagated into callback contexts** — parentSpan = trace.SpanFromContext(ctx) is captured once in start() and injected (trace.ContextWithSpan) into both the per-callback timeout context and the drain context, preserving trace linkage after the original ctx is cancelled. (`parentSpan := trace.SpanFromContext(ctx); ctx = trace.ContextWithSpan(ctx, parentSpan)`)
**mu guards OnFlushSuccess and eventsClose to prevent channel-close races** — OnFlushSuccess acquires mu before sending to events; Close acquires mu before eventsClose(). Never close the events channel outside this lock in a custom handler. eventsClose/stopChanClose/drainDoneClose are sync.OnceFunc to prevent double-close. (`f.mu.Lock(); defer f.mu.Unlock(); f.eventsClose()`)
**Per-handler OTel metrics under sink.flush_handler.<name>.*** — Each handler builds its own Int64Counter/Int64Histogram via newMetrics(name, meter): events_received, events_processed, events_failed, event_channel_full, event_processing_time_ms. Keep handler names lowercase alphanumeric+dash so metric names are well-formed. (`metrics, err := newMetrics(opts.Name, opts.MetricMeter)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `types.go` | Defines the FlushEventHandler interface (OnFlushSuccess, Start, WaitForDrain, Close) and the FlushCallback type alias — the sole contract every implementation satisfies. | Any new handler must implement all four methods, and Start must be non-blocking (launch a goroutine internally). |
| `handler.go` | Concrete async flushEventHandler: buffered events channel (size 1000), stopChan, drainDone, atomic isShutdown, and sync.Mutex preventing channel-close races; drain loop on a fresh context after the parent ctx is cancelled. | invokeCallbackWithTimeout uses context.Background()+CallbackTimeout (not the caller ctx) so callbacks can still reach external systems after cancellation. isShutdown.Swap guards double-close. OnFlushSuccess retries once when the channel is full before failing. |
| `mux.go` | FlushEventHandlers multiplexer fanning OnFlushSuccess/Start/Close/WaitForDrain to all registered handlers (errors.Join on fan-out) and running OnDrainComplete callbacks after all drain. Implements FlushEventHandler itself. | WaitForDrain runs onDrainComplete only after all handlers drain — register post-drain cleanup via OnDrainComplete, not Close. AddHandler must be called before Start (Start iterates handlers once). |
| `meters.go` | OTel metric initialisation for a single handler; metric names embed the handler name via fmt.Sprintf("sink.flush_handler.%s....", handlerName). | Handler names with special characters produce malformed metric names — keep names lowercase alphanumeric+dash. |

## Anti-Patterns

- Implementing FlushEventHandler with a blocking Start — it must launch a goroutine and return immediately
- Calling handler.Close() without handler.WaitForDrain() before process exit — in-flight buffered messages are dropped
- Adding handlers to FlushEventHandlers after Start() — Start iterates handlers once synchronously
- Closing the events channel outside the mu lock — data races with concurrent OnFlushSuccess callers
- Bypassing FlushEventHandlers and wiring a FlushEventHandler directly to the sink — breaks fan-out and drain ordering

## Decisions

- **Buffered channel (size 1000) with two-phase shutdown (stopChan signal + drain loop) rather than a simple WaitGroup** — OnFlushSuccess must not block the Kafka consumer hot path; the drain loop guarantees no committed-flush notifications are silently dropped on graceful shutdown.
- **FlushEventHandlers multiplexer implements FlushEventHandler itself** — The sink worker holds a single FlushEventHandler reference; adding or removing downstream handlers requires no change at the call site.
- **Trace parent span captured at Start and injected into background callback contexts** — Callbacks run after the HTTP/Kafka request context is cancelled; capturing the span once at Start preserves trace linkage without keeping the original ctx alive.

## Example: Register a post-flush handler and wire its full lifecycle through the multiplexer

```
import (
	"context"
	"github.com/openmeterio/openmeter/openmeter/sink/flushhandler"
	"github.com/openmeterio/openmeter/openmeter/sink/models"
)

mux := flushhandler.NewFlushEventHandlers()
handler, err := flushhandler.NewFlushEventHandler(flushhandler.FlushEventHandlerOptions{
	Name:        "my-handler",
	Callback:    func(ctx context.Context, msgs []models.SinkMessage) error { return nil },
	Logger:      logger,
	MetricMeter: meter,
})
if err != nil { return err }
mux.AddHandler(handler)
// ...
```

<!-- archie:ai-end -->
