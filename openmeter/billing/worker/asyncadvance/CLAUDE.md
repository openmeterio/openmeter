# asyncadvance

<!-- archie:ai-start -->

> Watermill event handler for async invoice advancement: receives a billing.AdvanceStandardInvoiceEvent from the system Kafka topic and calls billing.Service.AdvanceInvoice once, treating ErrInvoiceCannotAdvance as a benign late-message condition. Requires billing service configured with ForegroundAdvancementStrategy to prevent infinite retry loops.

## Patterns

**Config.Validate() guards strategy mismatch** — New() calls c.Validate() which rejects any BillingService not using ForegroundAdvancementStrategy — prevents infinite event loops where background advancement re-emits the same advance event. (`if c.BillingService.GetAdvancementStrategy() != billing.ForegroundAdvancementStrategy { return errors.New("billing service must have foreground advancement strategy...") }`)
**Handle() has a single-event signature** — Handler.Handle(ctx, *billing.AdvanceStandardInvoiceEvent) is the only exported method beyond New() — matches the Watermill NoPublisherHandler closure signature expected by the worker router. (`func (h *Handler) Handle(ctx context.Context, event *billing.AdvanceStandardInvoiceEvent) error`)
**ErrInvoiceCannotAdvance → warn + nil return** — Late Kafka messages for already-advanced invoices must log a warning and return nil (no requeue); all other errors propagate normally. (`if errors.Is(err, billing.ErrInvoiceCannotAdvance) { h.logger.WarnContext(...); return nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `asyncadvance.go` | Entire package — Config, Config.Validate(), New(), Handler.Handle(). No other files. | Forgetting the strategy validation in Validate() creates an infinite loop: background strategy re-emits advance events, which this handler re-processes forever. |

## Anti-Patterns

- Wiring this handler with a BillingService that has BackgroundAdvancementStrategy — Validate() will catch it at startup, not at runtime
- Returning ErrInvoiceCannotAdvance from Handle() — it will cause Watermill to nack and potentially requeue
- Adding state to Handler — the struct is intentionally stateless beyond service + logger

## Decisions

- **Strategy guard in Config.Validate() rather than a runtime check inside Handle()** — Misconfiguration (background strategy + this handler) creates an infinite loop that is hard to detect at runtime; failing fast at construction time is safer.

<!-- archie:ai-end -->
