# asyncadvance

<!-- archie:ai-start -->

> Watermill event handler for async invoice advancement: receives billing.AdvanceStandardInvoiceEvent from the system Kafka topic and calls billing.Service.AdvanceInvoice once, treating ErrInvoiceCannotAdvance as a benign late-message condition. Requires the billing service configured with ForegroundAdvancementStrategy to avoid infinite retry loops.

## Patterns

**Config.Validate() guards strategy mismatch at construction** — New() calls Validate() which rejects any BillingService not using ForegroundAdvancementStrategy — prevents an infinite loop where background advancement re-emits the same event. (`if c.BillingService.GetAdvancementStrategy() != billing.ForegroundAdvancementStrategy { return errors.New("billing service must have foreground advancement strategy or we are creating an infinite loop") }`)
**Single-event Handle() matching the NoPublisherHandler closure** — Handler.Handle(ctx, *billing.AdvanceStandardInvoiceEvent) is the only exported method beyond New(), matching the worker router's expected closure signature. (`func (h *Handler) Handle(ctx context.Context, event *billing.AdvanceStandardInvoiceEvent) error`)
**ErrInvoiceCannotAdvance -> warn + nil** — Late Kafka messages for already-advanced invoices log a warning and return nil (no requeue); all other errors propagate. (`if errors.Is(err, billing.ErrInvoiceCannotAdvance) { h.logger.WarnContext(ctx, "invoice cannot advance (most probably a late message)", "invoice_id", event.Invoice); return nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `asyncadvance.go` | Entire package — Config, Config.Validate(), New(), Handler.Handle(). | Forgetting the strategy validation creates an infinite loop: background strategy re-emits advance events this handler re-processes forever; Handler is intentionally stateless beyond service + logger. |

## Anti-Patterns

- Wiring this handler with a BackgroundAdvancementStrategy BillingService — creates an infinite Kafka consume-emit loop.
- Returning ErrInvoiceCannotAdvance from Handle() — causes Watermill to nack and requeue indefinitely.
- Adding mutable state to Handler — the struct is intentionally stateless.
- Using context.Background() inside Handle() instead of the event's ctx parameter.

## Decisions

- **Strategy guard in Config.Validate() rather than a runtime check in Handle().** — Misconfiguration creates a hard-to-detect runtime infinite loop; failing fast at construction surfaces the bug on startup.

## Example: Wire asyncadvance handler into a Watermill grouphandler

```
import (
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/asyncadvance"
)
handler, err := asyncadvance.New(asyncadvance.Config{BillingService: foregroundBillingSvc, Logger: logger})
if err != nil { return err }
// grouphandler.NewGroupEventHandler(handler.Handle)
```

<!-- archie:ai-end -->
