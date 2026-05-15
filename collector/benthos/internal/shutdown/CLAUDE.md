# shutdown

<!-- archie:ai-start -->

> Provides the Signaller primitive for two-tier graceful shutdown (at-leisure then now) plus a HasClosed signal. Components own a Signaller; orchestrators call CloseAtLeisure/CloseNow; components call ShutdownComplete when done.

## Patterns

**Two-tier shutdown: CloseAtLeisure then CloseNow** — CloseAtLeisure closes closeAtLeisureChan (finish current work, start no new work). CloseNow calls CloseAtLeisure first then closes closeNowChan (stop immediately). Each is guarded by sync.Once so concurrent calls are safe. (`sig.CloseAtLeisure() // soft stop
sig.CloseNow()       // hard stop — always calls CloseAtLeisure first`)
**Chan-based and ctx-based wait helpers per tier** — Each tier exposes *Chan() returning a read-only channel for select, and *Ctx() wrapping it in a context. Use *Ctx() when combining shutdown with other cancellation sources. (`ctx, cancel := sig.CloseNowCtx(parentCtx)
defer cancel()`)
**ShutdownComplete signals component termination** — The component calls sig.ShutdownComplete() once its goroutines have exited. Orchestrators wait on HasClosedChan() or HasClosedCtx() to confirm safe teardown. Forgetting this call blocks the orchestrator forever. (`defer sig.ShutdownComplete()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `signaler.go` | Entire package: Signaller struct with CloseAtLeisure, CloseNow, ShutdownComplete and their *Chan/*Ctx/ShouldClose* accessors. | CloseNow always calls CloseAtLeisure first — ShouldCloseAtLeisure is true whenever ShouldCloseNow is true; checking only the Now tier misses the leisure signal. |

## Anti-Patterns

- Forgetting to call ShutdownComplete — orchestrators block forever waiting on HasClosedChan
- Adding a third shutdown tier — the two-tier model (leisure / now) is the intentional contract
- Using CloseAtLeisureCtx when CloseNowCtx is needed — leisure ctx fires on either tier; now ctx fires only on the urgent tier
- Closing shutdown channels directly — always call CloseAtLeisure/CloseNow; direct close panics on repeat calls

## Decisions

- **sync.Once guards each close channel** — Closing a closed channel panics; Once makes CloseAtLeisure and CloseNow safe to call multiple times from concurrent goroutines
- **Separate Signaller per component rather than a global shutdown signal** — Allows fine-grained teardown sequencing — orchestrator can drain outputs before closing inputs by shutting down Signallers in dependency order

## Example: Component using Signaller for graceful shutdown

```
import "github.com/openmeterio/openmeter/collector/benthos/internal/shutdown"

sig := shutdown.NewSignaller()
go func() {
	defer sig.ShutdownComplete()
	for {
		select {
		case <-sig.CloseAtLeisureChan():
			// finish in-flight work, then return
			return
		}
	}
}()
// orchestrator triggers and waits
sig.CloseNow()
// ...
```

<!-- archie:ai-end -->
