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

Credit rating uses a strict storage-time cutoff at whole-second precision. For an ingest batch,
the worker waits until the next whole second after `StoredAt` before looking up active demand.
A first pull can register during that second and calculate a balance that excludes the batch.
Resolving demand after the cutoff lets the same notification invalidate that result.
Older batches proceed immediately, and cancellation interrupts the wait.

`rte.upstream.events` distinguishes `no_target`, `routed` and `error`. A routed fact was passed to
demand processing, where it may still be a duplicate or match no active row. Compare with
`rte.demand.invalidated` for affected keys and `rte.calculations` for executions.
`rte.upstream.lag` measures time from envelope publication to processing. A newly published batch
can contain much older usage.
