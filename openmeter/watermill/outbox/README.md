# System event outbox

System events are stored in PostgreSQL within the caller's transaction (or a
standalone transaction). Savepoints share the outer transaction ID; rollbacks
discard their events. `Publish` means stored, not delivered to Kafka. Other topics
still publish directly.

## Delivery and ordering

After commit, workers drain the shared queue. They also retry periodically
(default: one minute). `events.outbox` configures drain limit, timeout, concurrency,
retry interval, and `maxAttempts` (default: 10).

Workers claim a source transaction's pending rows together and send them in
sequence-ID order. If B fails in A → B → C, A is removed, B's attempt count is
persisted, and C waits for a later drain. Other transactions can progress. Once B
reaches `maxAttempts`, it is logged and retained for inspection, skipped by future
drains, and C can proceed. Raising the limit makes retained rows eligible again.

Acknowledged events are **hard-deleted**. A crash or failed database commit can
resend them, so consumers must tolerate duplicates. Failed-attempt counts survive
successful drain commits; crashes or database failures can cause extra attempts.
Finite retries mean some events may never be delivered. Ordering applies within
a source transaction, until an event is abandoned; there is no ordering guarantee
between transactions or across Kafka partitions.

## Concurrency and shutdown

`FOR UPDATE SKIP LOCKED` claims transaction heads without waiting for other
workers. Domain transactions never wait for Kafka. Drain message limits are
checked between transaction batches; a large batch can exceed the message limit.
The drain timeout still applies.

`Close` cancels workers and waits; it does not flush pending events. Workers use
the application context, and synchronous Kafka I/O follows the underlying
publisher's timeouts, which can delay shutdown.
