# flushhandler

<!-- archie:ai-start -->

> Provides the post-flush event handler infrastructure for the sink worker: a buffered async callback framework (FlushEventHandler interface + flushEventHandler impl) and a fan-out multiplexer (FlushEventHandlers) that dispatches ClickHouse-flushed SinkMessages to downstream handlers. The primary constraint is graceful drain-before-exit semantics so no committed-flush notifications are dropped on shutdown.

## Patterns

**NewFlushEventHandler with mandatory validation** — All FlushEventHandler instances must be created via NewFlushEventHandler(FlushEventHandlerOptions). Name, Callback, Logger, and MetricMeter are required; missing any returns an error. CallbackTimeout and DrainTimeout default to defaultCallbackTimeout (30s) if zero. (`handler, err := flushhandler.NewFlushEventHandler(flushhandler.FlushEventHandlerOptions{Name: "ingest-notification", Callback: cb, Logger: logger, MetricMeter: meter})`)
**Non-blocking Start that launches internal goroutine** — Start(ctx) must return immediately after launching the internal event loop goroutine. The goroutine calls drainDoneClose() via defer so WaitForDrain unblocks after shutdown completes. (`func (f *flushEventHandler) Start(ctx context.Context) error { go f.start(ctx); return nil }`)
**Two-phase shutdown: Close then WaitForDrain** — Close() sets isShutdown, signals stopChan, then closes the events channel under mu. WaitForDrain blocks on drainDone until the drain loop empties the buffered channel. Always call both in sequence before process exit. (`handler.Close(); handler.WaitForDrain(ctx)`)
**FlushEventHandlers as the single fan-out entry point** — Compose multiple FlushEventHandler implementations via NewFlushEventHandlers() + AddHandler(). FlushEventHandlers itself implements FlushEventHandler, so callers (sink worker) hold a single interface reference. Register post-drain callbacks with OnDrainComplete, not in Close. (`mux := flushhandler.NewFlushEventHandlers(); mux.AddHandler(ingestHandler); mux.OnDrainComplete(func() { close(done) })`)
**Trace parent span captured at Start and propagated into callback contexts** — parentSpan is captured from the Start ctx via trace.SpanFromContext and injected into both the callback timeout context and the drain context via trace.ContextWithSpan, preserving trace linkage after the original ctx is cancelled. (`parentSpan := trace.SpanFromContext(ctx); drainContext = trace.ContextWithSpan(drainContext, parentSpan)`)
**mu guards both OnFlushSuccess and eventsClose to prevent channel close races** — OnFlushSuccess acquires mu before sending to the events channel. Close acquires mu before calling eventsClose(). Never close the events channel outside this lock in a custom handler. (`f.mu.Lock(); defer f.mu.Unlock(); f.eventsClose()`)
**OTel metrics per handler name using sink.flush_handler.<name>.* prefix** — Each handler creates its own Int64Counter and Int64Histogram instruments via newMetrics(name, meter). Metric names are deterministic: events_received, events_processed, events_failed, event_channel_full, event_processing_time_ms. Keep handler names lowercase alphanumeric+dash. (`metrics, err := newMetrics(opts.Name, opts.MetricMeter)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `types.go` | Defines the FlushEventHandler interface (OnFlushSuccess, Start, WaitForDrain, Close) and FlushCallback type alias. This is the sole contract all implementations must satisfy. | Any new handler must implement all four methods; Start must be non-blocking (launch goroutine internally). |
| `handler.go` | Concrete async implementation of FlushEventHandler. Manages a buffered events channel (size 1000), stopChan, drainDone, atomic isShutdown flag, and sync.Mutex to prevent channel close races. | mu guards both OnFlushSuccess and eventsClose — never close the channel outside that lock. isShutdown.Swap prevents double-close of stopChan/events. invokeCallbackWithTimeout uses context.Background() + CallbackTimeout, not the caller ctx. |
| `mux.go` | FlushEventHandlers multiplexer that fans out OnFlushSuccess/Start/Close/WaitForDrain to all registered handlers and serialises their lifecycle. Implements FlushEventHandler itself. | WaitForDrain calls onDrainComplete callbacks only after all handlers have drained — register post-drain cleanup via OnDrainComplete, not in Close. AddHandler must be called before Start. |
| `meters.go` | OTel metric initialisation for a single handler instance. Called once from NewFlushEventHandler; metric names embed the handler name. | Handler names with special characters produce malformed metric names — keep names lowercase alphanumeric+dash. |

## Anti-Patterns

- Implementing FlushEventHandler without making Start non-blocking — it must launch a goroutine and return immediately.
- Calling handler.Close() without handler.WaitForDrain() before process exit — in-flight messages in the buffer will be dropped.
- Adding handlers to FlushEventHandlers after Start() has been called — Start iterates handlers once synchronously.
- Closing the events channel outside the mu lock — causes data races with concurrent OnFlushSuccess callers.
- Bypassing FlushEventHandlers and wiring FlushEventHandler implementations directly to the sink — breaks fan-out and drain ordering.

## Decisions

- **Buffered channel (size 1000) with two-phase shutdown (stopChan signal + drain loop) rather than a simple WaitGroup** — OnFlushSuccess must not block the Kafka consumer hot path; the drain loop ensures no committed-flush notifications are silently dropped on graceful shutdown.
- **FlushEventHandlers multiplexer implements FlushEventHandler itself** — Callers (sink worker) hold a single FlushEventHandler reference; adding or removing downstream handlers requires no changes at the call site.
- **Trace parent span captured at Start and injected into background callback contexts** — Callbacks run after the HTTP/Kafka request context is cancelled; capturing the span once at Start preserves trace linkage without keeping the original ctx alive.

## Example: Registering a new post-flush handler and wiring its full lifecycle

```
import (
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

// ...
```

<!-- archie:ai-end -->
