# System event outbox

The shared publisher stores system events in PostgreSQL before sending them to
Kafka. An event joins the caller's transaction, including nested savepoints;
without a caller transaction, enqueueing uses its own transaction. Other topics
keep their direct publication behavior.

`Publish` succeeds when the event has been stored in the current transaction,
not when Kafka has received it. An outer rollback removes the event along with
the domain changes. Serialization and validation still happen before enqueueing.

## Delivery and retries

After the outer commit, a notification wakes the publisher's in-process
drainers. Each wake attempts a bounded amount of work from the shared table,
including older pending events. There is no polling timer, startup sweep, or
independent retry service. A pending event may remain undelivered until another
system event is published, including after a process restart or broker outage.

Drainers use their application context rather than the completed request's
context. They remove each row only after the broker acknowledges it. A failed
send, delete, or commit leaves the row eligible for a later attempt. Crashing
after broker acknowledgment can cause duplicate delivery; retries preserve the
original message ID, payload, metadata, topic, and Kafka message key.

The outbox is a delivery queue, not an audit log. Delivered rows are physically
removed. Consumers must tolerate duplicate delivery.

## Ordering and isolation

Ordering follows the existing Kafka message key, normally a customer. Events
without a key share an unkeyed lane. A transaction-scoped advisory lock
serializes enqueueing for each topic/key until the outer transaction ends;
this prevents an earlier allocated ID from committing after a later event was
already delivered. Multi-key operations must acquire keys in a consistent order,
as with other transactional locks.

Drainers claim only the oldest pending event for a key. `FOR UPDATE SKIP LOCKED`
allows other keys to progress while that row is being sent. A failed event keeps
later events for its key pending, but the drain continues with unrelated keys.
There is no global ordering guarantee across keys.

Delivery is asynchronous and uses a fixed, small number of workers per process.
No domain transaction waits for Kafka. A relay transaction locks only its claimed
outbox row while sending; it never acquires the enqueue advisory lock. Drain
attempts have a context deadline and a message limit, and broker I/O retains the
underlying publisher's timeout behavior.
