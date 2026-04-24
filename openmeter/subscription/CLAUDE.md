# subscription

<!-- archie:ai-start -->

> Core subscription domain package: defines SubscriptionSpec (the mutable spec driving phase/item/RateCard layout), the patch system (AppliesToSpec), uniqueness validation, annotation helpers, and the Service interface. Primary constraint: all mutations go through the patch system applied to SubscriptionSpec, never direct field writes.

## Patterns

**SubscriptionSpec as mutable spec object** — SubscriptionSpec is the central object manipulated by patches, addons, and the service layer. It holds phases (map[string]*SubscriptionPhaseSpec) keyed by phase key. Always use GetSortedPhases() for ordered iteration — phase ordering is by StartAfter, not map insertion order. (`spec.GetSortedPhases() returns []*SubscriptionPhaseSpec sorted by StartAfter; spec.GetCurrentPhaseAt(t) returns the active phase at time t.`)
**AppliesToSpec patch interface** — All mutations (patches, addon applications) implement AppliesToSpec with an ApplyTo(spec *SubscriptionSpec, actx ApplyContext) method. The method must only be called through SubscriptionSpec.ApplyX wrapper methods, never invoked directly. (`type AppliesToSpec interface { ApplyTo(spec *SubscriptionSpec, actx ApplyContext) error }`)
**pg_advisory_lock via GetCustomerLock** — Per-customer serialization uses lockr.Key from GetCustomerLock(customerId). The lock must be acquired inside an active Postgres transaction before any write operation. (`lockr.NewKey("customer", customerId, "subscription") — pass this to billing.Service.WithLock before invoice/charge mutations.`)
**ValidateUniqueConstraintByFeatures cross-subscription check** — Before persisting a new or updated subscription, ValidateUniqueConstraintByFeatures([]SubscriptionSpec) must be called to prevent two overlapping billable subscriptions from covering the same feature+customer simultaneously. (`subscription.ValidateUniqueConstraintByFeatures([]subscription.SubscriptionSpec{s1, s2}) returns ValidationIssues with JSONPath selectors on overlap.`)
**Annotation helpers via annotationParser** — Subscription-level metadata (e.g. PreviousSubscriptionID, SupersedingSubscriptionID, entitlement counts) is stored in models.Annotations as typed keys. Always use the annotationParser getter/setter methods — never read raw annotation map keys. (`annotationParser{}.GetPreviousSubscriptionID(annotations) — returns *string, nil if not set.`)
**SubscriptionSpec.GetAlignedBillingPeriodAt for billing period calculation** — For aligned subscriptions, billing periods must be computed via GetAlignedBillingPeriodAt(at time.Time) rather than manually advancing the plan cadence, to handle phase reanchoring correctly. (`spec.GetAlignedBillingPeriodAt(now) returns (timeutil.ClosedPeriod, error).`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/subscription/apply.go` | Defines AppliesToSpec interface and ApplyContext. All patches and addon applications implement this interface. | The FIXME comment notes ApplyTo should be private but isn't yet — never call ApplyTo directly, always go through SubscriptionSpec.ApplyX. |
| `openmeter/subscription/spec.go` | Defines SubscriptionSpec and SubscriptionPhaseSpec. Contains GetSortedPhases, GetCurrentPhaseAt, GetAlignedBillingPeriodAt, HasEntitlements, HasBillables. | Phase map is keyed by phase key string — must call GetSortedPhases() for ordered traversal; direct map iteration gives undefined phase order. |
| `openmeter/subscription/locks.go` | Provides GetCustomerLock(customerId) for pg_advisory_lock key construction. | Lock must be used inside an active Postgres transaction; lockr.Locker fails if no transaction is present in ctx. |
| `openmeter/subscription/uniqueness.go` | Implements ValidateUniqueConstraintByFeatures for cross-subscription overlap detection on billable features. | Must be called with all active+new subscriptions for a customer, not just the two being compared — otherwise partial overlap cases are missed. |
| `openmeter/subscription/annotation.go` | Typed annotation getter/setter helpers (PreviousSubscriptionID, SupersedingSubscriptionID, entitlement counts). | Never read annotation keys directly — always use annotationParser methods to avoid key string typos and type assertion errors. |

## Anti-Patterns

- Calling patch.ApplyTo(spec, actx) directly — always route through SubscriptionSpec.ApplyX wrapper methods so subsequent validation runs.
- Iterating over SubscriptionSpec.Phases map directly — always call GetSortedPhases() to get deterministic phase order.
- Skipping ValidateUniqueConstraintByFeatures before persisting a new or updated subscription covering a billable feature.
- Reading or writing annotation map keys as raw strings instead of using annotationParser methods.
- Calling subscription.Service directly for create/change/migrate — always route through PlanSubscriptionService or WorkflowService.

## Decisions

- **Patch system (AppliesToSpec) as the only mutation path** — Ensures all changes go through a validated, ordered application pipeline so that SubscriptionSpec invariants (phase ordering, feature uniqueness, billing cadence alignment) are always checked after every mutation.
- **SubscriptionSpec as an in-memory spec object separate from persisted Subscription** — Allows the service layer to build, validate, and diff specs before writing to the DB, supporting EditRunning and ChangeToPlan operations that need to compute diffs against the current live state.
- **pg_advisory_lock per customer for subscription serialization** — Prevents concurrent subscription creates/edits for the same customer from producing conflicting billing line items or overlapping entitlement grants.

## Example: Applying a patch to a SubscriptionSpec and validating uniqueness

```
// openmeter/subscription/service/service.go (conceptual)
spec, err := svc.GetSpecForCustomer(ctx, customerID)
if err != nil { return err }

// Apply the patch through the spec wrapper (not patch.ApplyTo directly)
if err := spec.ApplyPatch(patch, subscription.ApplyContext{CurrentTime: now}); err != nil {
    return err
}

// Validate cross-subscription uniqueness before persisting
allSpecs, err := svc.GetAllActiveSpecsForCustomer(ctx, customerID)
if err != nil { return err }
if err := subscription.ValidateUniqueConstraintByFeatures(append(allSpecs, spec)); err != nil {
    return err
}
// ...
```

<!-- archie:ai-end -->
