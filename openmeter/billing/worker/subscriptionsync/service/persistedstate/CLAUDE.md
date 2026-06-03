# persistedstate

<!-- archie:ai-start -->

> Provides a typed snapshot of subscription-related billing state already in the DB (invoice lines, split-line hierarchies, flat-fee and usage-based charges) keyed by ChildUniqueReferenceID, so the subscription-sync reconciler can diff persisted vs target state without re-querying inside the diff loop.

## Patterns

**Sealed Item type hierarchy** — All four concrete types (persistedLine, persistedSplitLineHierarchy, persistedUsageBasedCharge, persistedFlatFeeCharge) implement Item but are package-private; callers use exported ItemAs* helpers returning typed errors. (`line, err := persistedstate.ItemAsLine(item) // error if item is not line-backed`)
**Construction-time validation via private constructors** — Private constructors (newPersistedLine, etc.) call Validate()/nil-check before returning. Public entry points are NewItemFromLineOrHierarchy and NewChargeItemFromChargeType. (`item, err := newPersistedUsageBasedCharge(charge) // calls charge.Validate() inside`)
**MinimumWindowSizeDuration normalization on load** — normalizePersistedLineOrHierarchy and normalizeSubscriptionReference truncate timestamps to streaming.MinimumWindowSizeDuration on read; apply to every line/hierarchy from GetLinesForSubscription. (`period.Truncate(streaming.MinimumWindowSizeDuration)`)
**Unified ByUniqueID map** — State.ByUniqueID merges lines/hierarchies and charges into one map keyed by ChildUniqueReferenceID; duplicate keys are a load-time error. (`if _, ok := byUniqueID[uniqueID]; ok { return State{}, fmt.Errorf("duplicate unique ids") }`)
**Nil chargeService guard** — Loader.loadChargesForSubscription returns an empty map when chargeService is nil, allowing construction without charges enabled. (`if l.chargeService == nil { return map[string]Item{}, nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `item.go` | Item interface, four concrete impls, typed getter interfaces, and public factories NewItemFromLineOrHierarchy / NewChargeItemFromChargeType. | Each concrete type has a var _ Item = ... guard. A new item type needs all Item methods, a matching ItemAs* helper, and a NewChargeItemFromChargeType case. Credit-purchase charges are explicitly rejected. |
| `loader.go` | Loads state via billingService.GetLinesForSubscription and chargeService.ListCharges, normalizes timestamps, assembles unified State. | normalizePersistedLineOrHierarchy must apply to every loaded line; subscription-tied credit-purchase charges return an error and must never be silently ignored. |
| `state.go` | State and Invoices types with Validate; Invoices.IsGatheringInvoice is the only query method. | State.Validate errors if either map is nil — always build via LoadForSubscription, never hand-construct State{}. |

## Anti-Patterns

- Constructing a concrete persistedXxx struct directly instead of via the private constructors / factories.
- Skipping normalizePersistedLineOrHierarchy on load — sub-second timestamps cause phantom no-op repairs.
- Accessing backing data via direct type assertion instead of ItemAs* helpers.
- Adding credit-purchase charges to a subscription's persisted state — explicitly unsupported, must error.
- Hand-constructing State{} with nil maps — Validate() rejects it.

## Decisions

- **Item interface with private concrete types and public ItemAs* accessors.** — Keeps construction-time validation inside the package and forces callers to handle the wrong-type error path, preventing silent nil-dereferences in the reconciler.
- **Timestamp normalization on read rather than at write time.** — Historical rows carry sub-second precision; normalizing on load avoids a migration while ensuring diffs compare at MinimumWindowSizeDuration resolution.

## Example: Load persisted state and extract a line

```
import "github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"

loader := persistedstate.NewLoader(billingService, chargeService)
state, err := loader.LoadForSubscription(ctx, subs)
if err != nil { return err }
item, ok := state.ByUniqueID[uniqueID]
if !ok { /* not yet persisted */ }
if line, err := persistedstate.ItemAsLine(item); err == nil { /* use line */ }
```

<!-- archie:ai-end -->
