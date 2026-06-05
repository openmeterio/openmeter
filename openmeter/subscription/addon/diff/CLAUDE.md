# diff

<!-- archie:ai-start -->

> Pure-functional engine that computes how a SubscriptionAddon transforms a SubscriptionSpec (apply) and how to undo it (restore). It bridges subscription addons into the spec model by splitting/merging items per phase so the addon's rate cards are present for its cadence; constraint: this package is internal to subscription and never persists.

## Patterns

**Diffable apply/restore pair** — Every addon transform is expressed as a Diffable with GetApplies() and GetRestores() returning subscription.AppliesToSpec; the two operations must be exact inverses (restore_test asserts SpecsEqual after apply+restore). (`type Diffable interface { GetApplies() subscription.AppliesToSpec; GetRestores() subscription.AppliesToSpec }`)
**AppliesToSpec closures, never direct mutation** — Spec changes are wrapped in subscription.NewAppliesToSpec(func(spec *SubscriptionSpec, actx ApplyContext) error {...}) and aggregated via subscription.NewAggregateAppliesToSpec; callers invoke spec.Apply(...). Do not mutate the spec outside an AppliesToSpec. (`return subscription.NewAppliesToSpec(func(spec *subscription.SubscriptionSpec, _ subscription.ApplyContext) error { ... })`)
**Per-phase gap/intersection splitting** — getApplyForRateCard walks spec.GetSortedPhases(), intersects the addon cadence with each phase via timeutil.OpenPeriod, tracks uncovered gaps, splits existing items into difference+intersection parts, and creates new items for gaps using rc.Apply Quantity times. (`addInPhase := pCad.AsPeriod().Intersection(addPer); gaps := []timeutil.OpenPeriod{*addInPhase}`)
**Quantity-driven rate card application** — rc.Apply(rateCard, annotations) is invoked exactly d.addon.Quantity times (gaps use Quantity-1 since the base card already exists); Quantity==0 yields a no-op AppliesToSpec. Merge math lives in subscriptionaddon, not here. (`for range d.addon.Quantity { err := rc.Apply(inst.RateCard, inst.Annotations) }`)
**Relative cadence via setItemRelativeCadence** — Item cadences are stored as ActiveFromOverrideRelativeToPhaseStart / ActiveToOverrideRelativeToPhaseStart durations computed with datetime.ISODurationBetween(phaseStart, target); overrides equal to the phase boundary are left nil. (`item.ActiveFromOverrideRelativeToPhaseStart = &diff // datetime.ISODurationBetween(phaseCadence.ActiveFrom, *target.From)`)
**Restore deletes only effectively-zero items** — restore() undoes rc.Restore Quantity times, then deletes an item only if zeroRateCardCheck.CanDelete() (zero price/entitlement AND not owned by OwnerSubscriptionSubSystem), and re-merges subsequent identical items. (`chk := zeroRateCardCheck{itemAnnotations: item.Annotations, rc: target}; if chk.CanDelete() { rmIdxs = append(rmIdxs, idx) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `diff.go` | Defines Diffable interface and someDiffable (closure-backed Apply/Restore fns) | GetApplies/GetRestores must wrap fns in NewAppliesToSpec; keep the two symmetric |
| `addon.go` | GetDiffableFromAddon builds a someDiffable from an addon's instances; per-instance diffable aggregates per-rate-card applies | Returns (nil, nil) when no instances/quantities — callers must nil-check the Diffable (see create.go list flow) |
| `apply.go` | getApplyForRateCard: the phase-splitting algorithm; setItemRelativeCadence | Gaps use Quantity-1 applies; phases iterated only between phaseAtCadenceStart and phaseAtCadenceEnd; sort newItems by cadence |
| `restore.go` | Inverse of apply: undo rc effects, delete zero items, re-merge subsequent identical items | Merge predicate compares RateCard.Equal, Annotations DeepEqual, BillingBehaviorOverride, and subsequency; empty ItemsByKey entries are deleted |
| `zeroratecard.go` | zeroRateCardCheck.CanDelete — decides if a restored item is safe to remove | Only measures fields rc.Apply/Restore touches (Price, EntitlementTemplate); non-flat price or owner annotation blocks deletion |
| `affected.go` | GetAffectedItemIDs maps rate-card key -> affected subscription item IDs (used by http mapping) | Marked FIXME for placement; uses AddonRateCardMatcherForAGivenPlanRateCard and dedupes with lo.Uniq |

## Anti-Patterns

- Mutating SubscriptionSpec phases/items outside a NewAppliesToSpec closure
- Making GetApplies and GetRestores non-inverse (breaks restore_test SpecsEqual assertions)
- Calling rc.Apply a number of times other than addon.Quantity (gaps use Quantity-1)
- Persisting or reaching into repos/services from this package — it is pure spec math, internal to subscription
- Setting absolute item cadences instead of relative ActiveFrom/ActiveToOverrideRelativeToPhaseStart durations

## Decisions

- **Express addon application as reversible Diffable apply/restore rather than mutating persisted state** — Lets the workflow layer compose addon changes onto an in-memory spec and undo them deterministically; tests verify symmetry
- **Item splitting/merging done by period arithmetic (timeutil.OpenPeriod gaps/difference/intersection)** — An addon must cover its exact cadence within each phase, possibly splitting pre-existing items; period algebra makes coverage provable
- **RateCard merge/restore math delegated to subscriptionaddon package** — Keeps this package focused on spec topology; README explicitly states merging logic lives in subscriptionaddon

<!-- archie:ai-end -->
