# patch

<!-- archie:ai-start -->

> Provides concrete Patch implementations (PatchAddItem, PatchRemoveItem, PatchAddPhase, PatchRemovePhase, PatchStretchPhase, PatchUnscheduleEdit) that mutate a *SubscriptionSpec in-memory. These are the only authorised mutation primitives for subscription specs.

## Patterns

**Patch interface compliance assertion** — Each concrete patch asserts subscription.Patch or subscription.ValuePatch[T] via blank var _ at file bottom. (`var _ subscription.ValuePatch[subscription.SubscriptionItemSpec] = PatchAddItem{}`)
**ApplyTo operates on *SubscriptionSpec** — Every patch implements ApplyTo(spec *subscription.SubscriptionSpec, actx subscription.ApplyContext) error and is the sole mutation point for the spec. (`func (a PatchAddItem) ApplyTo(spec *subscription.SubscriptionSpec, actx subscription.ApplyContext) error { ... }`)
**Temporal guard: no past-phase mutations** — All add/remove/stretch patches return PatchForbiddenError if the target phase starts before the current phase computed from actx.CurrentTime. (`if phaseStartTime.Before(currentPhaseStartTime) { return &subscription.PatchForbiddenError{...} }`)
**Current-phase item versioning: close then append** — When adding or removing items in the current phase, the existing item is soft-closed (ActiveToOverrideRelativeToPhaseStart set) and a new spec entry appended — never deleted — to preserve audit history. (`itemToClose.ActiveToOverrideRelativeToPhaseStart = a.CreateInput.ActiveFromOverrideRelativeToPhaseStart
phase.ItemsByKey[a.ItemKey] = append(phase.ItemsByKey[a.ItemKey], &a.CreateInput)`)
**ISO duration arithmetic for phase spacing** — AddPhase and StretchPhase compute phase spacing adjustments using datetime.ISODuration.Add/Subtract to preserve relative gaps, not wall-clock arithmetic. (`sa, err := p.StartAfter.Add(diff)
sortedPhases[i].StartAfter = sa`)
**Test suite pattern: testsuite[T] + testcase[T]** — All patch tests use a generic testsuite[T subscription.AppliesToSpec] with GetSpec / GetExpectedSpec lambdas; call suite.Run(t) to iterate cases. (`suite := testsuite[patch.PatchAddItem]{TT: []testcase[patch.PatchAddItem]{...}}
suite.Run(t)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `additem.go` | PatchAddItem: adds a new item version to a phase; handles current-phase auto-close and ActiveFromOverrideRelativeToPhaseStart defaulting to now. | If ActiveFromOverrideRelativeToPhaseStart is nil when adding to the current phase, the patch auto-computes it from phaseStart to actx.CurrentTime — callers need not pre-fill it. |
| `removeitem.go` | PatchRemoveItem: soft-removes last item version in current phase or hard-deletes from future phase. | Future-phase removal deletes the entry; current-phase removal only sets ActiveTo. Both cases leave prior versions intact. |
| `addphase.go` | PatchAddPhase: inserts a new future phase and shifts all later phases by the new phase's duration. | Duration field on CreateSubscriptionPhaseInput drives the shift; omitting it leaves no gap before the next phase. |
| `removephase.go` | PatchRemovePhase: deletes a future phase with Shift=Next (compress) or Shift=Prev (leave gap) semantics. | Only future phases can be removed; current or past phases return PatchForbiddenError. |
| `stretchphase.go` | PatchStretchPhase: shifts all phases after the target by a signed ISO duration. | Negative durations are allowed (shrink); guard checks that no phase is compressed to zero length. |
| `unscheduleedit.go` | PatchUnscheduleEdit: removes all future scheduled item versions from the current phase, leaving only the currently active version. | Operates only on the current phase; no-ops gracefully if no future edits exist. |
| `patch_test.go` | Shared test infrastructure: testsuite/testcase generics, getDefaultSpec helper, TestRemoveAdd integration. | getDefaultSpec calls subscriptiontestutils.GetExamplePlanInput; tests assume the default plan has at least 3 phases named test_phase_1/2/3. |

## Anti-Patterns

- Mutating spec.Phases directly outside an ApplyTo method — bypasses the temporal and versioning guards.
- Using wall-clock time.Duration arithmetic instead of datetime.ISODuration for phase spacing adjustments.
- Returning a plain error instead of PatchValidationError/PatchForbiddenError/PatchConflictError — callers distinguish these types.
- Hard-deleting items from the current phase instead of soft-closing them.

## Decisions

- **Items in the current phase are soft-closed rather than deleted.** — Cannot falsify history; billing sync reads item cadences to determine billable periods, so deleting would lose the active window.
- **Phase spacing adjustments use ISO duration arithmetic, not absolute timestamps.** — Subscription phases use relative StartAfter durations from subscription.ActiveFrom; preserving them as ISO durations avoids DST and calendar edge cases.

## Example: Apply a sequence of remove-then-add patches atomically to a spec

```
import (
	"github.com/samber/lo"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
)

patches := []subscription.Patch{
	&patch.PatchRemoveItem{PhaseKey: "phase-1", ItemKey: "feature-key"},
	&patch.PatchAddItem{PhaseKey: "phase-1", ItemKey: "feature-key", CreateInput: newSpec},
}
err := spec.ApplyMany(
	lo.Map(patches, subscription.ToApplies),
	subscription.ApplyContext{CurrentTime: clock.Now()},
)
```

<!-- archie:ai-end -->
