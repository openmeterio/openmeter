# System event outbox

System events are stored in PostgreSQL within the caller's transaction (or a
standalone transaction). Outer and savepoint rollbacks discard their events.
`Publish` means stored, not delivered to Kafka; other topics still publish directly.

## Delivery

After commit, a small fixed set of workers drains the shared queue, including
older pending events. Productive batches continue draining. Workers also retry
periodically (one minute by default), so pending events recover after a failure
or restart without new traffic. Drain limit, timeout, concurrency, and retry
interval are configured under `events.outbox`.

Rows are deleted after broker acknowledgment. Delivery is **at least once**:
failed deletion or commit can resend the original ID, payload, headers, topic,
and Kafka message key. Consumers must tolerate duplicates.

## Concurrency and shutdown

Workers claim pending rows with `FOR UPDATE SKIP LOCKED`. A locked or failed
row does not hold back other rows, including those with the same Kafka key.
Delivery order is not guaranteed; consumers must tolerate reordered events.
Domain transactions never wait for Kafka, and workers use the application context.

`Close` cancels workers and waits for them; it does not flush the backlog. Pending
rows remain durable. Drain passes have message/time limits, but synchronous
Kafka I/O follows the underlying publisher's timeouts and can delay shutdown.
