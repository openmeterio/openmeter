# reconciler

<!-- archie:ai-start -->

> Computes and applies a Plan/Apply two-phase diff between the desired subscription target state and persisted billing artifacts (invoice lines, split-line hierarchies, charges). Routes each diff operation to the correct PatchCollection backend based on settlement mode, price type, and credits configuration; its invoiceupdater/ child is the write-side that translates invoice patches into billing.Service calls.

## Patterns

**Plan/Apply two-phase split with pure in-memory Plan** — Plan() builds Plan{InvoicePatches, ChargePatches} without touching the DB; Apply() executes it; DryRun skips execution and only logs. Never interleave planning and execution. (`plan, err := s.Plan(ctx, PlanInput{...}); if err != nil { return err }; return s.Apply(ctx, ApplyInput{Plan: plan, DryRun: false, ...})`)
**patchCollectionRouter dispatches by item type and settlement mode** — GetCollectionFor(existing) routes existing items by persistedstate.ItemType; ResolveDefaultCollection(target) routes new items by credits-enabled + settlement mode + price type. Never assign a patch to a collection manually. (`collection, err := patchCollections.GetCollectionFor(existingLine); ... ; collection, err := router.ResolveDefaultCollection(target)`)
**diffItem emits exactly one of five patch operations** — diffItem(target, existing, collection) emits AddCreate, AddDelete, AddProrate (invoice flat-fee only), AddShrink, or AddExtend; charge-backed collections return error for AddProrate. (`case targetPeriod.To.Before(existingPeriod.To): return collection.AddShrink(...)\ncase targetPeriod.To.After(existingPeriod.To): return collection.AddExtend(...)`)
**Charge collections via struct embedding of chargePatchCollection** — flatFeeChargeCollection and usageBasedChargeCollection embed chargePatchCollection and override only AddCreate/AddShrink/AddExtend; AddDelete and AddProrate (always error) come from the base. (`type flatFeeChargeCollection struct { chargePatchCollection }\nfunc newFlatFeeChargeCollection(cap int) *flatFeeChargeCollection { return &flatFeeChargeCollection{chargePatchCollection: newChargePatchCollection(billing.LineEngineTypeChargeFlatFee, persistedstate.ItemTypeChargeFlatFee, cap)} }`)
**Emulated replacement for CreditOnly shrink/extend** — For non-CreditThenInvoice settlement, period changes call addEmulatedReplacement (PatchDelete with RefundAsCreditsDeletePolicy + new create intent); for CreditThenInvoice, native chargesmeta.NewPatchShrink/NewPatchExtend is used. (`if existingCharge.GetFlatFeeCharge().Intent.SettlementMode != productcatalog.CreditThenInvoiceSettlementMode { return c.addEmulatedReplacement(existing, intent) }`)
**Config-validate-before-construct with required-by-flag deps** — patchCollectionRouterConfig.Validate() runs in newPatchCollectionRouter; capacity, invoices, and featureGate are mandatory. credits resolution goes through featureGate.EvaluateBool(ns, creditsFlag). (`func newPatchCollectionRouter(cfg patchCollectionRouterConfig) (*patchCollectionRouter, error) { if err := cfg.Validate(); err != nil { return nil, err } ... }`)
**addPatch enforces one patch per chargeID per cycle** — chargePatchCollection.addPatch errors on a duplicate chargeID in PatchesByChargeID; addCreate requires a non-empty UniqueReferenceID via intent.GetUniqueReferenceID(). (`if _, exists := c.patches.PatchesByChargeID[chargeID]; exists { return fmt.Errorf("patch for charge ID %s already exists", chargeID) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `reconciler.go` | Entry point: Service struct, Plan() diff loop, Apply() execution, diffItem() five-operation selector, filterInScopeLines() pre-filter; hosts the Reconciler interface. | Plan() derives creditsEnabled from chargesService != nil; enableCreditThenInvoice is true only when both the config flag and chargesService are non-nil. Do not pass a non-nil chargesService unless credits are truly enabled. |
| `patch.go` | PatchCollection interfaces (InvoicePatch, ChargePatchCollection, PatchCollection) and patchCollectionRouter with GetCollectionFor + ResolveDefaultCollection + CollectInvoicePatches/CollectChargePatches. | ResolveDefaultCollection falls back to lineCollection when credits or CreditThenInvoice is disabled — a new settlement mode must be added here and in GetCollectionFor. |
| `patchcharge.go` | chargePatchCollection base: holds charges.ApplyPatchesInput, implements AddDelete/AddProrate(error)/addEmulatedReplacement, addPatch()/addCreate() helpers, newChargeIntentBaseFromTargetState. | addPatch errors on duplicate chargeID; addCreate requires non-empty UniqueReferenceID; AddProrate always returns unsupportedOperationError. |
| `patchchargeflatfee.go` | flatFeeChargeCollection.AddCreate builds chargesflatfee.Intent; AddShrink/AddExtend branch on SettlementMode for emulated-replacement vs native patch. | AmountBeforeProration must use flatPrice.Amount (pre-proration rate-card value), not a derived post-proration value. |
| `patchchargeusagebased.go` | usageBasedChargeCollection.AddCreate builds chargesusagebased.Intent; AddShrink/AddExtend branch on SettlementMode same as flat fee. | Price is dereferenced via *price — the nil-price guard in newUsageBasedChargeIntent must stay or it panics. |
| `patchinvoiceline.go` | lineInvoicePatchCollection: AddCreate/Delete/Shrink/Extend/Prorate for invoice lines; usage-based period changes go through getPatchesForUpdateUsageBasedLine. | AddProrate is flat-fee only (invoiceupdater.IsFlatFee check) and updates service period + per-unit amount atomically; usage-based returns error. |
| `patchinvoicelinehierarchy.go` | lineHierarchyPatchCollection: SplitLineHierarchy artifacts; AddCreate/AddProrate unsupported; AddDelete/Shrink/Extend walk child lines. | AddDelete skips the whole hierarchy if any child carries AnnotationSubscriptionSyncIgnore; Shrink sorts children and updates only the last one crossing the new boundary. |
| `prorate.go` | semanticProrateDecision compares existing flat-fee per-unit amount to expected and returns ShouldProrate plus both amounts for AddProrate. | Valid only for ItemTypeInvoiceLine; split-line-group and charge item types must not call it. |

## Anti-Patterns

- Calling billing.Adapter or charges.Adapter directly — go through billing.Service (invoiceupdater.Updater) and charges.Service.ApplyPatches
- Adding a settlement mode without updating both GetCollectionFor and ResolveDefaultCollection in patch.go
- Implementing AddProrate for charge collections — charge backends recompute amounts from shrink/extend; explicit prorate is invoice-only
- Skipping filterInScopeLines before the Plan() diff — non-billable targets emit phantom creates
- Using context.Background() instead of propagating ctx through billing.Service and charges.Service calls in Apply()

## Decisions

- **Plan/Apply two-phase split with a pure in-memory Plan struct** — Enables dry-run logging, DB-independent testable diff output, and Apply retry without re-running the expensive Plan walk over all subscription items.
- **patchCollectionRouter picks backend at planning time by settlement mode + price type** — Existing items route by persisted ItemType and new items by ResolveDefaultCollection, so invoice-line and charge-backed artifacts coexist safely during the credits feature rollout.
- **Prorate is a distinct PatchOperation only for invoice-backed flat-fee lines** — Invoice lines need atomic period+amount updates; usage-based amounts derive from usage and charge backends recompute internally, so explicit prorate is unsupported there and returns an error.

## Example: Add a patch collection for a new charge type

```
// 1. Add persistedstate.ItemTypeNewCharge + billing.LineEngineTypeNewCharge constants
type newChargeCollection struct { chargePatchCollection }
func newNewChargeCollection(cap int) *newChargeCollection {
	return &newChargeCollection{chargePatchCollection: newChargePatchCollection(billing.LineEngineTypeNewCharge, persistedstate.ItemTypeNewCharge, cap)}
}
func (c *newChargeCollection) AddCreate(target targetstate.StateItem) error {
	intent, err := newNewChargeIntent(target)
	if err != nil { return err }
	return c.addCreate(intent)
}
// 2. Add field to patchCollectionRouter; wire in GetCollectionFor + ResolveDefaultCollection; include in CollectChargePatches
```

<!-- archie:ai-end -->
