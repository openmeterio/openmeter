# System event outbox

System events are stored in PostgreSQL within the caller's transaction (or a
standalone transaction). Outer and savepoint rollbacks discard their events.
`Publish` means stored, not delivered to Kafka; other topics still publish directly.

## Delivery

After commit, a small fixed set of workers drains the shared queue, including
older pending events. Productive batches continue draining. There is no timer or
startup sweep: after a failure or restart, pending events may wait for new traffic.

Rows are deleted after broker acknowledgment. Delivery is **at least once**:
failed deletion or commit can resend the original ID, payload, headers, topic,
and Kafka message key. Consumers must tolerate duplicates.

## Ordering and shutdown

Ordering is per topic/Kafka message key; unkeyed events share one lane.
Transaction-scoped advisory locks serialize enqueueing until commit, preventing
later events from overtaking uncommitted predecessors. Multi-key transactions
must acquire keys consistently to avoid deadlocks.

Workers claim only each key's oldest row with `FOR UPDATE SKIP LOCKED`. Failed
heads hold back their key; unrelated keys can progress. Domain transactions never
wait for Kafka, and workers use the application context, not the request context.

`Close` cancels workers and waits for them; it does not flush the backlog. Pending
rows remain durable. Drain passes have message/time limits, but synchronous
Kafka I/O follows the underlying publisher's timeouts and can delay shutdown.
