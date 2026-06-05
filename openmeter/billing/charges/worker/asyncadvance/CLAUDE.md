# asyncadvance

<!-- archie:ai-start -->

> Event-driven counterpart to advance/: a Watermill/Kafka-style Handler that advances charges for a single customer in response to one charges.AdvanceChargesEvent. It maps the event's Namespace/CustomerID into a charges.AdvanceChargesInput and delegates to charges.ChargeService.

## Patterns

**Config.Validate() + New constructor** — Config exposes a Validate() error method (returns errors.New on nil Logger / ChargesService); New(c) calls c.Validate() first and returns (*Handler, error). This is the validating-config idiom, distinct from advance/'s inline checks. (`func New(c Config) (*Handler, error) { if err := c.Validate(); err != nil { return nil, err }; ... }`)
**Single-event Handle delegating to service** — Handle(ctx, *charges.AdvanceChargesEvent) builds customer.CustomerID{Namespace, ID} from the event and calls chargesService.AdvanceCharges, discarding the result value and returning the error to let the consumer drive retry. (`h.chargesService.AdvanceCharges(ctx, charges.AdvanceChargesInput{Customer: customer.CustomerID{Namespace: event.Namespace, ID: event.CustomerID}})`)
**Return error for consumer-level retry** — On failure Handle logs WarnContext (not Error) with namespace/customer_id and returns the error so the message pipeline can retry; it does not swallow or aggregate. (`h.logger.WarnContext(ctx, "failed to advance charges", slog.String("namespace", event.Namespace), ...); return err`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `asyncadvance.go` | Defines Handler, Config (+Validate), New, and Handle(ctx, *charges.AdvanceChargesEvent). | Handle operates on exactly one customer per event — there is no batching here (that belongs in advance/). The error is returned (for retry), unlike a fire-and-forget consumer; keep it propagating. |

## Anti-Patterns

- Swallowing the error in Handle — the returned error is the message-bus retry signal; returning nil on failure silently drops advancement.
- Adding multi-customer/batch iteration here; per-customer batching lives in the sibling advance/ package.
- Constructing Handler directly without New/Validate, bypassing the nil-dependency guards.

## Decisions

- **Split async (per-event) advancement into its own package separate from advance/ (batch sweep).** — The two have different triggers and error semantics: the batch sweep aggregates and continues, the event handler propagates a single error for bus-level retry.

## Example: Advance charges for one customer from an AdvanceChargesEvent

```
func (h *Handler) Handle(ctx context.Context, event *charges.AdvanceChargesEvent) error {
	_, err := h.chargesService.AdvanceCharges(ctx, charges.AdvanceChargesInput{
		Customer: customer.CustomerID{Namespace: event.Namespace, ID: event.CustomerID},
	})
	if err != nil {
		h.logger.WarnContext(ctx, "failed to advance charges", slog.String("namespace", event.Namespace), slog.String("customer_id", event.CustomerID), slog.String("error", err.Error()))
		return err
	}
	return nil
}
```

<!-- archie:ai-end -->
