# service

<!-- archie:ai-start -->

> Core implementation of subscription.Service: orchestrates Create/Update/Delete/Cancel/Continue/GetView/List/ExpandViews by coordinating repos, EntitlementAdapter, CustomerService, and event publishing inside pg-advisory-locked transactions.

## Patterns

**NewSubscriptionOperationContext on every public method** — Each exported method calls ctx = subscription.NewSubscriptionOperationContext(ctx) first to set a context key preventing hook re-entrancy. (`ctx = subscription.NewSubscriptionOperationContext(ctx)`)
**pg-advisory lock per customer before mutations** — Create and Cancel acquire s.lockCustomer(ctx, customerId) inside the transaction. (`if err := s.lockCustomer(ctx, spec.CustomerId); err != nil { return def, err }`)
**Hook fan-out with errors.Join** — Before/After hooks fan out to all registered SubscriptionCommandHooks via errors.Join(lo.Map(s.Hooks, ...)). (`err = errors.Join(lo.Map(s.Hooks, func(v subscription.SubscriptionCommandHook, _ int) error { return v.BeforeCreate(ctx, namespace, spec) })...)`)
**sync() as the universal diff-apply engine** — Update, Cancel, and Continue all build a new spec and call s.sync(ctx, currentView, newSpec) — never mutate entities directly. (`spec.ActiveTo = lo.ToPtr(cancelTime); sub, err := s.sync(ctx, view, spec)`)
**State machine transition guard** — Delete/Cancel check subscription.NewStateMachine(status).CanTransitionOrErr() before the transaction. (`if err := subscription.NewStateMachine(status).CanTransitionOrErr(ctx, subscription.SubscriptionActionDelete); err != nil { return err }`)
**Publish event after every mutation** — After a successful mutation the service publishes via s.Publisher (NewCreatedEvent, NewUpdatedEvent, ...). (`err = s.Publisher.Publish(ctx, subscription.NewCreatedEvent(ctx, view))`)
**ServiceConfig constructor injection** — New(ServiceConfig) takes all deps as struct fields; the unique-constraint validator self-registers via RegisterHook inside New(). (`svc := &service{ServiceConfig: conf}; svc.RegisterHook(val)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Create, Update, Delete, Cancel, Continue, Get, GetView, List, ExpandViews, UpdateAnnotations, RegisterHook. | ExpandViews requires all subs to share one customer ID; mixed customers produce incorrect views. |
| `servicevalidation.go` | validateCreate/Update/Cancel/Continue — pure validation of state transitions, currency, timing. | validateCreate checks customer.Currency vs spec.Currency only when spec.HasBillables() is true. |
| `sync.go` | sync() diffs current view vs new spec and issues create/delete/update on repos and EntitlementAdapter. | sync() is the single authorised spec→DB path; do not call repo methods directly from mutators. |
| `synchelpers.go` | createPhase/deletePhase/createItem/deleteItem building blocks for sync(). | createItem calls EntitlementAdapter.ScheduleEntitlement for items with entitlement templates; missing wiring causes nil-deref. |
| `service_test.go` | Integration tests needing live Postgres via subscriptiontestutils.SetupDBDeps + NewService. | Existing tests use context.Background(); prefer t.Context() for new tests. |

## Anti-Patterns

- Calling repo methods directly from service methods instead of routing through sync().
- Adding a mutation without NewSubscriptionOperationContext(ctx) at entry.
- Omitting the customer advisory lock for operations that write subscription or item rows.
- Adding a hook without errors.Join fan-out — partial failure would be swallowed.
- Publishing events before the transaction commits.

## Decisions

- **Cancel and Continue reuse sync() by building a new spec with modified ActiveTo.** — sync() is the single truth for spec→DB reconciliation; duplicating logic would drift and introduce partial-write bugs.
- **pg-advisory lock per customer ID acquired inside the transaction.** — Serialises concurrent subscription mutations for the same customer; the lock releases automatically on commit/rollback.

## Example: Wire up a new subscription.Service via ServiceConfig

```
svc, err := service.New(service.ServiceConfig{
  SubscriptionRepo:      repo.NewSubscriptionRepo(dbClient),
  SubscriptionPhaseRepo: repo.NewSubscriptionPhaseRepo(dbClient),
  SubscriptionItemRepo:  repo.NewSubscriptionItemRepo(dbClient),
  CustomerService:       customerSvc,
  EntitlementAdapter:    entitlementAdapter,
  TransactionManager:    dbClient,
  Publisher:             eventbusPublisher,
  Lockr:                 locker,
  // ...
})
```

<!-- archie:ai-end -->
