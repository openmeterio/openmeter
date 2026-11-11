# Event and demand processing

The worker uses one Kafka consumer group across distinct ingest and system-event topics. It runs
the message router alongside a loop that drains ready push work. A successful push calculation
immediately polls for the next key. An empty poll or failure waits for the poll interval. The
[demand repository](../demand/README.md) owns the durable work, so another instance can take ready
keys after a restart.

| Message | Worker action |
| --- | --- |
| Calculation request without a freshness policy | Decode the concrete key and execute an exact calculation |
| Calculation request with a freshness policy | Call `EnsureFresh`. A clean, sufficiently recent state can satisfy the request without emitting an update |
| Recognized upstream fact | Resolve affected demand, then invalidate it with the original CloudEvents source and ID |
| Calculation update | A downstream notification. The worker takes no further calculation action |

The source receipt belongs to invalidation. A redelivered explicit calculation request can execute
again. Consumers waiting for updates on the shared topic must match the calculation kind and
semantic key, and handle duplicates or out-of-order delivery. A freshness request served from
storage delivers no acknowledgement through the update event.

For a credit-affecting ingest batch, the sink has confirmed persistence, but credit rating applies
a strict storage-time cutoff at whole-second precision. The worker waits until the next whole
second after the batch’s `StoredAt` before marking keys dirty. Otherwise a fast push calculation
could omit the new rows and mark that result fresh. A metered-only batch bypasses this wait.
Cancellation interrupts it.

`rte.upstream.events` distinguishes `no_target`, `routed` and `error`. A routed fact was passed to
demand processing, where it may still be a duplicate or match no active row. Compare with
`rte.demand.invalidated` for affected keys and `rte.calculations` for executions.
`rte.upstream.lag` measures time from envelope publication to processing. A newly published batch
can contain much older usage.
