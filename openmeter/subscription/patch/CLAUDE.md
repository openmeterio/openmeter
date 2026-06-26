# patch

<!-- archie:ai-start -->

> Concrete declarative patches that transform a SubscriptionSpec in-memory (add/remove items and phases, stretch phase, unschedule edit). Each patch enforces timing-safety invariants (cannot edit the past) before mutating the spec.

## Patterns

**Patch implements the subscription patch interface** — Each patch type asserts conformance to subscription.Patch or subscription.ValuePatch[T] and implements Op(), Path(), Validate(), ValueAsAny(), ApplyTo(). (`var _ subscription.ValuePatch[subscription.SubscriptionItemSpec] = PatchAddItem{}`)
**Validate before apply** — Validate() chains Path().Validate(), Op().Validate(), then the value's Validate(); wrap nested value errors with the FieldDescriptor prefix. (`return models.ErrorWithFieldPrefix(a.FieldDescriptor(), err)`)
**Mutate via spec.Phases / ItemsByKey, never the DB** — ApplyTo operates only on the in-memory *subscription.SubscriptionSpec (Phases map, ItemsByKey slices). No repo or DB calls in this package. (`phase.ItemsByKey[a.ItemKey] = append(phase.ItemsByKey[a.ItemKey], &a.CreateInput)`)
**Enforce past-immutability against ApplyContext.CurrentTime** — Use spec.GetCurrentPhaseAt(actx.CurrentTime) and phase StartAfter.AddTo(spec.ActiveFrom) to forbid edits to past phases; return typed PatchForbiddenError / PatchValidationError / PatchConflictError. (`return &subscription.PatchForbiddenError{Msg: fmt.Sprintf("cannot add item to phase %s which starts before current phase", a.PhaseKey)}`)
**Close-then-add for current-phase items** — Adding an item to the current phase closes the prior version by setting its ActiveToOverrideRelativeToPhaseStart to the new item's ActiveFrom override (history is never falsified). (`itemToClose.ActiveToOverrideRelativeToPhaseStart = a.CreateInput.ActiveFromOverrideRelativeToPhaseStart`)
**Express times as ISODuration relative to phase start** — Item timing is stored as ActiveFrom/ActiveToOverrideRelativeToPhaseStart using datetime.ISODurationBetween / AddTo, not absolute times. (`diff := datetime.ISODurationBetween(phaseStartTime, actx.CurrentTime)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `additem.go` | PatchAddItem: appends an item version under a phase key with past/future-phase guards | Branch logic distinguishes no-current-phase, current-phase (sets ActiveFrom to now if unset), future-phase (key must be empty). When closing a current item it errors if the existing item's scheduled end is after the new start (user must delete first). |
| `addphase.go` | PatchAddPhase: inserts a future phase and re-spaces later phases by a signed diff | Only future phases allowed (vST.After(CurrentTime)); recomputes StartAfter of all later phases via Duration.Subtract to preserve relative spacing. PatchConflictError if phase key already exists. |
| `removeitem.go` | PatchRemoveItem: removes last version for a key, or closes it if in current phase | Current-phase removal sets ActiveToOverrideRelativeToPhaseStart to now (soft-close); future-phase removal pops the slice and deletes the key when empty. Cannot remove from past phases. |
| `stretchphase.go / unscheduleedit.go / removephase.go` | Additional spec transforms (extend a phase, revert scheduled edits, drop a phase) | Follow the same Validate/ApplyTo + typed-error contract as the other patches. |
| `patch_test.go` | Shared test harness: testcase[T]/testsuite[T], getDefaultSpec, SpecsEqual | Tests call Patch.ApplyTo directly and compare via subscriptiontestutils.SpecsEqual; getDefaultSpec builds a 3-phase spec from GetExamplePlanInput. errors.As is used for typed-error assertions. |

## Anti-Patterns

- Calling a repository or DB from a patch's ApplyTo — patches are pure spec transforms.
- Returning a bare error instead of PatchValidationError/PatchForbiddenError/PatchConflictError, which downstream maps to API status codes.
- Allowing edits to phases/items that start before ApplyContext.CurrentTime (rewriting history).
- Using absolute timestamps for item activity instead of ISODuration overrides relative to phase start.

## Decisions

- **Item edits in the current phase close the previous version rather than mutate it** — Subscription history must remain immutable for billing correctness; closing then re-adding preserves the audit trail of rate-card versions.
- **Adding a phase re-spaces later phases by a signed diff** — Keeps the relative spacing of subsequent phases intact when a new phase is inserted, so existing schedules are not silently shifted.

## Example: Define a spec patch with Validate + ApplyTo

```
type PatchAddItem struct {
	PhaseKey    string
	ItemKey     string
	CreateInput subscription.SubscriptionItemSpec
}

var _ subscription.ValuePatch[subscription.SubscriptionItemSpec] = PatchAddItem{}

func (a PatchAddItem) ApplyTo(spec *subscription.SubscriptionSpec, actx subscription.ApplyContext) error {
	phase, ok := spec.Phases[a.PhaseKey]
	if !ok {
		return &subscription.PatchValidationError{Msg: fmt.Sprintf("phase %s not found", a.PhaseKey)}
	}
	// ... timing guards against actx.CurrentTime ...
	phase.ItemsByKey[a.ItemKey] = append(phase.ItemsByKey[a.ItemKey], &a.CreateInput)
// ...
```

<!-- archie:ai-end -->
