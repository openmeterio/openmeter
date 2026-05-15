# service

<!-- archie:ai-start -->

> Core implementation of subscription.Service: orchestrates Create/Update/Delete/Cancel/Continue/GetView/List/ExpandViews by coordinating repos, EntitlementAdapter, CustomerService, and event publishing, all inside pg-advisory-locked transactions.

## Patterns

**NewSubscriptionOperationContext on every public method** — Every exported Service method calls ctx = subscription.NewSubscriptionOperationContext(ctx) before any operation to set a context key that prevents hook re-entrancy. (`func (s *service) Create(ctx context.Context, ...) {
	ctx = subscription.NewSubscriptionOperationContext(ctx)
	...`)
**pg-advisory lock per customer before mutations** — Create and Cancel acquire a per-customer advisory lock via s.lockCustomer(ctx, customerId) inside the transaction to serialise concurrent mutations. (`return transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (subscription.Subscription, error) {
	if err := s.lockCustomer(ctx, spec.CustomerId); err != nil { return def, err }
	...`)
**Hook fan-out with errors.Join** — BeforeCreate/AfterCreate (and equivalent Before/After* for Update/Delete/Cancel/Continue) are fanned out to all registered SubscriptionCommandHooks using errors.Join(lo.Map(s.Hooks, ...)). (`err = errors.Join(lo.Map(s.Hooks, func(v subscription.SubscriptionCommandHook, _ int) error {
	return v.BeforeCreate(ctx, namespace, spec)
})...)`)
**sync() as the universal diff-apply engine** — Update, Cancel, and Continue all build a new spec and call s.sync(ctx, currentView, newSpec) — never manipulate entities directly. sync.go contains the diff logic. (`spec.ActiveTo = lo.ToPtr(cancelTime)
sub, err := s.sync(ctx, view, spec)`)
**State machine transition guard before mutations** — Delete and Cancel check subscription.NewStateMachine(status).CanTransitionOrErr() before entering the transaction. (`if err := subscription.NewStateMachine(view.Subscription.GetStatusAt(currentTime)).CanTransitionOrErr(ctx, subscription.SubscriptionActionDelete); err != nil { return err }`)
**Publish event after every mutation** — After each successful mutation, the service publishes a domain event (subscription.NewCreatedEvent, NewUpdatedEvent, NewCancelledEvent, etc.) via s.Publisher. (`err = s.Publisher.Publish(ctx, subscription.NewCreatedEvent(ctx, view))`)
**ServiceConfig constructor injection** — New(ServiceConfig) accepts all dependencies as fields in a config struct; the UniqueConstraintValidator is self-registered via RegisterHook inside New(). (`func New(conf ServiceConfig) (subscription.Service, error) {
	svc := &service{ServiceConfig: conf}
	val, _ := subscriptionvalidators.NewSubscriptionUniqueConstraintValidator(...)
	svc.RegisterHook(val)
	return svc, nil
}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Main service implementation: Create, Update, Delete, Cancel, Continue, Get, GetView, List, ExpandViews, UpdateAnnotations, RegisterHook. | ExpandViews requires all subs to share the same customer ID; calling with mixed customers produces incorrect views. |
| `servicevalidation.go` | validateCreate/validateUpdate/validateCancel/validateContinue — pure validation helpers that check state machine transitions, currency consistency, and timing constraints. | validateCreate checks customer.Currency vs spec.Currency only when spec.HasBillables() is true. |
| `sync.go` | sync() and helpers: diffs current view against new spec and issues create/delete/update calls on repos and EntitlementAdapter. | sync() is the single authorised path for spec→DB reconciliation; do not call repo methods directly from service mutators. |
| `synchelpers.go` | createPhase, deletePhase, createItem, deleteItem — building-blocks called by sync(). | createItem calls EntitlementAdapter.ScheduleEntitlement for items with entitlement templates; missing EntitlementAdapter wiring causes runtime nil-deref. |
| `service_test.go` | Integration tests requiring a live Postgres DB; set up via subscriptiontestutils.SetupDBDeps + NewService. | Tests use context.WithCancel(context.Background()) — prefer t.Context() for new tests per AGENTS.md. |

## Anti-Patterns

- Calling repo methods directly from service methods instead of routing through sync().
- Adding a new mutation operation without calling NewSubscriptionOperationContext(ctx) at entry.
- Omitting the customer advisory lock for operations that write subscription or item rows.
- Adding a new hook without using errors.Join fan-out — partial hook failure would be silently swallowed.
- Publishing events before the transaction commits — event must be emitted after transaction.Run succeeds.

## Decisions

- **Cancel and Continue reuse sync() by building a new spec with modified ActiveTo.** — sync() is the single truth for spec→DB reconciliation; duplicating logic in Cancel/Continue would cause drift and introduce partial-write bugs.
- **pg-advisory lock per customer ID acquired inside the transaction.** — Concurrent subscription mutations for the same customer must be serialised to prevent double-active or conflicting phase timelines; advisory lock releases automatically on tx commit/rollback.

## Example: Wire up a new subscription.Service using ServiceConfig

```
import (
	"github.com/openmeterio/openmeter/openmeter/subscription/service"
	"github.com/openmeterio/openmeter/openmeter/subscription/repo"
)

svc, err := service.New(service.ServiceConfig{
	SubscriptionRepo:      repo.NewSubscriptionRepo(dbClient),
	SubscriptionPhaseRepo: repo.NewSubscriptionPhaseRepo(dbClient),
	SubscriptionItemRepo:  repo.NewSubscriptionItemRepo(dbClient),
	CustomerService:       customerSvc,
	FeatureService:        featureSvc,
	EntitlementAdapter:    entitlementAdapter,
	TransactionManager:    dbClient,
	Publisher:             eventbusPublisher,
	Lockr:                 locker,
// ...
```

<!-- archie:ai-end -->
