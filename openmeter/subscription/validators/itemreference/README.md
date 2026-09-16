# Subscription item reference validation

A subscription item reference identifies one physical item materialization by
namespace, subscription ID, phase ID, and item ID. The validator protects
consumers that persist those IDs independently from accidentally combining
otherwise valid rows from different branches of the subscription graph.

The reference is valid only when all of the following belong to the same
namespace-scoped chain:

- the subscription exists in the namespace
- the phase belongs to that subscription
- the item belongs to that phase

This is a subscription-domain invariant. Consumers use the validator from
their service boundary before persisting a reference. Persistence adapters do
not depend on the validator; they remain responsible for storage-specific
constraints such as conditional updates and affected-row checks. For example,
[charge subscription-reference repair](../../../billing/charges/README.md#intent-layers)
validates the target chain in the charge service, while its adapter prevents
the repair from changing the charge's subscription identity.

## Materialization lifecycle

[Subscription materialization](../../service/README.md) may archive and
recreate phases and items when desired state changes. Physical child IDs are
therefore not stable logical identity, but persisted downstream history can
still legitimately reference an archived materialization.

Reference validation intentionally includes archived subscriptions, phases,
and items. Archival controls lifecycle visibility; it does not invalidate the
structural relationship between already-persisted rows. The validator does not
decide whether a reference is current, active at a particular time, or
semantically equivalent to a replacement.

It also does not validate price, feature, tax, cadence, or other item content.
Those facts are outside the structural subscription-to-phase-to-item ownership
chain and remain the responsibility of their owning domain operations.

## Transactions and failures

Validation starts a transaction when the caller has not supplied one and
reuses a transaction carried by the caller's context. A service can therefore
validate a target reference and persist it in the same transaction.

Missing input identifiers are validation errors. A complete tuple that does
not resolve to one ownership chain is a precondition failure. Repository
failures are propagated with operation context so callers can distinguish an
invalid reference from an unavailable validation query.
