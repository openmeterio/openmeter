# asyncadvance

<!-- archie:ai-start -->

> Watermill/event consumer (package asyncadvance) for billing.AdvanceStandardInvoiceEvent. It synchronously advances one invoice per event by calling billing.Service.AdvanceInvoice, serving as the async counterpart to the batch advancer.

## Patterns

**Config.Validate guards against advancement-strategy infinite loop** — Config.Validate() requires Logger and BillingService non-nil AND asserts BillingService.GetAdvancementStrategy() == billing.ForegroundAdvancementStrategy — a background strategy would re-emit the same event and loop forever. (`if c.BillingService.GetAdvancementStrategy() != billing.ForegroundAdvancementStrategy { return errors.New("... infinite loop") }`)
**New() runs Validate before constructing Handler** — New(Config) calls c.Validate() and returns its error before building the Handler; never instantiate Handler directly. (`func New(c Config) (*Handler, error) { if err := c.Validate(); err != nil { return nil, err }; ... }`)
**Handle swallows ErrInvoiceCannotAdvance** — Handle returns nil (not an error) on billing.ErrInvoiceCannotAdvance, logging a WarnContext about a likely late/out-of-order message, so the consumer acks instead of redelivering forever. (`if errors.Is(err, billing.ErrInvoiceCannotAdvance) { h.logger.WarnContext(...); return nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `asyncadvance.go` | Entire package: Handler struct, Config + Validate + New constructor, and Handle(ctx, *billing.AdvanceStandardInvoiceEvent). | The ForegroundAdvancementStrategy invariant in Validate is load-bearing — removing it lets the handler re-emit the event it is consuming and spin an infinite loop. Handle ignores the returned invoice (`_, err := ...`); the event only carries event.Invoice (an InvoiceID). |

## Anti-Patterns

- Wiring this handler with a BillingService that uses a non-foreground advancement strategy — Validate exists specifically to reject this.
- Returning an error on ErrInvoiceCannotAdvance — it would cause endless message redelivery for an invoice that legitimately cannot advance yet.
- Bypassing New() and building Handler directly, skipping the strategy validation.

## Decisions

- **Enforce ForegroundAdvancementStrategy at construction time.** — AdvanceInvoice under a background strategy would publish another AdvanceStandardInvoiceEvent, which this same handler consumes — Validate makes the loop unconstructable rather than relying on runtime caution.

<!-- archie:ai-end -->
