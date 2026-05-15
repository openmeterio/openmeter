# reconciler

<!-- archie:ai-start -->

> Computes and applies a diff plan (Plan/Apply two-phase protocol) between the desired subscription target state and persisted billing artifacts (invoice lines, split-line hierarchies, charges). Routes each diff operation to the correct PatchCollection backend based on settlement mode, price type, and credits configuration.

## Patterns

**Plan/Apply two-phase split** — Plan() builds a pure in-memory Plan{InvoicePatches, ChargePatches} without touching the DB; Apply() executes it. DryRun skips execution and only logs. Always separate planning from execution. (`plan, err := s.Plan(ctx, PlanInput{...}); if err != nil { return err }; return s.Apply(ctx, ApplyInput{Plan: plan, DryRun: false, ...})`)
**patchCollectionRouter dispatches by persisted item type and settlement mode** — GetCollectionFor(existing) routes by persistedstate.ItemType for existing items; ResolveDefaultCollection(target) routes new items by settlement mode + price type + credits/creditThenInvoice flags. Never assign a patch to a collection manually. (`collection, err := patchCollections.GetCollectionFor(existingLine); s.diffItem(&targetLine, existingLine, collection)`)
**diffItem emits exactly one of five patch operations** — diffItem(target, existing, collection) emits AddCreate, AddDelete, AddProrate (flat-fee invoice-backed period+amount change), AddShrink, or AddExtend. Charge-backed collections return unsupportedOperationError for AddProrate. (`case targetPeriod.To.Before(existingPeriod.To): return patches.AddShrink(...)
case targetPeriod.To.After(existingPeriod.To): return patches.AddExtend(...)`)
**Semantic prorate gate applies only to invoice-backed flat-fee lines** — semanticProrateDecision() returns ShouldProrate=true only for ItemTypeInvoiceLine with flat-fee price and a period or amount change. Charge-backed collections skip this gate via patches.GetLineEngineType().IsCharge() check in diffItem. (`if patches.GetLineEngineType() == billing.LineEngineTypeInvoice { if decision.ShouldProrate { return patches.AddProrate(...) } }`)
**chargePatchCollection struct embedding for concrete charge types** — flatFeeChargeCollection and usageBasedChargeCollection both embed chargePatchCollection. AddCreate is the only method overridden; AddDelete/AddShrink/AddExtend/AddProrate are satisfied by the embedded base (AddProrate always returns error). (`type flatFeeChargeCollection struct { chargePatchCollection }
func (c *flatFeeChargeCollection) AddCreate(target targetstate.StateItem) error { ... }`)
**Emulated replacement for CreditOnly shrink/extend** — For CreditOnly settlement mode, period changes emit addEmulatedReplacement (delete existing + create new intent) instead of native PatchShrink/PatchExtend. For CreditThenInvoice, native chargesmeta.NewPatchShrink/NewPatchExtend is used. (`if existingCharge.GetFlatFeeCharge().Intent.SettlementMode != productcatalog.CreditThenInvoiceSettlementMode { return c.addEmulatedReplacement(existing, intent) }`)
**Config.Validate() before construction** — Config structs expose Validate() checked in New(). ChargesService is required only when EnableCreditThenInvoice is true. Do not skip validation in tests. (`func New(config Config) (*Service, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `reconciler.go` | Entry point: Service struct, Plan() diff loop, Apply() execution, diffItem() five-operation selector, filterInScopeLines() pre-filter. Hosts Reconciler interface. | Plan() sets creditsEnabled based on chargesService != nil — do not pass a non-nil chargesService unless credits are truly enabled in config. enableCreditThenInvoice is true only when both config flag AND chargesService are non-nil. |
| `patch.go` | PatchCollection interfaces (InvoicePatch, ChargePatchCollection, PatchCollection) and patchCollectionRouter; routes items to the right collection by type and settlement mode. | ResolveDefaultCollection falls back to lineCollection when credits disabled or CreditThenInvoice disabled — new settlement modes must be added here and in GetCollectionFor switch. |
| `patchcharge.go` | chargePatchCollection base: holds charges.ApplyPatchesInput, implements AddDelete/AddProrate (returns error)/addEmulatedReplacement, provides addPatch() and addCreate() helpers. | addPatch() returns error on duplicate chargeID in PatchesByChargeID map — one patch per charge per reconciliation cycle. addCreate() requires non-empty UniqueReferenceID. |
| `patchchargeflatfee.go` | flatFeeChargeCollection.AddCreate builds chargesflatfee.Intent from target; AddShrink/AddExtend branch on SettlementMode to emit emulated replacement or native patch. | AmountBeforeProration must use flatPrice.Amount (pre-proration value from rate card), not any post-proration derived value. |
| `patchchargeusagebased.go` | usageBasedChargeCollection.AddCreate builds chargesusagebased.Intent; AddShrink/AddExtend branch on SettlementMode same as flat fee. | Price is dereferenced with *price — nil price guard exists in newUsageBasedChargeIntent but removing it panics. |
| `patchinvoiceline.go` | lineInvoicePatchCollection: AddCreate/Delete/Shrink/Extend/Prorate for flat invoice lines; uses getPatchesForUpdateUsageBasedLine for period changes on usage-based lines. | AddProrate only applies to flat-fee lines — enforced by invoiceupdater.IsFlatFee check; returns error for usage-based. Prorate updates both service period and per-unit amount atomically. |
| `patchinvoicelinehierarchy.go` | lineHierarchyPatchCollection: handles SplitLineHierarchy artifacts; AddCreate and AddProrate are unsupported (return error). AddDelete/Shrink/Extend walk child lines. | AddDelete short-circuits and skips the entire hierarchy if ANY child line has AnnotationSubscriptionSyncIgnore. Shrink sorts children then updates only the last child that crosses the new boundary. |
| `prorate.go` | semanticProrateDecision: compares existing flat-fee per-unit amount to expected; returns ShouldProrate and both amounts for AddProrate. | Only valid for ItemTypeInvoiceLine — ItemTypeInvoiceSplitLineGroup returns an error. ItemTypeCharge* types must not call this function. |

## Anti-Patterns

- Calling billing.Adapter or charges.Adapter methods directly from reconciler code — always go through billing.Service (via invoiceupdater.Updater) and charges.Service.ApplyPatches
- Adding a new settlement mode without updating both patchCollectionRouter.GetCollectionFor and ResolveDefaultCollection in patch.go
- Implementing AddProrate for charge collections — charge-backed reconciliation handles amount changes implicitly via shrink/extend (or emulated replacement); explicit prorate is invoice-only
- Skipping filterInScopeLines before Plan() diff — non-billable targets and invoice-backed targets without an expected line must be excluded before the diff loop or phantom creates are emitted
- Using context.Background() instead of propagating the ctx parameter through all billing.Service and charges.Service calls in Apply()

## Decisions

- **Plan/Apply two-phase split with a pure in-memory Plan struct** — Enables dry-run logging, testable diff output independent of DB side-effects, and retry of Apply without re-running the potentially expensive Plan computation that walks all subscription items.
- **patchCollectionRouter selects backend at planning time based on settlement mode and price type** — Allows graceful migration of existing invoice-line artifacts to charge-backed provisioning: existing items are routed by their persisted ItemType, new items use ResolveDefaultCollection, so mixed states coexist safely during feature rollout.
- **Prorate is a distinct PatchOperation only for invoice-backed flat-fee lines** — Invoice lines require atomic period+amount updates to avoid billing discrepancies; usage-based lines only need period updates (amount is derived from usage); charge backends recompute amounts from period changes internally so explicit prorate is unsupported and returns an error.

## Example: Adding a new patch collection for a hypothetical new charge type

```
// 1. Add ItemType constant in persistedstate package
// 2. Add LineEngineType constant in billing package
// 3. Create new collection embedding chargePatchCollection:
type newChargeCollection struct { chargePatchCollection }
func newNewChargeCollection(cap int) *newChargeCollection {
	return &newChargeCollection{chargePatchCollection: newChargePatchCollection(billing.LineEngineTypeNewCharge, persistedstate.ItemTypeNewCharge, cap)}
}
func (c *newChargeCollection) AddCreate(target targetstate.StateItem) error {
	intent, err := newNewChargeIntent(target)
	if err != nil { return err }
	return c.addCreate(intent)
}
// 4. Add field to patchCollectionRouter, construct in newPatchCollectionRouter
// 5. Wire in GetCollectionFor switch and ResolveDefaultCollection switch
// 6. Include in CollectChargePatches via charges.ConcatenateApplyPatchesInputs
```

<!-- archie:ai-end -->
