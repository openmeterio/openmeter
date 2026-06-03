# subscription

<!-- archie:ai-start -->

> Core subscription domain managing lifecycle (Create, Update, Delete, Cancel, Continue, UpdateAnnotations) against a versioned plan-phase-RateCard model. The root defines SubscriptionSpec (the mutable in-memory spec), the AppliesToSpec patch system (the only authorised mutation path), uniqueness validation, typed annotations, lifecycle events, and the Service interface; children split into patch (mutation primitives), service/repo (orchestration + Ent), entitlement (bridge), workflow (higher-level orchestration), addon, hooks, validators, and testutils.

## Patterns

**AppliesToSpec patch system is the only mutation path** — All SubscriptionSpec mutations implement AppliesToSpec.ApplyTo and must be invoked through SubscriptionSpec.Apply/ApplyX wrappers (never patch.ApplyTo directly) so post-mutation validation and ordering always run. Concrete patches live in patch/; addon transforms in addon/diff. (`spec.Apply(patch, subscription.ApplyContext{CurrentTime: now}); NewAggregateAppliesToSpec batches patches.`)
**Spec built/validated in-memory, then synced through service** — service/ uses sync() as the universal diff-apply engine; Cancel/Continue reuse sync() by building a new spec with modified ActiveTo. Persistence goes through SubscriptionRepository/Phase/Item repos (TransactingRepo triad), and all DB writes from outside route through subscriptionworkflow.Service, never raw repo calls. (`service.sync() diffs target vs live spec; workflow/service orchestrates CreateFromPlan/EditRunning/ChangeToPlan.`)
**Per-customer pg_advisory_lock inside a transaction on every write** — Every public Service method opens with NewSubscriptionOperationContext(ctx), then acquires lockr.Key from GetCustomerLock(customerID) via LockForTX inside the active Ent transaction before any subscription/item write. Workflow methods lockCustomer too. (`ctx = subscription.NewSubscriptionOperationContext(ctx); key,_ := subscription.GetCustomerLock(cid); locker.LockForTX(ctx, key)`)
**Uniqueness + typed errors at the spec boundary** — ValidateUniqueConstraintByFeatures must run with ALL active+new specs for a customer before persist to block overlapping billable features. Domain errors are ValidationIssue sentinels / typed NotFound errors with IsXxx predicates and HTTP status attributes; workflow wraps spec errors via MapSubscriptionErrors. (`subscription.ValidateUniqueConstraintByFeatures(append(allSpecs, newSpec)); workflow.MapSubscriptionErrors(err)`)
**Typed annotation access + hooks/validators registered in app/common** — Annotation map keys (previous/superseding subscription IDs, boolean entitlement count, owner subsystem) are read/written only via subscription.AnnotationParser. SubscriptionCommandHooks (hooks/) and cross-domain validators (validators/) embed noop bases, write via repositories (never Service), and are registered through RegisterHook/RegisterRequestValidator in app/common. (`AnnotationParser.SetSupersedingSubscriptionID(annotations, id); validators/customer blocks delete when active subs exist.`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `apply.go` | AppliesToSpec interface, ApplyContext, NewAppliesToSpec, NewAggregateAppliesToSpec. | ApplyTo is public only by FIXME — never call it directly; always go through SubscriptionSpec.Apply. |
| `subscriptionspec.go` | SubscriptionSpec (Phases map[string]*SubscriptionPhaseSpec) and GetSortedPhases(). | Direct map iteration is unordered — use GetSortedPhases(); never write Phases directly. |
| `errors.go` | ValidationIssue sentinels + typed NotFound errors with IsXxx predicates and HTTP status attributes. | Use Is*NotFoundError predicates; never plain fmt.Errorf at the boundary. |
| `annotations.go` | AnnotationParser getters/setters for previous/superseding IDs, boolean count, owner subsystem. | Never read annotation keys by raw string — typos and bad type assertions. |
| `hook.go` | SubscriptionCommandHook interface + NoOpSubscriptionCommandHook base. | Embed the NoOp base; hooks must never call Service write methods (re-entrant invocation). |
| `locks.go` | GetCustomerLock(customerID) → lockr.Key. | Must be used inside an active Postgres tx (entutils.TransactingRepo). |
| `uniqueness.go` | ValidateUniqueConstraintByFeatures for cross-subscription billable-feature overlap. | Call with ALL active+new specs — comparing only two misses partial overlaps. |
| `item.go` | SubscriptionItem with custom RateCard polymorphism JSON. | UnmarshalJSON switch must handle every RateCard type (FlatFee vs UsageBased). |

## Anti-Patterns

- Calling patch.ApplyTo(spec, actx) directly instead of routing through SubscriptionSpec.Apply/ApplyX wrappers.
- Iterating SubscriptionSpec.Phases map directly instead of GetSortedPhases(), or writing Phases without a patch.
- Reading/writing annotation map keys as raw strings instead of subscription.AnnotationParser.
- Calling subscription.Service write methods from inside a SubscriptionCommandHook (or workflow create/change without sync) — re-entrant or unvalidated writes.
- Skipping ValidateUniqueConstraintByFeatures, the per-customer advisory lock, or NewSubscriptionOperationContext at a mutating operation.

## Decisions

- **AppliesToSpec patch system as the sole SubscriptionSpec mutation path.** — Forces every change through a validated, ordered pipeline so phase ordering, feature uniqueness, and cadence invariants are re-checked after each mutation.
- **SubscriptionSpec is an in-memory object separate from the persisted Subscription.** — Lets the service build/validate/diff specs before writing — required for EditRunning and ChangeToPlan.
- **pg_advisory_lock per customer for serialization.** — Prevents concurrent creates/edits for one customer from producing conflicting billing lines or overlapping entitlement grants.

## Example: Applying a patch and validating cross-subscription uniqueness before persisting

```
import "github.com/openmeterio/openmeter/openmeter/subscription"

ctx = subscription.NewSubscriptionOperationContext(ctx)

key, err := subscription.GetCustomerLock(customerID)
if err != nil { return err }
// locker.LockForTX(ctx, key) inside entutils.TransactingRepo

if err := spec.Apply(patch, subscription.ApplyContext{CurrentTime: now}); err != nil {
    return err
}
if err := subscription.ValidateUniqueConstraintByFeatures(append(allSpecs, spec)); err != nil {
    return err
}
```

<!-- archie:ai-end -->
