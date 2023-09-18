# ClickHouse

To store historical usage data and to pre-aggregate events we choose ClickHouse.

## Context and Problem Statement

OpenMeter faces the need to store historical usage data and enable efficient querying of this data.
Additionally, our initial choice of the streaming processor, ksqlDB, revealed limitations when scaling for small to medium-sized producers.
The constraints stemmed from ksqlDB's limited capacity to run persisted queries per instance and its lack of support for clusterization.

## Considered Options

1. **ClickHouse**: A columnar database optimized for real-time analytics.
2. **PostgreSQL**: A popular open-source relational database system.
3. **Timescale**: An extension for PostgreSQL optimized for time-series data.
4. **VictoriaMetrics**: A time-series database and monitoring solution.

## Decision Outcome

After careful consideration, we have selected ClickHouse in conjunction with Kafka Connect to address our requirements.

- **ClickHouse**: Chosen as the primary database to store historical usage data.
- **Kafka Connect**: Employed to facilitate data movement from Kafka topics to ClickHouse tables.
- **Pre-aggregation**: Utilization of ClickHouse's `AggregatingMergeTree` and `MaterializedView` features to pre-aggregate usage data into tumbling windows.

To efficiently move data in batches, we will make use of the [ClickHouse Kafka Connect Sink](https://github.com/openmeterio/clickhouse-kafka-connect) plugin.

In our future efforts to alleviate the write load on ClickHouse, particularly for high-volume producers,
we will explore the implementation of streaming aggregation for busy topics.
This exploration will involve leveraging a more scalable stream processing technology like [Arroyo](https://www.arroyo.dev).

### Consequences

- **Pros**: ClickHouse is purpose-built for real-time analytics, offering robust performance.
- **Pros**: Kafka Connect provides a reliable and scalable data movement mechanism.
- **Cons**: Storing raw events in ClickHouse may lead to substantial data growth, requiring efficient data management strategies.
- **Cons**: The ClickHouse Kafka Connect Sink plugin is relatively new and may require ongoing development and support.
- **Cons**: Users on pre 1.0.0 releases need to migrate from ksqlDB
