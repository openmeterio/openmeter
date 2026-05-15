# targetstate

<!-- archie:ai-start -->

> Computes the desired (target) set of billing line items for a subscription as of a given point in time, by iterating over subscription phases and rate-card cadences via PhaseIterator, then correcting period boundaries against persisted state, so the reconciler knows exactly what should exist in the database.

## Patterns

**PhaseIterator per-phase generation** — Builder.collectUpcomingLines creates one PhaseIterator per subscription phase. Each iterator advances through aligned billing periods up to a generationLimit derived from the current billing period boundary, never past the current period's end. HasInvoicableItems() gates generation for empty or zero-length phases. (`iterator, err := NewPhaseIterator(b.logger, b.tracer, subs, phase.SubscriptionPhase.Key)
if !iterator.HasInvoicableItems() { continue }
items, err := iterator.Generate(ctx, generationLimit)`)
**UniqueID as subID/phaseKey/itemKey/v[N]/period[N] composite key** — Every generated SubscriptionItemWithPeriods carries a UniqueID composed of subscription ID, phase key, item key, version index, and period index joined with '/'. This format MUST match the ChildUniqueReferenceID format used in persistedstate so the diff can correlate target and persisted items. (`UniqueID: strings.Join([]string{sub.ID, phase.Spec.PhaseKey, item.Spec.ItemKey, fmt.Sprintf("v[%d]", version), fmt.Sprintf("period[%d]", periodIdx)}, "/")`)
**MinimumWindowSizeDuration truncation of all periods** — truncateItemsIfNeeded applies streaming.MinimumWindowSizeDuration truncation to ServicePeriod, BillingPeriod, and FullServicePeriod on every generated item. Non-flat-fee items with empty service periods after truncation are dropped. (`item.ServicePeriod = item.ServicePeriod.Truncate(streaming.MinimumWindowSizeDuration)`)
**iterationEnd end-of-second extension before generateAligned** — Generate() truncates iterationEnd to MinimumWindowSizeDuration then adds (MinimumWindowSizeDuration - time.Nanosecond) so the last 1-second window is fully included before passing to generateAligned. Never pass a raw asOf directly as iterationEnd. (`iterationEnd = iterationEnd.Truncate(streaming.MinimumWindowSizeDuration).Add(streaming.MinimumWindowSizeDuration - time.Nanosecond)`)
**Period-start continuity correction from persisted state** — correctPeriodStartForUpcomingLines adjusts the start of a generated period to align with the persisted previous period's end when the previous line carries AnnotationSubscriptionSyncIgnore + AnnotationSubscriptionSyncForceContinuousLines. The carried boundary is also truncated to MinimumWindowSizeDuration. (`continuousStart := previousServicePeriod.To.Truncate(streaming.MinimumWindowSizeDuration)
inScopeLines[idx].ServicePeriod.From = continuousStart`)
**GetExpectedLine returns nil for zero-amount lines, not an error** — StateItem.GetExpectedLine returns (nil, nil) when a pro-rated flat-fee amount rounds to zero. Callers must explicitly check for nil line — it means 'do not create'. Use GetExpectedLineOrErr only when a line is mandatory. (`line, err := item.GetExpectedLine()
if err != nil { return err }
if line == nil { /* skip — zero amount */ }`)
**tracex.Start spans on all public methods** — Every non-trivial method on Builder and PhaseIterator wraps its body in a tracex.Start span for end-to-end OTel tracing of subscription-sync generation timing. (`span := tracex.Start[State](ctx, b.tracer, "billing.worker.subscription.sync.targetstate.Build")
return span.Wrap(func(ctx context.Context) (State, error) { ... })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `targetstate.go` | Entry point: Builder.Build validates input, caps subscription view at customer deletion, iterates phases via collectUpcomingLines, corrects period starts, and returns a fully assembled State. | generationLimit logic in collectUpcomingLines is subtle — it advances to currBillingPeriod.To unless asOf is before subscription start or the phase hasn't started yet. Changes here break the lookahead window. Customer-deleted-at capping (withActiveTo) is the only sanctioned guard against generating lines past the customer lifecycle. |
| `phaseiterator.go` | Generates SubscriptionItemWithPeriods for all billable items in a single phase, handling recurring cadences, one-time items, aligned billing periods, and truncation. | maxSafeIter=1000 guards against infinite loops. iterationEnd must be extended to end-of-second before passing to generateAligned — never pass raw asOf. UniqueID composite key format subID/phaseKey/itemKey/v[N]/period[N] must be preserved exactly. |
| `targetstateitem.go` | Defines StateItem embedding SubscriptionItemWithPeriods and exposing GetExpectedLine, IsBillable, shouldProrate with all proration and price dispatch logic. | GetExpectedLine returns (nil, nil) for zero-amount pro-rated flat fee — callers must check for nil separately from error. shouldProrate returns false when subscription ends within the service period (prevents over-proration on cancellation). |
| `phaseiterator_test.go` | Table-driven tests for PhaseIterator covering aligned/unaligned cadences, phase boundaries, version splits, and truncation edge cases. | Uses suite-level Assertions and testify/suite. New test cases must add to the tcs slice and call suite.Run. UniqueID format in expected Key fields must match the format in phaseiterator.go exactly. |

## Anti-Patterns

- Passing a raw asOf time as iterationEnd to PhaseIterator.Generate — must extend to end-of-second via Add(MinimumWindowSizeDuration - time.Nanosecond) first
- Constructing SubscriptionItemWithPeriods.UniqueID with a different separator or field order — the format subID/phaseKey/itemKey/v[N]/period[N] must match persistedstate exactly
- Returning an error from GetExpectedLine when the expected line is nil — nil means 'do not create', not a failure; use GetExpectedLineOrErr only when a line is mandatory
- Skipping correctPeriodStartForUpcomingLines — continuity annotations (AnnotationSubscriptionSyncForceContinuousLines) will be ignored and sync will propose spurious period-start repairs
- Adding generation logic that creates lines past the customer's deleted-at timestamp — Builder.Build capping via withActiveTo is the only sanctioned guard

## Decisions

- **PhaseIterator generates target state purely from the subscription view without DB access** — Keeping generation stateless (no DB calls) means the diff between target and persisted state is a pure in-memory computation, making it idempotent and safe to retry.
- **correctPeriodStartForUpcomingLines patches generated period boundaries from persisted state** — When a previous period was manually frozen (SyncIgnore + ForceContinuousLines), the next period must start exactly where the frozen period ends, even if the iterator would compute a different start from the subscription cadence.
- **StateItem.GetExpectedLine returns nil for zero-amount lines instead of filtering before returning** — Separating 'nothing to bill' (nil) from 'error' lets the reconciler distinguish legitimate zero-value periods from calculation failures without a mandatory IsBillable pre-check.

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
