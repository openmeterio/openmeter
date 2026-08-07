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
only orders and invokes those hooks; it does not determine whether an event is
transactionally valid or durable. System-event publication currently has a
known problem where an event can be sent from a transaction which is later
rolled back. Correctly coupling publication to commit, for example through an
outbox or an after-commit mechanism, is outside the scope of service hooks and
must be handled by the owning service and event infrastructure.

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

var hooks servicehooks.Registry[CustomerLifecycleEvent]
```

The registry is responsible for:

- validating and storing named hook registrations;
- ordering hooks by priority and registration order;
- freezing registration before request processing;
- invoking hooks synchronously with caller context propagation;
- detecting recursive hook invocation in the causal context chain;
- attaching hook identity and priority to returned errors.

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
err := hooks.Register(
    "ledger-accounts",
    servicehooks.HookFunc[CustomerLifecycleEvent](provisionLedgerAccounts),
    servicehooks.WithPriority(servicehooks.PriorityHigh),
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
- assume equal-priority hooks run in parallel; invocation is sequential.

The registry intentionally does not recover panics. A panic indicates a
programming defect, while expected service failures must be returned as
errors. The active cycle frame is still deactivated during panic unwinding.

## Registration and priority

Every registration has a non-empty name that is unique within the registry.
Names are normalized by trimming whitespace and are included in invocation and
cycle errors.

Priority is an integer from `0` through `100`; lower values run first. Named
priorities are:

- `PriorityHighest` (`0`)
- `PriorityHigh` (`25`)
- `PriorityDefault` (`50`)
- `PriorityLow` (`75`)
- `PriorityLowest` (`100`)

Registrations without `WithPriority` use `PriorityDefault`. Hooks with equal
priority retain registration order. Invocation is sequential even when hooks
have equal priority, so fail-fast behavior remains deterministic.

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
6. Traverse registrations in priority order, skipping only active
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
`InvocationError` containing its name and priority while preserving
`errors.Is` and `errors.As`. Hooks later in the ordered chain do not run.

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
registration is skipped, while remaining hooks still run in priority order.
The explicit option makes intentional cycle breaking visible at registration.

Cycle frames use an atomic running flag and are deactivated when a hook
returns. A context retained beyond the callback therefore does not permanently
suppress or reject later invocation, although retaining callback contexts is
still outside the hook contract.
