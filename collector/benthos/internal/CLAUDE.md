# internal

<!-- archie:ai-start -->

> Internal utility packages shared by all collector/benthos plugins: a logr bridge (logging), a Batch/Transaction primitive for delivery-guaranteed message routing (message), and a two-tier graceful shutdown primitive (shutdown). None of these packages contain Benthos plugin registrations or business logic.

## Patterns

**logging bridge: Benthos logger to logr.LogSink** — logging.NewLogrLogger wraps *service.Logger as a logr.LogSink. Consumers call ctrllog.SetLogger exactly once at startup so controller-runtime and klog route through Benthos. WithValues/WithName are intentional no-ops. (`ctrlLogger := logging.NewLogrLogger(res.Logger())
ctrllog.SetLogger(ctrlLogger)`)
**Transaction-based delivery guarantee** — message.NewTransaction(batch, resChan) pairs a MessageBatch with a buffered chan<-error (cap 1). Server sends a Transaction into a channel; ReadBatch returns the batch and an AckFunc that sends on resChan. Ack must be called exactly once. (`resChan := make(chan error, 1)
in.transactions <- message.NewTransaction(batch, resChan)
res := <-resChan`)
**Two-tier shutdown: CloseAtLeisure then CloseNow** — Components own a *shutdown.Signaller. Orchestrators call CloseAtLeisure() then CloseNow(). Components call ShutdownComplete() when goroutine exits. Callers block on HasClosedChan(). (`defer in.shutSig.ShutdownComplete()
<-in.shutSig.CloseAtLeisureChan() // drain
<-in.shutSig.CloseNowChan()       // forced stop`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `internal/logging/logging.go` | Implements logr.LogSink backed by *service.Logger. Must be initialized once per component via SetupKlog. | WithValues and WithName are no-ops — do not rely on them for structured field propagation. Call SetupKlog before any k8s client-go or controller-runtime code runs. |
| `internal/message/transaction.go` | Defines message.Transaction holding a MessageBatch and a buffered chan<-error ack mechanism. Used by otel_log to bridge gRPC Export to ReadBatch. | Ack must be called exactly once. The channel is buffered (cap 1) to avoid blocking the sender if the receiver has already exited. |
| `internal/shutdown/signaller.go` | Provides shutdown.Signaller with two close tiers plus a HasClosed gate. | Forgetting ShutdownComplete() causes HasClosedChan to block forever. CloseAtLeisureCtx fires on either tier; CloseNowCtx fires only on CloseNow — use the right one for drain logic. |

## Anti-Patterns

- Adding business logic or Benthos plugin registrations to internal/ — it is a pure utilities layer.
- Calling logging.SetupKlog more than once — klog global state is not idempotent.
- Calling Transaction.Ack multiple times — the buffered channel send is not idempotent and will block or panic.
- Forgetting shutdown.Signaller.ShutdownComplete() in a component goroutine — orchestrators block on HasClosedChan indefinitely.
- Implementing WithValues/WithName in the logr bridge with real state — Benthos logger is format-string based; structured fields are not supported.

## Decisions

- **Three separate sub-packages (logging, message, shutdown) rather than a single internal package.** — Each primitive has a single responsibility and independent consumers; splitting them prevents accidental coupling and keeps import graphs minimal.
- **message.Transaction uses a buffered chan<-error (cap 1) for the ack path.** — Buffering allows the sender (gRPC handler) to send without blocking if the receiver (ReadBatch loop) has already moved on, avoiding a goroutine leak on timeout.

<!-- archie:ai-end -->
