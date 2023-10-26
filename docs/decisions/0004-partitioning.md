# Partitioning

## Context and Problem Statement

Kafka and ksqlDB achieve horizontal scalability through partitions, which can be assigned to dedicated Kafka brokers and ksqlDB instances.

## Considered Options

We need to address the following challenges related to partitioning:

- Defining partitioning key for producers, tables, and streams
- Ensuring co-partitioning for certain ksqlDB operations like joins.
- Making usage data retrievable by subject and group by properties
- Determining the default number of partitions

## Decision Outcome

- Choose `subject` as the key for Kafka producers.
- Include `subject` and group by properties in ksqlDB meter keys.
- Make the number of partitions configurable.
- Default number of partitions: 100.

### Consequences

In certain cases, such as deduplication or group by conditions, our choices to define keys are limited.

## Pros and Cons of the Options

### Number of partitions

Since OpenMeter will be used with varying loads and numbers of meters, it is challenging to pre-define the ideal partition size. Considering common use cases and the cost of managed solutions, we have chosen `100` as the default number of partitions.

### ksqlDB Partitioning

Tables are always partitioned by their `PRIMARY KEY`, and ksqlDB does not allow repartitioning of tables. Streams, on the other hand, do not have primary keys but can have an optional KEY column, which defines the partitioning column. For streams and tables using existing topics like events, we do not need to define partitions as they reuse the Kafka topic configuration.
