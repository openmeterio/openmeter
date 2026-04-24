# persistedstate

<!-- archie:ai-start -->

> Provides a typed snapshot of subscription-related billing state already stored in the database (invoice lines, split-line hierarchies, flat-fee charges, usage-based charges), keyed by ChildUniqueReferenceID, so the subscription-sync reconciler can diff persisted state against target state without re-querying inside the diff loop.

## Patterns

**Sealed Item type hierarchy** — All concrete item types (persistedLine, persistedSplitLineHierarchy, persistedUsageBasedCharge, persistedFlatFeeCharge) implement the Item interface but are package-private. Callers access backing data through typed getter interfaces (LineGetter, SplitLineHierarchyGetter, UsageBasedChargeGetter, FlatFeeChargeGetter) via type assertions using the exported ItemAs* helpers. (`line, err := persistedstate.ItemAsLine(item)  // returns error if item is not line-backed`)
**Construction-time validation** — Private constructors (newPersistedLine, newPersistedUsageBasedCharge, newPersistedFlatFeeCharge, newPersistedSplitLineHierarchy) validate the wrapped value before returning; callers receive an error rather than an invalid Item. (`item, err := newPersistedUsageBasedCharge(charge)  // calls charge.Validate() inside`)
**MinimumWindowSizeDuration normalization on load** — normalizePersistedLineOrHierarchy and normalizeSubscriptionReference truncate all timestamps to streaming.MinimumWindowSizeDuration on read so legacy sub-second precision in the DB never leaks into the diff. Must be applied to every line/hierarchy loaded from billing service. (`period.Truncate(streaming.MinimumWindowSizeDuration)`)
**Unified ByUniqueID map merging lines and charges** — State.ByUniqueID merges invoice lines/hierarchies and charges into a single map keyed by ChildUniqueReferenceID. Duplicate keys across lines and charges are treated as an error; the loader enforces this during LoadForSubscription. (`if _, ok := byUniqueID[uniqueID]; ok { return State{}, fmt.Errorf("duplicate unique ids") }`)
**Nil chargeService guard** — Loader.loadChargesForSubscription returns an empty map when chargeService is nil, allowing callers to construct a Loader without a charge service for contexts where charges are not yet enabled. (`if l.chargeService == nil { return map[string]Item{}, nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `item.go` | Defines the Item interface, all four concrete implementations, typed getter interfaces, and the public factory functions NewItemFromLineOrHierarchy and NewChargeItemFromChargeType. | All concrete types have compile-time interface guards (var _ Item = ...). Adding a new item type requires implementing all Item methods and adding a matching ItemAs* helper and NewChargeItemFromChargeType case. |
| `loader.go` | Loads persisted state for a subscription by calling billingService.GetLinesForSubscription and chargeService.ListCharges, normalizes timestamps, and assembles the unified State. | normalizePersistedLineOrHierarchy must be applied to every line returned from GetLinesForSubscription; credit purchase charges tied to subscriptions are explicitly unsupported and return an error. |
| `state.go` | Defines the State and Invoices types and their Validate methods; Invoices.IsGatheringInvoice is the only query method on the invoice map. | State.Validate returns an error if either map is nil; callers must always initialize via LoadForSubscription, not by hand-constructing a State{} without populating both fields. |

## Anti-Patterns

- Constructing a concrete persistedLine/persistedXxx struct directly — use the private constructors via NewItemFromLineOrHierarchy or NewChargeItemFromChargeType to get construction-time validation
- Skipping normalizePersistedLineOrHierarchy when loading lines — sub-second timestamps will leak into diff decisions and produce phantom no-op repairs
- Accessing backing data via direct type assertion to the concrete struct type — use the exported ItemAs* helpers which return a typed error
- Adding credit purchase charges to a subscription's persisted state — explicitly unsupported; return an error and fail loudly
- Hand-constructing State{} with nil maps — Validate() will reject it; always build via Loader.LoadForSubscription

## Decisions

- **Item interface with private concrete types and public ItemAs* accessors** — Keeps construction-time validation inside the package and forces callers to handle the error path when the item is not of the expected type, preventing silent nil-dereferences in the reconciler.
- **Timestamp normalization on read rather than at write time** — Historical rows in the DB already carry sub-second precision; normalizing on load avoids a migration dependency while ensuring the reconciler always compares at MinimumWindowSizeDuration resolution.

## Example: Load persisted state for a subscription and extract a line from an Item

```
import (
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
)

loader := persistedstate.NewLoader(billingService, chargeService)
state, err := loader.LoadForSubscription(ctx, subs)
if err != nil { return err }

item, ok := state.ByUniqueID[uniqueID]
if !ok { /* not yet persisted */ }
if line, err := persistedstate.ItemAsLine(item); err == nil {
	// use line
}
```

<!-- archie:ai-end -->
