# Entitlements

Entitlements answer whether a customer may use a feature and, when relevant,
return the state behind that decision. They connect product configuration to a
customer's usage attribution, but they do not own raw usage or accounting
entries.

## Domain model

An entitlement belongs to a customer and a feature. At most one entitlement for
the same feature is active for a customer at a given time.

| Type | Meaning |
| --- | --- |
| `boolean` | grants access while active |
| `static` | grants access and carries an immutable JSON configuration |
| `metered` | derives access from metered usage and grant balance |

Entitlement values are time-bound. An entitlement is active from its configured
start, or creation time, until its configured end or deletion. The end is
exclusive. Outside that interval it provides no access.

## Ownership and lifecycle

Entitlements may be created directly or derived from subscriptions.
[Subscription](../subscription/README.md) owns the customer-specific schedule
that materializes and supersedes subscription-managed entitlements.
Entitlement owns their persisted lifecycle and value resolution.

Definitions are historical. Deletion ends an entitlement at a timestamp rather
than erasing it, and overriding supersedes the previous definition rather than
changing its meaning in the past.

An entitlement identifies the customer-feature relationship. Metered value
calculations resolve that customer's usage attribution when querying the
feature's meter.

## Metered entitlements

A metered entitlement connects a feature's meter to grants owned by the
entitlement. Metered usage consumes active grants in priority order. Uncovered
usage becomes overage.

Access is allowed while balance remains. A soft limit continues to allow access
after the balance is exhausted; it does not prevent usage or overage from being
calculated.

Measurement begins at the entitlement's measurement start. Usage before that
time is outside the entitlement calculation.

Metered values inherit [Credit](../credit/README.md)'s one-minute resolution.
They must not be used to express sub-minute access transitions.

## Usage periods and resets

Usage periods partition a metered entitlement's usage over time. The initial
period begins at measurement start and later periods follow the configured
recurrence.

An explicit reset ends the current period and starts a new one. The reset may
retain the original recurrence anchor or establish a new one. Grant rollover
and preserved-overage behavior are part of the credit reset, not separate
entitlement balances.

See [Credit](../credit/README.md) for grant consumption, reset, history, and
snapshot semantics.

## Value events

Entitlement value reads calculate the value at the requested time. Separately,
the balance worker recalculates values when usage or entitlement state changes
and publishes value events for notifications and downstream consumers.

Those event payloads are sometimes called snapshots, but they are not the
durable credit balance snapshots used to resume calculations. Event handlers
are retried, so consumers must process them idempotently.

Crossing a future entitlement or grant activation or expiry time does not by
itself produce a value event. Reads reflect the new state, while event consumers
observe it after another recalculation trigger or a scheduled recalculation.

## Boundaries

- Features and meters define what is measured; entitlement selects the feature,
  customer attribution, and access policy.
- Streaming owns raw usage and meter queries.
- [Credit](../credit/README.md) owns grants, balance calculation, resets,
  history, and balance snapshots for metered entitlements.
- [Subscriptions](../subscription/README.md) may materialize and supersede
  subscription-managed entitlements.
- Notifications consume entitlement value events; those events are not a
  transactional source of balance truth.
- Billing and [Ledger](../ledger/README.md) own monetary and accounting state.
