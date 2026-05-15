# asyncadvance

<!-- archie:ai-start -->

> Watermill event handler for async invoice advancement: receives a billing.AdvanceStandardInvoiceEvent from the system Kafka topic and calls billing.Service.AdvanceInvoice once, treating ErrInvoiceCannotAdvance as a benign late-message condition. Requires billing service configured with ForegroundAdvancementStrategy to prevent infinite retry loops.

## Patterns

**Config.Validate() guards strategy mismatch at construction time** — New() calls c.Validate() which rejects any BillingService not using ForegroundAdvancementStrategy — prevents infinite event loops where background advancement re-emits the same advance event. (`if c.BillingService.GetAdvancementStrategy() != billing.ForegroundAdvancementStrategy { return errors.New("billing service must have foreground advancement strategy or we are creating an infinite loop") }`)
**Single-event Handle() signature matching Watermill NoPublisherHandler closure** — Handler.Handle(ctx, *billing.AdvanceStandardInvoiceEvent) is the only exported method beyond New() — matches the expected closure signature for the worker router's NoPublisherHandler. (`func (h *Handler) Handle(ctx context.Context, event *billing.AdvanceStandardInvoiceEvent) error`)
**ErrInvoiceCannotAdvance → warn + nil return** — Late Kafka messages for already-advanced invoices must log a warning and return nil (no requeue); all other errors propagate normally. (`if errors.Is(err, billing.ErrInvoiceCannotAdvance) { h.logger.WarnContext(ctx, "invoice cannot advance (most probably a late message has occurred)", "invoice_id", event.Invoice); return nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `asyncadvance.go` | Entire package — Config, Config.Validate(), New(), Handler.Handle(). No other files. | Forgetting the strategy validation in Validate() creates an infinite loop: background strategy re-emits advance events, which this handler re-processes forever. Handler is intentionally stateless beyond service + logger — do not add mutable state. |

## Anti-Patterns

- Wiring this handler with a BillingService that has BackgroundAdvancementStrategy — Validate() catches it at startup; missing the check creates an infinite Kafka consume-emit loop
- Returning ErrInvoiceCannotAdvance from Handle() — causes Watermill to nack and potentially requeue indefinitely
- Adding mutable state to Handler — the struct is intentionally stateless beyond service + logger
- Using context.Background() inside Handle() instead of the event's ctx parameter

## Decisions

- **Strategy guard in Config.Validate() rather than a runtime check inside Handle()** — Misconfiguration (background strategy + this handler) creates an infinite loop that is hard to detect at runtime; failing fast at construction time is safer and surfaces the bug immediately on startup.

## Example: Wire asyncadvance handler into a Watermill NoPublisherHandler

```
import (
    "github.com/openmeterio/openmeter/openmeter/billing"
    "github.com/openmeterio/openmeter/openmeter/billing/worker/asyncadvance"
)

handler, err := asyncadvance.New(asyncadvance.Config{
    BillingService: foregroundBillingSvc, // must be ForegroundAdvancementStrategy
    Logger:         logger,
})
if err != nil {
    return err
}
// Register with watermill grouphandler:
// grouphandler.NewGroupEventHandler(handler.Handle)
```

<!-- archie:ai-end -->
