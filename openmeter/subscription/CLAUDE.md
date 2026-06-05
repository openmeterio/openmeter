# subscription

<!-- archie:ai-start -->

> Subscription domain root: bridges product-catalog plans to live customer subscriptions and forward into billing + entitlement. The root holds the in-memory SubscriptionSpec model, views, the declarative patch system, the apply/sync primitives, the state machine, timing/phase/item models, and the billing/entitlement bridge types. repo/ persists, service/+workflow/ orchestrate, addon/ is the addon sub-system, hooks/+validators/ enforce invariants.

## Patterns

**Spec is the unit of manipulation** — SubscriptionSpec (CreateSubscriptionPlanInput + CreateSubscriptionCustomerInput + Phases map of pointers) is the central mutable object. Patches and addons mutate it via AppliesToSpec.ApplyTo(spec, ApplyContext); the service syncs the spec to the DB rather than editing rows directly. (`type AppliesToSpec interface { ApplyTo(spec *SubscriptionSpec, actx ApplyContext) error }`)
**Phases use pointer maps and cumulative StartAfter** — Phases is map[string]*SubscriptionPhaseSpec so patches can mutate in place; phase timing is expressed as StartAfter ISODuration added to ActiveFrom (GetPhaseCadence), never absolute timestamps. (`phaseStartTime, _ := phase.StartAfter.AddTo(s.ActiveFrom)`)
**Typed patch errors mapped to API status** — Patches return *PatchValidationError / *PatchForbiddenError / *PatchConflictError; the Patch interface couples AppliesToSpec + Validate + Op() + Path(). Apply aggregation tolerates issues tagged AllowedDuringApplyingToSpecError. (`type Patch interface { AppliesToSpec; Validate() error; Op() PatchOperation; Path() SpecPath }`)
**State machine gates lifecycle actions** — NewStateMachine(status) (qmuntal/stateless) defines the legal transitions (Inactive->Create->Active, Active reentry Update/ChangeAddons, Active->Cancel->Canceled, Canceled->Continue->Active, Scheduled->Delete). CanTransitionOrErr returns models.NewGenericForbiddenError when disallowed. (`sm.Configure(SubscriptionStatusActive).PermitReentry(SubscriptionActionUpdate).Permit(SubscriptionActionCancel, SubscriptionStatusCanceled)`)
**Three-tier spec inputs** — Plan-derived fields use the *PlanInput suffix, customer-derived fields the *CustomerInput suffix, and the merged result the *Spec suffix; SubscriptionSpec embeds both inputs inline. (`CreateSubscriptionPlanInput + CreateSubscriptionCustomerInput composed into SubscriptionSpec`)
**Operation context marker** — subscription-driven mutations carry NewSubscriptionOperationContext(ctx); downstream hooks check IsSubscriptionOperation(ctx) to avoid re-entrant side effects. (`func IsSubscriptionOperation(ctx context.Context) bool`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `subscriptionspec.go` | SubscriptionSpec + Create*PlanInput/Create*CustomerInput, ToCreateSubscriptionEntityInput, GetPhaseCadence/GetSortedPhases | Phases is a pointer map for in-place patch mutation; phase cadence is derived from StartAfter+ActiveFrom and clamped by ActiveTo |
| `apply.go` | AppliesToSpec interface, ApplyContext, NewAggregateAppliesToSpec, AllowedDuringApplyingToSpecError | ApplyTo should only be invoked via spec.Apply so post-apply validation runs; some issues are tolerated only when tagged allowed-during-apply |
| `patch.go` | Patch/ValuePatch interfaces, PatchOperation enum, typed patch errors | Return PatchValidationError/PatchForbiddenError/PatchConflictError (not bare errors) so HTTP maps to 4xx; concrete patches live in patch/ |
| `state.go` | SubscriptionStatus/SubscriptionAction enums and the stateless state machine | Gate every lifecycle action through CanTransitionOrErr before mutating; Cancel/Continue are real transitions, Update is a reentry |
| `billing.go` | BillingBehaviorOverride (RestartBillingPeriod anchor at item ActiveFrom) | ProratingBehavior is intentionally not yet wired; restart anchors to the item's ActiveFrom |
| `entitlement.go` | EntitlementAdapter contract for scheduling/fetching item-linked entitlements | Subscription depends on this interface, not entitlement internals; impl is in subscription/entitlement |
| `subscriptionview.go` | SubscriptionView (read projection) with AsSpec() round-trip | Mutate via view.AsSpec() -> spec edit -> Service.Update, never edit view items directly |
| `context.go` | Subscription operation context marker | Set it on subscription-originated writes so hooks can skip re-entrant work |

## Anti-Patterns

- Editing persisted subscription/phase/item rows directly instead of building a target SubscriptionSpec and running sync
- Computing phase/item activity as absolute timestamps instead of StartAfter/ISODuration relative to ActiveFrom
- Returning bare errors from patches instead of the typed PatchValidationError/PatchForbiddenError/PatchConflictError
- Performing a lifecycle action without passing it through SubscriptionStateMachine.CanTransitionOrErr
- Adding DB or HTTP code to the root package — persistence is in repo/, transport via the v3 handlers, orchestration in service/ and workflow/

## Decisions

- **All mutation flows through a declarative spec + sync diff rather than imperative row edits** — Patches, addons, plan-changes, cancel, and continue can all be expressed as transforms on a single SubscriptionSpec, so one sync algorithm reconciles target state and history immutability uniformly
- **Lifecycle legality is encoded in a stateless state machine, not scattered if-checks** — Create/Update/Cancel/Continue/Delete/ChangeAddons have a small fixed transition graph; centralizing it makes forbidden transitions return a consistent GenericForbiddenError
- **Phase timing is relative (StartAfter ISODuration), not absolute** — A spec can be re-anchored to any ActiveFrom (migrations, plan changes) without rewriting every phase timestamp

## Example: Aggregate patches onto a spec, tolerating only allowed-during-apply issues

```
func NewAggregateAppliesToSpec(applieses []AppliesToSpec) AppliesToSpec {
	return NewAppliesToSpec(func(spec *SubscriptionSpec, actx ApplyContext) error {
		for i, applies := range applieses {
			if err := spec.Apply(applies, actx); err != nil {
				issues, e := models.AsValidationIssues(err)
				if e != nil { return models.ErrorWithComponent(models.ComponentName(fmt.Sprintf("patch at idx %d", i)), e) }
				if lo.EveryBy(issues, func(is models.ValidationIssue) bool {
					return IsValidationIssueWithBoolAttr(is, subscriptionPatchErrAttrNameAllowedDuringApplyingToSpecError)
				}) { continue }
				return models.ErrorWithComponent(models.ComponentName(fmt.Sprintf("patch at idx %d", i)), err)
			}
		}
		return nil
	})
}
```

<!-- archie:ai-end -->
