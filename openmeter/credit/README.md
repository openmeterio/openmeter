# Credit

Credit calculates how metered usage consumes grants over time. Its primary
owner is a metered entitlement, but the calculation model is generic: an owner
provides usage, grants, usage periods, and reset behavior.

Credit balances are derived state. They are not money and they are not an
accounting ledger. See [Ledger](../ledger/README.md) for accounting state and
[Entitlements](../entitlement/README.md) for access semantics.

## Grants

A grant is an immutable, time-effective allocation of credit. Creation adds a
new grant; voiding ends its effective period without rewriting its earlier
meaning.

Usage is burnt in deterministic order:

1. lower numeric priority first;
2. earlier expiry first;
3. earlier creation first.

Recurring grants replenish their own balance on their recurrence. Owner usage
resets are different: they apply each active grant's rollover limits and begin a
new usage period.

Only active grants contribute to a balance. Expired and voided grants remain
part of historical calculations for times when they were active.

## Time semantics

Credit has one-minute temporal resolution. This is a domain constraint inherited
from minute-windowed metering, not merely a snapshot or database detail.

Grant effective times, recurrence anchors, voids, and resets are normalized to
minute boundaries. A balance read at a sub-minute timestamp rounds up to include
usage from that partial minute. Callers must not rely on ordering or balance
changes within the same minute.

History boundaries and persisted snapshots must use the same minute alignment.
Changing this resolution requires changing the metering and credit time model
together.

## Balance calculation

A balance at time `t` is calculated from:

- the owner's metering and reset configuration;
- grants effective during the calculation;
- usage-period and explicit reset boundaries;
- metered usage up to `t`.

The engine burns usage from active grants in priority order. Usage that cannot
be covered becomes overage. The result contains the remaining grant balances,
overage, and the usage-period state needed to continue the calculation.

An engine run also produces a contiguous burn-down history anchored to its
starting balance. The anchor is part of the history's meaning: the segments
alone are not enough to reconstruct balances.

## Resets

A reset closes one usage period and starts another. Active grant balances are
first constrained by their rollover limits. If overage is preserved, it is then
burnt from those rolled-over balances in the new period.

The burn-down history represents that reset transition explicitly. Rollover and
preserved overage burn must not be hidden by changing the meaning of the usage
segments on either side.

Resetting changes both balance state and the owner's usage-period timeline.
Those changes are serialized for the owner so readers cannot observe only half
of the transition.

## Balance snapshots

A balance snapshot is a complete calculation checkpoint at a specific time. It
is a cache of derived state, not an event and not part of the source history.

### Creation

Balance calculations may save a snapshot at an eligible history breakpoint.
Snapshots are kept behind a configurable grace window so recent usage remains
recalculable when events arrive late. A reset saves the balance at the reset
boundary as part of the reset transaction.

`LATEST` balances are not persisted as snapshots because a latest value is not
reusable cumulative state.

### Use

A balance calculation starts from the newest usable snapshot at or before the
query time and evaluates only the inputs after it. If no snapshot is usable,
calculation starts from the owner's measurement start.

Snapshot presence affects cost, never meaning. Removing intermediate snapshots,
removing every snapshot before the latest one, or inserting an earlier snapshot
must not change the result obtained from a later checkpoint.

Snapshots are independent checkpoints rather than a chain. Each one contains
all state required to continue the calculation, and incompatible snapshots are
ignored.

### Invalidation

A time-effective input can make saved derived state stale. The operation that
changes grants, resets, usage, or another calculation input is responsible for
invalidating affected snapshots.

If a change affects the state represented by a snapshot, that snapshot must be
replaced or invalidated together with every later snapshot. Invalidated
snapshots are excluded from selection; the next read recomputes from an earlier
usable checkpoint or from measurement start.

This rule is what makes snapshot pruning safe and prevents balance correctness
from depending on which checkpoints happen to exist.

## Boundaries

- The owner supplies metering, usage attribution, reset timing, and conversion
  behavior.
- Streaming owns raw usage; credit consumes meter results.
- Credit owns grants, burn order, rollover, overage, calculation history, and
  balance snapshots.
- Entitlement turns the calculated balance into an access decision.
- Billing and ledger consumers must not treat feature credit as money.
