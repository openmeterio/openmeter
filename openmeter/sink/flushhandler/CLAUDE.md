# flushhandler

<!-- archie:ai-start -->

> Provides the post-flush event handler infrastructure for the sink worker: a buffered async callback framework (FlushEventHandler interface + flushEventHandler impl) and a fan-out multiplexer (FlushEventHandlers) that dispatches ClickHouse-flushed SinkMessages to downstream handlers like ingestnotification. The primary constraint is graceful shutdown with drain-before-exit semantics.

## Patterns

**NewFlushEventHandler constructor with mandatory validation** — All FlushEventHandler instances are created via NewFlushEventHandler(FlushEventHandlerOptions). Name, Callback, Logger, and MetricMeter are required; missing any returns an error. CallbackTimeout and DrainTimeout default to defaultCallbackTimeout if zero. (`handler, err := flushhandler.NewFlushEventHandler(flushhandler.FlushEventHandlerOptions{Name: "ingest-notification", Callback: cb, Logger: logger, MetricMeter: meter})`)
**Buffered async channel with two-phase shutdown** — OnFlushSuccess enqueues to a buffered channel (size 1000). Close() signals stopChan and closes events channel under mu. Start() drains the queue after shutdown via a separate context with DrainTimeout. WaitForDrain blocks until drainDone is closed. (`handler.Start(ctx); /* flush loop calls */ handler.OnFlushSuccess(ctx, msgs); /* shutdown: */ handler.Close(); handler.WaitForDrain(ctx)`)
**FlushEventHandlers as the fan-out multiplexer** — Use NewFlushEventHandlers() + AddHandler() to compose multiple FlushEventHandler implementations. FlushEventHandlers itself implements FlushEventHandler so callers hold a single interface. OnDrainComplete registers post-drain callbacks. (`mux := flushhandler.NewFlushEventHandlers(); mux.AddHandler(ingestHandler); mux.OnDrainComplete(func() { close(done) })`)
**OTel metrics per handler name** — Each handler creates its own counters and histogram prefixed with sink.flush_handler.<name>.* via newMetrics. Metric names are deterministic: events_received, events_processed, events_failed, event_channel_full, event_processing_time_ms. (`metrics, err := newMetrics(opts.Name, opts.MetricMeter)`)
**Trace context propagation through drain** — parentSpan is captured from the Start ctx and propagated into both the callback timeout context and the drain context via trace.ContextWithSpan, so callback spans are linked to the parent trace even after the original ctx is cancelled. (`parentSpan := trace.SpanFromContext(ctx); drainContext = trace.ContextWithSpan(drainContext, parentSpan)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `types.go` | Defines FlushEventHandler interface (OnFlushSuccess, Start, WaitForDrain, Close) and FlushCallback type alias. This is the only contract new implementations must satisfy. | Any new handler must implement all four methods; Start must be non-blocking (launch goroutine internally). |
| `handler.go` | Concrete async implementation of FlushEventHandler. Manages a buffered events channel, stopChan, drainDone, atomic isShutdown flag, and a sync.Mutex to prevent channel close races. | mu guards both OnFlushSuccess and eventsClose — never close the channel outside that lock. isShutdown.Swap prevents double-close of stopChan/events. |
| `mux.go` | FlushEventHandlers multiplexer that fans out to all registered handlers and serialises their lifecycle (Start, Close, WaitForDrain). Implements FlushEventHandler itself. | WaitForDrain calls onDrainComplete callbacks after all handlers drain — register post-drain cleanup here, not in Close. |
| `meters.go` | OTel metric initialisation for a single handler instance. Called once from NewFlushEventHandler; metric names embed the handler name. | If handler name contains special characters the metric name will be malformed — keep names lowercase alphanumeric+dash. |

## Anti-Patterns

- Implementing FlushEventHandler without making Start non-blocking — it must launch a goroutine and return immediately.
- Calling handler.Close() without handler.WaitForDrain() before process exit — in-flight messages in the buffer will be dropped.
- Adding handlers to FlushEventHandlers after Start() has been called — Start iterates handlers once synchronously.
- Closing the events channel outside the mu lock in a custom handler — causes data races with concurrent OnFlushSuccess callers.
- Bypassing FlushEventHandlers and wiring FlushEventHandler implementations directly to the sink — breaks fan-out and drain ordering.

## Decisions

- **Buffered channel (size 1000) with a two-phase shutdown (stopChan signal + drain loop) rather than a simple WaitGroup** — The sink flush is on the hot path; OnFlushSuccess must not block the Kafka consumer. The drain loop ensures no committed-flush notifications are silently dropped on graceful shutdown.
- **FlushEventHandlers multiplexer implements FlushEventHandler itself** — Callers (sink worker) hold a single FlushEventHandler reference; adding or removing downstream handlers requires no changes at the call site.
- **Trace parent span captured at Start() and injected into background callback contexts** — Callbacks run after the HTTP/Kafka request context is cancelled; capturing the span once at Start() preserves trace linkage without keeping the original ctx alive.

## Example: Registering a new post-flush handler and wiring lifecycle

```
import (
	"github.com/openmeterio/openmeter/openmeter/sink/flushhandler"
	"github.com/openmeterio/openmeter/openmeter/sink/models"
)

mux := flushhandler.NewFlushEventHandlers()

handler, err := flushhandler.NewFlushEventHandler(flushhandler.FlushEventHandlerOptions{
	Name:        "my-handler",
	Callback:    func(ctx context.Context, msgs []models.SinkMessage) error { /* ... */ return nil },
	Logger:      logger,
	MetricMeter: meter,
})
if err != nil { return err }

// ...
```

<!-- archie:ai-end -->
