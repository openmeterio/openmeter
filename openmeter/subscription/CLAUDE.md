# subscription

<!-- archie:ai-start -->

> Core subscription domain: defines SubscriptionSpec (the mutable in-memory spec driving phase/item/RateCard layout), the AppliesToSpec patch system (the only authorised mutation path), uniqueness validation, annotation helpers, lifecycle events, and the Service interface. Primary constraint: all spec mutations go through the patch system — never direct field writes to SubscriptionSpec.Phases.

## Patterns

**AppliesToSpec patch interface — only authorised mutation path** — All mutations implement AppliesToSpec with ApplyTo(spec *SubscriptionSpec, actx ApplyContext). Must only be invoked through SubscriptionSpec.Apply/ApplyX wrappers, never directly — subsequent validation and ordering logic runs in the wrapper. (`type AppliesToSpec interface { ApplyTo(spec *SubscriptionSpec, actx ApplyContext) error }; spec.Apply(patch, ApplyContext{CurrentTime: now})`)
**GetSortedPhases for ordered phase iteration** — SubscriptionSpec.Phases is a map[string]*SubscriptionPhaseSpec keyed by phase key. Direct map iteration gives undefined order. Always call GetSortedPhases() which returns phases sorted by StartAfter. (`for _, phase := range spec.GetSortedPhases() { /* ordered by StartAfter */ }`)
**pg_advisory_lock per customer via GetCustomerLock** — Per-customer serialization uses lockr.Key from GetCustomerLock(customerId). Lock must be acquired inside an active Postgres transaction before any subscription write. (`key, err := subscription.GetCustomerLock(customerID); locker.LockForTX(ctx, key)`)
**ValidateUniqueConstraintByFeatures before persisting** — Before persisting a new or updated subscription, call ValidateUniqueConstraintByFeatures with all active+new subscriptions for the customer to prevent two overlapping billable subscriptions from covering the same feature+customer simultaneously. (`subscription.ValidateUniqueConstraintByFeatures(append(allSpecs, newSpec))`)
**AnnotationParser for all annotation map access** — Subscription-level metadata (PreviousSubscriptionID, SupersedingSubscriptionID, BooleanEntitlementCount, OwnerSubSystem) is stored in models.Annotations. Always use subscription.AnnotationParser getter/setter methods — never read raw annotation map keys by string literal. (`subscription.AnnotationParser.GetPreviousSubscriptionID(annotations); subscription.AnnotationParser.SetSupersedingSubscriptionID(annotations, subID)`)
**NewSubscriptionOperationContext on every Service method entry** — Every public Service method calls NewSubscriptionOperationContext(ctx) at entry to mark the ctx as inside a subscription operation. IsSubscriptionOperation(ctx) lets validators and hooks detect re-entrant calls. (`ctx = subscription.NewSubscriptionOperationContext(ctx); // then acquire lock and begin transaction`)
**Typed error constructors — never plain fmt.Errorf for domain errors** — Use NewSubscriptionNotFoundError, NewPhaseNotFoundError, NewItemNotFoundError, ErrOnlySingleSubscriptionAllowed (ValidationIssue), and models.NewGenericPreConditionFailedError for HTTP-mappable errors. (`return subscription.NewSubscriptionNotFoundError(id) // wraps models.GenericNotFoundError → 404`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/subscription/apply.go` | Defines AppliesToSpec interface, ApplyContext, NewAppliesToSpec helper, and NewAggregateAppliesToSpec for batching patches. | FIXME comment: ApplyTo should be private but isn't — never call patch.ApplyTo directly; always route through SubscriptionSpec.Apply wrappers. |
| `openmeter/subscription/errors.go` | All subscription domain errors as ValidationIssue sentinels (with ErrorCode constants and IsXxx predicates) and typed NotFound errors. | ValidationIssue errors use commonhttp.WithHTTPStatusCodeAttribute for HTTP status mapping. Use IsSubscriptionNotFoundError/IsPhaseNotFoundError/IsItemNotFoundError for type-safe detection. |
| `openmeter/subscription/annotations.go` | Typed annotation getter/setter helpers for PreviousSubscriptionID, SupersedingSubscriptionID, BooleanEntitlementCount, OwnerSubSystem. Exported as subscription.AnnotationParser. | Never read annotation map keys by string literal — always use AnnotationParser methods to avoid key typos and type assertion errors. |
| `openmeter/subscription/hook.go` | Defines SubscriptionCommandHook interface (Before/AfterCreate, BeforeUpdate, BeforeDelete, AfterCancel, AfterContinue) and NoOpSubscriptionCommandHook base struct. | All hook implementations must embed NoOpSubscriptionCommandHook to avoid implementing all methods manually. Hook methods must never call subscription.Service write methods (re-entrant hook calls). |
| `openmeter/subscription/locks.go` | Provides GetCustomerLock(customerId) returning lockr.Key for pg_advisory_lock key construction. | Lock must be used inside an active Postgres transaction (inside entutils.TransactingRepo). Calling LockForTX outside a transaction fails. |
| `openmeter/subscription/uniqueness.go` | Implements ValidateUniqueConstraintByFeatures for cross-subscription overlap detection on billable features. | Must be called with ALL active+new specs for a customer — passing only the two being compared misses partial overlap cases. |
| `openmeter/subscription/events.go` | All subscription lifecycle events (CreatedEvent, UpdatedEvent, CancelledEvent, ContinuedEvent, SubscriptionSyncEvent) with EventName() and EventMetadata(). | All events use metadata.GetEventName with EventSubsystem prefix — ensures correct Kafka topic routing via eventbus.GeneratePublishTopic. |
| `openmeter/subscription/item.go` | SubscriptionItem domain type with custom JSON unmarshaller for RateCard polymorphism (FlatFeeRateCard vs UsageBasedRateCard discriminated by type field). | UnmarshalJSON must handle both RateCard types — adding a new RateCard type requires updating the switch case here. |

## Anti-Patterns

- Calling patch.ApplyTo(spec, actx) directly — always route through SubscriptionSpec.Apply/ApplyX wrappers so subsequent validation runs.
- Iterating over SubscriptionSpec.Phases map directly — always call GetSortedPhases() for deterministic phase order.
- Reading or writing annotation map keys as raw strings instead of using subscription.AnnotationParser methods.
- Calling subscription.Service write methods from inside a SubscriptionCommandHook — creates re-entrant hook invocations.
- Skipping ValidateUniqueConstraintByFeatures before persisting a new or updated subscription covering a billable feature.

## Decisions

- **Patch system (AppliesToSpec) as the only mutation path for SubscriptionSpec** — Ensures all changes go through a validated, ordered application pipeline so SubscriptionSpec invariants (phase ordering, feature uniqueness, billing cadence alignment) are always checked after every mutation.
- **SubscriptionSpec as an in-memory spec object separate from the persisted Subscription** — Allows the service layer to build, validate, and diff specs before writing to the DB — supports EditRunning and ChangeToPlan which need to compute diffs against the current live state.
- **pg_advisory_lock per customer for subscription serialization** — Prevents concurrent subscription creates/edits for the same customer from producing conflicting billing line items or overlapping entitlement grants.

## Example: Applying a patch to a SubscriptionSpec and validating cross-subscription uniqueness before persisting

```
import (
    "github.com/openmeterio/openmeter/openmeter/subscription"
)

// Mark context as inside a subscription operation
ctx = subscription.NewSubscriptionOperationContext(ctx)

// Acquire per-customer advisory lock inside transaction
key, err := subscription.GetCustomerLock(customerID)
if err != nil { return err }
// (locker.LockForTX called inside entutils.TransactingRepo)

// Apply patch through spec wrapper — never call patch.ApplyTo directly
if err := spec.Apply(patch, subscription.ApplyContext{CurrentTime: now}); err != nil {
    return err
// ...
```

<!-- archie:ai-end -->
