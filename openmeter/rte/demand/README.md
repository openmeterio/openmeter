# Calculation demand

Demand identifies which stored calculation keys need maintenance. Push activation
keeps a key active until deactivation; optional refresh intervals account for
time-dependent results. A pull keeps invalidation interest for its TTL and
calculates only when `EnsureFresh` is called. Pull-only keys never calculate in
the background. `Get` still reads stored state without tracking or calculating.

Each key has a dirty version and a completed version. Independent invalidations
advance dirtiness; timers also start a new dirty version when due. One successful
calculation can cover several invalidations.
These versions describe handled demand, not a source revision or a transactional
snapshot of every calculator input. `EnsureFresh` accepts a clean version, with
an optional maximum age, or synchronously calculates it. A freshness call already
in progress can finish after deactivation; deactivation removes future interest.

An invalidation receipt uses the immutable source and event ID. Inserting its
receipt and marking all matching active keys happen in one transaction. Duplicate
events do nothing; an independent late event still dirties the key. A selector
can span a namespace/kind, owners, or exact semantic keys. An empty owner/target
list deliberately selects the entire namespace/kind. Producers must include all
affected selectors on the first delivery. Keys activated after a fact calculate
from current sources instead of replaying its receipt.

Receipts are retained for seven days by default, configurable independently of
calculation freshness. Maintenance deletes bounded batches of 1,000 old receipts,
normally once per minute per service instance. Full batches retry after one
second to drain a backlog. Replay after deletion safely triggers extra calculation;
size retention to the expected source replay window to avoid that extra work.

A worker locks one ready push key with `FOR UPDATE SKIP LOCKED`. The lock spans
calculation, stored state, its update outbox entry, and completion. Another worker
can work on another key. An invalidation arriving during calculation waits for
the lock, then advances dirtiness after completion. Successful state and
completion commit together; a crash rolls both back, leaving the work ready.

Failed attempts roll back their writes to a savepoint and commit bounded
exponential retry backoff. A pull can retry immediately. Polls order work by its
next attempt time and ID, so one failing key does not monopolize the queue.
Each attempt has a configured timeout, while retry bookkeeping uses the outer
context. A canceled database read can close its connection. After rolling that
transaction back, an owning worker records backoff on a fresh connection only
if its dirty/completed versions and attempt count are unchanged. A caller-owned
transaction instead receives the failure and must roll back. The transaction
holds a connection and row lock during calculation, including external reads.
