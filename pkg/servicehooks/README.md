# Service hooks

`servicehooks` runs ordered, synchronous reactions to typed service events. A
registry is an in-process dispatcher owned by a service; it is not an event
bus, transaction manager, retry mechanism, or background worker.

## Motivation

The primary use case is cross-domain validation without leaking one domain
into another. The domain performing an operation owns and invokes its lifecycle
event. Application wiring can register validation from another domain without
making the source domain import or depend on that domain's service.

For example, a customer service can expose a deletion event while an
entitlement hook rejects deletion for customers that still own entitlements.
Customer lifecycle remains owned by the customer domain, and entitlement rules
remain owned by the entitlement domain.

Hooks can also produce system or audit events. The registry does not make
those effects transactionally valid or durable: a system event can currently
be sent from a transaction that later rolls back. Commit coupling through an
outbox or after-commit mechanism belongs to the owning service and event
infrastructure, not this package.

## Ownership and event boundaries

Each service defines the event type containing the lifecycle and state its
hooks need. One type can represent related operations:

```go
type CustomerLifecycleEvent struct {
    Operation CustomerOperation
    Before    *customer.Customer
    After     *customer.Customer
}

type CustomerService struct {
    servicehooks.Registry[CustomerLifecycleEvent]

    repository customer.Repository
}
```

Embedding promotes the registry's complete method set, including `Register`,
`Invoke`, `Seal`, and `IsSealed`. If invocation and sealing must remain private
to the service, store the registry in an unexported field and expose an explicit
`Register` wrapper instead.

The registry owns named registration, registration order, sealing,
synchronous invocation, context propagation, cycle detection, and hook error
attribution. The service owns:

- event semantics and valid event shapes;
- the invocation point relative to persistence and commit;
- whether hook failure rejects or rolls back an operation;
- transaction separation, retries, idempotency, compensation, and
  observability.

A registry handles one Go event type, not necessarily one resource operation.
Events can share a registry when they have the same ordering, atomicity,
failure, and concurrency guarantees. Hooks with different guarantees should
use separate event types and registries. In particular, do not put a
transactional guard and an external after-commit notification in the same
fail-fast chain.

See [the example service test](example_service_test.go) for create, update, and
delete operations sharing one lifecycle event registry.

## Registration and invocation

A hook handles one typed event:

```go
type Hook[T any] interface {
    Handle(context.Context, T) error
}
```

`HookFunc` adapts a function when a dedicated implementation type would not
clarify domain intent.

Registration names are non-empty and unique within the registry. Invocation
order is exactly the order in which `Register` succeeds; the registry never
reorders hooks. A Wire composition provider must therefore encode the order in
its calls rather than relying on declaration order in `wire.NewSet`:

```go
func registerCustomerHooks(
    service *CustomerService,
    entitlementValidation servicehooks.Hook[CustomerLifecycleEvent],
    audit servicehooks.Hook[CustomerLifecycleEvent],
) error {
    if err := service.Register("entitlement-validation", entitlementValidation); err != nil {
        return err
    }

    return service.Register("audit", audit)
}
```

Registration is a startup phase. `Seal`, or the first call to `Invoke`, makes
registrations immutable; later registration returns `ErrRegistrySealed`.
Callbacks then run without the registry lock, and invocation reuses the sealed
registrations without cloning them. The zero value is ready for use, but a
registry must not be copied after first use.

Invocation is sequential and fail-fast. Cancellation is checked before
dispatch and before each hook, but an already-running hook must observe the
propagated context itself. Hook errors are wrapped in `InvocationError` with
the registration name while preserving `errors.Is` and `errors.As`.

The registry does not undo completed hooks. Place invocation inside an
appropriate transaction or provide compensation when atomicity is required.
Multiple goroutines can invoke a sealed registry concurrently, so hooks and
mutable data referenced by events must provide their own synchronization.

## Cycle handling

While a hook runs, its private registration identity is added to the causal
context chain. Nested service calls must propagate that context for cycle
detection to work.

The default `CyclePolicyError` rejects a nested invocation with `CycleError`
before any hook in that invocation runs. An intentionally re-entrant hook can
use `WithCyclePolicy(CyclePolicySkip)` when skipping only its active
registration is semantically safe; unrelated hooks continue in registration
order. Cycle tracking is per registration rather than suppressing the entire
registry.

## Hook implementation contract

Hooks should:

- treat events and referenced domain state as immutable unless the event
  contract explicitly permits mutation;
- propagate context, return domain errors, and keep synchronous work bounded;
- be concurrency-safe and idempotent when the service operation can be
  retried.

Hooks should not:

- register hooks during invocation or after application startup;
- retain events or contexts, or launch uncoordinated goroutines;
- panic for expected failures or assume the registry provides rollback,
  retries, logging, or compensation;
- perform irreversible external effects inside a transaction-bound registry;
- call the same registry recursively without an explicit re-entry design.

The registry intentionally does not recover panics. Programming defects may
panic; expected service failures must be returned as errors.
