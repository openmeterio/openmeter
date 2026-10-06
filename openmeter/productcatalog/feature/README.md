# Feature persistence

Feature repository operations join the Ent transaction carried by the caller's
context. Reads, writes, eager meter loading, pagination counts, and follow-up
queries use that transaction's connection and observe its uncommitted changes.
This is required when subscriptions materialize entitlements and expand their
views while holding a customer lock: acquiring another pooled connection can
stall the lock holder behind transactions waiting for that lock.

The caller owns transaction creation, commit, and rollback. Repository methods
do not commit independently or create a transaction when none is present;
without a context transaction they use the normal database client. Atomicity
across operations belongs to the owning service or workflow. An incompatible
context transaction returns an error instead of falling back to the pool.

The feature connector wraps create, update, and archive commands in a
transaction, joining an existing transaction when present. Validation reads,
persistence, and event publication receive the same transaction context. With
the system-event outbox publisher, feature changes and queued events commit
or roll back together. Read-only connector operations join a caller transaction
without starting one.

The adapter binds a new instance to the transaction and preserves its logger.
The shared repository instance remains usable by concurrent callers.
