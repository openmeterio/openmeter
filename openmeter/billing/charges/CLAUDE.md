# charges

<!-- archie:ai-start -->

> Top-level domain contract package for the charges sub-system: defines the tagged-union Charge/ChargeIntent types (private discriminator accessed via AsFlatFeeCharge/AsUsageBasedCharge/AsCreditPurchaseCharge), the composite Service interface (ChargeService + CreditPurchaseFacadeService), and Adapter (ChargesSearchAdapter + TxCreator). All charge lifecycle must flow through charges.Service — never via sub-package adapters directly.

## Patterns

**Tagged-union construction via NewCharge[T]/NewChargeIntent[T]** — Charge and ChargeIntent hold a private `t meta.ChargeType` discriminator set only by NewCharge[T] / NewChargeIntent[T]. Accessors (AsFlatFeeCharge, AsUsageBasedCharge, AsCreditPurchaseCharge) return typed values or errors. Never use struct literals. (`charge := charges.NewCharge(flatfee.Charge{...})
ff, err := charge.AsFlatFeeCharge()`)
**Input.Validate() on every cross-boundary struct** — All Input types implement models.Validator. Validate() accumulates sub-errors via errors.Join and wraps with models.NewNillableGenericValidationError. Service implementations call Validate() before any business logic. (`if err := input.Validate(); err != nil { return nil, err }`)
**ValidationIssue sentinels declared in errors.go** — Package-level errors (ErrChargeNotFound, ErrChargeNamespaceEmpty, ErrChargeInvalid, ErrCreditRealizationsAlreadyAllocated) are models.ValidationIssue vars with ErrorCode constants, severity, and HTTP status via commonhttp.WithHTTPStatusCodeAttribute. Never use raw fmt.Errorf for these conditions. (`return charges.NewChargeNotFoundError(namespace, id)`)
**ChargeIntents.ByType() for per-type dispatch** — ByType() returns ChargeIntentsByType with pre-split slices (FlatFee, CreditPurchase, UsageBased each as []WithIndex[T]). Iterate ByType() output rather than switching on individual items in a loop. (`byType, err := intents.ByType()
for _, ff := range byType.FlatFee { /* ff.Value, ff.Index */ }`)
**NewLockKeyForCharge for per-charge advisory locks** — Per-charge advisory locks must be obtained via charges.NewLockKeyForCharge(chargeID), which validates the ID before constructing the lockr.Key. Never hand-construct lockr.Key strings inline. (`key, err := charges.NewLockKeyForCharge(chargeID)`)
**AdvanceChargesEvent for async dispatch** — Async charge advancement is triggered by publishing AdvanceChargesEvent (EventName uses metadata.GetEventName, not a raw string). Event carries Namespace + CustomerID; metadata source/subject via metadata.ComposeResourcePath. (`evt := charges.AdvanceChargesEvent{Namespace: ns, CustomerID: cid}
publisher.Publish(ctx, evt)`)
**Adapter exposes read-only ChargesSearchAdapter only** — charges.Adapter embeds ChargesSearchAdapter (GetByIDs, ListCharges, ListCustomersToAdvance — all read-only) plus entutils.TxCreator. Write operations belong to sub-package adapters invoked through the service orchestration layer, never called directly by callers. (`var _ charges.Adapter = (*adapter.Adapter)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines Service interface (ChargeService + CreditPurchaseFacadeService) and all Input types with Validate(). Primary contract for callers. | Input Validate() must accumulate all sub-errors via errors.Join and call sub-field Validate(); never short-circuit on first error. |
| `charge.go` | Defines Charge, ChargeIntent, Charges, ChargeIntents, ChargeIntentsByType, and WithIndex generic wrapper. All charge-type dispatch routes through switch cases here. | NewCharge/NewChargeIntent are the ONLY constructors — struct literal Charge{} leaves field `t` empty; all accessor methods will return errors. |
| `adapter.go` | Defines Adapter interface (ChargesSearchAdapter + TxCreator), ChargeSearchItem, ChargeSearchItems with Validate(). | ChargesSearchAdapter methods are read-only; writes must go through sub-package adapters via the service layer. |
| `errors.go` | All package-level ValidationIssue sentinels and ErrorCode constants with embedded HTTP status codes. | Adding a new domain error as fmt.Errorf instead of models.NewValidationIssue breaks HTTP status mapping in the error encoder chain (produces 500). |
| `patch.go` | Defines ApplyPatchesInput (CustomerID + Creates + PatchesByChargeID map) and ConcatenateApplyPatchesInputs. | PatchesByChargeID enforces one patch per charge ID; ConcatenateApplyPatchesInputs returns error on duplicate IDs — do not bypass with direct map merges. |
| `events.go` | AdvanceChargesEvent for async per-customer charge advancement via event bus. | EventName must use metadata.GetEventName; constructing the name string manually breaks topic routing in eventbus.GeneratePublishTopic. |
| `lock.go` | NewLockKeyForCharge produces the lockr.Key for per-charge advisory locking, with chargeID validation. | Always call NewLockKeyForCharge — raw string keys can collide across namespaces. |

## Anti-Patterns

- Constructing Charge{} or ChargeIntent{} with struct literals instead of NewCharge[T]/NewChargeIntent[T] — leaves discriminator field `t` empty, all accessor methods error.
- Bypassing charges.Service by calling flatfee/usagebased/creditpurchase sub-package adapters directly from app/common or billing/worker — breaks namespace lockdown check and TransactingRepo discipline.
- Returning raw fmt.Errorf instead of models.NewValidationIssue for domain error conditions — breaks HTTP status code mapping, produces 500.
- Adding business orchestration logic (gathering-line creation, auto-advance) to adapter.go — adapter is pure data-access; orchestration belongs in charges/service.
- Omitting Validate() on new Input types or skipping the call in service implementations — allows denormalized/invalid inputs to reach the adapter layer.

## Decisions

- **Charge and ChargeIntent are tagged unions (private discriminator, NewCharge[T] constructor) rather than interfaces or separate top-level types.** — Prevents callers from constructing partial charge values and forces all type dispatch through a single switch pattern, making coverage gaps visible when adding new charge types.
- **charges.Adapter only exposes read/search methods (ChargesSearchAdapter); write operations belong to sub-package adapters invoked through the service orchestration layer.** — Keeps the top-level package as a pure aggregation point with no write side-effects, ensuring all mutations flow through the service with namespace lockdown and TransactingRepo checks.
- **ValidationIssue sentinels with embedded HTTP status codes declared at package level in errors.go.** — Centralises HTTP status mapping so the error encoder chain detects them by type assertion; inline fmt.Errorf calls silently produce 500 responses.

## Example: Creating charges and dispatching async advancement

```
import (
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
)

intent := charges.NewChargeIntent(flatfee.Intent{ /* ... */ })
input := charges.CreateInput{
	Namespace: ns,
	Intents:   charges.ChargeIntents{intent},
}
if err := input.Validate(); err != nil {
	return err
}
created, err := chargeService.Create(ctx, input)
if err != nil {
// ...
```

<!-- archie:ai-end -->
