# asyncadvance

<!-- archie:ai-start -->

> Event-driven (async) handler that advances charges for a single customer in response to a charges.AdvanceChargesEvent; intended to be wired as a Watermill message handler for the billing-worker.

## Patterns

**Config.Validate() before construction** — Config struct carries a Validate() error method that checks all required fields; New calls it before constructing the Handler. (`func New(c Config) (*Handler, error) {
    if err := c.Validate(); err != nil { return nil, err }
    return &Handler{chargesService: c.ChargesService, logger: c.Logger}, nil
}`)
**Single-responsibility Handle method** — Handler exposes exactly one Handle(ctx, *AdvanceChargesEvent) error method; no batch logic, no pagination — those live in the advance package. (`func (h *Handler) Handle(ctx context.Context, event *charges.AdvanceChargesEvent) error {
    _, err := h.chargesService.AdvanceCharges(ctx, charges.AdvanceChargesInput{Customer: customer.CustomerID{Namespace: event.Namespace, ID: event.CustomerID}})
    return err
}`)
**CustomerID constructed from event fields** — Build customer.CustomerID{Namespace: event.Namespace, ID: event.CustomerID} inline from the event — no helper, no global state. (`charges.AdvanceChargesInput{Customer: customer.CustomerID{Namespace: event.Namespace, ID: event.CustomerID}}`)
**Warn-then-return on service error** — Log a WarnContext entry with namespace/customer_id/error keys before returning the error so the upstream Watermill router can apply retry/dead-letter logic. (`h.logger.WarnContext(ctx, "failed to advance charges",
    slog.String("namespace", event.Namespace),
    slog.String("customer_id", event.CustomerID),
    slog.String("error", err.Error()),
)
return err`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `asyncadvance.go` | Sole file; defines Handler, Config (with Validate), New constructor, and Handle method. | Handler must not swallow errors — always return them so the message bus can apply retry/dead-letter logic. Never add batch or pagination logic here. |

## Anti-Patterns

- Adding batch/pagination logic here — that belongs in the advance package
- Calling Ent or adapter code directly instead of going through charges.ChargeService
- Returning nil on service error to suppress retries
- Introducing context.Background() instead of propagating the event's ctx
- Accumulating errors across multiple events — this handler processes one event per call

## Decisions

- **Separate package from synchronous advance** — Async (event-driven, single-customer) and sync (batch, all-customers) advancement have different error-handling and retry contracts; keeping them in separate packages avoids conflating them.
- **Errors returned, not swallowed** — Watermill's router applies retry and dead-letter semantics based on the handler's return value; swallowing errors would silently drop failed advancements.

## Example: Handle a single AdvanceChargesEvent from the event bus

```
import (
    "context"
    "log/slog"

    "github.com/openmeterio/openmeter/openmeter/billing/charges"
    "github.com/openmeterio/openmeter/openmeter/customer"
)

func (h *Handler) Handle(ctx context.Context, event *charges.AdvanceChargesEvent) error {
    _, err := h.chargesService.AdvanceCharges(ctx, charges.AdvanceChargesInput{
        Customer: customer.CustomerID{
            Namespace: event.Namespace,
            ID:        event.CustomerID,
        },
    })
// ...
```

<!-- archie:ai-end -->
