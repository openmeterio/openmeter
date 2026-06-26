# reconciler

<!-- archie:ai-start -->

> The planning side of the subscription->billing sync bridge: diffs subscription target state against persisted billing state and produces a Plan of invoice-line patches and charge patches, then applies it via billing.Service / charges.Service. It decides WHAT must change; the child invoiceupdater package executes invoice writes. Primary constraint: every diffed item must route to exactly one backend (invoice line, split-line hierarchy, flat-fee charge, or usage-based charge) and the chosen backend must stay consistent across runs.

## Patterns

**Reconciler = Plan then Apply** — Service.Plan(ctx, PlanInput) builds a *Plan (pure, no writes) and Service.Apply(ctx, ApplyInput) executes it. DryRun in ApplyInput logs patches via invoiceUpdater.LogPatches + logChargesPatches and returns without writing. (`plan, _ := s.Plan(ctx, PlanInput{Target: t, Persisted: p}); s.Apply(ctx, ApplyInput{Plan: plan, Customer: cid, Currency: cc})`)
**PatchCollection interface per backend** — Every backend collection implements PatchCollection (AddCreate/AddDelete/AddShrink/AddExtend/AddProrate + GetLineEngineType). diffItem dispatches one of these methods based on target/existing presence and period-end comparison. New backends must satisfy the full interface. (`func (c *flatFeeChargeCollection) AddShrink(_ string, existing persistedstate.Item, target targetstate.StateItem) error`)
**Router resolves the backend, not the diff** — patchCollectionRouter.GetCollectionFor(item) maps a persisted Item.Type() to its collection; ResolveDefaultCollection(target) picks the backend for a NEW target from settlement mode + credits gate + price type. Plan routes EXISTING items by item type, falling back to the default collection only for brand-new uniqueIDs (graceful invoice->charge migration). (`case persistedstate.ItemTypeChargeFlatFee: return c.flatFeeChargeCollection, nil`)
**diffItem create/delete/shrink/extend/prorate decision tree** — nil target + existing => AddDelete; target + nil existing => AddCreate; otherwise compare ServicePeriod.To: Before=>AddShrink, After=>AddExtend, equal=>no-op. Flat-fee invoice lines additionally use semanticProrateDecision (LineEngineTypeInvoice only) to emit AddProrate when amount or period drifts. (`case targetPeriod.To.Before(existingPeriod.To): return patches.AddShrink(target.UniqueID, existing, *target)`)
**Charge period changes: emulated replacement vs native patch by settlement mode** — In flat/usage charge collections, AddShrink/AddExtend emit a native chargesmeta.NewPatchShrink/NewPatchExtend ONLY when the existing charge's Intent.SettlementMode == CreditThenInvoiceSettlementMode; otherwise they call addEmulatedReplacement (delete patch + fresh create intent). AddProrate is unsupported for charges (returns unsupportedOperationError). (`if existingCharge.GetFlatFeeCharge().Intent.SettlementMode != productcatalog.CreditThenInvoiceSettlementMode { return c.addEmulatedReplacement(existing, intent) }`)
**filterInScopeLines before diffing** — Plan filters target items first: drops !IsBillable(); charge-backed targets pass on billability alone; invoice-backed targets are dropped when GetExpectedLine() returns nil. This makes non-billable/non-realizing targets behave as absent and reconcile to delete/no-op. (`if defaultCollection.GetLineEngineType().IsCharge() { out = append(out, line); continue }`)
**Config.Validate gates construction** — New(Config) requires non-nil BillingService, Logger, FeatureGate; ChargesService is required only when EnableCreditThenInvoice. CreditOnly settlement is rejected at Plan if chargesService is nil. Collections build via newPatchCollectionRouter whose config.Validate enforces capacity>0, invoices!=nil, featureGate!=nil. (`if c.EnableCreditThenInvoice && c.ChargesService == nil { return fmt.Errorf("charges service is required when credit then invoice is enabled") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `reconciler.go` | Public Reconciler interface, Service, Config, Plan/Apply, diffItem decision tree, filterInScopeLines. Owns create/delete/shrink/extend/prorate routing. | Plan routes existing items by GetCollectionFor (existing Item.Type) and only new uniqueIDs by ResolveDefaultCollection. Apply must apply invoice patches via invoiceUpdater BEFORE charge patches, and errors if ChargePatches non-empty while chargesService is nil. |
| `patch.go` | Patch/InvoicePatch/PatchCollection interfaces and patchCollectionRouter (holds the 4 backend collections + credits gating). CollectInvoicePatches / CollectChargePatches aggregate plan output. | ResolveDefaultCollection assumes price is non-nil (guaranteed by filterInScopeLines). isCreditsEnabled returns false when creditsEnabled is off; nil featureGate or empty creditsFlag => enabled. |
| `patchcharge.go` | chargePatchCollection base shared by flat/usage charge collections: addCreate/addPatch dedup by chargeID, addEmulatedReplacement (delete+create), newChargeIntentBaseFromTargetState builds chargesmeta.Intent from target state. | addPatch rejects duplicate chargeID; AddProrate is intentionally unsupported. newChargeIntentBaseFromTargetState clones Annotations/Metadata and sets ManagedBy=SubscriptionManagedLine + UniqueReferenceID=target.UniqueID. |
| `patchchargeflatfee.go` | flatFeeChargeCollection: AddCreate/AddShrink/AddExtend; newFlatFeeChargeIntent converts rate card price.AsFlat() into a chargesflatfee.Intent. | Shrink/Extend emit a native patch only for CreditThenInvoice settlement, else emulated replacement. existing must implement persistedstate.FlatFeeChargeGetter or it errors. |
| `patchchargeusagebased.go` | usageBasedChargeCollection: same create/shrink/extend shape for usage-based charges; newUsageBasedChargeIntent carries Price + Discounts. | existing must satisfy persistedstate.UsageBasedChargeGetter; settlement-mode branching mirrors flat fee. |
| `patchinvoiceline.go / patchinvoicelinehierarchy.go` | lineInvoicePatchCollection and lineHierarchyPatchCollection implement invoice-line and split-line-group diffs, emitting invoiceupdater.New*Patch. Hierarchy mutates child lines + updates the SplitLineGroup via ToUpdate(). | Both compile-time assert _ PatchCollection / _ InvoicePatchCollection. Hierarchy AddCreate/AddProrate are unsupported (error). Shrink/Extend assert target end strictly before/after existing end. ServicePeriod.Truncate(streaming.MinimumWindowSizeDuration).IsEmpty() => delete patch instead of update. |
| `patchhelpers.go` | shouldSkipLinePatch / shouldSkipHierarchyPatch (AnnotationSubscriptionSyncIgnore + SubscriptionManagedLine guards) and getPatchesForUpdateUsageBasedLine (clone, set service period, set invoice-at on gathering invoices). | Skips any line not managed by SubscriptionManagedLine or flagged AnnotationSubscriptionSyncIgnore. Returns nil (no-op) when nothing changed. |
| `prorate.go` | semanticProrateDecision: flat-fee-only proration decision comparing existing vs expected per-unit amount and service period; errors for split-line hierarchies. | Only meaningful for invoice flat-fee lines; returns empty decision (ShouldProrate=false) for usage-based/non-flat-fee. diffItem calls it only when LineEngineType==Invoice. |

## Anti-Patterns

- Diffing by target backend instead of existing item type: Plan must use GetCollectionFor(existing) for items already in persisted state and only fall back to ResolveDefaultCollection for brand-new uniqueIDs, or invoice->charge migrations break.
- Adding a charge AddProrate or a hierarchy AddCreate/AddProrate path: these are deliberately unsupported and return errors; period changes for charges must flow through shrink/extend.
- Emitting a native charge shrink/extend patch for a non-CreditThenInvoice charge: those must become an emulated delete+create replacement via addEmulatedReplacement.
- Mutating invoice lines directly here instead of producing invoiceupdater.Patch values (NewCreateLinePatch/NewUpdateLinePatch/NewDeleteLinePatch/NewUpdateSplitLineGroupPatch) for the invoiceupdater child to apply.
- Skipping the SubscriptionManagedLine / AnnotationSubscriptionSyncIgnore guards (shouldSkipLinePatch/shouldSkipHierarchyPatch) and patching customer-owned or sync-ignored lines.

## Decisions

- **Plan is pure and Apply is the only writer; DryRun logs instead of writing.** — Separating computation from side effects makes the diff deterministic and testable, and lets the worker preview drift (DryRun) before committing invoice/charge mutations.
- **A router selects one of four backend collections per item, gated by settlement mode + credits feature flag + price type.** — The same subscription item can be realized as an invoice line, a split-line hierarchy, or a flat/usage charge depending on credits/settlement config; routing keeps each artifact's lifecycle isolated and enables graceful migration between backends.
- **Charge period changes are emulated replacements except under CreditThenInvoice, and proration is delegated entirely to the charge domain.** — Only CreditThenInvoice charges expose native shrink/extend semantics; for other modes a delete+recreate keeps the charge state correct, and the charge stack recomputes effective amounts from updated periods so the reconciler never prorates charges itself.

## Example: diffItem dispatching the correct PatchCollection method based on presence and period drift

```
func (s *Service) diffItem(target *targetstate.StateItem, existing persistedstate.Item, patches PatchCollection) error {
	switch {
	case target == nil && existing == nil:
		return nil
	case target == nil && existing != nil:
		return patches.AddDelete(lo.FromPtr(existing.ChildUniqueReferenceID()), existing)
	case target != nil && existing == nil:
		return patches.AddCreate(*target)
	}
	existingPeriod := existing.ServicePeriod()
	targetPeriod := target.GetServicePeriod()
	if patches.GetLineEngineType() == billing.LineEngineTypeInvoice {
		if decision, err := semanticProrateDecision(existing, *target); err != nil {
			return err
		} else if decision.ShouldProrate {
// ...
```

<!-- archie:ai-end -->
