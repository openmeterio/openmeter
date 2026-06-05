# internal

<!-- archie:ai-start -->

> Private helper package (imported only by httpdriver) hosting TestEventGenerator, which fabricates representative EventPayloads for the 'test rule' endpoint across all event types.

## Patterns

**TestEventGenerator dispatch by EventType** — Generate validates EventGeneratorInput then switches on EventType: balance-threshold and entitlement-reset return hardcoded fixtures; invoice types build a real payload via billingService.SimulateInvoice. (`switch in.EventType { case notification.EventTypeBalanceThreshold: return t.newTestBalanceThresholdPayload(), nil; ... }`)
**Balance payload derives from reset payload** — newTestBalanceThresholdPayload calls newTestEntitlementResetPayload, flips Type, and converts EntitlementReset into a BalanceThresholdPayload by casting EntitlementValuePayloadBase(*payload.EntitlementReset). (`payload.BalanceThreshold = &notification.BalanceThresholdPayload{EntitlementValuePayloadBase: notification.EntitlementValuePayloadBase(*payload.EntitlementReset), Threshold: ...}`)
**Invoice fixture uses real billing simulation** — newTestInvoicePayload constructs a customer + flat-fee line, calls billingService.SimulateInvoice, wraps via billing.NewEventStandardInvoice, then maps with billinghttp.MapEventInvoiceToAPI — exercising the real billing path rather than a stub. (`invoice, err := t.billingService.SimulateInvoice(ctx, billing.SimulateInvoiceInput{...}); eventInvoice, _ := billing.NewEventStandardInvoice(invoice)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `rule.go` | TestEventGenerator{billingService}, NewTestEventGenerator, EventGeneratorInput{Validate}, Generate + per-type fixture builders | Invoice fixtures require a live billing.Service (SimulateInvoice); the entitlement/feature/subject fixtures are hardcoded ULIDs/values — adding a new event type needs a new case in Generate or it errors 'unsupported event type'. |

## Anti-Patterns

- Importing this package outside httpdriver (it is intentionally internal)
- Hand-rolling an invoice fixture instead of going through billingService.SimulateInvoice
- Adding an EventType to the system without a Generate case (TestRule will fail)

## Decisions

- **Invoice test payloads use real SimulateInvoice rather than canned JSON** — Guarantees the test event's invoice shape matches what production billing actually emits, so channel integrations can be validated end-to-end.

## Example: Generating a type-specific test event payload

```
func (t *TestEventGenerator) Generate(ctx context.Context, in EventGeneratorInput) (notification.EventPayload, error) {
	if err := in.Validate(); err != nil { return notification.EventPayload{}, err }
	switch in.EventType {
	case notification.EventTypeBalanceThreshold:
		return t.newTestBalanceThresholdPayload(), nil
	case notification.EventTypeInvoiceCreated, notification.EventTypeInvoiceUpdated:
		return t.newTestInvoicePayload(ctx, in.Namespace, in.EventType)
	default:
		return notification.EventPayload{}, fmt.Errorf("unsupported event type: %s", in.EventType)
	}
}
```

<!-- archie:ai-end -->
