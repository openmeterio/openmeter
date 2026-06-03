# internal

<!-- archie:ai-start -->

> Internal utility packages shared by all collector/benthos plugins: logging (a Benthos-logger -> logr.LogSink bridge), message (Batch alias + Transaction delivery-guarantee primitive), and shutdown (a two-tier graceful-shutdown Signaller). None of these contain Benthos plugin registrations or business logic.

## Patterns

**logging bridge: Benthos logger to logr.LogSink** — logging.NewLogrLogger wraps *service.Logger as a logr.LogSink; consumers call ctrllog.SetLogger once at startup so controller-runtime and klog route through Benthos. WithValues/WithName are intentional no-ops. (`ctrlLogger := logging.NewLogrLogger(res.Logger()); ctrllog.SetLogger(ctrlLogger)`)
**Transaction-based delivery guarantee** — message.NewTransaction(batch, resChan) pairs a MessageBatch with a buffered chan<-error (cap 1); a sender sends a Transaction into a channel and ReadBatch returns the batch with an AckFunc that sends on resChan. Ack must be called exactly once. (`resChan := make(chan error, 1); in.transactions <- message.NewTransaction(batch, resChan); res := <-resChan`)
**Two-tier shutdown: CloseAtLeisure then CloseNow** — Components own a *shutdown.Signaller; orchestrators call CloseAtLeisure() then CloseNow(); components call ShutdownComplete() on goroutine exit; callers block on HasClosedChan(). (`defer in.shutSig.ShutdownComplete()
<-in.shutSig.CloseAtLeisureChan() // drain
<-in.shutSig.CloseNowChan()       // forced stop`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `logging/logging.go` | logr.LogSink backed by *service.Logger; initialize once per component via SetupKlog. | WithValues/WithName are no-ops — do not rely on them for structured fields; call SetupKlog before any client-go/controller-runtime code runs. |
| `message/transaction.go` | message.Transaction holding a MessageBatch and a buffered chan<-error ack; used by otel_log to bridge gRPC Export to ReadBatch. | Ack must be called exactly once; the channel is buffered (cap 1) to avoid blocking the sender if the receiver already exited. |
| `shutdown/signaller.go` | shutdown.Signaller with two close tiers plus a HasClosed gate. | Forgetting ShutdownComplete() blocks HasClosedChan forever; CloseAtLeisureCtx fires on either tier, CloseNowCtx only on CloseNow — pick the right one for drain logic. |

## Anti-Patterns

- Adding business logic or Benthos plugin registrations to internal/ — it is a pure utilities layer.
- Calling logging.SetupKlog more than once — klog global state is not idempotent.
- Calling Transaction.Ack multiple times — the buffered channel send is not idempotent and will block or panic.
- Forgetting shutdown.Signaller.ShutdownComplete() in a component goroutine — orchestrators block on HasClosedChan indefinitely.
- Implementing WithValues/WithName in the logr bridge with real state — the Benthos logger is format-string based; structured fields are not supported.

## Decisions

- **Three separate sub-packages (logging, message, shutdown) rather than one internal package.** — Each primitive has a single responsibility and independent consumers; splitting prevents accidental coupling and keeps import graphs minimal.
- **message.Transaction uses a buffered chan<-error (cap 1) for the ack path.** — Buffering lets the sender (gRPC handler) send without blocking if the receiver (ReadBatch loop) already moved on, avoiding a goroutine leak on timeout.

<!-- archie:ai-end -->
