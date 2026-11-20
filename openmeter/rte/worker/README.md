# RTE worker

The worker consumes the system and ingest topics under one Kafka consumer group. Message
handlers run alongside background slots that poll PostgreSQL for pending demand.

| Message | Worker action |
| --- | --- |
| Calculation request | Run the requested calculator. With a freshness policy, first check whether a stored result satisfies it. An unconditional request may execute again on redelivery. |
| Comparison request | Replay the sampled operation and record paired durations on success. Observations expire after one minute. |
| Supported source event | Resolve affected keys and mark matching active demand dirty once per Kafka position. |
| Calculation update | No further calculation. This event tells downstream consumers that RTE accepted a result. |

## Source offsets

Watermill handles one record at a time per partition and waits for acknowledgement before
dispatching the next. A successful source invalidation records its offset even when no key
matched. Unrelated messages on the same topic can leave gaps. Commands and comparison requests
retain their own delivery semantics. The Kafka consumer group still manages where consumption starts.

When Sarama assigns a partition, the worker checks the broker's exclusive log end before
reporting the claim ready. If the saved offset is at or beyond that end, it clears the
checkpoint and logs the reset. A failed check aborts the claim.

The checkpoint belongs to the topic name and partition. A recreated topic that has already
grown past its old checkpoint looks like the original. When replacing topics, stop the old
consumers and clear those topics' checkpoint rows before restarting. An intentional Kafka
rewind also needs a checkpoint reset if the aim is to apply the old invalidations again.

## Background polling

Each slot claims and processes one eligible key before looking for another. It checks again
immediately after finishing work, including an attempt that scheduled a retry. An empty or
failed claim waits for `rte.pollInterval`. With four idle slots and a one-second interval,
one process makes roughly four claim attempts per second.

A slot holds one transaction through calculation, including external usage reads. Size
`rte.calculationConcurrency` with database capacity in mind. This setting bounds background
claims per process. Requests handled from Kafka can run alongside them. Shutdown cancels the
slots and waits for in-flight attempts to finish. [Queue order and locking](../demand/README.md#claims-and-queue-order)

## When stored usage becomes visible

Credit rating uses an exclusive storage-time cutoff at whole-second precision. An initial
calculation can miss a batch written earlier in the same second. The worker waits for the
notification's storage second to end **before resolving demand**, so a key activated during
that second can still be invalidated. Older batches can be routed immediately. Occurrence
time determines which service periods the usage affects. [Source routing](../upstream/README.md)
