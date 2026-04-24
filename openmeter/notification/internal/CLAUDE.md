# internal

<!-- archie:ai-start -->

> Internal test-helper package that generates synthetic notification event payloads (balance threshold, entitlement reset, invoice) for use in notification rule tests and test event dispatch — never used in production code paths.

## Patterns

**TestEventGenerator struct with injected billing.Service** — The sole exported type wraps billing.Service to generate realistic invoice payloads via billing.SimulateInvoice; entitlement payloads are built from hardcoded fixture values. (`func NewTestEventGenerator(billingService billing.Service) *TestEventGenerator { return &TestEventGenerator{billingService: billingService} }`)
**EventGeneratorInput.Validate() before switching on EventType** — Generate() validates the input struct before branching, surfacing empty namespace or event type errors early rather than panicking. (`func (t *TestEventGenerator) Generate(ctx context.Context, in EventGeneratorInput) (notification.EventPayload, error) { if err := in.Validate(); err != nil { return notification.EventPayload{}, err } switch in.EventType { ... } }`)
**Invoice payloads go through billing.SimulateInvoice + billinghttp.MapEventInvoiceToAPI** — Rather than constructing invoice API types by hand, newTestInvoicePayload calls the real billing simulation path and maps through the HTTP driver converter — keeping test fixtures aligned with production serialization. (`invoice, err := t.billingService.SimulateInvoice(ctx, billing.SimulateInvoiceInput{...}); eventInvoice, _ := billing.NewEventStandardInvoice(invoice); apiInvoice, _ := billinghttp.MapEventInvoiceToAPI(eventInvoice)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `rule.go` | Entire package. Defines TestEventGenerator, EventGeneratorInput, and three private payload builders for the three supported event types. | Entitlement/balance payloads use hardcoded ULID strings and timestamps — if notification payload schemas change, update the fixture literals here or test assertions will silently diverge. |

## Anti-Patterns

- Importing this package from production code — it is test-infrastructure only and imports billing HTTP driver types that have no place in the live dispatch path.
- Extending Generate() for a new event type without updating the EventPayloadMeta.Type field inside the returned payload — callers switch on Type and will misroute the payload.
- Replacing billing.SimulateInvoice with hand-crafted invoice structs — the simulation path exercises real serialization and keeps the test payload stable across invoice schema changes.

## Decisions

- **Use billing.SimulateInvoice for invoice test payloads instead of static fixtures** — Invoice API shape is complex and changes frequently; going through the real simulation + HTTP mapping ensures the test payload is always structurally valid.

<!-- archie:ai-end -->
