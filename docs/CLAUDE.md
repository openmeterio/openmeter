# docs

<!-- archie:ai-start -->

> Documentation-only root split into ADRs (docs/decisions/) recording immutable architectural choices and operator-facing migration guides (docs/migration-guides/) for breaking changes between versions, plus standalone developer/operator guides (database-migration, event-ingestion, seeder, stripe-dev). Contains no Go source; changes here affect operator understanding and historical record, not compile-time behavior.

## Patterns

**ADR immutability + supersession** — Existing ADRs in docs/decisions/ record a past decision and must never be edited to change it; create a new sequentially numbered ADR to supersede. (`// 0005-clickhouse.md documents the ksqlDB→ClickHouse switch; a reversal needs 0006-*.md, never an edit to 0005`)
**One migration guide per breaking change** — Each file in docs/migration-guides/ covers exactly one breaking change, with date-prefixed filename, Before/After API examples, and a deprecation timeline table. (`// 2025-08-12-subject-customer-consolidation.md covers only the subject→customer consolidation`)
**Audit queries before destructive SQL** — Migration guides that include SQL show a SELECT audit query before any UPDATE/DELETE so operators can verify scope before executing. (`-- Audit first: SELECT id, key FROM subjects WHERE customer_id IS NULL; -- Then migrate: UPDATE subjects SET ...`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `decisions/0001-event-streaming-platform.md` | ADR: choice of Kafka as the event streaming platform. | Do not edit to reflect a different decision; create a new 000N-*.md ADR to supersede. |
| `decisions/0005-clickhouse.md` | ADR: ClickHouse replaces ksqlDB for analytics — authoritative source for why ClickHouse is the analytics store. | Any ClickHouse architectural change requires a new superseding ADR, not an edit here. |
| `database-migration.md` | Operator guide covering autoMigrate modes (ent, migration, false) and the Atlas workflow. | Keep in sync with actual atlas.hcl and Makefile targets when migration commands change. |
| `event-ingestion.md` | User-facing explanation of CloudEvents format, JSONPath meter configuration, and the 32-day deduplication window. | Any change to ingest API contract or deduplication window must be reflected here. |
| `stripe-dev.md` | Developer guide for working against a Stripe test account including webhook setup and psql commands. | Keep webhook URL patterns in sync with actual API routes; update psql column names if the Ent schema changes. |

## Anti-Patterns

- Editing existing ADR files to change the recorded decision — always create a new sequentially numbered ADR.
- Combining multiple unrelated breaking changes in a single migration guide — one breaking change per file.
- Writing a migration guide without audit SELECT queries before destructive UPDATE/DELETE statements.
- Using docs/decisions for operational runbooks or migration SQL — those belong in docs/migration-guides.
- Omitting the 'Considered Options' and 'Cons' sections when writing a new ADR — every decision must document its tradeoffs.

## Decisions

- **ADRs in docs/decisions/ are append-only historical records.** — Immutable records prevent revisionist history and give an auditable trail of why the current architecture exists.
- **SQL migration scripts embedded directly in migration guides rather than separate .sql files.** — Self-contained guides are easier for operators to follow without cross-referencing, and inline context explains each statement.

<!-- archie:ai-end -->
