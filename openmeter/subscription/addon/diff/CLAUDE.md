# diff

<!-- archie:ai-start -->

> Pure in-memory spec mutation library that converts a SubscriptionAddon into a Diffable — a bidirectional apply/restore transform on SubscriptionSpec. Its sole constraint is invertibility: Apply followed by Restore must yield the original spec.

## Patterns

**Diffable interface** — Every spec mutation is represented as a Diffable{GetApplies() AppliesToSpec, GetRestores() AppliesToSpec}. Both methods return an AppliesToSpec (a func wrapper) that is passed to spec.Apply(). Never call repo or DB in this package. (`diffable.GetApplies() returns subscription.NewAppliesToSpec(fn); call spec.Apply(diffable.GetApplies(), actx)`)
**Gap-based item insertion** — apply.go tracks coverage gaps ([]timeutil.OpenPeriod) starting from the full addon cadence, subtracts each existing item's intersection, then creates new SubscriptionItemSpec for remaining gaps. Existing items that overlap the addon cadence are split into (difference, intersection) fragments. (`gaps := []timeutil.OpenPeriod{*addInPhase}; for _, item := range items { gaps = subtract(gaps, itemPer); newItems = append(newItems, splitItem...) }`)
**Restore merges adjacent equal items** — restore.go undoes addon effects per-item via AddonRateCard.Restore, deletes items that become zero (zeroRateCardCheck.CanDelete), then merges consecutive items whose RateCard+Annotations+BillingBehaviorOverride are equal and whose cadences are adjacent. (`canMerge := targetItem.RateCard.Equal(testItem.RateCard) && targetCadence.ActiveTo.Equal(testCadence.ActiveFrom)`)
**Relative cadence via ISODuration offsets** — setItemRelativeCadence converts absolute timestamps to ISO 8601 durations relative to phase start using datetime.ISODurationBetween, stored as ActiveFromOverrideRelativeToPhaseStart / ActiveToOverrideRelativeToPhaseStart. (`diff := datetime.ISODurationBetween(phaseCadence.ActiveFrom, *target.From); item.ActiveFromOverrideRelativeToPhaseStart = &diff`)
**Zero rate-card deletion guard** — zeroRateCardCheck in zeroratecard.go determines if a restored item can be deleted by inspecting FlatPrice amount, EntitlementTemplate type/value, and OwnerSubSystem annotations. An item is deletable only if all three checks pass. (`chk := zeroRateCardCheck{itemAnnotations: item.Annotations, rc: target}; if chk.CanDelete() { rmIdxs = append(rmIdxs, idx) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `diff.go` | Defines the Diffable interface and someDiffable helper struct. The entire package is built on this two-method contract. | Never add DB calls here; Diffable is a pure value type. |
| `addon.go` | GetDiffableFromAddon converts a SubscriptionAddon into a Diffable by mapping instances to per-instance diffables and aggregating them via NewAggregateAppliesToSpec. | Returns nil, nil when addon has no instances — callers must handle nil Diffable. |
| `apply.go` | Core apply algorithm: iterates phases in addon cadence range, manages gaps, splits items, applies rc.Apply() quantity times. | Item quantity application uses `for range d.addon.Quantity` (new gaps) vs `for range d.addon.Quantity - 1` (existing items that already have one application). Mixing these counts breaks quantity correctness. |
| `restore.go` | Inverse of apply: calls AddonRateCard.Restore per item, removes zero items, merges adjacent equal fragments. | Merge logic uses reflect.DeepEqual on Annotations and BillingBehaviorOverride — pointer fields must be comparable. |
| `zeroratecard.go` | Predicate for whether a restored RateCard is effectively empty and safe to delete. | Only inspects Price and EntitlementTemplate fields that AddonRateCard.Apply/Restore touches — other fields are ignored intentionally. |
| `affected.go` | GetAffectedItemIDs computes a map[rateCardKey][]itemID used by the HTTP mapping layer to populate AffectedSubscriptionItemIds in the API response. | Has a FIXME noting it belongs elsewhere; do not add business logic here. |

## Anti-Patterns

- Adding database or service calls inside Diffable.GetApplies / GetRestores — this is a pure in-memory transform layer
- Calling spec.Apply with a nil Diffable (GetDiffableFromAddon returns nil when no instances exist)
- Using absolute timestamps inside SubscriptionItemSpec cadence fields instead of ISODuration offsets relative to phase start
- Modifying phase.ItemsByKey in-place without rebuilding the slice — restore.go always rebuilds filteredItems and mergedItems slices
- Assuming Apply then Restore is always a no-op for items that existed before the addon — only items fully within the addon cadence and with zero restored value are deleted

## Decisions

- **Pure in-memory spec mutation with no I/O** — The diff/restore operations must be composable and testable without DB; callers (workflow service) own the persistence boundary.
- **Gap-based insertion rather than full spec rebuild** — Preserves existing item splits (e.g. from prior edits) and only touches the time range covered by the addon cadence.

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
