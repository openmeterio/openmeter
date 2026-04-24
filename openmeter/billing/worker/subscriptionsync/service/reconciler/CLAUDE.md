# reconciler

<!-- archie:ai-start -->

> Computes and applies a diff plan (Plan/Apply two-phase protocol) between desired subscription target state and persisted billing artifacts (invoice lines, split-line hierarchies, charges). Routes each diff operation to the correct PatchCollection backend based on settlement mode and price type.

## Patterns

**Plan/Apply two-phase protocol** — Plan() builds a pure in-memory Plan{InvoicePatches, ChargePatches} without touching the DB; Apply() executes it. DryRun skips execution and only logs. Always separate planning from execution. (`plan, err := s.Plan(ctx, PlanInput{...}); if err != nil { return err }; return s.Apply(ctx, ApplyInput{Plan: plan, ...})`)
**PatchCollectionRouter dispatches by persisted item type and settlement mode** — GetCollectionFor(existing) routes by persistedstate.ItemType; ResolveDefaultCollection(target) routes new items by settlement mode + price type. Never assign a patch to a collection manually. (`collection, err := patchCollections.GetCollectionFor(existingLine); s.diffItem(&target, existing, collection)`)
**diffItem drives all five patch operations** — diffItem(target, existing, collection) emits exactly one of: AddCreate, AddDelete, AddProrate (flat-fee period+amount change), AddShrink, or AddExtend. Charge-backed collections return unsupportedOperationError for AddProrate. (`case targetPeriod.To.Before(existingPeriod.To): return patches.AddShrink(...)
case targetPeriod.To.After(existingPeriod.To): return patches.AddExtend(...)`)
**Semantic prorate gate applies only to invoice-backed flat-fee lines** — semanticProrateDecision() returns ShouldProrate=true only for ItemTypeInvoiceLine with flat-fee price and a period or amount change. Charge-backed collections skip this gate entirely via GetLineEngineType().IsCharge() check. (`if patches.GetLineEngineType() == billing.LineEngineTypeInvoice { decision, _ := semanticProrateDecision(...); if decision.ShouldProrate { return patches.AddProrate(...) } }`)
**filterInScopeLines removes non-billable targets before diff** — Non-billable targets are silently excluded before planning so they reconcile to delete/no-op for any existing artifacts. Charge-backed targets pass if IsBillable(); invoice-backed also require GetExpectedLine() != nil. (`inScopeLines, err := filterInScopeLines(input.Target.Items, patchCollections)`)
**chargePatchCollection embeds base, concrete types embed chargePatchCollection** — flatFeeChargeCollection and usageBasedChargeCollection both embed chargePatchCollection via struct embedding. AddCreate is the only method overridden; all other PatchCollection methods are satisfied by the embedded base. (`type flatFeeChargeCollection struct { chargePatchCollection }
func (c *flatFeeChargeCollection) AddCreate(target targetstate.StateItem) error { ... }`)
**Config.Validate() before construction** — Config structs expose a Validate() method checked in New(). ChargesService is required only when EnableCreditThenInvoice is true. Do not skip validation in tests. (`func New(config Config) (*Service, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `reconciler.go` | Entry point: Service struct, Plan() diff loop, Apply() execution, diffItem() operation selector, filterInScopeLines() pre-filter | Plan() sets creditsEnabled based on chargesService != nil — do not pass a non-nil chargesService unless credits are truly enabled in config |
| `patch.go` | PatchCollection interfaces and patchCollectionRouter; routes items to the right collection by type and settlement mode | ResolveDefaultCollection falls back to lineCollection when credits disabled or CreditThenInvoice disabled — new settlement modes must be added here |
| `patchcharge.go` | chargePatchCollection base: holds charges.ApplyPatchesInput, implements AddDelete/AddShrink/AddExtend via chargesmeta patches, AddProrate returns error | addPatch() panics on duplicate chargeID in PatchesByChargeID map — one patch per charge per reconciliation cycle |
| `patchchargeflatfee.go` | flatFeeChargeCollection.AddCreate builds chargesflatfee.Intent from target; wires ProRating, PercentageDiscounts, PaymentTerm | AmountBeforeProration must use flatPrice.Amount (before any proration), not a post-proration value |
| `patchchargeusagebased.go` | usageBasedChargeCollection.AddCreate builds chargesusagebased.Intent; passes full Price and Discounts from rate card meta | Price is dereferenced with *price — nil price is guarded but still panics if guard is removed |
| `patchinvoiceline.go` | lineInvoicePatchCollection: AddCreate/Delete/Shrink/Extend/Prorate for flat invoice lines; uses getPatchesForUpdateUsageBasedLine for period changes | AddProrate only applies to flat-fee lines — enforced by invoiceupdater.IsFlatFee check; returns error for usage-based |
| `patchinvoicelinehierarchy.go` | lineHierarchyPatchCollection: handles SplitLineHierarchy artifacts; AddCreate and AddProrate are unsupported (return error) | AddDelete walks all child lines — if any child has AnnotationSubscriptionSyncIgnore the entire hierarchy is skipped |
| `prorate.go` | semanticProrateDecision: compares existing flat-fee amount to expected; returns ShouldProrate and both amounts for AddProrate | Flat fee prorate is only valid for ItemTypeInvoiceLine — ItemTypeInvoiceSplitLineGroup returns an error |

## Anti-Patterns

- Calling billing.Adapter or charges.Adapter methods directly from reconciler code — always go through billing.Service (via invoiceupdater.Updater) and charges.Service
- Adding a new settlement mode without updating patchCollectionRouterConfig.ResolveDefaultCollection routing logic in patch.go
- Implementing AddProrate for charge collections — charge-backed reconciliation handles amount changes implicitly via shrink/extend; explicit prorate is invoice-only
- Skipping filterInScopeLines before Plan() diff — non-billable targets must be excluded before the diff loop or phantom creates are emitted
- Using context.Background() instead of propagating the ctx parameter through all billing.Service and charges.Service calls in Apply()

## Decisions

- **Plan/Apply two-phase split with a pure in-memory Plan struct** — Enables dry-run logging, testable diff output independent of DB side-effects, and retry of Apply without re-running the potentially expensive Plan computation
- **patchCollectionRouter selects backend at planning time based on settlement mode and price type** — Allows graceful migration of existing invoice-line artifacts to charge-backed provisioning: existing items are routed by their persisted ItemType, new items use ResolveDefaultCollection, so mixed states coexist during rollout
- **Prorate is a distinct PatchOperation only for invoice-backed flat-fee lines** — Invoice lines require atomic period+amount updates to avoid billing discrepancies; usage-based lines only need period updates (amount is derived from usage); charge backends recompute amounts from period changes internally so prorate is unsupported there

## Example: Adding a new patch collection for a hypothetical new charge type

```
// 1. Add ItemType constant in persistedstate package
// 2. Add LineEngineType constant in billing package
// 3. Create new collection embedding chargePatchCollection:
type newChargeCollection struct { chargePatchCollection }
func newNewChargeCollection(cap int) *newChargeCollection {
	return &newChargeCollection{chargePatchCollection: newChargePatchCollection(billing.LineEngineTypeNewCharge, persistedstate.ItemTypeNewCharge, cap)}
}
func (c *newChargeCollection) AddCreate(target targetstate.StateItem) error { ... }

// 4. Add field to patchCollectionRouter, construct in newPatchCollectionRouter
// 5. Wire in GetCollectionFor switch and ResolveDefaultCollection switch
// 6. Include in CollectChargePatches via charges.ConcatenateApplyPatchesInputs
```

<!-- archie:ai-end -->
