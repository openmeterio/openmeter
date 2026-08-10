# Service hooks

`servicehooks` runs ordered, synchronous reactions to typed service events. A
registry is an in-process dispatcher owned by a service; it is not an event
bus, transaction manager, retry mechanism, or background worker.

## Motivation

The primary purpose of service hooks is cross-domain validation without
leaking one domain into another. The domain performing an operation owns its
lifecycle event and invokes the registry. Another domain can register a hook
which validates the operation without the source domain importing that domain
or depending on its service. Application wiring composes the two domains.

For example, a customer service can expose a customer-deletion lifecycle event
without depending on entitlements. The entitlement domain can register a hook
which rejects deletion while the customer still owns entitlements. The
customer domain remains responsible for customer lifecycle, while entitlement
rules remain owned by the entitlement domain.

The same mechanism can support other synchronous lifecycle reactions,
including producing system events and generating audit events. The registry
only preserves registration order and invokes those hooks; it does not
determine whether an event is transactionally valid or durable. System-event
publication currently has a known problem where an event can be sent from a
transaction which is later rolled back. Correctly coupling publication to
commit, for example through an outbox or an after-commit mechanism, is outside
the scope of service hooks and must be handled by the owning service and event
infrastructure.

## Ownership and event boundaries

The registry has one hook method instead of prescribing generic create,
update, and delete callbacks. Each service defines the event type containing
the lifecycle and state information its hooks need. One event type can
represent a family of related operations:

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

The registry is responsible for:

- validating and storing named hook registrations;
- preserving the order of successful `Register` calls;
- freezing registration before request processing;
- invoking hooks synchronously with caller context propagation;
- detecting recursive hook invocation in the causal context chain;
- attaching hook identity to returned errors.

The owning service is responsible for:

- defining event meaning and validating valid event shapes;
- deciding where invocation belongs relative to persistence and commit;
- providing before, after, input, actor, or operation data needed by hooks;
- deciding whether a hook failure rejects or rolls back the operation;
- separating transactional reactions from after-commit integrations;
- retries, idempotency, compensation, logging, metrics, and tracing.

A registry handles one Go event type, not necessarily one concrete resource
operation. Events that share ordering, atomicity, failure, and concurrency
semantics can share a registry. Hooks with different guarantees should use
different event types and registries. In particular, a transactional guard
should not share a fail-fast chain with an external after-commit notification:
rollback cannot compensate for an already-visible external effect.

See [the example service test](example_service_test.go) for create, update, and
delete operations sharing one lifecycle event registry.

## Hooks

A hook handles one typed event:

```go
type Hook[T any] interface {
    Handle(context.Context, T) error
}
```

`HookFunc` adapts a function when a dedicated implementation type would not
clarify domain intent:

```go
err := service.Register(
    "ledger-accounts",
    servicehooks.HookFunc[CustomerLifecycleEvent](provisionLedgerAccounts),
)
```

Event values are not cloned. Passing a struct copies only the struct itself;
contained pointers, maps, and slices can still reference shared state.

### Hook implementation contract

Hook implementations should:

- treat the event and referenced domain state as immutable unless the owning
  service explicitly defines ordered mutation as part of the event contract;
- propagate the provided context to every synchronous downstream operation;
- return errors with enough domain context for the service to classify them;
- remain safe for concurrent invocation by separate service requests;
- be deterministic and idempotent when the caller can retry the operation;
- keep work bounded because invocation holds up the service operation;
- use `CyclePolicySkip` only when re-entry is intentional and skipping the
  active hook is semantically safe.

Hook implementations should not:

- register hooks during invocation or after application startup;
- retain the event or context for asynchronous work;
- launch uncoordinated goroutines from a synchronous lifecycle hook;
- panic for expected failures;
- assume the registry provides rollback, retries, logging, or compensation;
- perform irreversible external effects inside a transaction-bound registry;
- call the same registry recursively without an explicit re-entry design;
- assume hooks run in parallel; invocation is sequential.

The registry intentionally does not recover panics. A panic indicates a
programming defect, while expected service failures must be returned as
errors. The active cycle frame is still deactivated during panic unwinding.

## Registration and explicit ordering

Every registration has a non-empty name that is unique within the registry.
Names are normalized by trimming whitespace and are included in invocation and
cycle errors.

Invocation order is exactly the order in which `Register` succeeds. The
registry never sorts or otherwise reorders registrations. This keeps ordering
visible at the composition root instead of distributing it across hook-owned
priority values. In a Wire application, a composition provider should encode
the required order in its `Register` calls; the order of declarations in a
`wire.NewSet` is not itself the hook order.

For example, validation can be made explicitly earlier than observation:

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

Invocation is sequential, so this explicit order also makes fail-fast behavior
deterministic. Adding or moving a hook requires changing the composition code
that owns the order.

Registration is a startup phase. Calling `Seal`, or invoking the registry for
the first time, makes the ordered registrations immutable. Later registration
returns `ErrRegistrySealed`. This prevents request-time changes to lifecycle
behavior and means callbacks run without a registry lock.

The registry's zero value is ready for use, and `NewRegistry` is available when
a pointer is more convenient. Registries contain synchronization state and
must not be copied after first use.

## Invocation internals

`Invoke` performs the following steps:

1. Reject a nil context.
2. Seal the registry and obtain its immutable ordered registrations.
3. Stop immediately if the context is canceled.
4. Inspect the causal context chain for active registrations.
5. Reject an active `CyclePolicyError` registration before any nested hook
   runs.
6. Traverse registrations in registration order, skipping only active
   `CyclePolicySkip` registrations.
7. Check cancellation before each hook.
8. Invoke the hook with a derived context containing its active cycle frame.
9. Stop at the first hook error.

The registry lock protects registration and sealing only. The immutable
registration slice is reused after sealing; invocation does not clone it and
does not hold the registry lock while callbacks run.

Multiple goroutines can invoke a sealed registry concurrently. This protects
registry state, not hook internals or mutable data referenced by the event.
Hook implementations remain responsible for their own synchronization.

## Failure behavior

Invocation is synchronous and fail-fast. A hook error is returned as an
`InvocationError` containing its name while preserving `errors.Is` and
`errors.As`. Hooks later in the ordered chain do not run.

Cancellation is checked before dispatch and before each hook. The registry
cannot stop a hook already running; that hook must observe the propagated
context. The registry does not undo previously completed hooks. The service
must place invocation inside an appropriate transaction or provide explicit
compensation when atomicity is required.

## Cycle handling

Each registration has a private identity token. While its hook is running, the
registry adds an active frame containing that token to the derived context.
Nested service calls which propagate the context therefore retain the causal
hook chain.

Cycle tracking identifies individual active registrations instead of marking
the entire registry as already handled. `CyclePolicyError` rejects the nested
invocation, while `CyclePolicySkip` allows unrelated registrations to continue.

The default `CyclePolicyError` rejects a nested invocation before any hook in
that invocation runs and returns a `CycleError`. An intentionally idempotent
hook can opt into `CyclePolicySkip` with `WithCyclePolicy`; only that active
registration is skipped, while remaining hooks still run in registration
order. The explicit option makes intentional cycle breaking visible at
registration.

Cycle frames use an atomic running flag and are deactivated when a hook
returns. A context retained beyond the callback therefore does not permanently
suppress or reject later invocation, although retaining callback contexts is
still outside the hook contract.
