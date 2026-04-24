# charges

<!-- archie:ai-start -->

> Top-level domain package for the charges sub-system: defines the tagged-union Charge/ChargeIntent types spanning flatfee, usagebased, and creditpurchase, the composite Service and Adapter interfaces, all cross-cutting input types with Validate(), and the AdvanceChargesEvent. It is the contract boundary consumed by app/common, billing/worker, and v3 API handlers — all charge lifecycle must flow through charges.Service, never via sub-package adapters directly.

## Patterns

**Tagged-union Charge/ChargeIntent via private discriminator** — Charge and ChargeIntent hold a private `t meta.ChargeType` field; construction is exclusively via NewCharge[T] / NewChargeIntent[T]. Accessors (AsFlatFeeCharge, AsUsageBasedCharge, AsCreditPurchaseCharge) return typed values or errors — never nil-check the inner pointer directly. (`charge := charges.NewCharge(flatfee.Charge{...}); ff, err := charge.AsFlatFeeCharge()`)
**Input.Validate() on every cross-boundary struct** — All Input types (CreateInput, GetByIDInput, AdvanceChargesInput, ListChargesInput, ApplyPatchesInput, etc.) implement models.Validator. Validate() accumulates sub-errors via errors.Join and wraps with models.NewNillableGenericValidationError. Service implementations call Validate() before any business logic. (`if err := input.Validate(); err != nil { return nil, err }`)
**ValidationIssue sentinels in errors.go** — Package-level errors (ErrChargeNotFound, ErrChargeNamespaceEmpty, ErrChargeInvalid, ErrCreditRealizationsAlreadyAllocated) are declared as models.ValidationIssue vars with ErrorCode constants, field hints, severity, and HTTP status via commonhttp.WithHTTPStatusCodeAttribute. Never return raw fmt.Errorf for these conditions. (`return charges.NewChargeNotFoundError(namespace, id)`)
**ChargeIntents.ByType() for per-type dispatch** — ChargeIntents.ByType() returns a ChargeIntentsByType struct with pre-split slices (FlatFee, CreditPurchase, UsageBased each as []WithIndex[T]). Service dispatch iterates ByType() output rather than switching on individual items in a loop. (`byType, err := intents.ByType(); for _, ff := range byType.FlatFee { ... ff.Value ... ff.Index ... }`)
**Adapter interface composed of ChargesSearchAdapter + entutils.TxCreator** — charges.Adapter embeds ChargesSearchAdapter (GetByIDs, ListCharges, ListCustomersToAdvance) plus entutils.TxCreator. All implementations must honour the TransactingRepo pattern so ctx-carried transactions are rebound on every method. (`var _ charges.Adapter = (*adapter.Adapter)(nil)`)
**lockr.Key via NewLockKeyForCharge** — Per-charge advisory locks are obtained via charges.NewLockKeyForCharge(chargeID), which validates the ID before constructing the key. Never construct lockr.Key strings inline. (`key, err := charges.NewLockKeyForCharge(chargeID); billingService.WithLock(ctx, key, func() error { ... })`)
**AdvanceChargesEvent for async advance dispatch** — Async advancement is triggered by publishing AdvanceChargesEvent (implements metadata.EventName + EventMetadata + Validate()). The event carries Namespace + CustomerID; metadata source/subject are composed via metadata.ComposeResourcePath. (`evt := charges.AdvanceChargesEvent{Namespace: ns, CustomerID: cid}; publisher.Publish(ctx, evt)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines the Service interface (ChargeService + CreditPurchaseFacadeService) and all Input types with Validate(). This is the primary contract for callers. | Input Validate() must call sub-field Validate() and accumulate errors; never short-circuit on the first error. |
| `charge.go` | Defines Charge, ChargeIntent, Charges, ChargeIntents, ChargeIntentsByType, and the WithIndex generic wrapper. All charge-type dispatch routes through this file's switch cases. | NewCharge/NewChargeIntent are the only constructors — direct struct literal construction leaves the discriminator field `t` empty and all accessor methods will return errors. |
| `adapter.go` | Defines Adapter (ChargesSearchAdapter + entutils.TxCreator), ChargeSearchItem, ChargeSearchItems. Interface compliance assertions belong here. | ChargesSearchAdapter methods are read-only search methods; writes must go through sub-package adapters via the service layer. |
| `errors.go` | All package-level ValidationIssue sentinels and their ErrorCode constants. HTTP status codes are embedded here. | Adding a new error as fmt.Errorf instead of models.NewValidationIssue breaks HTTP status mapping in the error encoder chain. |
| `patch.go` | Defines ApplyPatchesInput (CustomerID + Creates + PatchesByChargeID map) and ConcatenateApplyPatchesInputs. Patch is a type alias for meta.Patch. | PatchesByChargeID enforces one patch per charge ID; ConcatenateApplyPatchesInputs returns an error on duplicate IDs — don't bypass this with direct map merges. |
| `events.go` | AdvanceChargesEvent used to trigger async per-customer charge advancement via the event bus. | EventName must use metadata.GetEventName to guarantee consistent topic routing; do not construct the name string manually. |
| `lock.go` | NewLockKeyForCharge produces the lockr.Key for per-charge advisory locking. | Always validate chargeID before constructing the lock key; raw string keys can collide across namespaces. |

## Anti-Patterns

- Constructing Charge{} or ChargeIntent{} with struct literals instead of NewCharge[T]/NewChargeIntent[T] — leaves discriminator field `t` empty, all accessor methods will error.
- Bypassing charges.Service by calling flatfee/usagebased/creditpurchase sub-package adapters directly from app/common or billing/worker — breaks the namespace lockdown check and TransactingRepo discipline.
- Returning raw fmt.Errorf instead of models.NewValidationIssue for domain error conditions — breaks HTTP status code mapping in the error encoder chain.
- Adding business orchestration logic (gathering-line creation, auto-advance) to adapter.go — adapter is pure data-access; orchestration belongs in charges/service.
- Omitting Validate() on new Input types or skipping the call in service implementations — allows denormalized/invalid inputs to reach the adapter layer.

## Decisions

- **Charge and ChargeIntent are tagged unions (private discriminator field, NewCharge[T] constructor) rather than interfaces or separate top-level types.** — Prevents callers from constructing partial charge values and forces all type dispatch through a single switch pattern, making it easy to audit coverage when adding new charge types.
- **charges.Adapter only exposes read/search methods (ChargesSearchAdapter); write operations belong to sub-package adapters invoked through the charges/service orchestration layer.** — Keeps the top-level package as a pure aggregation point with no write side-effects, ensuring all mutations flow through the service with namespace lockdown and TransactingRepo checks.
- **ValidationIssue sentinels with embedded HTTP status codes are declared at the package level in errors.go rather than inline in service methods.** — Centralises HTTP status mapping so the error encoder chain can detect them by type assertion; inline fmt.Errorf calls would silently produce 500 responses.

## Example: Creating charges and dispatching async advancement

```
import (
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
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
// ...
```

<!-- archie:ai-end -->
