# Calculation demand

RTE tracks whether a balance needs recalculating after its inputs change.

- **Background refresh** (`Activate`). When a tracked input changes, the worker recalculates
  the balance even if nobody asks for it.
- **On-demand refresh** (`EnsureFresh`). RTE remembers changes between requests for
  `PullTTL`. With background refresh off, those changes wait for the next `EnsureFresh`
  call, which recalculates if needed.

`Get` reads the last stored balance. Only `Activate` enables background refresh.

## Choosing a freshness policy

| Setting | What it controls |
| --- | --- |
| `EnsureFreshInput.PullTTL` | How long to track changes between calls to `EnsureFresh`. Must be positive. |
| `EnsureFreshInput.MaxAge` | How old a completed calculation `EnsureFresh` accepts. Zero accepts any age if the key is active and clean. |

For example, a client calls `EnsureFresh` every ten seconds with `PullTTL=1m` and
`MaxAge=30s`. If nothing changes, the second call can reuse the first result. If two usage
batches arrive between calls, one recalculation covers both. With background refresh off, a
gap longer than a minute means the next call recalculates because tracking expired.

Demand is shared per semantic key. Repeated activation keeps the existing registration.
`Deactivate` stops background refresh and change tracking for that key.
There is no per-caller ownership count. Coordinate deactivation when several
consumers depend on the same balance.

When a metered entitlement has a scheduled deletion, demand remains active until
its effective time. The worker checks the deadline for activated keys. For pull-only keys,
`EnsureFresh` checks it on the next call. Once deletion takes effect, processing clears
demand and leaves the last successful result
available through `Get`. An explicit `EnsureFresh` returns the entitlement's
not-found error.

## Coalescing and redelivery

Each key has a dirty version and a completed version. Independent invalidations
advance the dirty version in PostgreSQL, so pending changes survive a worker restart.
Each background worker slot claims one ready key at a time. One calculation can cover all changes
recorded before the worker claims that key. Each calculator reads its own sources
under their usual consistency rules.

`EnsureFresh` accepts a clean version, with an optional maximum age, or calculates
it synchronously. A freshness call already in progress can finish after
deactivation.

An invalidation receipt uses the immutable source and event ID. Inserting its
receipt and marking all matching active keys happen in one transaction. Duplicate
events do nothing. An independent late event still dirties the key. A selector
can span a namespace/kind, owners, or exact semantic keys. An empty owner/target
list deliberately selects the entire namespace/kind. Producers must include all
affected selectors on the first delivery. Keys activated after a fact calculate
from current sources instead of replaying its receipt.

Receipts are retained for seven days by default, configurable independently of
calculation freshness. Maintenance deletes bounded batches of 1,000 old receipts,
normally once per minute per service instance. Full batches retry after one
second to drain a backlog. Replay after deletion safely triggers extra calculation.
Size retention to the expected source replay window to avoid that extra work.

## Concurrent work and retries

A worker locks one key due for background refresh with `FOR UPDATE SKIP LOCKED`. The lock spans
calculation, stored state, its update outbox entry, and completion. Another worker
can work on another key. An invalidation arriving during calculation waits for
the lock, then advances dirtiness after completion. Successful state and
completion commit together. A crash rolls both back, leaving the work ready.

Failed attempts roll back their writes to a savepoint and commit bounded
exponential retry backoff. New input still advances the dirty version during backoff.
The next background attempt covers those changes. An explicit `EnsureFresh` can retry immediately.
Polls order work by its next attempt time and ID, so one failing key does not monopolize the queue.
Each attempt has a configured timeout, while retry bookkeeping uses the outer
context. A canceled database read can close its connection. After rolling that
transaction back, an owning worker records backoff on a fresh connection only
if its dirty/completed versions and attempt count are unchanged. A caller-owned
transaction instead receives the failure and must roll back. The transaction
holds a connection and row lock during calculation, including external reads.
