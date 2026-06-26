# decisions

<!-- archie:ai-start -->

> Architecture Decision Records (ADRs) capturing the foundational data-plane technology choices behind OpenMeter's usage-metering pipeline: event streaming, event format, idempotency, partitioning, and historical storage. Documentation only — no code; the records explain why the system is built the way it is.

## Patterns

**Sequential numbered ADR filenames** — Each file is `NNNN-kebab-title.md` with a zero-padded 4-digit prefix that monotonically increases; new decisions take the next free number. (`0005-clickhouse.md follows 0004-partitioning.md`)
**Fixed MADR-style section structure** — Every ADR uses the headings `## Context and Problem Statement`, `## Considered Options`, `## Decision Outcome`, and a `### Consequences` subsection listing Pros:/Cons: bullets. (`0001-event-streaming-platform.md lists 3 considered options then a Decision Outcome of 'Kafka with ksqlDB' followed by Pros/Cons`)
**Outcome stated as bullet list of concrete choices** — The Decision Outcome leads with the chosen option name, then itemizes the specific components/parameters adopted. (`0003-idempotency.md: 'Deduplication in ksqlDB via windowed tables', 'Events are unique by ID + Source', '32 days default deduplication window'`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `0001-event-streaming-platform.md` | Records choice of Kafka for event streaming + ksqlDB for stream processing; notes interfaces are kept generic for swappability. | Partly superseded by 0005 (ClickHouse) which lists 'migrate from ksqlDB' as a consequence — ksqlDB is no longer the sole processing path. |
| `0002-event-format.md` | Records CloudEvents as the ingestion event format (ID + Source uniqueness, multi-language, batch support). | The ID+Source uniqueness contract here is the premise for the idempotency decision in 0003 — keep them consistent. |
| `0003-idempotency.md` | Records deduplication via ksqlDB windowed tables (32-day window) instead of Redis; events unique by ID + Source. | Depends on 0002's uniqueness definition; window length has Kafka storage/backup tradeoffs. |
| `0004-partitioning.md` | Records `subject` as the Kafka producer key, subject+group-by in ksqlDB meter keys, configurable partitions defaulting to 100. | Tables partitioned by PRIMARY KEY and cannot be repartitioned — co-partitioning constraints for joins follow from this. |
| `0005-clickhouse.md` | Records ClickHouse + Kafka Connect Sink for historical storage and AggregatingMergeTree/MaterializedView pre-aggregation; addresses ksqlDB scaling limits. | Implies a migration path off ksqlDB for pre-1.0.0 users; mentions future Arroyo-based streaming aggregation as not-yet-implemented direction. |

## Anti-Patterns

- Renumbering or reusing an existing ADR number instead of appending the next sequential file
- Editing a past decision's outcome in place rather than adding a new superseding ADR
- Adding implementation code or config here — these are prose decision records only

## Decisions

- **Keep streaming interfaces generic rather than hard-coupling to Kafka/ksqlDB** — 0001 states this explicitly so OpenMeter can adapt to alternative streaming platforms (and indeed ClickHouse later supplements ksqlDB).
- **Idempotency via ksqlDB windowed dedup instead of Redis** — 0003 chose strong consistency and no extra system over Redis's simpler-but-inconsistent options, accepting increased Kafka storage.

<!-- archie:ai-end -->
