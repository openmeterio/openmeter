# decisions

<!-- archie:ai-start -->

> Architecture Decision Records (ADRs) documenting why OpenMeter chose its core infrastructure stack (Kafka, CloudEvents, ClickHouse, partitioning, idempotency). These are immutable historical records — constraints they document are baked into the codebase and must not be reversed without a new numbered ADR.

## Patterns

**ADR sequential numbering** — Files are named 000N-<slug>.md with zero-padded sequence numbers. Each document follows: Context/Problem Statement → Considered Options → Decision Outcome → Consequences. (`0001-event-streaming-platform.md, 0005-clickhouse.md`)
**Decision recorded with tradeoffs** — Every ADR must document Pros AND Cons of the chosen option AND rejected alternatives. Omitting Cons makes the ADR incomplete. (`0003-idempotency.md lists Redis key-expiration and bloom-filter alternatives with explicit cons before explaining the ksqlDB windowed-table choice.`)
**Supersession by new ADR** — When a later ADR overrides an earlier one (e.g., ClickHouse replacing ksqlDB), the new ADR is the authority. The old ADR is not edited — it remains as historical context. (`0005-clickhouse.md supersedes parts of 0001 and 0003; ClickHouse is now the analytics store, not ksqlDB.`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `0001-event-streaming-platform.md` | Documents why Kafka was chosen as the event streaming backbone. The decision to 'keep interfaces generic' is why streaming.Connector is an abstraction in openmeter/streaming/connector.go. | The ksqlDB choice here is superseded by 0005-clickhouse.md — do not treat ksqlDB as the current processor. |
| `0002-event-format.md` | Explains why CloudEvents is the ingest wire format. All ingest code in openmeter/ingest/ and openmeter/sink/ works with CloudEvents structs. | CloudEvents are unique by ID + Source — this uniqueness contract underlies the deduplication logic in openmeter/dedupe/. |
| `0003-idempotency.md` | Establishes the 32-day deduplication window and ID+Source uniqueness key. These constants appear in openmeter/sink/ and openmeter/dedupe/. | The ksqlDB deduplication described here is superseded by the Redis/in-memory deduplicator in openmeter/dedupe/ — the 32-day window is the enduring constraint. |
| `0004-partitioning.md` | Explains why subject is the Kafka partition key and why the default partition count is 100. | Co-partitioning requirements constrain ksqlDB join operations — relevant if adding new Kafka topics. |
| `0005-clickhouse.md` | Documents the replacement of ksqlDB with ClickHouse + Kafka Connect. The AggregatingMergeTree + MaterializedView pre-aggregation strategy is why meter queries in openmeter/streaming/clickhouse/ use those table types. | This ADR supersedes parts of 0001 and 0003 — ClickHouse is now the analytics store, not ksqlDB. |

## Anti-Patterns

- Editing existing ADR files to change the recorded decision — create a new numbered ADR instead
- Skipping the 'Considered Options' section when writing a new ADR
- Using this folder for operational runbooks or migration guides — those belong in docs/migration-guides
- Omitting the Cons section for the chosen option — every decision must document its tradeoffs

## Decisions

- **CloudEvents as the universal event format** — Enables integration with cloud infrastructure, multi-language SDKs, and provides built-in uniqueness via ID+Source without a custom format.
- **ClickHouse replaces ksqlDB for analytics** — ksqlDB had scaling limitations for small-to-medium producers and lacked clusterization; ClickHouse with AggregatingMergeTree handles pre-aggregation more efficiently.

<!-- archie:ai-end -->
