# charges

<!-- archie:ai-start -->

> Top-level domain contract package for the charges sub-system: defines the tagged-union Charge/ChargeIntent types (private meta.ChargeType discriminator accessed via AsFlatFeeCharge/AsUsageBasedCharge/AsCreditPurchaseCharge), the composite Service interface (ChargeService + CreditPurchaseFacadeService), and the read-only Adapter (ChargesSearchAdapter + entutils.TxCreator). All charge lifecycle must flow through charges.Service; the per-type implementations live in the flatfee/usagebased/creditpurchase children and orchestration in charges/service.

## Patterns

**Tagged-union construction via NewCharge[T]/NewChargeIntent[T]** — Charge and ChargeIntent hold a private `t meta.ChargeType` discriminator set only by the generic NewCharge[T]/NewChargeIntent[T] constructors; accessors return typed values or errors. Struct literals leave `t` empty and every accessor errors. (`charge := charges.NewCharge(flatfee.Charge{...}); ff, err := charge.AsFlatFeeCharge()`)
**Input.Validate() before any business logic** — Every Input type (CreateInput, GetByIDInput, ListChargesInput, ApplyPatchesInput, AdvanceChargesInput...) implements models.Validator, accumulating sub-errors via errors.Join and wrapping with models.NewNillableGenericValidationError; service implementations call Validate() first. (`if err := input.Validate(); err != nil { return nil, err }`)
**ValidationIssue sentinels declared in errors.go** — Domain errors (ErrChargeNamespaceEmpty, ErrChargeNotFound, ErrChargeInvalid, ErrCreditRealizationsAlreadyAllocated) are models.NewValidationIssue vars with ErrorCode constants and HTTP status via commonhttp.WithHTTPStatusCodeAttribute. Never use raw fmt.Errorf for these conditions. (`return charges.NewChargeNotFoundError(namespace, id)`)
**ChargeIntents.ByType() for per-type dispatch** — ByType() returns ChargeIntentsByType with pre-split FlatFee/CreditPurchase/UsageBased slices of WithIndex[T]; iterate the split output rather than re-switching on each item, preserving original indices. (`byType, err := intents.ByType(); for _, ff := range byType.FlatFee { /* ff.Value, ff.Index */ }`)
**NewLockKeyForCharge for per-charge advisory locks** — Per-charge advisory lock keys come from charges.NewLockKeyForCharge(chargeID), which validates the ChargeID before building a namespace-scoped lockr.Key. Never hand-construct lockr.Key strings. (`key, err := charges.NewLockKeyForCharge(chargeID)`)
**AdvanceChargesEvent for async dispatch** — Async per-customer advancement is triggered by publishing AdvanceChargesEvent; EventName() uses metadata.GetEventName with EventSubsystem "billing" and metadata.ComposeResourcePath, never a raw string, so eventbus routing works. (`publisher.Publish(ctx, charges.AdvanceChargesEvent{Namespace: ns, CustomerID: cid})`)
**Adapter exposes read-only ChargesSearchAdapter only** — charges.Adapter embeds ChargesSearchAdapter (GetByIDs, ListCharges, ListCustomersToAdvance) plus entutils.TxCreator. All write operations belong to sub-package adapters invoked through the service orchestration layer. (`var _ charges.Adapter = (*adapter.Adapter)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines Service (ChargeService + CreditPurchaseFacadeService) and all Input types with Validate(): GetByID, GetByIDs, Create, AdvanceCharges, ListCustomersToAdvance, ApplyPatches, ListCharges, HandleCreditPurchaseExternalPaymentStateTransition. | Input Validate() must accumulate every sub-error via errors.Join and call sub-field Validate(); ListChargesInput rejects StatusIn and StatusNotIn set simultaneously. |
| `charge.go` | Defines Charge, ChargeIntent, Charges, ChargeIntents, ChargeIntentsByType, and the WithIndex[T] generic wrapper; all charge-type dispatch routes through the switch on `t` here. | NewCharge/NewChargeIntent are the ONLY constructors. A struct-literal Charge{} leaves `t` empty and AsX/GetChargeID/Validate all return errors. |
| `adapter.go` | Defines the read-only Adapter interface (ChargesSearchAdapter + TxCreator), ChargeSearchItem and ChargeSearchItems with Validate(). | ChargesSearchAdapter methods are read-only; writes must go through sub-package adapters via the service layer. |
| `errors.go` | All package-level ValidationIssue sentinels and ErrorCode constants with embedded HTTP status codes (400/404). | Adding a domain error as fmt.Errorf instead of models.NewValidationIssue breaks HTTP status mapping in the error encoder chain and produces 500. |
| `patch.go` | Defines ApplyPatchesInput (CustomerID + Creates + PatchesByChargeID map), ConcatenateApplyPatchesInputs, and IsEmpty(); Patch is a type alias of meta.Patch. | PatchesByChargeID enforces one patch per charge ID; ConcatenateApplyPatchesInputs errors on duplicate IDs. Do not bypass with direct map merges. |
| `events.go` | AdvanceChargesEvent (Namespace + CustomerID) for async charge advancement; EventName/EventMetadata built from metadata helpers; EventSubsystem = "billing". | EventName must use metadata.GetEventName; constructing the name string manually breaks topic routing in eventbus.GeneratePublishTopic. |
| `lock.go` | NewLockKeyForCharge validates the ChargeID then builds a namespace+charge scoped lockr.Key. | Always use this helper; raw string keys can collide across namespaces. |
| `features.go` | Package-level feature flag CreditNotesSupportedByLineUpdater (default false) gating charge-backed immutable invoice-line proration. | Defaults false until the invoice line updater supports credit notes; flipping it without updater support corrupts immutable invoice history. |

## Anti-Patterns

- Constructing Charge{} or ChargeIntent{} with struct literals instead of NewCharge[T]/NewChargeIntent[T] — leaves discriminator `t` empty and all accessors error.
- Bypassing charges.Service by calling flatfee/usagebased/creditpurchase sub-package adapters directly from app/common or billing/worker — skips namespace lockdown and TransactingRepo discipline.
- Returning raw fmt.Errorf instead of a models.ValidationIssue sentinel for domain errors — breaks HTTP status code mapping and produces 500.
- Adding business orchestration (gathering-line creation, auto-advance) to adapter.go — adapter is pure read/search data-access; orchestration belongs in charges/service.
- Omitting Validate() on new Input types or skipping the call in service implementations — lets denormalized/invalid inputs reach the adapter layer.

## Decisions

- **Charge and ChargeIntent are tagged unions with a private discriminator and NewCharge[T]/NewChargeIntent[T] constructors rather than interfaces.** — Prevents partial charge construction and forces all type dispatch through a single switch, making coverage gaps visible when a new charge type is added.
- **charges.Adapter exposes only read/search methods (ChargesSearchAdapter); writes live in sub-package adapters invoked through the service.** — Keeps the top-level package a pure aggregation point with no write side-effects, ensuring all mutations flow through the service with namespace lockdown and TransactingRepo checks.
- **ValidationIssue sentinels with embedded HTTP status codes are declared at package level in errors.go.** — Centralizes HTTP status mapping so the error encoder chain detects them by type assertion; inline fmt.Errorf silently produces 500 responses.

## Example: Creating charges through the validated service entry point

```
import (
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
)

intent := charges.NewChargeIntent(flatfee.Intent{ /* ... */ })
input := charges.CreateInput{Namespace: ns, Intents: charges.ChargeIntents{intent}}
if err := input.Validate(); err != nil {
	return err
}
created, err := chargeService.Create(ctx, input)
```

<!-- archie:ai-end -->
