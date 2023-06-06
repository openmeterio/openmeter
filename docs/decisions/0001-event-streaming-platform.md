# Event Streaming Platform

To handle the aggregation of millions of events per second, we needed to select a stream processing solution.

## Context and Problem Statement

Usage metering requires real-time processing of large event volumes with high accuracy to support billing and data-intensive applications such as DevTool, AI, and IoT.
Balancing scale, accuracy, latency, and cost poses challenges:

- Monitoring systems fall short in terms of accuracy and consistency necessary for billing
- Scaling databases to handle large volumes of writes and real-time queries can be expensive
- Warehouses processing leads to stale usage data and longer feedback cycles

## Considered Options

1. Kafka with Kafka Connect.
2. Kafka with ksqlDB.
3. InfinyOn Cloud.

## Decision Outcome

We have chosen Kafka with ksqlDB.

- Kafka for event streaming.
- ksqlDB for stream processing.

To make OpenMeter adoptable to alternative streaming platforms we keep the interfaces generic.

### Consequences

- Pros: Kafka has been battle-tested at scale.
- Pros: Kafka is widely popular among developers.
- Pros: Kafka is available as a managed solution.
- Pros: ksqlDB provides a user-friendly interface.
- Cons: ksqlDB is licensed under the Confluent Community License.
- Cons: Neither Kafka nor ksqlDB offers message idempotency out of the box.
- Cons: The Kafka ecosystem primarily caters to Java, while OpenMeter is implemented in Go.
