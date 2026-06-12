# service

<!-- archie:ai-start -->

> Orchestration layer for the subscription lifecycle (Create/Update/Cancel/Continue/Delete/Get/List/ExpandViews). Owns the sync diff algorithm that reconciles a stored subscription to a target SubscriptionSpec, plus validation, hooks, eventing, and entitlement linkage.

## Patterns

**service implements subscription.Service via ServiceConfig DI** — New(ServiceConfig) wires repos, customer/feature services, EntitlementAdapter, TransactionManager, Publisher, Lockr, FeatureFlags, TaxCode; the unique-constraint validator is registered as a hook in New. (`var _ subscription.Service = &service{}`)
**validate -> transaction.Run -> hooks -> publish** — Mutating methods validate first (state machine + currency), then run inside transaction.Run, fire Before*/After* hooks around the work, and Publish a domain event (NewCreatedEvent, NewUpdatedEvent, etc.). (`err = s.Publisher.Publish(ctx, subscription.NewCreatedEvent(ctx, view))`)
**Lock customer and seed operation context** — Each public method calls subscription.NewSubscriptionOperationContext(ctx); customer-mutating ops call s.lockCustomer (Lockr.LockForTX on GetCustomerLock). (`ctx = subscription.NewSubscriptionOperationContext(ctx); s.lockCustomer(ctx, spec.CustomerId)`)
**Gate transitions with the state machine** — Use subscription.NewStateMachine(view.Subscription.GetStatusAt(now)).CanTransitionOrErr(ctx, action) before Create/Update/Cancel/Continue/Delete. (`subscription.NewStateMachine(subscription.SubscriptionStatusInactive).CanTransitionOrErr(ctx, subscription.SubscriptionActionCreate)`)
**sync = three-phase reconcile with touched map** — sync(view, newSpec) deletes changed/removed, recreates changed, creates new; change detection compares ToCreateSubscription*EntityInput via Equal, and dirty (touched) tracks parent/child invalidation by SpecPath. (`dirty.mark(subscription.NewPhasePath(currentPhaseView.SubscriptionPhase.Key))`)
**Cancel/Continue reuse sync by mutating spec.ActiveTo** — Cancel resolves timing to a cancel time and sets spec.ActiveTo; Continue sets spec.ActiveTo = nil; both then call sync so all cadences re-derive from the subscription cadence. (`spec.ActiveTo = lo.ToPtr(cancelTime); sub, err := s.sync(ctx, view, spec)`)
**Reject immutable-field changes in sync** — sync errors if CustomerId, PlanRef, ActiveFrom, or SettlementMode differ between view and newSpec. (`if !view.Subscription.PlanRef.NilEqual(newSpec.Plan) { return def, fmt.Errorf("cannot change plan") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service facade: Create/Update/Delete/Cancel/Continue/Get/GetView/List/ExpandViews, hook registration, customer locking | Create persists phases via GetSortedPhases + GetPhaseCadence then updates customer currency if unset. GetView delegates to ExpandViews; ExpandViews only supports a single customer id at a time. |
| `sync.go` | Core reconcile algorithm and the touched map (SpecPath parent/child invalidation) | Marked TODO(OM-1074) cleanup. Item change comparison uses a deliberately-faked 'new' entitlement reused from the current item (commented 'This is a lie') because the new entitlement does not exist yet; touching a phase forces impossibleNamespacedId so child items recompute. Order matters: delete pass, recreate-changed pass, create-new pass. |
| `synchelpers.go` | createPhase/createItem (with createItemOptions), deletePhase/deleteItem, resolveTaxCode | createItem schedules the entitlement via EntitlementAdapter (stamping AnnotationSubscriptionID) BEFORE building the item entity input, and resolveTaxCode mutates the spec's RateCard in place — order is load-bearing (see the in-code comment). deleteItem deletes the entitlement first when present. |
| `servicevalidation.go` | validateCreate/Update/Cancel/Continue: state-machine + customer + currency checks | Currency mismatch is only enforced when spec.HasBillables(); returns models.NewGenericValidationError. validateCreate also asserts spec.CustomerId == cust.ID. |
| `service_test.go / sync_test.go` | DB-backed lifecycle tests using subscriptiontestutils.SetupDBDeps + NewService | Tests assert spec/view equivalence via subscriptiontestutils.ValidateSpecAndView and drive edits by mutating spec.Phases then calling Update. Use clock.SetTime/FreezeTime for deterministic timing. |

## Anti-Patterns

- Mutating subscription contents directly instead of building a target spec and calling Update/sync.
- Skipping the NewStateMachine transition check before a lifecycle action.
- Publishing an event or firing hooks outside the transaction.Run boundary, risking inconsistent state on rollback.
- Changing CustomerId/Plan/ActiveFrom/SettlementMode through sync (these are explicitly rejected).
- Building the item entity input before scheduling its entitlement / resolving tax code, which breaks the in-place RateCard enrichment.

## Decisions

- **Cancel and Continue are implemented on top of sync rather than as bespoke updates** — All phase/item cadences are derived from the subscription cadence, so adjusting spec.ActiveTo and re-syncing yields correct deactivation/reactivation for every sub-resource.
- **sync recreates a phase/item wholesale when any field changes, tracked via a touched SpecPath map** — Sub-resources are hard-linked to their parents; recreating the parent and re-linking children is simpler and safer than in-place field mutation, and parent/child invalidation falls out of the path-based touched map.
- **Validators are registered as hooks (e.g. unique-constraint validator) in New** — Lets cross-cutting invariants run inside the same Before/After hook pipeline as other behavior, keeping the service open for extension without branching its core methods.

## Example: Lifecycle method: validate, transact, hook, publish

```
func (s *service) Update(ctx context.Context, subscriptionID models.NamespacedID, newSpec subscription.SubscriptionSpec) (subscription.Subscription, error) {
	ctx = subscription.NewSubscriptionOperationContext(ctx)
	view, err := s.GetView(ctx, subscriptionID)
	if err != nil { return subscription.Subscription{}, err }
	if err := s.validateUpdate(ctx, view, newSpec); err != nil { return subscription.Subscription{}, err }
	return transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (subscription.Subscription, error) {
		s.mu.RLock(); defer s.mu.RUnlock()
		_ = errors.Join(lo.Map(s.Hooks, func(v subscription.SubscriptionCommandHook, _ int) error { return v.BeforeUpdate(ctx, subscriptionID, newSpec) })...)
		subs, err := s.sync(ctx, view, newSpec)
		if err != nil { return subs, err }
		updatedView, _ := s.GetView(ctx, subs.NamespacedID)
		return subs, s.Publisher.Publish(ctx, subscription.NewUpdatedEvent(ctx, updatedView))
	})
}
```

<!-- archie:ai-end -->
