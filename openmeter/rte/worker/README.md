# Event and demand processing

The worker uses one Kafka consumer group across distinct ingest and system-event topics. It runs
message handling alongside a bounded pool that drains the durable PostgreSQL demand queue.
`rte.calculationConcurrency` sets background slots per process (default 4). Each slot claims one
ready key with `FOR UPDATE SKIP LOCKED` and keeps the claim through calculation and completion.
Slots and replicas skip keys already held by another calculation.

After processing a key, including recording its retry, that slot immediately checks for more work.
An empty or failed claim waits `rte.pollInterval` before another attempt. Each idle slot polls
independently, so idle query volume grows with the configured concurrency. Size the pool with DB
connection capacity in mind: in-flight calculations hold a transaction open. Shutdown cancels all
slots and waits for them to finish. The worker owns execution and idle waiting; the
[demand repository](../demand/README.md) owns claims and retries.

This limit applies to background demand processing. Explicit requests and shadow comparisons run
in the message handlers and can calculate alongside the pool. Polling discovers dirty work and
retries; it does not periodically recalculate clean balances.

| Message | Worker action |
| --- | --- |
| Calculation request without a freshness policy | Decode the concrete key and execute an exact calculation |
| Calculation request with a freshness policy | Call `EnsureFresh`. A clean, sufficiently recent state can satisfy the request without emitting an update |
| Sampled balance comparison request | Replay the current read or calculation and record paired timings. Expired requests are discarded. |
| Recognized upstream fact | Resolve affected demand, then invalidate it with the original CloudEvents source and ID |
| Calculation update | A downstream notification. The worker takes no further calculation action |

The source receipt belongs to invalidation. A redelivered explicit calculation request can execute
again. Consumers waiting for updates on the shared topic must match the calculation kind and
semantic key, and handle duplicates or out-of-order delivery. A freshness request served from
storage delivers no acknowledgement through the update event.

Credit rating uses a strict storage-time cutoff at whole-second precision. For an ingest batch,
the worker waits until the next whole second after `StoredAt` before looking up active demand.
A first `EnsureFresh` call can start tracking a key during that second and calculate a balance that excludes the batch.
Resolving demand after the cutoff lets the same notification invalidate that result.
Older batches proceed immediately, and cancellation interrupts the wait.

`rte.upstream.events` distinguishes `no_target`, `routed` and `error`. A routed fact was passed to
demand processing, where it may still be a duplicate or match no active row. Compare with
`rte.demand.invalidated` for affected keys and `rte.calculations` for executions.
`rte.upstream.lag` measures time from envelope publication to processing. A newly published batch
can contain much older usage.
