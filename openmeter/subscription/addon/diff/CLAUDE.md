# diff

<!-- archie:ai-start -->

> Pure in-memory spec mutation library that converts a SubscriptionAddon into a bidirectional Apply/Restore transform on SubscriptionSpec. Its sole constraint is invertibility: Apply followed by Restore must yield the original spec with no DB or service calls.

## Patterns

**Diffable interface as the unit of work** — Every spec mutation is expressed as a Diffable{GetApplies() AppliesToSpec, GetRestores() AppliesToSpec}. Both methods return an AppliesToSpec passed to spec.Apply(). Never call any repo or domain service inside Diffable implementations. (`diffable, err := addondiff.GetDiffableFromAddon(subView, subsAdd)
if diffable == nil { return nil } // no instances
spec.Apply(diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now})`)
**Gap-based item insertion in apply.go** — apply.go tracks coverage gaps ([]timeutil.OpenPeriod) starting from the full addon cadence, subtracts each existing item's intersection, then creates new SubscriptionItemSpec for remaining gaps. Existing items that overlap are split into difference + intersection fragments. (`gaps := []timeutil.OpenPeriod{*addInPhase}
for _, item := range items {
    nGaps = append(nGaps, g.Difference(itemPer)...)
    // split item into diff + intersection
}`)
**Quantity application asymmetry** — For existing items within the addon cadence, rc.Apply is called Quantity times. For gap-created items (which start from the RateCard base), rc.Apply is called Quantity-1 times because the RateCard already represents one application. (`// existing item in addon cadence:
for range d.addon.Quantity { rc.Apply(inst.RateCard, inst.Annotations) }
// new gap item:
for range d.addon.Quantity - 1 { rc.Apply(inst.RateCard, inst.Annotations) }`)
**Relative cadence via ISODuration offsets** — setItemRelativeCadence converts absolute timestamps to ISO 8601 durations relative to phase start using datetime.ISODurationBetween, stored as ActiveFromOverrideRelativeToPhaseStart / ActiveToOverrideRelativeToPhaseStart. Never store absolute times in item cadence overrides. (`diff := datetime.ISODurationBetween(phaseCadence.ActiveFrom, *target.From)
item.ActiveFromOverrideRelativeToPhaseStart = &diff`)
**Restore merges adjacent equal items** — restore.go undoes addon effects per-item via AddonRateCard.Restore, deletes items whose zeroRateCardCheck.CanDelete() returns true, then merges consecutive items with equal RateCard + Annotations + BillingBehaviorOverride whose cadences are adjacent. (`canMerge := targetItem.RateCard.Equal(testItem.RateCard) &&
    reflect.DeepEqual(targetItem.Annotations, testItem.Annotations) &&
    targetCadence.ActiveTo.Equal(testCadence.ActiveFrom)`)
**Zero rate-card deletion guard** — zeroRateCardCheck in zeroratecard.go determines if a restored item can be deleted by inspecting FlatPrice amount and EntitlementTemplate type/value and OwnerSubSystem annotations. Only if all checks pass is the item safe to remove. (`chk := zeroRateCardCheck{itemAnnotations: item.Annotations, rc: target}
if chk.CanDelete() { rmIdxs = append(rmIdxs, idx) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `diff.go` | Defines the Diffable interface and someDiffable concrete helper. The two-method contract (GetApplies/GetRestores) is the only public API shape this package exports. | Never add I/O here; Diffable is a pure value type with no side-effects. |
| `addon.go` | GetDiffableFromAddon converts a SubscriptionAddon into a Diffable by mapping instances to per-instance diffables aggregated via NewAggregateAppliesToSpec. | Returns nil, nil when addon has no instances — callers must check for nil Diffable before calling spec.Apply. |
| `apply.go` | Core apply algorithm: iterates phases in addon cadence range, manages gaps, splits items, applies rc.Apply() the correct number of times. | Quantity application count differs for existing items (Quantity) vs gap-created items (Quantity-1). Mixing these breaks quantity correctness. |
| `restore.go` | Inverse of apply: calls AddonRateCard.Restore per item, removes zero items, merges adjacent equal fragments. | Merge logic uses reflect.DeepEqual on Annotations and BillingBehaviorOverride — pointer fields must be comparable via DeepEqual. |
| `zeroratecard.go` | Predicate for whether a restored RateCard is effectively empty and safe to delete. Only inspects Price and EntitlementTemplate fields. | Only measures fields that AddonRateCard.Apply/Restore touches — adding checks for other fields here risks false positives. |
| `affected.go` | GetAffectedItemIDs computes a map[rateCardKey][]itemID used by the HTTP layer to populate AffectedSubscriptionItemIds in API responses. | Has a FIXME noting it belongs elsewhere; do not add business logic here. |

## Anti-Patterns

- Adding database or service calls inside Diffable.GetApplies / GetRestores — this package is a pure in-memory transform layer
- Calling spec.Apply with a nil Diffable (GetDiffableFromAddon returns nil when no instances exist)
- Using absolute timestamps in SubscriptionItemSpec cadence fields instead of ISODuration offsets relative to phase start
- Modifying phase.ItemsByKey in-place without rebuilding the slice — restore.go always rebuilds filteredItems and mergedItems slices
- Assuming Apply then Restore is a no-op for all items — only items fully within the addon cadence with zero restored value are deleted

## Decisions

- **Pure in-memory spec mutation with no I/O** — The diff/restore operations must be composable and testable without DB; callers (workflow service) own the persistence boundary.
- **Gap-based insertion rather than full spec rebuild** — Preserves existing item splits from prior edits and only touches the time range covered by the addon cadence.

## Example: Apply addon to spec and later restore it

```
import (
	addondiff "github.com/openmeterio/openmeter/openmeter/subscription/addon/diff"
	"github.com/openmeterio/openmeter/openmeter/subscription"
)

diffable, err := addondiff.GetDiffableFromAddon(subView, subsAdd)
if err != nil { return err }
if diffable == nil { return nil } // no instances

spec := subView.Spec
if err := spec.Apply(diffable.GetApplies(), subscription.ApplyContext{CurrentTime: now}); err != nil {
	return err
}
// later, to undo:
if err := spec.Apply(diffable.GetRestores(), subscription.ApplyContext{CurrentTime: now}); err != nil {
// ...
```

<!-- archie:ai-end -->
