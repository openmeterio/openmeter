# Idempotency

## Context and Problem Statement

Event-driven ingestion requires idempotency to handle message retries, ensuring reliable processing.
Key considerations are:

- Distributed systems often guarantee least-once delivery.
- Event replays may be necessary for failure and recovery scenarios.
- CloudEvents are unique by ID + Source properties.

Neither Kafka nor ksqlDB natively support message idempotency.

## Considered Options

To achive idempotency we discovered event deduplication options both on producer (API Server) and consumer (ksqlDB) side:

1. Redis with key expiration
1. Redis with bloom filter
1. Deduplication in ksqlDB via windowed tables

## Decision Outcome

To maintain strong consistency and avoid complex multi-phase commits and state management, we chose deduplication via ksqlDB.

- Idempotency via deduplication
- Deduplication in ksqlDB via windowed tables
- Events are unique by ID + Source
- 32 days default deduplication window

### Consequences

- Pros: No need to run Redis.
- Pros: Strong consistency can be guaranteed.
- Cons: Increased Kafka storage for deduplication window.
- Cons: Longer retention window may impact backup and recovery time.

## Pros and Cons of the Options

### Redis with key expiration

- Pros: Simple implementation.
- Cons: Inconsistent with Kafka Producer.
- Cons: Requires Redis.

### Redis with bloom filter

- Same tradeoffs as Redis with key expiration.
- Pros: Efficient storage.
- Cons: Possible false positives leading to dropped events.

### Deduplication in ksqlDB via windowed tables

Deduplicating Kafka messages with ksqlDB requires an extra table where we count how many times we have seen a specific event. Then we only include events in streaming processing at their first occurrence.

See [How to find distinct values in a stream of events](https://developer.confluent.io/tutorials/finding-distinct-events/ksql.html) for more details.

- Pros: No separate system required.
- Pros: Guaranteed consistency.
- Cons: Increased Kafka storage for deduplication window.
- Cons: Longer retention window may impact backup and recovery time.
