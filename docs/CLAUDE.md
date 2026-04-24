# docs

<!-- archie:ai-start -->

> Documentation-only folder split into ADRs (docs/decisions/) recording immutable architectural choices and user-facing migration guides (docs/migration-guides/) for breaking changes. Neither subfolder contains Go source; changes here affect operator understanding, not compile-time behavior.

## Patterns

**ADR immutability** — Existing ADR files in docs/decisions/ record a past decision and must not be edited to change that decision; create a new sequentially numbered ADR instead. (`0005-clickhouse.md documents the switch from ksqlDB; a reversal would require 0006-*.md`)
**Migration guide per breaking change** — Each file in docs/migration-guides/ covers exactly one breaking change with SQL audit queries before destructive statements, Before/After API examples, and a deprecation timeline table. (`2025-08-12-subject-customer-consolidation.md covers the subject→customer consolidation with SQL scripts and a phased removal timeline`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `docs/decisions/0001-event-streaming-platform.md` | ADR: choice of Kafka as the event streaming platform | Do not edit to reflect a different decision; create 000N-*.md |
| `docs/decisions/0005-clickhouse.md` | ADR: ClickHouse replaces ksqlDB for analytics | This ADR is the authoritative source for why ClickHouse is the analytics store |
| `docs/database-migration.md` | Operator guide: autoMigrate modes (ent, migration, false) and Atlas workflow | Keep in sync with actual atlas.hcl and Makefile targets |
| `docs/event-ingestion.md` | User-facing explanation of CloudEvents format, JSONPath meter config, and deduplication semantics | Any change to ingest API contract must be reflected here |

## Anti-Patterns

- Editing existing ADR files to change the recorded decision — write a new numbered ADR
- Combining multiple unrelated breaking changes in a single migration guide file
- Omitting the 'Considered Options' section when writing a new ADR
- Using docs/decisions for operational runbooks or migration SQL — those belong in docs/migration-guides
- Writing a migration guide without audit SELECT queries before destructive UPDATE/DELETE statements

## Decisions

- **ADRs in docs/decisions/ are append-only historical records** — Immutable records prevent revisionist history and provide a clear audit trail of why the current architecture exists
- **SQL migration scripts embedded directly in docs/migration-guides/ rather than as separate .sql files** — Self-contained guides are easier for operators to follow without cross-referencing separate files

<!-- archie:ai-end -->
