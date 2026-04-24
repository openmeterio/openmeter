# targetstate

<!-- archie:ai-start -->

> Computes the desired (target) set of billing line items for a subscription as of a given point in time, by iterating over subscription phases and rate-card cadences via PhaseIterator, then correcting period boundaries against persisted state, so the reconciler knows exactly what should exist in the database.

## Patterns

**PhaseIterator per-phase generation** — Builder.collectUpcomingLines creates one PhaseIterator per subscription phase. Each iterator advances through aligned billing periods up to a generationLimit derived from the current billing period boundary, never past the current period's end. (`iterator, err := NewPhaseIterator(b.logger, b.tracer, subs, phase.SubscriptionPhase.Key)
if !iterator.HasInvoicableItems() { continue }
items, err := iterator.Generate(ctx, generationLimit)`)
**UniqueID as subID/phaseKey/itemKey/v[N]/period[N] composite key** — Every generated SubscriptionItemWithPeriods carries a UniqueID composed of subscription ID, phase key, item key, version index, and period index joined with '/'. This must match the ChildUniqueReferenceID format used in persistedstate so the diff can correlate target and persisted items. (`UniqueID: strings.Join([]string{sub.ID, phase.Spec.PhaseKey, item.Spec.ItemKey, fmt.Sprintf("v[%d]", version), fmt.Sprintf("period[%d]", periodIdx)}, "/")`)
**MinimumWindowSizeDuration truncation of all periods** — truncateItemsIfNeeded applies streaming.MinimumWindowSizeDuration truncation to ServicePeriod, BillingPeriod, and FullServicePeriod on every generated item. Non-flat-fee items with empty service periods after truncation are dropped. (`item.ServicePeriod = item.ServicePeriod.Truncate(streaming.MinimumWindowSizeDuration)`)
**Period-start continuity correction from persisted state** — correctPeriodStartForUpcomingLines adjusts the start of a generated period to align with the persisted previous period's end when the previous line carries AnnotationSubscriptionSyncIgnore + AnnotationSubscriptionSyncForceContinuousLines. The carried boundary is also truncated to MinimumWindowSizeDuration. (`continuousStart := previousServicePeriod.To.Truncate(streaming.MinimumWindowSizeDuration)
inScopeLines[idx].ServicePeriod.From = continuousStart`)
**Customer-deleted-at subscription view capping** — If the customer was deleted before the subscription's ActiveTo, Builder.Build caps the subscription view's ActiveTo at CustomerDeletedAt to prevent the iterator from generating lines past the customer's lifecycle. (`subs = withActiveTo(subs, *input.CustomerDeletedAt)`)
**StateItem.GetExpectedLine for gathering-line construction** — StateItem.GetExpectedLine converts a SubscriptionItemWithPeriods into a *billing.GatheringLine including proration for flat-fee items. Returns nil (not an error) when the computed amount rounds to zero — callers must handle nil as 'do not create line'. (`line, err := item.GetExpectedLine()
if err != nil { return err }
if line == nil { /* skip */ }`)
**tracex.Start spans on all public methods** — Every non-trivial method on Builder and PhaseIterator wraps its body in a tracex.Start span so OTel tracing captures subscription-sync generation timing end-to-end. (`span := tracex.Start[State](ctx, b.tracer, "billing.worker.subscription.sync.targetstate.Build")
return span.Wrap(func(ctx context.Context) (State, error) { ... })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `targetstate.go` | Entry point: Builder.Build validates input, caps subscription view at customer deletion, iterates phases via collectUpcomingLines, corrects period starts, and returns a fully assembled State. | generationLimit logic in collectUpcomingLines is subtle: it advances to currBillingPeriod.To unless asOf is before subscription start or the phase hasn't started yet; changes here break the lookahead window. |
| `phaseiterator.go` | Generates SubscriptionItemWithPeriods for all billable items in a single phase, handling recurring cadences (advancing at, periodIdx loop), one-time items, aligned billing periods, and truncation. | maxSafeIter=1000 guards against infinite loops; iterationEnd is extended to end-of-second before passing to generateAligned to compensate for 1s resolution. Do not pass a raw asOf as iterationEnd. |
| `targetstateitem.go` | Defines StateItem embedding SubscriptionItemWithPeriods and exposing GetExpectedLine, IsBillable, shouldProrate. Contains all proration and flat-fee/usage-based price dispatch logic. | GetExpectedLine returns (nil, nil) for zero-amount pro-rated flat fee — callers must check for nil line separately from error. shouldProrate returns false when the subscription ends within the service period. |
| `phaseiterator_test.go` | Table-driven tests for PhaseIterator covering aligned/unaligned cadences, phase boundaries, version splits, and truncation edge cases. | Uses suite-level Assertions and testify/suite; new test cases must add to the tcs slice and call suite.Run. |

## Anti-Patterns

- Passing a raw asOf time as iterationEnd to PhaseIterator.Generate — must extend to end-of-second via Add(MinimumWindowSizeDuration - time.Nanosecond) first
- Constructing SubscriptionItemWithPeriods.UniqueID with a different separator or field order — the format subID/phaseKey/itemKey/v[N]/period[N] must match persistedstate exactly
- Returning an error from GetExpectedLine when the expected line is nil — nil means 'do not create', not a failure; use GetExpectedLineOrErr only when a line is mandatory
- Skipping correctPeriodStartForUpcomingLines — continuity annotations (AnnotationSubscriptionSyncForceContinuousLines) will be ignored and sync will propose spurious period-start repairs
- Adding generation logic that creates lines past the customer's deleted-at timestamp — the Builder.Build capping via withActiveTo is the only sanctioned guard

## Decisions

- **PhaseIterator generates target state purely from the subscription view without DB access** — Keeping generation stateless (no DB calls) means the diff between target and persisted state is a pure in-memory computation, making it idempotent and safe to retry.
- **correctPeriodStartForUpcomingLines patches generated period boundaries from persisted state** — When a previous period was manually frozen (SyncIgnore + ForceContinuousLines), the next period must start exactly where the frozen period ends, even if the iterator would compute a different start from the subscription cadence.
- **StateItem.GetExpectedLine returns nil for zero-amount lines instead of filtering before returning** — Separating 'nothing to bill' (nil) from 'error' lets the reconciler distinguish legitimate zero-value periods from calculation failures without a separate IsBillable pre-check being mandatory.

## Example: Build target state and get expected gathering lines for all billable items

```
import (
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
)

builder := targetstate.NewBuilder(logger, tracer)
state, err := builder.Build(ctx, targetstate.BuildInput{
	AsOf:             asOf,
	SubscriptionView: subsView,
	Persisted:        persistedState,
})
if err != nil { return err }

for _, item := range state.Items {
	if !item.IsBillable() { continue }
// ...
```

<!-- archie:ai-end -->
