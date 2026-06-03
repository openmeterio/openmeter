# internal

<!-- archie:ai-start -->

> Internal test-helper package that generates synthetic notification event payloads (balance threshold, entitlement reset, invoice) for notification rule tests and test event dispatch — never imported from production code paths.

## Patterns

**TestEventGenerator struct with injected billing.Service** — The sole exported type wraps billing.Service to generate realistic invoice payloads via billing.SimulateInvoice; entitlement payloads are built from hardcoded fixtures. (`func NewTestEventGenerator(billingService billing.Service) *TestEventGenerator { return &TestEventGenerator{billingService: billingService} }`)
**EventGeneratorInput.Validate() before switching on EventType** — Generate() validates the input before branching, surfacing empty namespace or event type errors early. (`if err := in.Validate(); err != nil { return notification.EventPayload{}, err }; switch in.EventType { ... }`)
**Invoice payloads via billing.SimulateInvoice + billinghttp.MapEventInvoiceToAPI** — newTestInvoicePayload calls the real billing simulation path and maps through the HTTP driver converter, keeping fixtures aligned with production serialization. (`invoice, _ := t.billingService.SimulateInvoice(ctx, billing.SimulateInvoiceInput{...}); eventInvoice, _ := billing.NewEventStandardInvoice(invoice); apiInvoice, _ := billinghttp.MapEventInvoiceToAPI(eventInvoice)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `rule.go` | Entire package. Defines TestEventGenerator, EventGeneratorInput, and three private payload builders (balance threshold, entitlement reset, invoice). | Entitlement/balance payloads use hardcoded ULID strings and timestamps — update fixture literals if payload schemas change. New event types in Generate() must set EventPayloadMeta.Type correctly or callers misroute. |

## Anti-Patterns

- Importing this package from production code — it pulls in billing HTTP driver types that have no place in live dispatch
- Extending Generate() for a new event type without setting EventPayloadMeta.Type inside the returned payload
- Replacing billing.SimulateInvoice with hand-crafted invoice structs — the simulation path keeps the payload structurally valid

## Decisions

- **Use billing.SimulateInvoice for invoice test payloads instead of static fixtures** — Invoice API shape is complex and changes often; the real simulation + HTTP mapping keeps the test payload always valid and aligned with production serialization.

<!-- archie:ai-end -->
