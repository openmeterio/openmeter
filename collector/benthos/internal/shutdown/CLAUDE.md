# shutdown

<!-- archie:ai-start -->

> Provides the Signaller primitive for two-tier graceful shutdown (at-leisure then now) plus a HasClosed signal. Components own a Signaller and external orchestrators call CloseAtLeisure/CloseNow; components call ShutdownComplete when done.

## Patterns

**Two-tier shutdown: CloseAtLeisure then CloseNow** — CloseAtLeisure closes closeAtLeisureChan (finish current work, no new work). CloseNow also calls CloseAtLeisure then closes closeNowChan (stop immediately). Each is guarded by sync.Once. (`sig.CloseAtLeisure() // soft stop
sig.CloseNow()       // hard stop`)
**Chan-based and ctx-based wait helpers** — Each tier exposes *Chan() and *Ctx() variants. *Chan() returns a read-only channel suitable for select. *Ctx() wraps it in a context so callers can combine shutdown with other cancellation. (`ctx, cancel := sig.CloseNowCtx(parentCtx)
defer cancel()`)
**ShutdownComplete signals component termination** — The component calls sig.ShutdownComplete() once its goroutines have exited. Orchestrators wait on HasClosedChan() or HasClosedCtx() to confirm safe teardown. (`defer sig.ShutdownComplete()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `signaller.go` | Entire package: Signaller struct with CloseAtLeisure, CloseNow, ShutdownComplete and their *Chan / *Ctx / ShouldClose* accessors | CloseNow always calls CloseAtLeisure first — ShouldCloseAtLeisure is true whenever ShouldCloseNow is true; do not check only the Now tier |

## Anti-Patterns

- Forgetting to call ShutdownComplete — orchestrators will block forever waiting on HasClosedChan
- Adding a third shutdown tier — the two-tier model (leisure / now) is the intentional contract
- Using CloseAtLeisureCtx when you need CloseNowCtx — leisure ctx fires on either tier; now ctx fires only on the urgent tier

## Decisions

- **sync.Once guards each close channel** — Closing a closed channel panics; Once makes CloseAtLeisure and CloseNow safe to call multiple times from concurrent goroutines
- **Separate Signaller per component rather than a global shutdown signal** — Allows fine-grained teardown sequencing — orchestrator can drain outputs before closing inputs by shutting down Signallers in dependency order

<!-- archie:ai-end -->
