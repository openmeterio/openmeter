# Subscription materialization

The subscription service persists the desired `SubscriptionSpec` as
subscription-owned phase and item rows and keeps derived entitlements aligned
with them.

This is synchronous materialization inside a subscription command. It is
different from [billing subscription sync](../../billing/worker/subscriptionsync/README.md),
which reacts to committed subscription state and reconciles invoice or charge
artifacts asynchronously.

## Ownership

- The workflow layer decides the product operation, resolves its timing, and
  constructs a complete target spec.
- The subscription service validates lifecycle rules, runs command hooks,
  resolves item features and currencies, materializes the target spec, and
  publishes the resulting subscription event.
- The materializer owns subscription phases, items, and their entitlement
  scheduling. It does not create invoice lines, charges, or ledger entries.

Create, update, cancellation, continuation, and deletion run in the
subscription transaction. A failed materialization does not leave a partially
updated subscription graph.

## Reconciliation model

An update indexes the current `SubscriptionView` and complete target spec by
logical path. Phase, item-version, and entitlement nodes compare their own
materialized shape and produce an ordered archive/create plan. Archives run
from derived resources toward their parents; creates run from parents toward
derived resources so generated references are available when needed.

Phase keys identify logical phases. Within a phase, item key and slice position
identify a logical item version. When a phase changes, its items are
rematerialized with it. Items and their derived entitlements are also one
replacement group because the persisted item references the generated
entitlement ID.

Changed rows are deleted and recreated rather than updated field by field.
Their database IDs—and the IDs of entitlements derived from them—are therefore
not stable domain identity. Consumers should use subscription paths and
annotations intended for correlation, not persisted child IDs.

## Boundaries and invariants

- The target spec must keep the existing customer, plan reference,
  subscription start, and settlement mode.
- Item and entitlement cadence is derived from the target spec and cannot
  escape the owning phase or subscription.
- Entitlement creation and deletion follows the owning item within the same
  transaction.
- The materializer receives a complete desired state. It does not interpret
  the user's patch sequence or decide command timing.
- Feature references in create and update targets must resolve in the item
  namespace before persistence, whether or not the item creates an entitlement.
- Custom item currencies must be resolved by the service before persistence;
  the materializer verifies that their managed identity belongs to the item
  namespace without loading currency state itself.
- Events are published from the materialized view. Downstream consumers should
  derive work from that committed view and tolerate delivery retries.
