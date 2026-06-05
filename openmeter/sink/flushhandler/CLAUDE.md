# flushhandler

<!-- archie:ai-start -->

> Async, decoupling layer that runs side-effect callbacks after the sink successfully flushes a batch of usage events to ClickHouse. It owns the generic queue+drain machinery (flushEventHandler) and a fan-out multiplexer (FlushEventHandlers); the concrete flush effect (ingest notifications) lives in the ingestnotification child.

## Patterns

**Interface-first FlushEventHandler** — Everything is wired against the FlushEventHandler interface (OnFlushSuccess/Start/WaitForDrain/Close) in types.go. Concrete handlers and the multiplexer both assert `var _ FlushEventHandler = (*T)(nil)`. (`var _ FlushEventHandler = (*flushEventHandler)(nil)`)
**Constructor validates required deps then builds** — NewFlushEventHandler returns (FlushEventHandler, error) and rejects empty Name, nil Callback, nil Logger, nil MetricMeter before constructing; defaults applied for zero timeouts. No slog.Default() fallback. (`if opts.Logger == nil { return nil, errors.New("logger is required") }`)
**OnFlushSuccess is non-blocking and never re-does the write** — OnFlushSuccess only enqueues the batch onto the buffered events channel (defaultFlushChanSize=1000) and records metrics; the actual callback runs on the background goroutine. It must never reverse the already-committed ClickHouse write. (`case f.events <- event: f.metrics.eventsReceived.Add(ctx, 1)`)
**Background context with timeout for callbacks** — invokeCallbackWithTimeout deliberately uses context.Background() + CallbackTimeout (default 30s) so a canceled parent context still lets callbacks reach external systems; the parent trace span is re-attached via trace.ContextWithSpan. (`ctx, cancel := context.WithTimeout(context.Background(), f.callbackTimeout)`)
**Graceful drain on shutdown** — On ctx.Done/stopChan the loop calls Close() (closes events channel under mutex), then drains remaining batches with a fresh drainTimeout context before closing drainDone. WaitForDrain blocks until drainDone. (`for event := range f.events { f.invokeCallback(drainContext, event) }`)
**Idempotent close via sync.OnceFunc + atomic shutdown flag** — isShutdown (atomic.Bool) gates re-entry; all channel closes (eventsClose, stopChanClose, drainDoneClose) are wrapped in sync.OnceFunc so double-Close is safe. (`if f.isShutdown.Swap(true) { return nil }`)
**Per-handler namespaced metrics** — newMetrics builds Int64Counters/Histogram keyed by handler name (sink.flush_handler.<name>.events_received etc). New handlers get isolated metric names via opts.Name. (`meter.Int64Counter(fmt.Sprintf("sink.flush_handler.%s.events_received", handlerName))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `types.go` | Defines the FlushEventHandler interface and FlushCallback func type — the contract every handler and the mux satisfy. | Adding methods here forces updates to both flushEventHandler and FlushEventHandlers; keep the interface narrow. |
| `handler.go` | The generic async queue: NewFlushEventHandler, the start/drain goroutine loop, OnFlushSuccess enqueue, and timeout/trace handling. | events channel is closed under f.mu in Close(); OnFlushSuccess also takes f.mu — do not send on events outside this locking or you risk send-on-closed-channel. Callbacks intentionally run on context.Background(), not the request ctx. |
| `mux.go` | FlushEventHandlers fan-out: registers multiple handlers + OnDrainComplete hooks, joins per-handler errors on OnFlushSuccess/Close, runs onDrainComplete callbacks after all drains finish. | Start() short-circuits on the first error (returns immediately) whereas OnFlushSuccess/Close join all errors — the asymmetry is intentional. |
| `meters.go` | OTel metric set construction (eventsReceived/Processed/Failed, eventProcessingTime, eventChannelFull) namespaced by handler name. | Every counter creation can error; propagate it — don't ignore the err returns. |

## Anti-Patterns

- Doing the callback work synchronously inside OnFlushSuccess instead of enqueueing — it would block the sink's flush path.
- Returning an error from OnFlushSuccess that the caller treats as a flush failure, causing re-processing of already-committed ClickHouse writes.
- Using the inbound request ctx (or context.TODO) for the callback instead of the background-context+timeout pattern, breaking external calls when the parent is canceled.
- Sending on the events channel without holding f.mu, racing eventsClose() in Close() and panicking on send-on-closed-channel.
- Implementing FlushEventHandler from scratch in a downstream package instead of wrapping NewFlushEventHandler and supplying only a FlushCallback.

## Decisions

- **Run callbacks on a background goroutine fed by a buffered channel rather than inline in OnFlushSuccess.** — Keeps the sink's flush latency independent of downstream notification work and lets the sink commit to ClickHouse without waiting on Kafka/eventbus publishes.
- **Callbacks execute on context.Background() with an explicit CallbackTimeout; only the trace span is propagated.** — A canceled parent (shutdown/request cancel) must not abort an external side-effect mid-flight; the timeout bounds the work instead while traces stay linked.
- **Provide a FlushEventHandlers multiplexer that aggregates handlers and joins errors.** — Multiple independent post-flush effects (e.g. ingest notifications) can be registered and drained together without each implementing lifecycle plumbing.

## Example: Wrap a side-effect callback as a flush handler and register it in the mux

```
h, err := flushhandler.NewFlushEventHandler(flushhandler.FlushEventHandlerOptions{
    Name:        "ingest_notification",
    Logger:      logger,
    MetricMeter: metricMeter,
    Callback: func(ctx context.Context, msgs []models.SinkMessage) error {
        return publishIngestEvents(ctx, msgs)
    },
})
if err != nil { return err }
mux := flushhandler.NewFlushEventHandlers()
mux.AddHandler(h)
_ = mux.Start(ctx)
```

<!-- archie:ai-end -->
