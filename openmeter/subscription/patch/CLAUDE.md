# patch

<!-- archie:ai-start -->

> Concrete Patch implementations (PatchAddItem, PatchRemoveItem, PatchAddPhase, PatchRemovePhase, PatchStretchPhase, PatchUnscheduleEdit) that mutate a *SubscriptionSpec in-memory via ApplyTo. These are the only authorised mutation primitives for subscription specs.

## Patterns

**Patch interface compliance assertion** — Each patch asserts subscription.Patch or subscription.ValuePatch[T] via blank var _. (`var _ subscription.ValuePatch[subscription.SubscriptionItemSpec] = PatchAddItem{}`)
**ApplyTo is the sole mutation point** — Every patch implements ApplyTo(spec *subscription.SubscriptionSpec, actx subscription.ApplyContext) error. (`func (a PatchAddItem) ApplyTo(spec *subscription.SubscriptionSpec, actx subscription.ApplyContext) error { ... }`)
**Temporal guard: no past-phase mutations** — Add/remove/stretch patches return PatchForbiddenError if the target phase starts before the current phase computed from actx.CurrentTime. (`if phaseStartTime.Before(currentPhaseStartTime) { return &subscription.PatchForbiddenError{...} }`)
**Current-phase item versioning: close then append** — Items in the current phase are soft-closed (ActiveToOverrideRelativeToPhaseStart) and a new entry appended — never deleted. (`phase.ItemsByKey[a.ItemKey] = append(phase.ItemsByKey[a.ItemKey], &a.CreateInput)`)
**ISO duration arithmetic for phase spacing** — AddPhase/StretchPhase shift phases with datetime.ISODuration.Add/Subtract, not wall-clock arithmetic. (`sa, err := p.StartAfter.Add(diff); sortedPhases[i].StartAfter = sa`)
**Typed patch errors** — Return PatchValidationError / PatchForbiddenError / PatchConflictError so callers can distinguish failure modes. (`return &subscription.PatchValidationError{Msg: fmt.Sprintf("phase %s not found", a.PhaseKey)}`)
**Generic test suite** — Patch tests use testsuite[T subscription.AppliesToSpec] + testcase[T] with GetSpec/GetExpectedSpec lambdas; call suite.Run(t). (`suite := testsuite[patch.PatchAddItem]{...}; suite.Run(t)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `additem.go` | PatchAddItem: adds a new item version to a phase; handles current-phase auto-close. | If ActiveFromOverrideRelativeToPhaseStart is nil when adding to the current phase it is auto-computed from phaseStart to actx.CurrentTime. |
| `removeitem.go` | PatchRemoveItem: soft-removes last item version in current phase or hard-deletes from a future phase. | Future-phase removal deletes the entry; current-phase removal only sets ActiveTo. Prior versions stay intact. |
| `addphase.go` | PatchAddPhase: inserts a future phase and shifts later phases by its duration. | Duration on CreateSubscriptionPhaseInput drives the shift; omitting it leaves no gap. |
| `removephase.go` | PatchRemovePhase: deletes a future phase with Shift=Next (compress) or Shift=Prev (gap). | Only future phases removable; current/past return PatchForbiddenError. |
| `stretchphase.go` | PatchStretchPhase: shifts phases after the target by a signed ISO duration. | Negative durations (shrink) allowed; guard prevents compressing a phase to zero length. |
| `unscheduleedit.go` | PatchUnscheduleEdit: removes future scheduled item versions from the current phase. | Operates only on the current phase; no-ops if no future edits exist. |
| `patch_test.go` | Shared test infra: testsuite/testcase generics, getDefaultSpec, TestRemoveAdd. | getDefaultSpec uses subscriptiontestutils.GetExamplePlanInput; assumes >=3 phases (test_phase_1/2/3). |

## Anti-Patterns

- Mutating spec.Phases directly outside an ApplyTo method — bypasses temporal and versioning guards.
- Using wall-clock time.Duration for phase spacing instead of datetime.ISODuration.
- Returning a plain error instead of PatchValidationError/PatchForbiddenError/PatchConflictError.
- Hard-deleting items from the current phase instead of soft-closing them.

## Decisions

- **Current-phase items are soft-closed, not deleted.** — Billing sync reads item cadences to determine billable periods; deletion would lose the active window and falsify history.
- **Phase spacing uses ISO duration arithmetic, not absolute timestamps.** — Phases use relative StartAfter durations from ActiveFrom; ISO durations avoid DST and calendar edge cases.

## Example: Apply a remove-then-add patch sequence atomically

```
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
