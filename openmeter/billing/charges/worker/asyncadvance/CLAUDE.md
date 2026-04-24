# asyncadvance

<!-- archie:ai-start -->

> Event-driven (async) handler that advances charges for a single customer in response to a `charges.AdvanceChargesEvent`; intended to be wired as a Watermill/event-bus message handler.

## Patterns

**Config.Validate() before construction** — Config struct carries a `Validate() error` method that checks all required fields; `New` calls it before constructing the Handler. (`func New(c Config) (*Handler, error) { if err := c.Validate(); err != nil { return nil, err } ... }`)
**Single-responsibility Handle method** — Handler exposes exactly one `Handle(ctx, *AdvanceChargesEvent) error` method; no batch logic, no pagination — those live in the advance package. (`func (h *Handler) Handle(ctx context.Context, event *charges.AdvanceChargesEvent) error { _, err := h.chargesService.AdvanceCharges(ctx, ...) }`)
**CustomerID constructed from event fields** — Build `customer.CustomerID{Namespace: event.Namespace, ID: event.CustomerID}` inline — no helper, no global state. (`charges.AdvanceChargesInput{Customer: customer.CustomerID{Namespace: event.Namespace, ID: event.CustomerID}}`)
**Warn-then-return on service error** — Log a WarnContext entry with namespace/customer_id/error keys before returning the error so the upstream router can decide on retry. (`h.logger.WarnContext(ctx, "failed to advance charges", slog.String("namespace", ...), slog.String("error", err.Error())); return err`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `asyncadvance.go` | Sole file; defines Handler, Config (with Validate), New constructor, and Handle method. | Handler does not discard errors — it returns them so the message bus can apply retry/dead-letter logic. Never swallow the error here. |

## Anti-Patterns

- Adding batch/pagination logic here — that belongs in the advance package
- Calling Ent or adapter code directly instead of charges.ChargeService
- Returning nil on service error to suppress retries
- Introducing context.Background() instead of propagating the event's ctx

## Decisions

- **Separate package from synchronous advance** — Async (event-driven, single-customer) and sync (batch, all-customers) advancement have different error-handling and retry contracts; keeping them in separate packages avoids conflating them.

## Example: Handle a single AdvanceChargesEvent

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
