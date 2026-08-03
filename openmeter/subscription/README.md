# Subscription

A subscription is OpenMeter's desired commercial schedule for a customer. It
combines plan-derived phases and rate cards with customer-specific timing,
currency, settlement mode, and customizations.

The subscription domain owns that schedule and keeps any derived entitlements
aligned with it. It does not own invoices, charge realization, or ledger
entries. Those are reconciled asynchronously from subscription state by
[Subscription sync](../billing/worker/subscriptionsync/README.md).

## The model

A subscription has a customer, active interval, billing cadence and anchor,
currency, settlement mode, and optional plan reference.

Its complete desired shape is a `SubscriptionSpec`:

- phases are keyed and ordered by their offset from the subscription start
- a phase ends when the next phase begins, or when the subscription ends
- items are grouped by key within a phase
- each item carries a rate card, optional relative timing overrides, and the
  information needed to schedule an entitlement

`ItemsByKey[key]` is a time-ordered version history. Its slice index identifies
a logical revision of the item; it is not quantity. Removing or inserting an
element changes the identity used by downstream subscription reconciliation.

A `SubscriptionView` is the hydrated read model: the subscription, its current
spec, customer, phases, items, features, and entitlements. The spec is the
desired state used for editing; the persisted child entities and expanded
references show how that state is currently materialized. A valid view keeps
the duplicated identity, cadence, currency, plan, and settlement facts
consistent.

## Lifecycle and edits

Subscription status is derived at a point in time from its active interval,
deletion time, and whether cancellation has set an end. The lifecycle supports
scheduled creation, active updates, cancellation, continuation, and deletion;
the state machine decides which action is legal from the current status.

Cancellation shortens the subscription and the cadences of affected phases,
items, and entitlements. Continuation removes a cancellable end and extends the
derived children again. Deletion is distinct from cancellation: deleted
subscriptions leave the normal read path but remain available to cleanup
consumers that explicitly include deleted records.

Updating a subscription reconciles its current view with a new spec. Customer,
plan reference, subscription start, and settlement mode are not mutable through
this path. Changed phases or items may be deleted and recreated, including
their entitlements. Consumers must not treat a persisted phase, item, or
entitlement ID as the durable identity of a logical spec path.

Higher-level plan workflows create a spec from a plan and express running
changes as patches. Addons also modify the spec, but an addon-bearing
subscription cannot be edited through the general running-edit workflow; use
the addon workflow so its overlay remains reversible. See
[Subscription addons](addon/README.md).

## Workflow, patches, and service

The workflow layer owns customer-facing operations such as creating from a
plan, editing a running subscription, changing plans, restoring, and applying
addons. It resolves command timing, constructs the target spec, and coordinates
operations that span more than one subscription.

[Patches](patch/README.md) are ordered transformations of a
`SubscriptionSpec`. They express how a workflow changes desired state; they do
not persist subscription entities or billing artifacts.

The core service owns subscription lifecycle validation, hooks, persistence,
and event publication. Given a complete target spec, its
[materialization](service/README.md) reconciles subscription-owned phase, item,
and entitlement state. The service does not decide which plan workflow or
patches produced that target.

## Cadence and effective time

`BillingAnchor` and `BillingCadence` define aligned billing periods. Periods
are clipped to phase and subscription boundaries. The plan workflow defaults
the anchor to `ActiveFrom`, but the stored anchor is an explicit subscription
fact and must be carried through replacements and plan changes.

`Timing` describes when a command should take effect. It contains exactly one
of a custom timestamp or a supported enum. `immediate` resolves from the
subscription clock; `next_billing_cycle` resolves to the end of the aligned
billing period containing that clock time. The resolved time becomes the
effective time used while applying patches.

Allowed timing depends on the action. Running edits accept immediate or
next-billing-cycle timing but not arbitrary custom timestamps. A custom
cancellation time for an aligned subscription must fall on a billing-period
boundary. Creation may be scheduled with a custom timestamp, subject to its
past-time tolerance.

Phase offsets and item timing overrides remain relative to the subscription
and phase; do not persist a derived absolute end as independent source truth.

## Neighboring domains

- The product catalog supplies the plan shape used to build the initial spec.
  The subscription owns the resulting customer-specific schedule, not the
  mutable plan definition.
- [Entitlements](../entitlement/README.md) are derived from
  entitlement-bearing rate cards and share the effective cadence of their
  subscription item. Recreating an item may recreate its entitlement.
- Subscription commands publish lifecycle events. Successful subscription
  mutation means the desired schedule was committed; it does not mean billing
  artifacts have already been reconciled.
- Billing behavior such as line generation, invoice collection, charge
  realization, proration materialization, and immutable-invoice handling
  belongs outside this package.

## Invariants

- phase order is determined from the full spec; a phase row alone cannot tell
  you its end
- item slice position is version identity, not quantity
- subscription, spec, and expanded view must agree on shared top-level facts
- item and entitlement cadences cannot escape their phase or subscription
- downstream work must be derived from the committed view and tolerate event
  retries
