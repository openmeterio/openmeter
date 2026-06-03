# targetstate

<!-- archie:ai-start -->

> Computes the desired (target) set of billing line items for a subscription as of a point in time — iterating subscription phases and rate-card cadences via PhaseIterator, then correcting period boundaries against persisted state — so the reconciler knows exactly what should exist in the DB.

## Patterns

**PhaseIterator per-phase generation** — collectUpcomingLines creates one PhaseIterator per phase, advancing through aligned billing periods up to generationLimit (never past the current period's end). HasInvoicableItems() gates empty/zero-length phases. (`iterator, err := NewPhaseIterator(b.logger, b.tracer, subs, phase.SubscriptionPhase.Key); if !iterator.HasInvoicableItems() { continue }; items, err := iterator.Generate(ctx, generationLimit)`)
**UniqueID composite key** — Every generated item carries UniqueID = subID/phaseKey/itemKey/v[N]/period[N] joined by '/'. This MUST match the ChildUniqueReferenceID format in persistedstate so the diff can correlate. (`UniqueID: strings.Join([]string{sub.ID, phase.Spec.PhaseKey, item.Spec.ItemKey, fmt.Sprintf("v[%d]", version), fmt.Sprintf("period[%d]", periodIdx)}, "/")`)
**MinimumWindowSizeDuration truncation of periods** — truncateItemsIfNeeded truncates ServicePeriod, BillingPeriod, FullServicePeriod on every item; non-flat-fee items with empty service periods after truncation are dropped. (`item.ServicePeriod = item.ServicePeriod.Truncate(streaming.MinimumWindowSizeDuration)`)
**iterationEnd end-of-second extension** — Generate() truncates iterationEnd then adds (MinimumWindowSizeDuration - time.Nanosecond) so the last window is fully included. Never pass raw asOf as iterationEnd. (`iterationEnd = iterationEnd.Truncate(streaming.MinimumWindowSizeDuration).Add(streaming.MinimumWindowSizeDuration - time.Nanosecond)`)
**Period-start continuity from persisted state** — correctPeriodStartForUpcomingLines aligns a period's start to the persisted previous period's end when it carries AnnotationSubscriptionSyncIgnore + ForceContinuousLines; the boundary is truncated to MinimumWindowSizeDuration. (`inScopeLines[idx].ServicePeriod.From = previousServicePeriod.To.Truncate(streaming.MinimumWindowSizeDuration)`)
**GetExpectedLine returns nil (not error) for zero-amount** — StateItem.GetExpectedLine returns (nil, nil) when a pro-rated flat-fee amount rounds to zero — meaning 'do not create'. Use GetExpectedLineOrErr only when a line is mandatory. (`line, err := item.GetExpectedLine(); if err != nil { return err }; if line == nil { /* skip — zero amount */ }`)
**tracex.Start spans on public methods** — Non-trivial Builder/PhaseIterator methods wrap their body in tracex.Start for OTel tracing of generation timing. (`span := tracex.Start[State](ctx, b.tracer, "...targetstate.Build"); return span.Wrap(func(ctx) (State, error) { ... })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `targetstate.go` | Builder.Build entry point: validates input, caps view at customer deletion, iterates phases, corrects period starts, returns assembled State. | generationLimit logic advances to currBillingPeriod.To unless asOf precedes subscription start or phase hasn't started. Customer-deleted-at capping (withActiveTo) is the only sanctioned guard against generating past the customer lifecycle. |
| `phaseiterator.go` | Generates SubscriptionItemWithPeriods for billable items in one phase (recurring, one-time, aligned periods, truncation). | maxSafeIter=1000 guards infinite loops. Extend iterationEnd to end-of-second before generateAligned. Preserve the UniqueID composite format exactly. |
| `targetstateitem.go` | StateItem with GetExpectedLine, IsBillable, shouldProrate and price dispatch. | GetExpectedLine returns (nil, nil) for zero-amount flat fee — check nil separately from error. shouldProrate returns false when the subscription ends within the service period (avoids over-proration on cancellation). |
| `phaseiterator_test.go` | Table-driven testify/suite tests for aligned/unaligned cadences, phase boundaries, version splits, truncation. | New cases append to tcs and call suite.Run; expected Key UniqueID format must match phaseiterator.go exactly. |

## Anti-Patterns

- Passing raw asOf as iterationEnd — must extend to end-of-second first.
- Constructing UniqueID with a different separator or field order than subID/phaseKey/itemKey/v[N]/period[N].
- Returning an error from GetExpectedLine when the line is nil — nil means 'do not create'.
- Skipping correctPeriodStartForUpcomingLines — continuity annotations get ignored and spurious repairs proposed.
- Generating lines past the customer's deleted-at — withActiveTo capping is the only sanctioned guard.

## Decisions

- **PhaseIterator generates target state purely from the subscription view without DB access.** — Stateless generation makes the target-vs-persisted diff a pure in-memory computation, idempotent and safe to retry.
- **correctPeriodStartForUpcomingLines patches boundaries from persisted state.** — When a previous period was manually frozen (SyncIgnore + ForceContinuousLines), the next period must start exactly where it ended regardless of cadence-computed start.
- **GetExpectedLine returns nil for zero-amount lines instead of pre-filtering.** — Separating 'nothing to bill' (nil) from 'error' lets the reconciler distinguish legitimate zero-value periods from failures without a mandatory IsBillable pre-check.

## Example: Build target state and iterate billable items

```
import "github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"

builder := targetstate.NewBuilder(logger, tracer)
state, err := builder.Build(ctx, targetstate.BuildInput{AsOf: asOf, SubscriptionView: subsView, Persisted: persistedState})
if err != nil { return err }
for _, item := range state.Items {
	if !item.IsBillable() { continue }
	// ...
}
```

<!-- archie:ai-end -->
